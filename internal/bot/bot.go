package bot

import (
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"

	"seshhub/internal/auth"
	"seshhub/internal/skater"
)

const shrugEmoji = "🤷‍♂️"
const frameEmoji = "🖼"

// ponytail: ring of 50, persist if the log needs to survive a restart.
const logCap = 50

type Settings struct {
	ChannelID     string
	HomeChannelID string
	News          bool
	Forum         bool
	Access        bool
	Mentions      bool
}

type Channel struct {
	ID   string
	Name string
}

type Status struct {
	Configured bool
	State      string
	Username   string
	Latency    time.Duration
	GuildOK    bool
	LastError  string
	Since      time.Time
}

type Event struct {
	At   time.Time
	Kind string
	Text string
	Err  string
}

type Bot struct {
	db      *sql.DB
	token   string
	guildID string
	sess    *discordgo.Session
	post    func(channelID, content string) error
	get     func(rawURL string) (io.ReadCloser, error)
	history func(channelID, beforeID string, limit int) ([]*discordgo.Message, error)
	react   func(channelID, messageID, emoji string) error

	mu      sync.Mutex
	state   string
	user    string
	selfID  string
	since   time.Time
	lastErr string
	guildOK bool
	log     []Event
}

// Open starts the gateway when token is set. A bad token leaves the bot disconnected
// so the HTTP server can still serve /admin/bot.
func Open(db *sql.DB, token, guildID string) *Bot {
	token = strings.TrimPrefix(strings.TrimSpace(token), "Bot ")
	b := &Bot{db: db, token: token, guildID: guildID, state: "not configured"}
	if token == "" {
		return b
	}
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		b.state = "disconnected"
		b.lastErr = err.Error()
		slog.Error("discord bot", "err", err)
		return b
	}
	dg.Identify.Intents = discordgo.IntentGuilds | discordgo.IntentGuildMessages | discordgo.IntentMessageContent
	dg.AddHandler(b.onReady)
	dg.AddHandler(b.onDisconnect)
	dg.AddHandler(b.onMessage)
	b.sess = dg
	b.post = func(channelID, content string) error {
		_, err := dg.ChannelMessageSend(channelID, content)
		return err
	}
	b.get = discordGet
	b.history = func(channelID, beforeID string, limit int) ([]*discordgo.Message, error) {
		return dg.ChannelMessages(channelID, limit, beforeID, "", "")
	}
	b.react = func(channelID, messageID, emoji string) error {
		return dg.MessageReactionAdd(channelID, messageID, emoji)
	}
	b.state = "connecting"
	if err := dg.Open(); err != nil {
		b.state = "disconnected"
		b.lastErr = err.Error()
		b.push("disconnect", "", err.Error())
		slog.Error("discord bot", "err", err)
	}
	return b
}

func (b *Bot) Close() {
	if b == nil || b.sess == nil {
		return
	}
	_ = b.sess.Close()
}

func (b *Bot) onReady(_ *discordgo.Session, r *discordgo.Ready) {
	name := ""
	if r.User != nil {
		name = r.User.Username
	}
	ok := false
	for _, g := range r.Guilds {
		if b.guildID != "" && g.ID == b.guildID {
			ok = true
		}
	}
	b.mu.Lock()
	b.state = "ready"
	b.user = name
	if r.User != nil {
		b.selfID = r.User.ID
	}
	if b.since.IsZero() {
		b.since = time.Now()
	}
	b.lastErr = ""
	b.guildOK = ok
	b.mu.Unlock()
	b.push("connect", name, "")
}

func (b *Bot) onDisconnect(_ *discordgo.Session, _ *discordgo.Disconnect) {
	b.mu.Lock()
	if b.state == "not configured" {
		b.mu.Unlock()
		return
	}
	b.state = "disconnected"
	b.mu.Unlock()
	b.push("disconnect", "", "")
}

// Channels are text and announcement channels in the guild where the bot can view and send.
func (b *Bot) Channels() ([]Channel, error) {
	if b == nil || b.sess == nil || b.guildID == "" {
		return nil, nil
	}
	me, err := b.sess.User("@me")
	if err != nil {
		return nil, err
	}
	guild, err := b.sess.Guild(b.guildID)
	if err != nil {
		return nil, err
	}
	member, err := b.sess.GuildMember(b.guildID, me.ID)
	if err != nil {
		return nil, err
	}
	raw, err := b.sess.GuildChannels(b.guildID)
	if err != nil {
		return nil, err
	}
	userID := ""
	if member.User != nil {
		userID = member.User.ID
	}
	return postable(guild, raw, userID, member.Roles), nil
}

func postable(guild *discordgo.Guild, raw []*discordgo.Channel, userID string, roles []string) []Channel {
	cats := map[string]*discordgo.Channel{}
	var posts []*discordgo.Channel
	for _, c := range raw {
		if c.Type == discordgo.ChannelTypeGuildCategory {
			cats[c.ID] = c
			continue
		}
		if c.Type != discordgo.ChannelTypeGuildText && c.Type != discordgo.ChannelTypeGuildNews {
			continue
		}
		if !canPost(guild, c, userID, roles) {
			continue
		}
		posts = append(posts, c)
	}
	sort.Slice(posts, func(i, j int) bool {
		pi, pj := catPos(cats, posts[i]), catPos(cats, posts[j])
		if pi != pj {
			return pi < pj
		}
		if posts[i].Position != posts[j].Position {
			return posts[i].Position < posts[j].Position
		}
		return posts[i].Name < posts[j].Name
	})
	out := make([]Channel, len(posts))
	for i, c := range posts {
		name := "#" + c.Name
		if parent := cats[c.ParentID]; parent != nil && parent.Name != "" {
			name = parent.Name + " / " + name
		}
		out[i] = Channel{ID: c.ID, Name: name}
	}
	return out
}

func catPos(cats map[string]*discordgo.Channel, c *discordgo.Channel) int {
	if p := cats[c.ParentID]; p != nil {
		return p.Position
	}
	return -1
}

func canPost(guild *discordgo.Guild, ch *discordgo.Channel, userID string, roles []string) bool {
	need := int64(discordgo.PermissionViewChannel | discordgo.PermissionSendMessages)
	return perms(guild, ch, userID, roles)&need == need
}

// Discord overwrite order: @everyone, roles, channel @everyone, role overwrites, member overwrite.
func perms(guild *discordgo.Guild, ch *discordgo.Channel, userID string, roles []string) int64 {
	if userID != "" && userID == guild.OwnerID {
		return discordgo.PermissionAll
	}
	var p int64
	for _, role := range guild.Roles {
		if role.ID == guild.ID {
			p |= role.Permissions
			break
		}
	}
	for _, role := range guild.Roles {
		for _, id := range roles {
			if role.ID == id {
				p |= role.Permissions
				break
			}
		}
	}
	if p&discordgo.PermissionAdministrator != 0 {
		p |= discordgo.PermissionAll
	}
	for _, ow := range ch.PermissionOverwrites {
		if ow.ID == guild.ID {
			p &^= ow.Deny
			p |= ow.Allow
			break
		}
	}
	var deny, allow int64
	for _, ow := range ch.PermissionOverwrites {
		if ow.Type != discordgo.PermissionOverwriteTypeRole {
			continue
		}
		for _, id := range roles {
			if id == ow.ID {
				deny |= ow.Deny
				allow |= ow.Allow
				break
			}
		}
	}
	p &^= deny
	p |= allow
	for _, ow := range ch.PermissionOverwrites {
		if ow.Type == discordgo.PermissionOverwriteTypeMember && ow.ID == userID {
			p &^= ow.Deny
			p |= ow.Allow
			break
		}
	}
	if p&discordgo.PermissionAdministrator != 0 {
		p |= discordgo.PermissionAllChannel
	}
	return p
}

func (b *Bot) onMessage(_ *discordgo.Session, m *discordgo.MessageCreate) {
	if m == nil || m.Message == nil || m.GuildID != b.guildID {
		return
	}
	s, err := b.Settings()
	if err != nil || s.HomeChannelID == "" {
		return
	}
	b.mu.Lock()
	self := b.selfID
	b.mu.Unlock()
	if !hears(s.HomeChannelID, self, m.Message) {
		return
	}
	if !mentioned(self, m.Message) {
		return
	}
	b.ingestGallery(m.Message)
}

func (b *Bot) ingestGallery(m *discordgo.Message) {
	if b == nil || m == nil || m.Author == nil {
		return
	}
	u, err := auth.GetByDiscordID(b.db, m.Author.ID)
	if err != nil || !auth.HasPublicRoster(u.Role) {
		return
	}
	name := strings.TrimSpace(u.Username)
	if name == "" {
		name = strings.TrimSpace(u.DisplayName)
	}
	p, err := skater.Get(b.db, "user_id", u.ID)
	if err == sql.ErrNoRows {
		p, err = skater.EnsureForUser(b.db, u.ID, name)
	}
	if err != nil {
		b.fail("gallery", err)
		return
	}
	src := m
	urls := jpegPNGURLs(src)
	if len(urls) == 0 {
		if b.history == nil {
			return
		}
		hist, err := b.history(m.ChannelID, m.ID, 50)
		if err != nil {
			b.fail("gallery", err)
			return
		}
		src = nearestPhoto(lastByAuthor(hist, m.Author.ID, 3))
		if src == nil {
			if b.react != nil {
				_ = b.react(m.ChannelID, m.ID, shrugEmoji)
			}
			return
		}
		urls = jpegPNGURLs(src)
	}
	n := 0
	for _, rawURL := range urls {
		body, err := b.fetch(rawURL)
		if err != nil {
			b.fail("gallery", err)
			continue
		}
		_, err = skater.AddGalleryDisplace(b.db, p.ID, body)
		_ = body.Close()
		if err != nil {
			b.fail("gallery", err)
			continue
		}
		n++
	}
	if n > 0 {
		if b.react != nil {
			_ = b.react(m.ChannelID, src.ID, frameEmoji)
		}
		b.push("gallery", strconv.Itoa(n)+" photos", "")
	}
}

func (b *Bot) fetch(rawURL string) (io.ReadCloser, error) {
	if b.get == nil {
		return nil, fmt.Errorf("no get")
	}
	rc, err := b.get(rawURL)
	if err != nil {
		return nil, err
	}
	return &limitedClose{Reader: io.LimitReader(rc, int64(skater.GalleryBytes)+1), c: rc}, nil
}

type limitedClose struct {
	io.Reader
	c io.Closer
}

func (l *limitedClose) Close() error { return l.c.Close() }

func (b *Bot) fail(kind string, err error) {
	if err == nil {
		return
	}
	b.mu.Lock()
	b.lastErr = err.Error()
	b.mu.Unlock()
	b.push(kind, "", err.Error())
}

func mentioned(selfID string, m *discordgo.Message) bool {
	if selfID == "" || m == nil {
		return false
	}
	for _, u := range m.Mentions {
		if u != nil && u.ID == selfID {
			return true
		}
	}
	return false
}

func lastByAuthor(msgs []*discordgo.Message, authorID string, n int) []*discordgo.Message {
	var out []*discordgo.Message
	for _, m := range msgs {
		if m == nil || m.Author == nil || m.Author.ID != authorID || m.Author.Bot {
			continue
		}
		out = append(out, m)
		if len(out) == n {
			break
		}
	}
	return out
}

func nearestPhoto(msgs []*discordgo.Message) *discordgo.Message {
	for _, m := range msgs {
		if len(jpegPNGURLs(m)) > 0 {
			return m
		}
	}
	return nil
}

func jpegPNGURLs(m *discordgo.Message) []string {
	if m == nil {
		return nil
	}
	var out []string
	for _, a := range m.Attachments {
		if a == nil || !isJPEGPNG(a) || !discordCDN(a.URL) {
			continue
		}
		out = append(out, a.URL)
	}
	return out
}

func isJPEGPNG(a *discordgo.MessageAttachment) bool {
	t := strings.ToLower(a.ContentType)
	if i := strings.IndexByte(t, ';'); i >= 0 {
		t = t[:i]
	}
	t = strings.TrimSpace(t)
	if t == "image/jpeg" || t == "image/jpg" || t == "image/png" {
		return true
	}
	n := strings.ToLower(a.Filename)
	return strings.HasSuffix(n, ".jpg") || strings.HasSuffix(n, ".jpeg") || strings.HasSuffix(n, ".png")
}

func discordHost(h string) bool {
	h = strings.ToLower(h)
	return h == "cdn.discordapp.com" || h == "media.discordapp.net"
}

func discordCDN(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && u.Scheme == "https" && discordHost(u.Hostname())
}

var cdnClient = &http.Client{
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if req.URL == nil || !discordHost(req.URL.Hostname()) {
			return http.ErrUseLastResponse
		}
		return nil
	},
}

func discordGet(raw string) (io.ReadCloser, error) {
	if !discordCDN(raw) {
		return nil, fmt.Errorf("not discord cdn")
	}
	resp, err := cdnClient.Get(raw)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("cdn %d", resp.StatusCode)
	}
	return resp.Body, nil
}

// hears is true in the home channel, and elsewhere only when the bot is @mentioned or replied to.
func hears(homeID, selfID string, m *discordgo.Message) bool {
	if m == nil || m.Author == nil || m.Author.Bot || homeID == "" {
		return false
	}
	if m.ChannelID == homeID {
		return true
	}
	if selfID == "" {
		return false
	}
	for _, u := range m.Mentions {
		if u != nil && u.ID == selfID {
			return true
		}
	}
	return m.ReferencedMessage != nil && m.ReferencedMessage.Author != nil && m.ReferencedMessage.Author.ID == selfID
}

func (b *Bot) Settings() (Settings, error) {
	var s Settings
	var news, forum, access, mentions int
	err := b.db.QueryRow(`SELECT channel_id, IFNULL(home_channel_id,''), news, forum, access, mentions FROM bot_settings WHERE id = 1`).
		Scan(&s.ChannelID, &s.HomeChannelID, &news, &forum, &access, &mentions)
	if err != nil {
		return s, err
	}
	s.News, s.Forum, s.Access, s.Mentions = news != 0, forum != 0, access != 0, mentions != 0
	return s, nil
}

func (b *Bot) SaveSettings(s Settings) error {
	s.ChannelID = strings.TrimSpace(s.ChannelID)
	s.HomeChannelID = strings.TrimSpace(s.HomeChannelID)
	_, err := b.db.Exec(`UPDATE bot_settings SET channel_id=?, home_channel_id=?, news=?, forum=?, access=?, mentions=? WHERE id=1`,
		s.ChannelID, s.HomeChannelID, bit(s.News), bit(s.Forum), bit(s.Access), bit(s.Mentions))
	return err
}

func bit(on bool) int {
	if on {
		return 1
	}
	return 0
}

func enabled(s Settings, kind string) bool {
	switch kind {
	case "news":
		return s.News
	case "forum":
		return s.Forum
	case "access":
		return s.Access
	case "mention":
		return s.Mentions
	default:
		return false
	}
}

// Announce posts text when that kind is on and a channel is set. Otherwise it does nothing.
func (b *Bot) Announce(kind, text string) {
	if b == nil || b.post == nil {
		return
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	s, err := b.Settings()
	if err != nil {
		b.mu.Lock()
		b.lastErr = err.Error()
		b.mu.Unlock()
		b.push(kind, text, err.Error())
		return
	}
	if strings.TrimSpace(s.ChannelID) == "" || !enabled(s, kind) {
		return
	}
	if err := b.post(s.ChannelID, text); err != nil {
		b.mu.Lock()
		b.lastErr = err.Error()
		b.mu.Unlock()
		b.push(kind, text, err.Error())
		slog.Error("discord announce", "kind", kind, "err", err)
		return
	}
	b.push(kind, text, "")
}

func (b *Bot) Status() Status {
	if b == nil {
		return Status{State: "not configured"}
	}
	b.mu.Lock()
	st := Status{
		Configured: b.token != "",
		State:      b.state,
		Username:   b.user,
		GuildOK:    b.guildOK,
		LastError:  b.lastErr,
		Since:      b.since,
	}
	sess := b.sess
	b.mu.Unlock()
	if sess != nil && st.State == "ready" {
		st.Latency = sess.HeartbeatLatency()
	}
	return st
}

func (b *Bot) Activity() []Event {
	if b == nil {
		return nil
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]Event, len(b.log))
	for i := range b.log {
		out[len(b.log)-1-i] = b.log[i]
	}
	return out
}

func (b *Bot) push(kind, text, err string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.log = append(b.log, Event{At: time.Now(), Kind: kind, Text: text, Err: err})
	if len(b.log) > logCap {
		b.log = append([]Event(nil), b.log[len(b.log)-logCap:]...)
	}
}
