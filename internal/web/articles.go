package web

import (
	"database/sql"
	"html/template"
	"net/http"
	"strings"

	"seshhub/internal/article"
	"seshhub/internal/auth"
	"seshhub/internal/skater"
)

func formArticle(r *http.Request, authorID string) article.Article {
	_ = r.ParseForm()
	return article.Article{
		ID:         r.FormValue("id"),
		Slug:       r.FormValue("slug"),
		Title:      r.FormValue("title"),
		Excerpt:    r.FormValue("excerpt"),
		ContentRaw: r.FormValue("content_raw"),
		ImageURL:   r.FormValue("featured_image_url"),
		AuthorID:   authorID,
		Status:     r.FormValue("status"),
		Tags:       r.FormValue("tags"),
	}
}

func (s *Server) news(w http.ResponseWriter, r *http.Request) {
	list, err := article.ListPublished(s.db)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.render(w, r, "articles_list.html", map[string]any{"Title": "News", "Path": "/news", "Articles": list})
}

func (s *Server) articleDetail(w http.ResponseWriter, r *http.Request) {
	a, err := article.Get(s.db, "slug", r.PathValue("slug"))
	if err == sql.ErrNoRows {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if a.Status != "published" {
		u := UserFrom(r)
		if u == nil || !article.CanEdit(u.Role, u.ID, a) {
			http.NotFound(w, r)
			return
		}
	}
	s.render(w, r, "article_detail.html", map[string]any{
		"Title": a.Title, "Path": "/news", "A": a, "HTML": template.HTML(a.ContentHTML),
		"Tags": skater.Lines(strings.ReplaceAll(a.Tags, ",", "\n")), "Minutes": article.ReadingMinutes(a.ContentRaw),
	})
}

func (s *Server) adminArticles(w http.ResponseWriter, r *http.Request) {
	u := UserFrom(r)
	if u == nil || u.Role != auth.RoleAdmin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	list, err := article.ListAll(s.db)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.render(w, r, "admin_articles.html", map[string]any{"Title": "Articles", "Path": "/admin/articles", "Articles": list})
}

func (s *Server) dashboardArticles(w http.ResponseWriter, r *http.Request) {
	u := UserFrom(r)
	if u == nil || u.Role != auth.RoleSkater {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	list, err := article.ListByAuthor(s.db, u.ID)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.render(w, r, "admin_articles.html", map[string]any{"Title": "Drafts", "Path": "/dashboard/articles", "Articles": list, "Skater": true})
}

func (s *Server) articleCreate(admin bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := UserFrom(r)
		if u == nil || (admin && u.Role != auth.RoleAdmin) || (!admin && u.Role != auth.RoleSkater) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if r.Method != http.MethodPost {
			s.render(w, r, "article_form.html", map[string]any{
				"Title": "New article", "Path": pathFor(admin), "A": article.Article{Status: "draft"}, "Action": pathFor(admin), "Admin": admin,
			})
			return
		}
		a := formArticle(r, u.ID)
		a.ID = ""
		saved, err := article.Save(s.db, a, admin)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Redirect(w, r, pathFor(admin)+"/"+saved.ID, http.StatusSeeOther)
	}
}

func (s *Server) articleEdit(admin bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := UserFrom(r)
		if u == nil {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		a, err := article.Get(s.db, "id", r.PathValue("id"))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if admin && u.Role != auth.RoleAdmin {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if !admin && !article.CanEdit(u.Role, u.ID, a) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if r.Method == http.MethodPost {
			np := formArticle(r, a.AuthorID)
			np.ID = a.ID
			if _, err := article.Save(s.db, np, admin); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			http.Redirect(w, r, pathFor(admin), http.StatusSeeOther)
			return
		}
		s.render(w, r, "article_form.html", map[string]any{"Title": "Edit " + a.Title, "Path": pathFor(admin), "A": a, "Action": pathFor(admin) + "/" + a.ID, "Admin": admin})
	}
}

func (s *Server) articleDelete(w http.ResponseWriter, r *http.Request) {
	u := UserFrom(r)
	if u == nil || u.Role != auth.RoleAdmin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	_ = article.Delete(s.db, r.PathValue("id"))
	http.Redirect(w, r, "/admin/articles", http.StatusSeeOther)
}

func pathFor(admin bool) string {
	if admin {
		return "/admin/articles"
	}
	return "/dashboard/articles"
}
