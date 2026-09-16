package web

import (
	"net/http"

	"seshhub/internal/article"
	"seshhub/internal/spot"
)

const previewMax = 256 << 10

func (s *Server) preview(w http.ResponseWriter, r *http.Request) {
	if UserFrom(r) == nil {
		http.Error(w, "auth", http.StatusUnauthorized)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, previewMax)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "too large", http.StatusRequestEntityTooLarge)
		return
	}
	raw := r.FormValue("content_raw")
	if r.FormValue("fill") == "spot" {
		if vals, ok := spot.Latest(); ok {
			raw = spot.Fill(raw, vals)
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Sesh-Preview", "1")
	_, _ = w.Write([]byte(article.Render(raw)))
}
