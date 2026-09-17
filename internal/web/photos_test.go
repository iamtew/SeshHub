package web

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"seshhub/internal/db"
)

func TestPhotosEmpty(t *testing.T) {
	sqldb, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqldb.Close() })
	if err := db.Migrate(sqldb, "../db/migrations"); err != nil {
		t.Fatal(err)
	}
	s := &Server{db: sqldb, webDir: filepath.Join("..", "..", "web")}
	rec := httptest.NewRecorder()
	s.photos(rec, httptest.NewRequest(http.MethodGet, "/photos", nil))
	body := rec.Body.String()
	if rec.Code != 200 || !strings.Contains(body, "No photos yet") || !strings.Contains(body, `href="/photos"`) {
		t.Fatalf("%d %s", rec.Code, body)
	}
}
