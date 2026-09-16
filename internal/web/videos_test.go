package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"seshhub/internal/auth"
	"seshhub/internal/db"
)

func TestVideoPage(t *testing.T) {
	per, page, offset, from, to := videoPage(9, 1, 248)
	if per != 9 || page != 1 || offset != 0 || from != 1 || to != 9 {
		t.Fatalf("p1 got per=%d page=%d off=%d %d–%d", per, page, offset, from, to)
	}
	per, page, offset, from, to = videoPage(9, 28, 248)
	if per != 9 || page != 28 || offset != 243 || from != 244 || to != 248 {
		t.Fatalf("last got per=%d page=%d off=%d %d–%d", per, page, offset, from, to)
	}
	per, page, _, from, to = videoPage(99, 0, 248)
	if per != 9 || page != 1 || from != 1 || to != 9 {
		t.Fatalf("clamp got per=%d page=%d %d–%d", per, page, from, to)
	}
	per, page, _, from, to = videoPage(18, 100, 248)
	if per != 18 || page != 14 || from != 235 || to != 248 {
		t.Fatalf("oversize got per=%d page=%d %d–%d", per, page, from, to)
	}
	per, page, offset, from, to = videoPage(27, 1, 0)
	if per != 27 || page != 1 || offset != 0 || from != 0 || to != 0 {
		t.Fatalf("empty got per=%d page=%d off=%d %d–%d", per, page, offset, from, to)
	}
}

func TestVideosFeedFilter(t *testing.T) {
	sqldb, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqldb.Close() })
	if err := db.Migrate(sqldb, "../db/migrations"); err != nil {
		t.Fatal(err)
	}
	_, err = sqldb.Exec(`INSERT INTO youtube_videos (id, channel_id, channel_title, title, published_at, thumbnail_url, tags, category) VALUES
		('v1','ch','Sofa TV','Wheel Session','2026-01-01','http://t','street','session'),
		('v2','ch','Sofa TV','Other','2026-01-02','http://t','','short')`)
	if err != nil {
		t.Fatal(err)
	}
	u, err := auth.UpsertDiscord(sqldb, "d1", "u", "U", "", auth.RoleMember, false)
	if err != nil {
		t.Fatal(err)
	}
	s := &Server{db: sqldb, webDir: filepath.Join("..", "..", "web")}
	withUser := func(req *http.Request) *http.Request {
		return req.WithContext(context.WithValue(req.Context(), userKey, &u))
	}

	rec := httptest.NewRecorder()
	s.videos(rec, httptest.NewRequest(http.MethodGet, "/videos", nil))
	body := rec.Body.String()
	if rec.Code != 200 || strings.Contains(body, "Feed filter") || !strings.Contains(body, "Wheel Session") || !strings.Contains(body, "Other") {
		t.Fatalf("guest %d %s", rec.Code, body)
	}

	rec = httptest.NewRecorder()
	s.videos(rec, withUser(httptest.NewRequest(http.MethodGet, "/videos", nil)))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "Feed filter") {
		t.Fatalf("logged in %d %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/videos/filter", strings.NewReader("action=save&field=title&value=wheel"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	s.videosFilter(rec, withUser(req))
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("save %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	s.videos(rec, withUser(httptest.NewRequest(http.MethodGet, "/videos", nil)))
	body = rec.Body.String()
	if !strings.Contains(body, "Wheel Session") || strings.Contains(body, ">Other<") {
		t.Fatalf("saved filter %s", body)
	}

	rec = httptest.NewRecorder()
	s.videos(rec, httptest.NewRequest(http.MethodGet, "/videos", nil))
	body = rec.Body.String()
	if rec.Code != 200 || !strings.Contains(body, "Wheel Session") || strings.Contains(body, ">Other<") || strings.Contains(body, "Feed filter") {
		t.Fatalf("guest filtered %d %s", rec.Code, body)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/videos/filter", strings.NewReader("action=test&field=title&value=wheel"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	s.videosFilter(rec, withUser(req))
	body = rec.Body.String()
	if rec.Code != 200 || !strings.Contains(body, "Keep") || !strings.Contains(body, "Hidden") || !strings.Contains(body, "Other") {
		t.Fatalf("test %d %s", rec.Code, body)
	}
}
