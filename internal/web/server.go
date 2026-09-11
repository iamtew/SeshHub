package web

import (
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"path/filepath"

	"seshhub/internal/config"
)

type Server struct {
	webDir string
	mux    *http.ServeMux
}

func New(cfg config.Config) *Server {
	s := &Server{webDir: cfg.WebDir, mux: http.NewServeMux()}
	s.mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir(filepath.Join(cfg.WebDir, "static")))))
	s.mux.HandleFunc("GET /healthz", s.healthz)
	s.mux.HandleFunc("GET /{$}", s.home)
	s.mux.HandleFunc("GET /team", s.team)
	s.mux.HandleFunc("GET /news", s.news)
	s.mux.HandleFunc("GET /videos", s.videos)
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("ok\n"))
}

func (s *Server) render(w http.ResponseWriter, page string, data map[string]any) {
	if data == nil {
		data = map[string]any{}
	}
	files := []string{
		filepath.Join(s.webDir, "templates", "layouts", "base.html"),
		filepath.Join(s.webDir, "templates", "partials", "nav.html"),
		filepath.Join(s.webDir, "templates", "partials", "footer.html"),
		filepath.Join(s.webDir, "templates", "pages", page),
	}
	t, err := template.ParseFiles(files...)
	if err != nil {
		slog.Error("parse templates", "err", err, "page", page)
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, "base.html", data); err != nil {
		slog.Error("execute template", "err", err, "page", page)
	}
}

func (s *Server) home(w http.ResponseWriter, _ *http.Request) {
	s.render(w, "index.html", map[string]any{"Title": "Sesh Sofa", "Path": "/"})
}

func (s *Server) team(w http.ResponseWriter, _ *http.Request) {
	s.render(w, "skaters_list.html", map[string]any{"Title": "Team", "Path": "/team"})
}

func (s *Server) news(w http.ResponseWriter, _ *http.Request) {
	s.render(w, "articles_list.html", map[string]any{"Title": "News", "Path": "/news"})
}

func (s *Server) videos(w http.ResponseWriter, _ *http.Request) {
	s.render(w, "videos_list.html", map[string]any{"Title": "Videos", "Path": "/videos"})
}

func Addr(port string) string {
	return fmt.Sprintf(":%s", port)
}
