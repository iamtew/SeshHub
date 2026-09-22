package bot

import (
	"database/sql"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

// ponytail: ring of 50, persist if the log needs to survive a restart.
const logCap = 50

type Settings struct {
	ChannelID string
	News      bool
	Forum     bool
	Access    bool
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

	mu      sync.Mutex
	state   string
	user    string
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
	dg.Identify.Intents = discordgo.IntentGuilds
	dg.AddHandler(b.onReady)
	dg.AddHandler(b.onDisconnect)
	b.sess = dg
	b.post = func(channelID, content string) error {
		_, err := dg.ChannelMessageSend(channelID, content)
		return err
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
	guild, err := b.sess.Guild(b.guildID)
	if err != nil {
		return nil, err
	}
	member, err := b.sess.UserGuildMember(b.guildID)
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

func (b *Bot) Settings() (Settings, error) {
	var s Settings
	var news, forum, access int
	err := b.db.QueryRow(`SELECT channel_id, news, forum, access FROM bot_settings WHERE id = 1`).
		Scan(&s.ChannelID, &news, &forum, &access)
	if err != nil {
		return s, err
	}
	s.News, s.Forum, s.Access = news != 0, forum != 0, access != 0
	return s, nil
}

func (b *Bot) SaveSettings(s Settings) error {
	s.ChannelID = strings.TrimSpace(s.ChannelID)
	_, err := b.db.Exec(`UPDATE bot_settings SET channel_id=?, news=?, forum=?, access=? WHERE id=1`,
		s.ChannelID, bit(s.News), bit(s.Forum), bit(s.Access))
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
