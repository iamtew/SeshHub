package web

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"seshhub/internal/auth"
	"seshhub/internal/skater"
)

func (s *Server) requireUser(w http.ResponseWriter, r *http.Request) *auth.User {
	u := UserFrom(r)
	if u == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return nil
	}
	return u
}

func (s *Server) accountUnlink(provider string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := s.requireUser(w, r)
		if u == nil {
			return
		}
		fresh, err := auth.GetUser(s.db, u.ID)
		if err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		switch provider {
		case "discord":
			if fresh.YouTubeChannelID == "" {
				http.Error(w, "connect YouTube first, or delete the account", http.StatusBadRequest)
				return
			}
			err = auth.UnlinkDiscord(s.db, fresh.ID)
		default:
			if fresh.DiscordID == "" {
				http.Error(w, "connect Discord first, or delete the account", http.StatusBadRequest)
				return
			}
			err = auth.UnlinkYouTube(s.db, fresh.ID)
		}
		if err != nil {
			http.Error(w, "unlink failed", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/account", http.StatusSeeOther)
	}
}

func (s *Server) accountDelete(w http.ResponseWriter, r *http.Request) {
	u := s.requireUser(w, r)
	if u == nil {
		return
	}
	if err := auth.DeleteUser(s.db, u.ID, ""); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: auth.CookieName, Path: "/", MaxAge: -1, HttpOnly: true})
	http.Redirect(w, r, "/", http.StatusFound)
}

func (s *Server) accountExport(w http.ResponseWriter, r *http.Request) {
	u := s.requireUser(w, r)
	if u == nil {
		return
	}
	fresh, err := auth.GetUser(s.db, u.ID)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	out := map[string]any{"account": fresh}
	if p, err := skater.Get(s.db, "user_id", fresh.ID); err == nil {
		out["profile"] = p
	} else if err != sql.ErrNoRows {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	if st, err := auth.AccessStatus(s.db, fresh.ID); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	} else if st != "" {
		out["access_request"] = map[string]string{"status": st}
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", `attachment; filename="seshhub-export.json"`)
	_ = json.NewEncoder(w).Encode(out)
}
