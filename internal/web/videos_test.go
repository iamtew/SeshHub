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

func TestOwnerFilterOnPublicVideos(t *testing.T) {
	sqldb, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqldb.Close() })
	if err := db.Migrate(sqldb, "../db/migrations"); err != nil {
		t.Fatal(err)
	}
	a, err := auth.UpsertDiscord(sqldb, "da", "a", "A", "", auth.RoleSkater, false)
	if err != nil {
		t.Fatal(err)
	}
	b, err := auth.UpsertDiscord(sqldb, "db", "b", "B", "", auth.RoleMember, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := auth.LinkYouTube(sqldb, a.ID, "cha", "A TV", "", "r"); err != nil {
		t.Fatal(err)
	}
	if _, err := auth.LinkYouTube(sqldb, b.ID, "chb", "B TV", "", "r"); err != nil {
		t.Fatal(err)
	}
	_, err = sqldb.Exec(`INSERT INTO youtube_videos (id, channel_id, channel_title, title, published_at, thumbnail_url, tags, category) VALUES
		('v1','cha','A TV','Wheel Session','2026-01-01','http://t','street','session'),
		('v2','cha','A TV','Other','2026-01-02','http://t','','short'),
		('v3','chb','B TV','B Clip','2026-01-03','http://t','','session')`)
	if err != nil {
		t.Fatal(err)
	}
	s := &Server{db: sqldb, webDir: filepath.Join("..", "..", "web")}
	with := func(u auth.User, req *http.Request) *http.Request {
		return req.WithContext(context.WithValue(req.Context(), userKey, &u))
	}

	rec := httptest.NewRecorder()
	s.videos(rec, httptest.NewRequest(http.MethodGet, "/videos", nil))
	body := rec.Body.String()
	if rec.Code != 200 || strings.Contains(body, "YouTube Feed Filter") || !strings.Contains(body, "Wheel Session") || !strings.Contains(body, "Other") || !strings.Contains(body, "B Clip") {
		t.Fatalf("guest all %d %s", rec.Code, body)
	}

	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/dashboard/profile/filter", strings.NewReader("action=save&field=title&op=contains&value=wheel"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	s.profileFilter(rec, with(a, req))
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("save %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	s.videos(rec, httptest.NewRequest(http.MethodGet, "/videos", nil))
	body = rec.Body.String()
	if !strings.Contains(body, "Wheel Session") || strings.Contains(body, ">Other<") || !strings.Contains(body, "B Clip") {
		t.Fatalf("public after A filter %s", body)
	}

	rec = httptest.NewRecorder()
	s.dashboardProfile(rec, with(a, httptest.NewRequest(http.MethodGet, "/dashboard/profile", nil)))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "YouTube Feed Filter") {
		t.Fatalf("profile %d %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/dashboard/profile/filter", strings.NewReader("action=test&field=title&op=contains&value=wheel"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	s.profileFilter(rec, with(a, req))
	body = rec.Body.String()
	if rec.Code != 200 || !strings.Contains(body, "Keep") || !strings.Contains(body, "Hidden") || !strings.Contains(body, "Other") {
		t.Fatalf("test %d %s", rec.Code, body)
	}
}
