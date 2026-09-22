package web

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"seshhub/internal/auth"
	"seshhub/internal/config"
	"seshhub/internal/db"
)

func TestForumGuestRedirect(t *testing.T) {
	sqldb, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqldb.Close() })
	if err := db.Migrate(sqldb, "../db/migrations"); err != nil {
		t.Fatal(err)
	}
	s := New(config.Config{WebDir: filepath.Join("..", "..", "web")}, sqldb)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/forum", nil))
	if rec.Code != http.StatusFound {
		t.Fatalf("code %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/login?next=%2Fforum" {
		t.Fatalf("loc %q", loc)
	}
}

func TestForumLoggedIn(t *testing.T) {
	sqldb, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqldb.Close() })
	if err := db.Migrate(sqldb, "../db/migrations"); err != nil {
		t.Fatal(err)
	}
	u, err := auth.UpsertDiscord(sqldb, "d1", "alice", "Alice", "", auth.RoleFriend, false)
	if err != nil {
		t.Fatal(err)
	}
	tok, err := auth.CreateSession(sqldb, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	s := New(config.Config{WebDir: filepath.Join("..", "..", "web")}, sqldb)
	req := httptest.NewRequest(http.MethodGet, "/forum", nil)
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: tok})
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "Forum") {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "General") {
		t.Fatalf("missing section %s", rec.Body.String())
	}
}
