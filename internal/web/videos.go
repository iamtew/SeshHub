package web

import (
	"database/sql"
	"net/http"
	"strconv"

	"seshhub/internal/auth"
	"seshhub/internal/yt"
)

func (s *Server) videos(w http.ResponseWriter, r *http.Request) {
	list, err := yt.ListPublic(s.db)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	by, err := yt.OwnerFilters(s.db)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	// ponytail: scan all public rows then apply each owner's rules; SQL WHERE if the cache ever gets large
	shown := yt.FilterOwned(list, by)
	n, _ := strconv.Atoi(r.URL.Query().Get("n"))
	p, _ := strconv.Atoi(r.URL.Query().Get("p"))
	data := map[string]any{"Title": "Videos", "Path": "/videos"}
	data["Videos"] = videoPager("/videos", n, p, shown, data)
	s.render(w, r, "videos_list.html", data)
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

func videoPager(base string, n, p int, shown []yt.Video, data map[string]any) []yt.Video {
	per, page, offset, from, to := videoPage(n, p, len(shown))
	var pageItems []yt.Video
	if len(shown) > 0 {
		end := offset + per
		if end > len(shown) {
			end = len(shown)
		}
		pageItems = shown[offset:end]
	}
	data["PagerBase"] = base
	data["Per"], data["Page"], data["Total"] = per, page, len(shown)
	data["From"], data["To"] = from, to
	data["Prev"], data["Next"] = page-1, page+1
	data["HasPrev"], data["HasNext"] = page > 1, to < len(shown)
	return pageItems
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
