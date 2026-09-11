package web

import (
	"database/sql"
	"net/http"
	"strings"

	"seshhub/internal/auth"
	"seshhub/internal/skater"
)

func formProfile(r *http.Request) skater.Profile {
	_ = r.ParseForm()
	return skater.Profile{
		ID:              r.FormValue("id"),
		UserID:          strings.TrimSpace(r.FormValue("user_id")),
		Slug:            r.FormValue("slug"),
		SkaterName:      r.FormValue("skater_name"),
		RealName:        r.FormValue("real_name"),
		Bio:             r.FormValue("bio"),
		Stance:          r.FormValue("stance"),
		Status:          r.FormValue("status"),
		AvatarURL:       r.FormValue("avatar_url"),
		BannerURL:       r.FormValue("banner_url"),
		Location:        r.FormValue("location"),
		Sponsors:        r.FormValue("sponsors"),
		SocialLinks:     r.FormValue("social_links"),
		SignatureTricks: r.FormValue("signature_tricks"),
	}
}

func skaterView(p skater.Profile) map[string]any {
	return map[string]any{
		"P": p, "Sponsors": skater.Lines(p.Sponsors), "Tricks": skater.Lines(p.SignatureTricks),
		"Social": skater.Lines(p.SocialLinks),
	}
}

func (s *Server) team(w http.ResponseWriter, r *http.Request) {
	list, err := skater.List(s.db)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.render(w, r, "skaters_list.html", map[string]any{"Title": "Team", "Path": "/team", "Skaters": list})
}

func (s *Server) skatersAlias(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/team", http.StatusFound)
}

func (s *Server) skaterDetail(w http.ResponseWriter, r *http.Request) {
	p, err := skater.Get(s.db, "slug", r.PathValue("slug"))
	if err == sql.ErrNoRows {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	u := UserFrom(r)
	data := skaterView(p)
	data["Title"] = p.SkaterName
	data["Path"] = "/team"
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
	s.render(w, r, "admin_skaters.html", map[string]any{"Title": "Roster", "Path": "/admin/skaters", "Skaters": list, "P": skater.Profile{Stance: "regular", Status: "active"}})
}

func (s *Server) adminSkaterCreate(w http.ResponseWriter, r *http.Request) {
	u := UserFrom(r)
	if u == nil || u.Role != auth.RoleAdmin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	p := formProfile(r)
	p.ID = ""
	if _, err := skater.Save(s.db, p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/admin/skaters", http.StatusSeeOther)
}

func (s *Server) adminSkaterEdit(w http.ResponseWriter, r *http.Request) {
	u := UserFrom(r)
	if u == nil || u.Role != auth.RoleAdmin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	p, err := skater.Get(s.db, "id", r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if r.Method == http.MethodPost {
		np := formProfile(r)
		np.ID = p.ID
		if _, err := skater.Save(s.db, np); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Redirect(w, r, "/admin/skaters", http.StatusSeeOther)
		return
	}
	s.render(w, r, "skater_form.html", map[string]any{"Title": "Edit " + p.SkaterName, "Path": "/admin/skaters", "P": p, "Action": "/admin/skaters/" + p.ID, "Admin": true})
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
		http.Error(w, "no profile linked", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if r.Method == http.MethodPost {
		np := formProfile(r)
		np.ID = p.ID
		np.UserID = p.UserID
		if u.Role != auth.RoleAdmin {
			np.Slug = p.Slug
			np.Status = p.Status
		}
		if _, err := skater.Save(s.db, np); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Redirect(w, r, "/team/"+p.Slug, http.StatusSeeOther)
		return
	}
	s.render(w, r, "skater_form.html", map[string]any{"Title": "Your profile", "Path": "/dashboard/profile", "P": p, "Action": "/dashboard/profile", "Admin": u.Role == auth.RoleAdmin})
}
