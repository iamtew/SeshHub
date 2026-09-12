package web

import (
	"net/http"

	"seshhub/internal/auth"
)

func (s *Server) requireAdmin(w http.ResponseWriter, r *http.Request) *auth.User {
	u := UserFrom(r)
	if u == nil || u.Role != auth.RoleAdmin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return nil
	}
	return u
}

func (s *Server) adminUsers(w http.ResponseWriter, r *http.Request) {
	if s.requireAdmin(w, r) == nil {
		return
	}
	list, err := auth.ListUsers(s.db)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.render(w, r, "admin_users.html", map[string]any{"Title": "Users", "Path": "/admin/users", "Users": list})
}

func (s *Server) adminUserMerge(w http.ResponseWriter, r *http.Request) {
	if s.requireAdmin(w, r) == nil {
		return
	}
	_ = r.ParseForm()
	if err := auth.MergeUsers(s.db, r.PathValue("id"), r.FormValue("from")); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

func (s *Server) adminUserUnlink(provider string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.requireAdmin(w, r) == nil {
			return
		}
		id := r.PathValue("id")
		var err error
		switch provider {
		case "discord":
			err = auth.UnlinkDiscord(s.db, id)
		default:
			err = auth.UnlinkYouTube(s.db, id)
		}
		if err != nil {
			http.Error(w, "unlink failed", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
	}
}

func (s *Server) adminUserDelete(w http.ResponseWriter, r *http.Request) {
	u := s.requireAdmin(w, r)
	if u == nil {
		return
	}
	if err := auth.DeleteUser(s.db, r.PathValue("id"), u.ID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}
