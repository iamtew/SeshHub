package web

import (
	"net/http"
	"strings"
)

const consentCookie = "seshhub_consent"

func consentFrom(r *http.Request) string {
	c, err := r.Cookie(consentCookie)
	if err != nil {
		return ""
	}
	return c.Value
}

func (s *Server) consent(w http.ResponseWriter, r *http.Request) {
	v := "no"
	if r.FormValue("choice") == "yes" {
		v = "yes"
	}
	http.SetCookie(w, &http.Cookie{
		Name:     consentCookie,
		Value:    v,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.cfg.CookieSecure(),
		MaxAge:   180 * 24 * 60 * 60,
	})
	to := r.FormValue("return")
	if !strings.HasPrefix(to, "/") || strings.HasPrefix(to, "//") {
		to = "/"
	}
	http.Redirect(w, r, to, http.StatusFound)
}
