package web

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"seshhub/internal/auth"
	"seshhub/internal/config"
	"seshhub/internal/db"
)

func TestAdminUsersSuperAdminOnly(t *testing.T) {
	sqldb, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqldb.Close() })
	if err := db.Migrate(sqldb, "../db/migrations"); err != nil {
		t.Fatal(err)
	}
	guild, err := auth.UpsertDiscord(sqldb, "guild", "mod", "Mod", "", auth.RoleAdmin, false)
	if err != nil {
		t.Fatal(err)
	}
	super, err := auth.UpsertDiscord(sqldb, "super", "root", "Root", "", auth.RoleAdmin, false)
	if err != nil {
		t.Fatal(err)
	}
	s := New(config.Config{WebDir: filepath.Join("..", "..", "web"), SuperAdminIDs: []string{"super"}}, sqldb)
	hit := func(u auth.User) int {
		tok, err := auth.CreateSession(sqldb, u.ID)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
		req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: tok})
		rec := httptest.NewRecorder()
		s.ServeHTTP(rec, req)
		return rec.Code
	}
	if code := hit(guild); code != http.StatusForbidden {
		t.Fatalf("guild admin %d", code)
	}
	if code := hit(super); code != http.StatusOK {
		t.Fatalf("superadmin %d", code)
	}
}
