package web

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"seshhub/internal/auth"
	"seshhub/internal/forum"
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

func accountDeleteWord(u *auth.User) string {
	if u == nil {
		return ""
	}
	if s := strings.TrimSpace(u.Username); s != "" {
		return s
	}
	return strings.TrimSpace(u.DisplayName)
}

func (s *Server) accountDelete(w http.ResponseWriter, r *http.Request) {
	u := s.requireUser(w, r)
	if u == nil {
		return
	}
	want := accountDeleteWord(u)
	if want == "" || r.FormValue("confirm") != want {
		http.Error(w, "type the confirmation word to delete", http.StatusBadRequest)
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
	raw, err := auth.GetVideoFilter(s.db, fresh.ID)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	out["video_filter"] = raw
	if p, err := skater.Get(s.db, "user_id", fresh.ID); err == nil {
		out["profile"] = p
		if photos, err := skater.ListByProfile(s.db, p.ID); err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		} else if len(photos) > 0 {
			urls := make([]string, len(photos))
			for i, ph := range photos {
				urls[i] = ph.URL
			}
			out["gallery"] = urls
		}
		if slugs, err := skater.FormerSlugs(s.db, p.ID); err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		} else if len(slugs) > 0 {
			out["former_slugs"] = slugs
		}
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
	if posts, err := forum.Export(s.db, fresh.ID); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	} else if len(posts) > 0 {
		out["forum"] = posts
	}
	if ments, err := forum.ExportMentions(s.db, fresh.ID); err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	} else if len(ments) > 0 {
		out["forum_mentions"] = ments
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", `attachment; filename="seshhub-export.json"`)
	_ = json.NewEncoder(w).Encode(out)
}

func (s *Server) accountPage(w http.ResponseWriter, r *http.Request) {
	if UserFrom(r) == nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	s.render(w, r, "account.html", map[string]any{"Title": "Account", "Path": "/account"})
}
