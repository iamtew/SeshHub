package web

import (
	"net/http"
	"strings"
	"time"

	"seshhub/internal/auth"
	"seshhub/internal/bot"
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
		ch := strings.TrimSpace(r.FormValue("channel_id"))
		if !channelID(ch) {
			http.Error(w, "channel id must be digits", http.StatusBadRequest)
			return
		}
		err := s.bot.SaveSettings(bot.Settings{
			ChannelID: ch,
			News:      r.FormValue("news") == "1",
			Forum:     r.FormValue("forum") == "1",
			Access:    r.FormValue("access") == "1",
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
	s.render(w, r, "admin_bot.html", map[string]any{
		"Title": "Discord bot", "Path": "/admin/bot",
		"Bot": botView(st, s.cfg.DiscordGuildID), "Settings": settings, "Activity": activityView(s.bot.Activity()),
	})
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

func (s *Server) publicURL(path string) string {
	return strings.TrimRight(s.cfg.BaseURL, "/") + path
}

func display(u *auth.User) string {
	if u.DisplayName != "" {
		return u.DisplayName
	}
	return u.Username
}
