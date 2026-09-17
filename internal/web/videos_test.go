package web

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"seshhub/internal/auth"
	"seshhub/internal/db"
	"seshhub/internal/skater"
)

func TestVideoPage(t *testing.T) {
	per, page, offset, from, to := videoPage(6, 1, 248)
	if per != 6 || page != 1 || offset != 0 || from != 1 || to != 6 {
		t.Fatalf("p1 got per=%d page=%d off=%d %d–%d", per, page, offset, from, to)
	}
	per, page, offset, from, to = videoPage(6, 42, 248)
	if per != 6 || page != 42 || offset != 246 || from != 247 || to != 248 {
		t.Fatalf("last got per=%d page=%d off=%d %d–%d", per, page, offset, from, to)
	}
	per, page, _, from, to = videoPage(99, 0, 248)
	if per != 6 || page != 1 || from != 1 || to != 6 {
		t.Fatalf("clamp got per=%d page=%d %d–%d", per, page, from, to)
	}
	per, page, _, from, to = videoPage(18, 100, 248)
	if per != 18 || page != 14 || from != 235 || to != 248 {
		t.Fatalf("oversize got per=%d page=%d %d–%d", per, page, from, to)
	}
	per, page, offset, from, to = videoPage(24, 1, 0)
	if per != 24 || page != 1 || offset != 0 || from != 0 || to != 0 {
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
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/dashboard/profile#youtube" {
		t.Fatalf("save %d %s", rec.Code, rec.Header().Get("Location"))
	}

	rec = httptest.NewRecorder()
	s.videos(rec, httptest.NewRequest(http.MethodGet, "/videos", nil))
	body = rec.Body.String()
	if !strings.Contains(body, "Wheel Session") || strings.Contains(body, ">Other<") || !strings.Contains(body, "B Clip") {
		t.Fatalf("public after A filter %s", body)
	}

	rec = httptest.NewRecorder()
	s.dashboardProfile(rec, with(a, httptest.NewRequest(http.MethodGet, "/dashboard/profile", nil)))
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "YouTube Feed Filter") || !strings.Contains(rec.Body.String(), "You have unsaved changes") || !strings.Contains(rec.Body.String(), `role="tablist"`) {
		t.Fatalf("profile %d %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/dashboard/profile/filter", strings.NewReader("action=test&field=title&op=contains&value=wheel"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	s.profileFilter(rec, with(a, req))
	body = rec.Body.String()
	if rec.Code != 200 || !strings.Contains(body, "Keep") || !strings.Contains(body, "Hidden") || !strings.Contains(body, "Other") || !strings.Contains(body, `data-initial-tab="youtube"`) {
		t.Fatalf("test %d %s", rec.Code, body)
	}
}

func TestRosterVideoPager(t *testing.T) {
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
	if _, err := auth.LinkYouTube(sqldb, u.ID, "cha", "A TV", "", "r"); err != nil {
		t.Fatal(err)
	}
	p, err := skater.EnsureForUser(sqldb, u.ID, "Alice")
	if err != nil {
		t.Fatal(err)
	}
	p.FeaturedVideoID = "clip00"
	if _, err := skater.Save(sqldb, p); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 11; i++ {
		_, err = sqldb.Exec(`INSERT INTO youtube_videos (id, channel_id, channel_title, title, published_at, thumbnail_url) VALUES (?,?,?,?,?,?)`,
			fmt.Sprintf("clip%02d", i), "cha", "A TV", fmt.Sprintf("Clip %02d", i), fmt.Sprintf("2026-01-%02d", i+1), "http://t")
		if err != nil {
			t.Fatal(err)
		}
	}
	s := &Server{db: sqldb, webDir: filepath.Join("..", "..", "web")}
	hit := func(q string) string {
		req := httptest.NewRequest(http.MethodGet, "/team/"+p.Slug+q, nil)
		req.SetPathValue("slug", p.Slug)
		rec := httptest.NewRecorder()
		s.skaterDetail(rec, req)
		if rec.Code != 200 {
			t.Fatalf("%s %d %s", q, rec.Code, rec.Body.String())
		}
		return rec.Body.String()
	}
	p1 := hit("")
	if !strings.Contains(p1, "Clip 00") || !strings.Contains(p1, "Clip 10") || strings.Contains(p1, "Clip 01") {
		t.Fatalf("page1 %s", p1)
	}
	if !strings.Contains(p1, "/team/"+p.Slug+"?n=6") || !strings.Contains(p1, "1–6 of 10") {
		t.Fatalf("pager %s", p1)
	}
	p2 := hit("?p=2")
	if !strings.Contains(p2, "Clip 00") || !strings.Contains(p2, "Clip 01") || strings.Contains(p2, "Clip 10") {
		t.Fatalf("page2 %s", p2)
	}
	if !strings.Contains(p2, "7–10 of 10") {
		t.Fatalf("range %s", p2)
	}
}

func TestProfileSaveStaysOnDashboard(t *testing.T) {
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
	p, err := skater.EnsureForUser(sqldb, u.ID, "Alice")
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("display_name", "Alice")
	_ = mw.WriteField("slug", p.Slug)
	_ = mw.WriteField("stance", "regular")
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/dashboard/profile", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req = req.WithContext(context.WithValue(req.Context(), userKey, &u))
	rec := httptest.NewRecorder()
	s := &Server{db: sqldb, webDir: filepath.Join("..", "..", "web")}
	s.dashboardProfile(rec, req)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/dashboard/profile#photo" {
		t.Fatalf("save %d %s", rec.Code, rec.Header().Get("Location"))
	}
}
