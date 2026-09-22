package bot

import (
	"database/sql"
	"log/slog"
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
