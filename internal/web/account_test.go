package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"seshhub/internal/auth"
	"seshhub/internal/db"
)

func TestAccountDeleteNeedsConfirm(t *testing.T) {
	sqldb, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqldb.Close() })
	if err := db.Migrate(sqldb, "../db/migrations"); err != nil {
		t.Fatal(err)
	}
	u, err := auth.UpsertDiscord(sqldb, "d1", "alice", "Alice", "", auth.RoleSkater, false)
	if err != nil {
		t.Fatal(err)
	}
	s := &Server{db: sqldb}
	with := func(req *http.Request) *http.Request {
		return req.WithContext(context.WithValue(req.Context(), userKey, &u))
	}
	post := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/account/delete", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		s.accountDelete(rec, with(req))
		return rec
	}
	if rec := post(""); rec.Code != http.StatusBadRequest {
		t.Fatalf("empty %d", rec.Code)
	}
	if rec := post("confirm=nope"); rec.Code != http.StatusBadRequest {
		t.Fatalf("wrong %d", rec.Code)
	}
	if _, err := auth.GetUser(sqldb, u.ID); err != nil {
		t.Fatal(err)
	}
	if rec := post("confirm=alice"); rec.Code != http.StatusFound {
		t.Fatalf("ok %d %s", rec.Code, rec.Body.String())
	}
	if _, err := auth.GetUser(sqldb, u.ID); err == nil {
		t.Fatal("still there")
	}
}
