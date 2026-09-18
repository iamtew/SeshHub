package web

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"seshhub/internal/auth"
	"seshhub/internal/skater"
	"seshhub/internal/yt"
)

type ctxKey int

const userKey ctxKey = 1

func UserFrom(r *http.Request) *auth.User {
	u, _ := r.Context().Value(userKey).(*auth.User)
	return u
}

func (s *Server) injectUser(r *http.Request) *http.Request {
	c, err := r.Cookie(auth.CookieName)
	if err != nil || c.Value == "" {
		return r
	}
	u, err := auth.UserByToken(s.db, c.Value)
	if err != nil {
		return r
	}
	s.attachProfile(&u)
	s.maybeSyncSkater(u.ID)
	return r.WithContext(context.WithValue(r.Context(), userKey, &u))
}

func (s *Server) attachProfile(u *auth.User) {
	u.ProviderAvatar = u.AvatarURL
	u.AvatarR1, u.AvatarR2, u.AvatarR3, u.AvatarR4 = 50, 50, 50, 50
	if !auth.HasPublicRoster(u.Role) {
		return
	}
	if u.Role != auth.RoleFriend && u.DiscordID == "" {
		return
	}
	name := u.Username
	if name == "" {
		name = u.DisplayName
	}
	p, err := skater.EnsureForUser(s.db, u.ID, name)
	if err != nil {
		slog.Error("skater profile", "err", err, "user", u.ID)
		return
	}
	if p.AvatarURL != "" {
		u.AvatarURL = p.AvatarURL
	}
	u.AvatarR1, u.AvatarR2, u.AvatarR3, u.AvatarR4, u.AvatarBorder, u.AvatarBorderBlur = p.AvatarR1, p.AvatarR2, p.AvatarR3, p.AvatarR4, p.AvatarBorder, p.AvatarBorderBlur
	u.AvatarBorderStyle, u.AvatarBorderColor = p.AvatarBorderStyle, p.AvatarBorderColor
	if p.Slug != "" {
		u.RosterURL = skater.RosterPath(u.Role) + "/" + p.Slug
	}
}

func (s *Server) maybeSyncSkater(userID string) {
	if !s.cfg.YouTubeEnabled() {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		if err := yt.MaybeSyncUser(ctx, s.db, userID, s.cfg.YouTubeClientID, s.cfg.YouTubeClientSecret); err != nil {
			slog.Error("skater youtube sync", "err", err, "user", userID)
		}
	}()
}

func (s *Server) loginPage(w http.ResponseWriter, r *http.Request) {
	next := safeNext(r.URL.Query().Get("next"))
	if UserFrom(r) != nil {
		loc := "/"
		if next != "" {
			loc = next
		}
		http.Redirect(w, r, loc, http.StatusSeeOther)
		return
	}
	s.render(w, r, "login.html", map[string]any{"Path": "/login", "AuthPage": true, "Next": next})
}

func (s *Server) startOAuth(provider string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if provider == "discord" && !s.cfg.DiscordEnabled() {
			http.Error(w, "discord oauth not configured", http.StatusNotFound)
			return
		}
		if provider == "youtube" && !s.cfg.YouTubeEnabled() {
			http.Error(w, "youtube oauth not configured", http.StatusNotFound)
			return
		}
		state := auth.RandomHex(16)
		verifier, challenge := auth.PKCE()
		http.SetCookie(w, s.oauthCookie(auth.FormatOAuthCookie(provider, state, verifier, safeNext(r.URL.Query().Get("next")))))
		var loc string
		switch provider {
		case "discord":
			loc = auth.DiscordAuthorizeURL(s.cfg.DiscordClientID, s.cfg.BaseURL+"/auth/discord/callback", state, challenge)
		default:
			loc = auth.YouTubeAuthorizeURL(s.cfg.YouTubeClientID, s.cfg.BaseURL+"/auth/youtube/callback", state, challenge)
		}
		http.Redirect(w, r, loc, http.StatusFound)
	}
}

func (s *Server) callbackDiscord(w http.ResponseWriter, r *http.Request) {
	verifier, err := s.checkOAuth(r, "discord")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	redirect := s.cfg.BaseURL + "/auth/discord/callback"
	token, err := auth.ExchangeDiscord(s.cfg.DiscordClientID, s.cfg.DiscordClientSecret, redirect, r.URL.Query().Get("code"), verifier)
	if err != nil {
		slog.Error("discord token", "err", err)
		http.Error(w, "discord login failed", http.StatusBadGateway)
		return
	}
	id, username, display, avatar, inGuild, roles, err := auth.FetchDiscord(token, s.cfg.DiscordGuildID)
	if err != nil {
		slog.Error("discord profile", "err", err)
		http.Error(w, "discord login failed", http.StatusBadGateway)
		return
	}
	role := auth.DiscordRole(id, inGuild, roles, s.cfg.SuperAdminIDs, s.cfg.DiscordHubAdminRoleID, s.cfg.DiscordSkaterRoleID, s.cfg.DiscordFriendsRoleID)
	host := auth.HasGuildRole(roles, s.cfg.DiscordHostsRoleID)
	ensure := func(userID string) {
		if !auth.HasGuildRole(roles, s.cfg.DiscordSkaterRoleID) && !auth.HasGuildRole(roles, s.cfg.DiscordFriendsRoleID) {
			return
		}
		if _, err := skater.EnsureForUser(s.db, userID, username); err != nil {
			slog.Error("skater profile", "err", err, "user", userID)
		}
	}
	if cur := UserFrom(r); cur != nil {
		if _, err := auth.LinkDiscord(s.db, cur.ID, id, username, display, avatar, role, host); err != nil {
			s.oauthLinkErr(w, err)
			return
		}
		ensure(cur.ID)
		s.clearOAuthCookie(w)
		http.Redirect(w, r, "/account", http.StatusFound)
		return
	}
	u, err := auth.UpsertDiscord(s.db, id, username, display, avatar, role, host)
	if err != nil {
		slog.Error("upsert discord", "err", err)
		http.Error(w, "login failed", http.StatusInternalServerError)
		return
	}
	ensure(u.ID)
	s.issueSession(w, r, u.ID)
}

func (s *Server) callbackYouTube(w http.ResponseWriter, r *http.Request) {
	verifier, err := s.checkOAuth(r, "youtube")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	redirect := s.cfg.BaseURL + "/auth/youtube/callback"
	token, refresh, err := auth.ExchangeGoogle(s.cfg.YouTubeClientID, s.cfg.YouTubeClientSecret, redirect, r.URL.Query().Get("code"), verifier)
	if err != nil {
		slog.Error("google token", "err", err)
		http.Error(w, "youtube login failed", http.StatusBadGateway)
		return
	}
	chID, title, avatar, err := auth.FetchYouTubeChannel(token)
	if err != nil {
		slog.Error("youtube channel", "err", err)
		http.Error(w, "youtube login failed", http.StatusBadGateway)
		return
	}
	if cur := UserFrom(r); cur != nil {
		u, err := auth.LinkYouTube(s.db, cur.ID, chID, title, avatar, refresh)
		if err != nil {
			s.oauthLinkErr(w, err)
			return
		}
		s.clearOAuthCookie(w)
		_ = yt.SyncWithToken(r.Context(), s.db, u.ID, token, chID)
		http.Redirect(w, r, "/account", http.StatusFound)
		return
	}
	u, err := auth.UpsertYouTube(s.db, chID, title, avatar, refresh)
	if err != nil {
		slog.Error("upsert youtube", "err", err)
		http.Error(w, "login failed", http.StatusInternalServerError)
		return
	}
	_ = yt.SyncWithToken(r.Context(), s.db, u.ID, token, chID)
	s.issueSession(w, r, u.ID)
}

func (s *Server) oauthLinkErr(w http.ResponseWriter, err error) {
	if err == auth.ErrTaken {
		http.Error(w, "that account is already linked to another user", http.StatusConflict)
		return
	}
	slog.Error("link oauth", "err", err)
	http.Error(w, "link failed", http.StatusInternalServerError)
}

func (s *Server) clearOAuthCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: auth.OAuthCookieName(), Path: "/", MaxAge: -1})
}

func (s *Server) checkOAuth(r *http.Request, provider string) (string, error) {
	c, err := r.Cookie(auth.OAuthCookieName())
	if err != nil {
		return "", fmt.Errorf("missing oauth cookie")
	}
	gotProvider, state, verifier, _, ok := auth.ParseOAuthCookie(c.Value)
	if !ok || gotProvider != provider || state != r.URL.Query().Get("state") || r.URL.Query().Get("code") == "" {
		return "", fmt.Errorf("invalid oauth state")
	}
	return verifier, nil
}

func (s *Server) issueSession(w http.ResponseWriter, r *http.Request, userID string) {
	token, err := auth.CreateSession(s.db, userID)
	if err != nil {
		slog.Error("session", "err", err)
		http.Error(w, "login failed", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     auth.CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.cfg.CookieSecure(),
		MaxAge:   30 * 24 * 60 * 60,
	})
	loc := "/"
	if c, err := r.Cookie(auth.OAuthCookieName()); err == nil {
		if _, _, _, next, ok := auth.ParseOAuthCookie(c.Value); ok {
			if n := safeNext(next); n != "" {
				loc = n
			}
		}
	}
	http.SetCookie(w, &http.Cookie{Name: auth.OAuthCookieName(), Path: "/", MaxAge: -1})
	http.Redirect(w, r, loc, http.StatusFound)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if u := UserFrom(r); u != nil {
		_ = auth.DeleteSessionsForUser(s.db, u.ID)
	}
	http.SetCookie(w, &http.Cookie{Name: auth.CookieName, Path: "/", MaxAge: -1, HttpOnly: true})
	http.Redirect(w, r, "/", http.StatusFound)
}

func (s *Server) oauthCookie(value string) *http.Cookie {
	return &http.Cookie{
		Name:     auth.OAuthCookieName(),
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.cfg.CookieSecure(),
		MaxAge:   int((10 * time.Minute).Seconds()),
	}
}
