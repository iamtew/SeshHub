package web

import (
	"database/sql"
	"net/http"
	"strconv"

	"seshhub/internal/auth"
	"seshhub/internal/yt"
)

func (s *Server) videos(w http.ResponseWriter, r *http.Request) {
	total, err := yt.CountPublic(s.db)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	n, _ := strconv.Atoi(r.URL.Query().Get("n"))
	p, _ := strconv.Atoi(r.URL.Query().Get("p"))
	per, page, offset, from, to := videoPage(n, p, total)
	list, err := yt.ListPublicPage(s.db, per, offset)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.render(w, r, "videos_list.html", map[string]any{
		"Title": "Videos", "Path": "/videos", "Videos": list,
		"Per": per, "Page": page, "Total": total, "From": from, "To": to,
		"Prev": page - 1, "Next": page + 1, "HasPrev": page > 1, "HasNext": to < total,
	})
}

func videoPage(n, p, total int) (per, page, offset, from, to int) {
	switch n {
	case 18, 27:
		per = n
	default:
		per = 9
	}
	if total <= 0 {
		return per, 1, 0, 0, 0
	}
	pages := (total + per - 1) / per
	page = p
	if page < 1 {
		page = 1
	}
	if page > pages {
		page = pages
	}
	offset = (page - 1) * per
	from = offset + 1
	to = offset + per
	if to > total {
		to = total
	}
	return
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
