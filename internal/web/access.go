package web

import (
	"net/http"

	"seshhub/internal/auth"
)

func (s *Server) accessPage(w http.ResponseWriter, r *http.Request) {
	u := UserFrom(r)
	if u == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	st, err := auth.AccessStatus(s.db, u.ID)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.render(w, r, "access.html", map[string]any{"Title": "Access", "Path": "/access", "AccessStatus": st})
}

func (s *Server) accessRequest(w http.ResponseWriter, r *http.Request) {
	u := UserFrom(r)
	if u == nil || u.Role != auth.RolePending {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if err := auth.RequestAccess(s.db, u.ID); err != nil {
		http.Error(w, "request failed", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/access", http.StatusSeeOther)
}

func (s *Server) adminAccess(w http.ResponseWriter, r *http.Request) {
	u := UserFrom(r)
	if u == nil || u.Role != auth.RoleAdmin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	list, err := auth.PendingAccess(s.db)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.render(w, r, "admin_access.html", map[string]any{"Title": "Access queue", "Path": "/admin/access", "Requests": list})
}

func (s *Server) adminDecide(status string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := UserFrom(r)
		if u == nil || u.Role != auth.RoleAdmin {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if err := auth.DecideAccess(s.db, r.PathValue("id"), u.ID, status); err != nil {
			http.Error(w, "update failed", http.StatusBadRequest)
			return
		}
		http.Redirect(w, r, "/admin/access", http.StatusSeeOther)
	}
}
