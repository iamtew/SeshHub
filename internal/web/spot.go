package web

import (
	"html/template"
	"net/http"

	"seshhub/internal/article"
	"seshhub/internal/skater"
	"seshhub/internal/spot"
)

func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	cfg, err := spot.Get(s.db)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	vals, _ := spot.Latest()
	before, after, hole := spot.SplitImages(cfg.ContentRaw)
	data := map[string]any{"Title": "Spot", "Path": "/"}
	var photos []skater.Photo
	if hole {
		if raw, ok := spot.LatestRaw(); ok {
			for _, u := range spot.Images(raw) {
				photos = append(photos, skater.Photo{URL: u})
			}
		}
	}
	if hole && len(photos) > 0 {
		data["HTMLBefore"] = template.HTML(article.Render(spot.Fill(before, vals)))
		data["HTMLAfter"] = template.HTML(article.Render(spot.Fill(after, vals)))
		data["Gallery"] = photos
		data["GalleryJSON"] = galleryJSON(photos)
		data["GalleryScript"] = "spot-gallery-json"
	} else {
		md := cfg.ContentRaw
		if hole {
			md = before + after
		}
		data["HTML"] = template.HTML(article.Render(spot.Fill(md, vals)))
	}
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
	items := []spot.Item{{Key: "spot_images", Placeholder: "{{spot_images}}", Val: "slideshow"}}
	vals, live := spot.Latest()
	if live {
		items = append(items, spot.Items(vals)...)
	}
	s.render(w, r, "admin_spot.html", map[string]any{
		"Title": "Spot", "Path": "/admin/spot", "Cfg": cfg, "Items": items, "Live": live, "Monaco": true, "Cancel": "/admin",
	})
}
