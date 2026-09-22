package web

import (
	"net/http"
	"strings"
	"time"

	"seshhub/internal/auth"
	"seshhub/internal/bot"
	"seshhub/internal/forum"
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
		if !channelID(home) || !channelID(ch) {
			http.Error(w, "channel id must be digits", http.StatusBadRequest)
			return
		}
		err := s.bot.SaveSettings(bot.Settings{
			ChannelID:     ch,
			HomeChannelID: home,
			News:          r.FormValue("news") == "1",
			Forum:         r.FormValue("forum") == "1",
			Access:        r.FormValue("access") == "1",
			Mentions:      r.FormValue("mentions") == "1",
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
	listed, homeListed := false, false
	for _, c := range channels {
		if c.ID == settings.ChannelID {
			listed = true
		}
		if c.ID == settings.HomeChannelID {
			homeListed = true
		}
	}
	s.render(w, r, "admin_bot.html", map[string]any{
		"Title": "Discord bot", "Path": "/admin/bot",
		"Bot": botView(st, s.cfg.DiscordGuildID), "Settings": settings, "Activity": activityView(s.bot.Activity()),
		"Channels": channels, "ChannelListed": listed, "HomeListed": homeListed,
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
	Configured bool
	State      string
	Username   string
	Latency    string
	Guild      string
	LastError  string
	Since      string
}

func botView(st bot.Status, guildID string) botStatusView {
	state := st.State
	if state == "ready" {
		state = "connected"
	}
	v := botStatusView{
		Configured: st.Configured,
		State:      state,
		Username:   orDash(st.Username),
		Latency:    "—",
		Guild:      "—",
		LastError:  orDash(st.LastError),
		Since:      "—",
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

func (s *Server) announceMentions(postID, title, path string, skip []string) {
	ids, err := forum.MentionDiscords(s.db, postID, skip)
	if err != nil || len(ids) == 0 {
		return
	}
	var b strings.Builder
	for i, id := range ids {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString("<@" + id + ">")
	}
	b.WriteString(" mentioned in **" + title + "**\n")
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
