package web

import (
	"database/sql"
	"html/template"
	"net/http"

	"seshhub/internal/article"
	"seshhub/internal/auth"
	"seshhub/internal/page"
)

func (s *Server) customPage(w http.ResponseWriter, r *http.Request) {
	p, err := page.Get(s.db, "slug", r.PathValue("slug"))
	if err == sql.ErrNoRows {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if !p.Published {
		u := UserFrom(r)
		if u == nil || u.Role != auth.RoleAdmin {
			http.NotFound(w, r)
			return
		}
	}
	s.render(w, r, "custom_page.html", map[string]any{
		"Title": p.Title, "Path": "/" + p.Slug, "HTML": template.HTML(article.Render(p.ContentRaw)), "CSS": template.CSS(p.CSS),
	})
}

func (s *Server) adminPages(w http.ResponseWriter, r *http.Request) {
	u := UserFrom(r)
	if u == nil || u.Role != auth.RoleAdmin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	list, err := page.List(s.db)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.render(w, r, "admin_pages.html", map[string]any{"Title": "Pages", "Path": "/admin/pages", "Pages": list})
}

func formPage(r *http.Request) page.Page {
	_ = r.ParseForm()
	return page.Page{
		ID:          r.FormValue("id"),
		Slug:        r.FormValue("slug"),
		Title:       r.FormValue("title"),
		ContentRaw:  r.FormValue("content_raw"),
		CSS:         r.FormValue("custom_css"),
		Published:   r.FormValue("published") == "1",
	}
}

func (s *Server) adminPageCreate(w http.ResponseWriter, r *http.Request) {
	u := UserFrom(r)
	if u == nil || u.Role != auth.RoleAdmin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodPost {
		s.render(w, r, "page_form.html", map[string]any{"Title": "New page", "Path": "/admin/pages", "P": page.Page{}, "Action": "/admin/pages", "Monaco": true})
		return
	}
	p := formPage(r)
	p.ID = ""
	saved, err := page.Save(s.db, p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/admin/pages/"+saved.ID, http.StatusSeeOther)
}

func (s *Server) adminPageEdit(w http.ResponseWriter, r *http.Request) {
	u := UserFrom(r)
	if u == nil || u.Role != auth.RoleAdmin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	p, err := page.Get(s.db, "id", r.PathValue("id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if r.Method == http.MethodPost {
		np := formPage(r)
		np.ID = p.ID
		if _, err := page.Save(s.db, np); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Redirect(w, r, "/admin/pages", http.StatusSeeOther)
		return
	}
	s.render(w, r, "page_form.html", map[string]any{"Title": "Edit " + p.Title, "Path": "/admin/pages", "P": p, "Action": "/admin/pages/" + p.ID, "Monaco": true})
}

func (s *Server) adminPageDelete(w http.ResponseWriter, r *http.Request) {
	u := UserFrom(r)
	if u == nil || u.Role != auth.RoleAdmin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	_ = page.Delete(s.db, r.PathValue("id"))
	http.Redirect(w, r, "/admin/pages", http.StatusSeeOther)
}
