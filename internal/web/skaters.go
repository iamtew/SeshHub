package web

import (
	"database/sql"
	"net/http"

	"seshhub/internal/auth"
	"seshhub/internal/skater"
	"seshhub/internal/yt"
)

func formProfile(r *http.Request, existing skater.Profile) skater.Profile {
	_ = r.ParseForm()
	existing.RealName = r.FormValue("display_name")
	existing.Bio = r.FormValue("bio")
	existing.Stance = r.FormValue("stance")
	existing.Location = r.FormValue("location")
	if _, ok := r.PostForm["featured_video_id"]; ok {
		existing.FeaturedVideoID = r.FormValue("featured_video_id")
	}
	return existing
}

func skaterView(p skater.Profile) map[string]any {
	return map[string]any{
		"P": p, "Sponsors": skater.Lines(p.Sponsors), "Tricks": skater.Lines(p.SignatureTricks),
		"Social": skater.Lines(p.SocialLinks),
	}
}

func (s *Server) team(w http.ResponseWriter, r *http.Request) {
	list, err := skater.ListTeam(s.db)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.render(w, r, "skaters_list.html", map[string]any{"Title": "FS Team", "Path": "/team", "Skaters": list})
}

func (s *Server) skatersAlias(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/team", http.StatusFound)
}

func (s *Server) skaterDetail(w http.ResponseWriter, r *http.Request) {
	p, err := skater.Get(s.db, "slug", r.PathValue("slug"))
	if err == sql.ErrNoRows || (err == nil && p.UserID == "") {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	u := UserFrom(r)
	name := p.PublicName()
	data := skaterView(p)
	data["Title"] = name
	data["Path"] = "/team"
	data["OGTitle"] = name
	data["OGDesc"] = p.Bio
	if p.AvatarURL != "" {
		data["OGImage"] = p.AvatarURL
	}
	if v, err := yt.Get(s.db, p.FeaturedVideoID); err == nil {
		data["Featured"] = v
	}
	clips := userChannelVideos(s.db, p.UserID, 6)
	if feat, ok := data["Featured"].(yt.Video); ok {
		var rest []yt.Video
		for _, c := range clips {
			if c.ID != feat.ID {
				rest = append(rest, c)
			}
		}
		data["Clips"] = rest
	} else {
		data["Clips"] = clips
	}
	if u != nil {
		data["CanEdit"] = skater.CanEdit(u.Role, u.ID, p.UserID)
	}
	s.render(w, r, "skater_detail.html", data)
}

func (s *Server) adminSkaters(w http.ResponseWriter, r *http.Request) {
	u := UserFrom(r)
	if u == nil || u.Role != auth.RoleAdmin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	list, err := skater.List(s.db)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.render(w, r, "admin_skaters.html", map[string]any{"Title": "FS Team", "Path": "/admin/skaters", "Skaters": list})
}

func (s *Server) adminSkaterEdit(w http.ResponseWriter, r *http.Request) {
	u := UserFrom(r)
	if u == nil || u.Role != auth.RoleAdmin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin/skaters", http.StatusSeeOther)
		return
	}
	p, err := skater.Get(s.db, "id", r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if st := r.FormValue("status"); st != "" {
		p.Status = st
	}
	if _, err := skater.Save(s.db, p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/admin/skaters", http.StatusSeeOther)
}

func (s *Server) adminSkaterDelete(w http.ResponseWriter, r *http.Request) {
	u := UserFrom(r)
	if u == nil || u.Role != auth.RoleAdmin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	_ = skater.Delete(s.db, r.PathValue("id"))
	http.Redirect(w, r, "/admin/skaters", http.StatusSeeOther)
}

func (s *Server) dashboardProfile(w http.ResponseWriter, r *http.Request) {
	u := UserFrom(r)
	if u == nil || (u.Role != auth.RoleSkater && u.Role != auth.RoleAdmin) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	p, err := skater.Get(s.db, "user_id", u.ID)
	if err == sql.ErrNoRows {
		name := u.Username
		if name == "" {
			name = u.DisplayName
		}
		p, err = skater.EnsureForUser(s.db, u.ID, name)
	}
	if err == sql.ErrNoRows {
		http.Error(w, "no profile linked", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if r.Method == http.MethodPost {
		if _, err := skater.Save(s.db, formProfile(r, p)); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Redirect(w, r, "/team/"+p.Slug, http.StatusSeeOther)
		return
	}
	vids := userChannelVideos(s.db, u.ID, 50)
	s.render(w, r, "skater_form.html", map[string]any{"Title": "Skater profile", "Path": "/dashboard/profile", "P": p, "Action": "/dashboard/profile", "Videos": vids})
}
