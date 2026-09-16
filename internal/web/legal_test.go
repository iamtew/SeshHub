package web

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestLegalPrivacy(t *testing.T) {
	s := &Server{webDir: filepath.Join("..", "..", "web")}
	rec := httptest.NewRecorder()
	s.legalPrivacy(rec, httptest.NewRequest(http.MethodGet, "/about/privacy", nil))
	body := rec.Body.String()
	if rec.Code != 200 || !strings.Contains(body, "tewmten@gmail.com") {
		t.Fatalf("%d %s", rec.Code, body)
	}
}
