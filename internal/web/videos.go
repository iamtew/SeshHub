package web

import (
	"database/sql"
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

func (s *Server) accountPage(w http.ResponseWriter, r *http.Request) {
	if UserFrom(r) == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	s.render(w, r, "account.html", map[string]any{"Title": "Account", "Path": "/account"})
}

func userChannelVideos(db *sql.DB, userID string, limit int) []yt.Video {
	if userID == "" {
		return nil
	}
	u, err := auth.GetUser(db, userID)
	if err != nil || u.YouTubeChannelID == "" {
		return nil
	}
	list, err := yt.ListByChannel(db, u.YouTubeChannelID, limit)
	if err != nil {
		return nil
	}
	return list
}
