package web

import (
	"log/slog"
	"net/http"

	"seshhub/internal/auth"
	"seshhub/internal/yt"
)

func (s *Server) videos(w http.ResponseWriter, r *http.Request) {
	list, err := yt.ListPublic(s.db)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.render(w, r, "videos_list.html", map[string]any{"Title": "Videos", "Path": "/videos", "Videos": list})
}

func (s *Server) adminYouTube(w http.ResponseWriter, r *http.Request) {
	u := UserFrom(r)
	if u == nil || u.Role != auth.RoleAdmin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	logs, err := yt.RecentLogs(s.db, 8)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.render(w, r, "admin_youtube.html", map[string]any{
		"Title": "YouTube sync", "Path": "/admin/youtube", "Logs": logs, "Enabled": s.cfg.YouTubeSyncEnabled(),
	})
}

func (s *Server) adminYouTubeSync(w http.ResponseWriter, r *http.Request) {
	u := UserFrom(r)
	if u == nil || u.Role != auth.RoleAdmin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if !s.cfg.YouTubeSyncEnabled() {
		http.Error(w, "youtube sync not configured", http.StatusBadRequest)
		return
	}
	c := yt.Client{Key: s.cfg.YouTubeAPIKey, Channel: s.cfg.YouTubeChannelID}
	if _, err := c.Sync(r.Context(), s.db); err != nil {
		slog.Error("youtube sync", "err", err)
	}
	http.Redirect(w, r, "/admin/youtube", http.StatusSeeOther)
}
