package web

import (
	"html/template"
	"net/http"

	"seshhub/internal/article"
	"seshhub/internal/spot"
)

func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	cfg, err := spot.Get(s.db)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	vals, _ := spot.Latest()
	md := spot.Fill(cfg.ContentRaw, vals)
	data := map[string]any{"Title": "Spot", "Path": "/", "HTML": template.HTML(article.Render(md))}
	if u := UserFrom(r); u != nil && u.Host {
		data["EditHref"] = "/admin/spot"
	}
	s.render(w, r, "index.html", data)
}

func (s *Server) adminSpot(w http.ResponseWriter, r *http.Request) {
	if s.requireHost(w, r) == nil {
		return
	}
	cfg, err := spot.Get(s.db)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if r.Method == http.MethodPost {
		_ = r.ParseForm()
		cfg.ContentRaw = r.FormValue("content_raw")
		if err := spot.Save(s.db, cfg); err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, afterSave(r, "/admin/spot", "/admin"), http.StatusSeeOther)
		return
	}
	var items []spot.Item
	if vals, ok := spot.Latest(); ok {
		items = spot.Items(vals)
	}
	s.render(w, r, "admin_spot.html", map[string]any{
		"Title": "Spot", "Path": "/admin/spot", "Cfg": cfg, "Items": items, "Monaco": true, "Cancel": "/admin",
	})
}
