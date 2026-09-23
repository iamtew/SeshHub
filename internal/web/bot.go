package web

import (
	"net/http"
	"strings"
	"time"

	"seshhub/internal/auth"
	"seshhub/internal/bot"
	"seshhub/internal/forum"
	"seshhub/internal/twitch"
)

func (s *Server) adminBot(w http.ResponseWriter, r *http.Request) {
	u := UserFrom(r)
	if u == nil || u.Role != auth.RoleAdmin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if s.bot == nil {
		http.Error(w, "bot unavailable", http.StatusInternalServerError)
		return
	}
	if r.Method == http.MethodPost {
		home := strings.TrimSpace(r.FormValue("home_channel_id"))
		ch := strings.TrimSpace(r.FormValue("channel_id"))
		twch := strings.TrimSpace(r.FormValue("twitch_channel_id"))
		if !channelID(home) || !channelID(ch) || !channelID(twch) {
			http.Error(w, "channel id must be digits", http.StatusBadRequest)
			return
		}
		err := s.bot.SaveSettings(bot.Settings{
			ChannelID:       ch,
			HomeChannelID:   home,
			TwitchChannelID: twch,
			News:            r.FormValue("news") == "1",
			Forum:           r.FormValue("forum") == "1",
			Access:          r.FormValue("access") == "1",
			Mentions:        r.FormValue("mentions") == "1",
			Twitch:          r.FormValue("twitch") == "1",
			TwitchMsg:       r.FormValue("twitch_msg"),
		})
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin/bot", http.StatusSeeOther)
		return
	}
	st := s.bot.Status()
	settings, err := s.bot.Settings()
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	channels, chErr := s.bot.Channels()
	listed, homeListed, twitchListed := false, false, false
	for _, c := range channels {
		if c.ID == settings.ChannelID {
			listed = true
		}
		if c.ID == settings.HomeChannelID {
			homeListed = true
		}
		if c.ID == settings.TwitchChannelID {
			twitchListed = true
		}
	}
	events := activityView(s.bot.Activity())
	if strings.TrimSpace(settings.TwitchMsg) == "" {
		settings.TwitchMsg = twitch.DefaultMsg
	}
	s.render(w, r, "admin_bot.html", map[string]any{
		"Title": "Discord bot", "Path": "/admin/bot",
		"Bot": botView(st, s.cfg.DiscordGuildID, events), "Settings": settings, "Activity": events,
		"HomeName": homeName(settings.HomeChannelID, channels),
		"Channels": channels, "ChannelListed": listed, "HomeListed": homeListed, "TwitchListed": twitchListed,
		"ChannelNote": channelNote(st, s.cfg.DiscordGuildID, channels, chErr),
	})
}

func channelNote(st bot.Status, guildID string, channels []bot.Channel, err error) string {
	if err != nil {
		return err.Error()
	}
	if !st.Configured || len(channels) > 0 {
		return ""
	}
	if guildID == "" {
		return "Set DISCORD_GUILD_ID to load channels."
	}
	if st.State != "ready" {
		return "Connect the bot to load channels."
	}
	return "The bot cannot post in any text channel."
}

func channelID(s string) bool {
	if s == "" {
		return true
	}
	if len(s) < 5 || len(s) > 22 {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

type botStatusView struct {
	Configured  bool
	State       string
	Username    string
	SelfID      string
	Latency     string
	Guild       string
	LastError   string
	Since       string
	LastMention string
}

func homeName(id string, channels []bot.Channel) string {
	if id == "" {
		return "none — @mentions are ignored until you pick one"
	}
	for _, c := range channels {
		if c.ID == id {
			return c.Name
		}
	}
	return id
}

func botView(st bot.Status, guildID string, events []botEventView) botStatusView {
	state := st.State
	if state == "ready" {
		state = "connected"
	}
	v := botStatusView{
		Configured:  st.Configured,
		State:       state,
		Username:    orDash(st.Username),
		SelfID:      orDash(st.SelfID),
		Latency:     "—",
		Guild:       "—",
		LastError:   orDash(st.LastError),
		Since:       "—",
		LastMention: "—",
	}
	if state == "connected" {
		v.Latency = st.Latency.Round(time.Millisecond).String()
	}
	if !st.Since.IsZero() {
		v.Since = st.Since.Format("2006-01-02 15:04:05")
	}
	switch {
	case !st.Configured:
		v.Guild = "—"
	case guildID == "":
		v.Guild = "DISCORD_GUILD_ID is empty"
	case st.Since.IsZero() && state != "connected":
		v.Guild = guildID
	case st.GuildOK:
		v.Guild = "in " + guildID
	default:
		v.Guild = "not in " + guildID
	}
	for _, e := range events {
		if e.Kind == "gallery" {
			v.LastMention = strings.TrimSpace(e.When + " " + e.Text + " " + e.Err)
			break
		}
	}
	return v
}

type botEventView struct {
	When string
	Kind string
	Text string
	Err  string
}

func activityView(events []bot.Event) []botEventView {
	out := make([]botEventView, len(events))
	for i, e := range events {
		out[i] = botEventView{When: e.At.Format("2006-01-02 15:04:05"), Kind: e.Kind, Text: e.Text, Err: e.Err}
	}
	return out
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

func (s *Server) announce(kind, text string) {
	if s == nil || s.bot == nil {
		return
	}
	go s.bot.Announce(kind, text)
}

func (s *Server) announceMentions(author *auth.User, postID, title, path string, skip []string) {
	ids, err := forum.MentionDiscords(s.db, postID, skip)
	if err != nil || len(ids) == 0 || author == nil {
		return
	}
	who := "**" + display(author) + "**"
	if author.DiscordID != "" && channelID(author.DiscordID) {
		who = "<@" + author.DiscordID + ">"
	}
	var b strings.Builder
	b.WriteString("User " + who + " mentioned ")
	for i, id := range ids {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString("<@" + id + ">")
	}
	b.WriteString(" in forum thread: **" + title + "**\n")
	b.WriteString(s.publicURL(path))
	s.announce("mention", b.String())
}

func (s *Server) publicURL(path string) string {
	return strings.TrimRight(s.cfg.BaseURL, "/") + path
}

func display(u *auth.User) string {
	if u.DisplayName != "" {
		return u.DisplayName
	}
	return u.Username
}
