package web

import (
	"html"
	"net/http"
	"strings"

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
		vals, live := spot.Latest()
		before, after, hole := spot.SplitImages(raw)
		fill := func(s string) string {
			if live {
				return spot.Fill(s, vals)
			}
			return s
		}
		if hole {
			var imgs strings.Builder
			if body, ok := spot.LatestRaw(); ok {
				for _, u := range spot.Images(body) {
					imgs.WriteString(`<p><img src="`)
					imgs.WriteString(html.EscapeString(u))
					imgs.WriteString(`" alt=""></p>`)
				}
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("X-Sesh-Preview", "1")
			_, _ = w.Write([]byte(article.Render(fill(before)) + imgs.String() + article.Render(fill(after))))
			return
		}
		raw = fill(raw)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Sesh-Preview", "1")
	_, _ = w.Write([]byte(article.Render(raw)))
}
