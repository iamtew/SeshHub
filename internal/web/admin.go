package web

import (
	"net/http"

	"seshhub/internal/admin"
	"seshhub/internal/auth"
)

func (s *Server) adminHome(w http.ResponseWriter, r *http.Request) {
	u := UserFrom(r)
	if u == nil || u.Role != auth.RoleAdmin {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	st, err := admin.StatsFrom(s.db)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	s.render(w, r, "admin.html", map[string]any{"Title": "Admin", "Path": "/admin", "Stats": st})
}
