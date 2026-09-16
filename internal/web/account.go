package web

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"seshhub/internal/auth"
	"seshhub/internal/skater"
	"seshhub/internal/yt"
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
	raw, err := auth.GetVideoFilter(s.db, fresh.ID)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	out["video_filter"] = raw
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

type clipTest struct {
	yt.Video
	Keep bool
}

func (s *Server) accountPage(w http.ResponseWriter, r *http.Request) {
	u := s.requireUser(w, r)
	if u == nil {
		return
	}
	s.renderAccount(w, r, u, nil, false)
}

func (s *Server) accountFilter(w http.ResponseWriter, r *http.Request) {
	u := s.requireUser(w, r)
	if u == nil {
		return
	}
	_ = r.ParseForm()
	rows := yt.FormRules(r.Form["field"], r.Form["value"])
	switch r.FormValue("action") {
	case "clear":
		if err := auth.SetVideoFilter(s.db, u.ID, ""); err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/account", http.StatusSeeOther)
	case "save":
		if err := auth.SetVideoFilter(s.db, u.ID, yt.EncodeRules(rows)); err != nil {
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/account", http.StatusSeeOther)
	case "add":
		if len(rows) < yt.MaxRules {
			rows = append(rows, yt.Rule{})
		}
		s.renderAccount(w, r, u, rows, false)
	default:
		s.renderAccount(w, r, u, yt.WithBlank(rows), true)
	}
}

func (s *Server) renderAccount(w http.ResponseWriter, r *http.Request, u *auth.User, rows []yt.Rule, test bool) {
	if rows == nil {
		raw, _ := auth.GetVideoFilter(s.db, u.ID)
		rows = yt.WithBlank(yt.ParseRules(raw))
	}
	rules := yt.Normalize(rows)
	data := map[string]any{"Title": "Account", "Path": "/account", "FilterRows": rows, "Test": test}
	if test {
		var items []clipTest
		for _, v := range userChannelVideos(s.db, u.ID, 50) {
			items = append(items, clipTest{Video: v, Keep: yt.Match(v, rules)})
		}
		data["TestClips"] = items
	}
	s.render(w, r, "account.html", data)
}
