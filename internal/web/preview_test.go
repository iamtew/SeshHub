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
	}
	for _, page := range []string{"article_form.html", "page_form.html", "admin_spot.html"} {
		files := append(append([]string{}, shared...), filepath.Join(root, "pages", page))
		if _, err := template.ParseFiles(files...); err != nil {
			t.Fatal(page, err)
		}
	}
}
