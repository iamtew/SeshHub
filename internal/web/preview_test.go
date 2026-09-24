package web

import (
	"context"
	"html/template"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"seshhub/internal/auth"
)

func TestPreviewAuthAndMarkdown(t *testing.T) {
	s := &Server{}

	anon := httptest.NewRequest(http.MethodPost, "/preview", strings.NewReader("content_raw=**x**"))
	anon.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	s.preview(rec, anon)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("anon: %d", rec.Code)
	}

	req := httptest.NewRequest(http.MethodPost, "/preview", strings.NewReader("content_raw=**x**"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req = req.WithContext(context.WithValue(req.Context(), userKey, &auth.User{ID: "u"}))
	rec = httptest.NewRecorder()
	s.preview(rec, req)
	if rec.Code != http.StatusOK || rec.Header().Get("X-Sesh-Preview") != "1" {
		t.Fatalf("user: %d %s", rec.Code, rec.Body.String())
	}
	html := rec.Body.String()
	if !strings.Contains(html, "<strong>") && !strings.Contains(html, "<b>") {
		t.Fatalf("markdown: %s", html)
	}
}

func TestPageCloseTo(t *testing.T) {
	list := httptest.NewRequest(http.MethodPost, "/admin/pages/x", strings.NewReader("after=close"))
	list.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if got := afterSave(list, "/admin/pages/x", pageCloseTo(list)); got != "/admin/pages" {
		t.Fatalf("list: %s", got)
	}
	view := httptest.NewRequest(http.MethodPost, "/admin/pages/x", strings.NewReader("after=close&next=/team"))
	view.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if got := afterSave(view, "/admin/pages/x", pageCloseTo(view)); got != "/team" {
		t.Fatalf("view: %s", got)
	}
	bad := httptest.NewRequest(http.MethodPost, "/admin/pages/x", strings.NewReader("after=close&next=https://evil.example/"))
	bad.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if got := pageCloseTo(bad); got != "/admin/pages" {
		t.Fatalf("bad next: %s", got)
	}
}

func TestEditorTemplatesParse(t *testing.T) {
	root := filepath.Join("..", "..", "web", "templates")
	shared := []string{
		filepath.Join(root, "layouts", "base.html"),
		filepath.Join(root, "partials", "nav.html"),
		filepath.Join(root, "partials", "signin.html"),
		filepath.Join(root, "partials", "footer.html"),
		filepath.Join(root, "partials", "consent.html"),
		filepath.Join(root, "partials", "md_editor.html"),
		filepath.Join(root, "partials", "pager.html"),
		filepath.Join(root, "partials", "forum_badge.html"),
		filepath.Join(root, "partials", "forum_md.html"),
		filepath.Join(root, "partials", "twitch_live.html"),
	}
	for _, page := range []string{"article_form.html", "page_form.html", "admin_spot.html", "skater_form.html", "forum_index.html", "forum_section.html", "forum_thread.html", "forum_mentions.html", "forum_form.html", "admin_forum.html"} {
		files := append(append([]string{}, shared...), filepath.Join(root, "pages", page))
		if _, err := template.ParseFiles(files...); err != nil {
			t.Fatal(page, err)
		}
	}
}
