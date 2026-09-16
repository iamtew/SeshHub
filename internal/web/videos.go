package web

import (
	"database/sql"
	"html/template"
	"net/http"
	"strconv"

	"seshhub/internal/auth"
	"seshhub/internal/yt"
)

type feedItem struct {
	yt.Video
	Keep bool
}

func (s *Server) videos(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	s.renderVideos(w, r, filterFromRequest(s.db, r))
}

func (s *Server) videosFilter(w http.ResponseWriter, r *http.Request) {
	u := s.requireUser(w, r)
	if u == nil {
		return
	}
	_ = r.ParseForm()
	rows := yt.FormRules(r.Form["field"], r.Form["value"])
	switch r.FormValue("action") {
	case "clear":
		if err := yt.SetFilter(s.db, ""); err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/videos", http.StatusSeeOther)
	case "save":
		if err := yt.SetFilter(s.db, yt.EncodeRules(rows)); err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/videos", http.StatusSeeOther)
	case "add":
		if len(rows) < yt.MaxRules {
			rows = append(rows, yt.Rule{})
		}
		s.renderVideos(w, r, filterState{rules: yt.Normalize(rows), rows: rows, open: true})
	default:
		s.renderVideos(w, r, filterState{rules: yt.Normalize(rows), rows: yt.WithBlank(rows), test: true, open: true})
	}
}

type filterState struct {
	rules []yt.Rule
	rows  []yt.Rule
	test  bool
	open  bool
}

func filterFromRequest(db *sql.DB, r *http.Request) filterState {
	raw, _ := yt.GetFilter(db)
	saved := yt.ParseRules(raw)
	if UserFrom(r) != nil && r.FormValue("test") == "1" {
		fields, values := r.Form["field"], r.Form["value"]
		if len(fields) > 0 || len(values) > 0 {
			rows := yt.FormRules(fields, values)
			return filterState{rules: yt.Normalize(rows), rows: yt.WithBlank(rows), test: true, open: true}
		}
		return filterState{rules: saved, rows: yt.WithBlank(saved), test: true, open: true}
	}
	return filterState{rules: saved, rows: yt.WithBlank(saved), open: len(saved) > 0}
}

func (s *Server) renderVideos(w http.ResponseWriter, r *http.Request, st filterState) {
	list, err := yt.ListPublic(s.db)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	shown := list
	if !st.test {
		// ponytail: scan all public rows then filter; SQL WHERE if the cache ever gets large
		shown = yt.Filter(list, st.rules)
	}
	n, _ := strconv.Atoi(r.FormValue("n"))
	p, _ := strconv.Atoi(r.FormValue("p"))
	per, page, offset, from, to := videoPage(n, p, len(shown))
	pageItems := shown
	if len(shown) == 0 {
		pageItems = nil
	} else {
		end := offset + per
		if end > len(shown) {
			end = len(shown)
		}
		pageItems = shown[offset:end]
	}
	items := make([]feedItem, len(pageItems))
	for i, v := range pageItems {
		items[i] = feedItem{Video: v, Keep: yt.Match(v, st.rules)}
	}
	pagerQ := ""
	if st.test {
		pagerQ = yt.Query(st.rules)
	}
	s.render(w, r, "videos_list.html", map[string]any{
		"Title": "Videos", "Path": "/videos", "Videos": items,
		"Per": per, "Page": page, "Total": len(shown), "From": from, "To": to,
		"Prev": page - 1, "Next": page + 1, "HasPrev": page > 1, "HasNext": to < len(shown),
		"FilterRows": st.rows, "FilterOpen": st.open, "FilterOn": len(st.rules) > 0 && !st.test,
		"Test": st.test, "PagerQ": template.URL(pagerQ),
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
