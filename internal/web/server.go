package web

import (
	"database/sql"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"

	"seshhub/internal/auth"
	"seshhub/internal/config"
	"seshhub/internal/yt"
)

type Server struct {
	cfg    config.Config
	db     *sql.DB
	webDir string
	mux    *http.ServeMux
}

func New(cfg config.Config, db *sql.DB) *Server {
	yt.RefreshAccess = auth.RefreshGoogle
	s := &Server{cfg: cfg, db: db, webDir: cfg.WebDir, mux: http.NewServeMux()}
	s.mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir(filepath.Join(cfg.WebDir, "static")))))
	s.mux.HandleFunc("GET /media/avatars/{file}", s.mediaAvatar)
	s.mux.HandleFunc("GET /media/gallery/{file}", s.mediaGallery)
	s.mux.HandleFunc("GET /healthz", s.healthz)
	s.mux.HandleFunc("GET /{$}", s.home)
	s.mux.HandleFunc("GET /team", s.team)
	s.mux.HandleFunc("GET /skaters", s.skatersAlias)
	s.mux.HandleFunc("GET /team/{slug}", s.skaterDetail)
	s.mux.HandleFunc("GET /friends", s.friends)
	s.mux.HandleFunc("GET /friends/{slug}", s.skaterDetail)
	s.mux.HandleFunc("GET /news", s.news)
	s.mux.HandleFunc("GET /news/{slug}", s.articleDetail)
	s.mux.HandleFunc("GET /videos", s.videos)
	s.mux.HandleFunc("GET /photos", s.photos)
	s.mux.HandleFunc("GET /episodes", s.episodes)
	s.mux.HandleFunc("GET /login", s.loginPage)
	s.mux.HandleFunc("GET /auth/discord", s.startOAuth("discord"))
	s.mux.HandleFunc("GET /auth/youtube", s.startOAuth("youtube"))
	s.mux.HandleFunc("GET /auth/discord/callback", s.callbackDiscord)
	s.mux.HandleFunc("GET /auth/youtube/callback", s.callbackYouTube)
	s.mux.HandleFunc("GET /auth/logout", s.logout)
	s.mux.HandleFunc("GET /account", s.accountPage)
	s.mux.HandleFunc("POST /account/unlink/discord", s.accountUnlink("discord"))
	s.mux.HandleFunc("POST /account/unlink/youtube", s.accountUnlink("youtube"))
	s.mux.HandleFunc("POST /account/delete", s.accountDelete)
	s.mux.HandleFunc("GET /account/export", s.accountExport)
	s.mux.HandleFunc("POST /consent", s.consent)
	s.mux.HandleFunc("POST /preview", s.preview)
	s.mux.HandleFunc("GET /access", s.accessPage)
	s.mux.HandleFunc("POST /access/request", s.accessRequest)
	s.mux.HandleFunc("GET /admin/access", s.adminAccess)
	s.mux.HandleFunc("POST /admin/access/{id}/approve", s.adminDecide("approved"))
	s.mux.HandleFunc("POST /admin/access/{id}/reject", s.adminDecide("rejected"))
	s.mux.HandleFunc("GET /admin/users", s.adminUsers)
	s.mux.HandleFunc("POST /admin/users/{id}/merge", s.adminUserMerge)
	s.mux.HandleFunc("POST /admin/users/{id}/unlink/discord", s.adminUserUnlink("discord"))
	s.mux.HandleFunc("POST /admin/users/{id}/unlink/youtube", s.adminUserUnlink("youtube"))
	s.mux.HandleFunc("POST /admin/users/{id}/delete", s.adminUserDelete)
	s.mux.HandleFunc("GET /admin/skaters", s.adminSkaters)
	s.mux.HandleFunc("GET /admin/skaters/{id}", s.adminSkaterEdit)
	s.mux.HandleFunc("POST /admin/skaters/{id}", s.adminSkaterEdit)
	s.mux.HandleFunc("POST /admin/skaters/{id}/delete", s.adminSkaterDelete)
	s.mux.HandleFunc("GET /dashboard/profile", s.dashboardProfile)
	s.mux.HandleFunc("POST /dashboard/profile", s.dashboardProfile)
	s.mux.HandleFunc("POST /dashboard/profile/filter", s.profileFilter)
	s.mux.HandleFunc("GET /dashboard/gallery", s.dashboardGallery)
	s.mux.HandleFunc("POST /dashboard/gallery", s.dashboardGalleryAdd)
	s.mux.HandleFunc("POST /dashboard/gallery/order", s.dashboardGalleryOrder)
	s.mux.HandleFunc("POST /dashboard/gallery/{id}/delete", s.dashboardGalleryDelete)
	s.mux.HandleFunc("GET /admin/articles", s.adminArticles)
	s.mux.HandleFunc("GET /admin/articles/new", s.articleCreate(true))
	s.mux.HandleFunc("POST /admin/articles", s.articleCreate(true))
	s.mux.HandleFunc("GET /admin/articles/{id}", s.articleEdit(true))
	s.mux.HandleFunc("POST /admin/articles/{id}", s.articleEdit(true))
	s.mux.HandleFunc("POST /admin/articles/{id}/delete", s.articleDelete)
	s.mux.HandleFunc("GET /dashboard/articles", s.dashboardArticles)
	s.mux.HandleFunc("GET /dashboard/articles/new", s.articleCreate(false))
	s.mux.HandleFunc("POST /dashboard/articles", s.articleCreate(false))
	s.mux.HandleFunc("GET /dashboard/articles/{id}", s.articleEdit(false))
	s.mux.HandleFunc("POST /dashboard/articles/{id}", s.articleEdit(false))
	s.mux.HandleFunc("GET /admin", s.adminHome)
	s.mux.HandleFunc("GET /admin/spot", s.adminSpot)
	s.mux.HandleFunc("POST /admin/spot", s.adminSpot)
	s.mux.HandleFunc("GET /admin/episodes", s.adminEpisodes)
	s.mux.HandleFunc("GET /admin/episodes/new", s.adminEpisodeCreate)
	s.mux.HandleFunc("POST /admin/episodes", s.adminEpisodeCreate)
	s.mux.HandleFunc("GET /admin/episodes/{n}", s.adminEpisodeEdit)
	s.mux.HandleFunc("POST /admin/episodes/{n}", s.adminEpisodeEdit)
	s.mux.HandleFunc("POST /admin/episodes/{n}/delete", s.adminEpisodeDelete)
	s.mux.HandleFunc("GET /admin/pages", s.adminPages)
	s.mux.HandleFunc("GET /admin/pages/new", s.adminPageCreate)
	s.mux.HandleFunc("POST /admin/pages", s.adminPageCreate)
	s.mux.HandleFunc("GET /admin/pages/{id}", s.adminPageEdit)
	s.mux.HandleFunc("POST /admin/pages/{id}", s.adminPageEdit)
	s.mux.HandleFunc("POST /admin/pages/{id}/delete", s.adminPageDelete)
	s.mux.HandleFunc("GET /about/privacy", s.legalPrivacy)
	s.mux.HandleFunc("GET /about/tos", s.legalTOS)
	s.mux.HandleFunc("GET /{slug...}", s.customPage)
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, s.injectUser(r))
}

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("ok\n"))
}

func (s *Server) render(w http.ResponseWriter, r *http.Request, page string, data map[string]any) {
	if data == nil {
		data = map[string]any{}
	}
	u := UserFrom(r)
	data["User"] = u
	data["DiscordLogin"] = s.cfg.DiscordEnabled()
	data["YouTubeLogin"] = s.cfg.YouTubeEnabled()
	if s.cfg.GTagID != "" {
		switch consentFrom(r) {
		case "yes":
			data["GTag"] = s.cfg.GTagID
		case "":
			data["ConsentAsk"] = true
		}
	}
	authPage, _ := data["AuthPage"].(bool)
	path, _ := data["Path"].(string)
	data["NavAbout"] = path == "/about" || strings.HasPrefix(path, "/about/")
	if !authPage {
		if u != nil {
			data["Fold"] = true
		} else if publicHero(path) {
			data["Hero"] = true
		}
	}
	files := []string{
		filepath.Join(s.webDir, "templates", "layouts", "base.html"),
		filepath.Join(s.webDir, "templates", "partials", "nav.html"),
		filepath.Join(s.webDir, "templates", "partials", "signin.html"),
		filepath.Join(s.webDir, "templates", "partials", "footer.html"),
		filepath.Join(s.webDir, "templates", "partials", "consent.html"),
		filepath.Join(s.webDir, "templates", "partials", "md_editor.html"),
		filepath.Join(s.webDir, "templates", "partials", "pager.html"),
		filepath.Join(s.webDir, "templates", "partials", "slideshow.html"),
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

func publicHero(path string) bool {
	if path == "" || path == "/account" || path == "/access" || path == "/login" {
		return false
	}
	if strings.HasPrefix(path, "/admin") || strings.HasPrefix(path, "/dashboard") {
		return false
	}
	return true
}

func Addr(port string) string {
	return fmt.Sprintf(":%s", port)
}

func afterSave(r *http.Request, stay, closeTo string) string {
	if r.FormValue("after") == "close" {
		return closeTo
	}
	return stay
}

func safeNext(s string) string {
	if !strings.HasPrefix(s, "/") || strings.HasPrefix(s, "//") || strings.ContainsAny(s, ":\\\n\r\t ") {
		return ""
	}
	return s
}

func pageCloseTo(r *http.Request) string {
	if n := safeNext(r.FormValue("next")); n != "" {
		return n
	}
	return "/admin/pages"
}
