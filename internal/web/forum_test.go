package web

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"seshhub/internal/auth"
	"seshhub/internal/config"
	"seshhub/internal/db"
	"seshhub/internal/forum"
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
	s := New(config.Config{WebDir: filepath.Join("..", "..", "web")}, sqldb, nil)
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
	s := New(config.Config{WebDir: filepath.Join("..", "..", "web")}, sqldb, nil)
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

func TestForumMentionsAndMedia(t *testing.T) {
	sqldb, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqldb.Close() })
	if err := db.Migrate(sqldb, "../db/migrations"); err != nil {
		t.Fatal(err)
	}
	alice, err := auth.UpsertDiscord(sqldb, "d1", "alice", "Alice", "", auth.RoleFriend, false)
	if err != nil {
		t.Fatal(err)
	}
	bob, err := auth.UpsertDiscord(sqldb, "d2", "bob", "Bob", "", auth.RoleFriend, false)
	if err != nil {
		t.Fatal(err)
	}
	tok, err := auth.CreateSession(sqldb, bob.ID)
	if err != nil {
		t.Fatal(err)
	}
	sec, err := forum.GetSection(sqldb, "general")
	if err != nil {
		t.Fatal(err)
	}
	th, err := forum.CreateThread(sqldb, sec.ID, alice.ID, "Hi", "hey @bob", nil)
	if err != nil {
		t.Fatal(err)
	}
	s := New(config.Config{WebDir: filepath.Join("..", "..", "web")}, sqldb, nil)
	req := httptest.NewRequest(http.MethodGet, "/forum/mentions", nil)
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: tok})
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	posts, err := forum.ListPosts(sqldb, th.ID)
	if err != nil || len(posts) != 1 {
		t.Fatalf("posts %d %v", len(posts), err)
	}
	want := forum.PostURL(sec.Slug, th.Slug, posts[0].ID)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "Hi") || !strings.Contains(rec.Body.String(), want) {
		t.Fatalf("mentions %d %s", rec.Code, rec.Body.String())
	}
	for i := 0; i < 24; i++ {
		if err := forum.Reply(sqldb, th.ID, alice.ID, "pad", "", nil); err != nil {
			t.Fatal(err)
		}
	}
	posts, err = forum.ListPosts(sqldb, th.ID)
	if err != nil || len(posts) != 25 {
		t.Fatalf("pad %d %v", len(posts), err)
	}
	last := posts[24]
	req = httptest.NewRequest(http.MethodGet, forum.PostURL(sec.Slug, th.Slug, last.ID), nil)
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: tok})
	rec = httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `id="p-`+last.ID+`"`) || strings.Contains(rec.Body.String(), `id="p-`+posts[0].ID+`"`) {
		t.Fatalf("post jump %d %s", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/forum/users?q=al", nil))
	if rec.Code != http.StatusFound {
		t.Fatalf("users guest %d", rec.Code)
	}
	req = httptest.NewRequest(http.MethodGet, "/forum/users?q=al", nil)
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: tok})
	rec = httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "alice") {
		t.Fatalf("users %d %s", rec.Code, rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodGet, "/forum/users?q=", nil)
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: tok})
	rec = httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "alice") || !strings.Contains(rec.Body.String(), "bob") || strings.Contains(rec.Body.String(), "deleted") {
		t.Fatalf("users empty %d %s", rec.Code, rec.Body.String())
	}
	id := strings.Repeat("a", 32)
	path := forum.PhotoPath(id, "jpg")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("jpg"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })
	rec = httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/media/forum/"+id+".jpg", nil))
	if rec.Code != http.StatusFound {
		t.Fatalf("media guest %d", rec.Code)
	}
	req = httptest.NewRequest(http.MethodGet, "/media/forum/"+id+".jpg", nil)
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: tok})
	rec = httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("media auth %d", rec.Code)
	}
	mp3 := append([]byte("ID3"), make([]byte, 20)...)
	if err := forum.Reply(sqldb, th.ID, alice.ID, "song", "", []forum.FileIn{{Name: `C:\mix\listen.mp3`, R: bytes.NewReader(mp3)}}); err != nil {
		t.Fatal(err)
	}
	posts, err = forum.ListPosts(sqldb, th.ID)
	if err != nil || len(posts) < 1 {
		t.Fatal(err)
	}
	var au forum.Photo
	for _, p := range posts {
		for _, ph := range p.Photos {
			if ph.IsAudio() {
				au = ph
			}
		}
	}
	if au.ID == "" || au.Name != "listen.mp3" {
		t.Fatalf("audio name %+v", au)
	}
	t.Cleanup(func() { forum.RemovePhotos([]string{au.ID + "." + au.Ext}) })
	req = httptest.NewRequest(http.MethodGet, au.URL, nil)
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: tok})
	rec = httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != 200 || !strings.Contains(rec.Header().Get("Content-Disposition"), "listen.mp3") || strings.Contains(rec.Header().Get("Content-Disposition"), "attachment") {
		t.Fatalf("inline %d %q", rec.Code, rec.Header().Get("Content-Disposition"))
	}
	req = httptest.NewRequest(http.MethodGet, au.URL+"?dl=1", nil)
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: tok})
	rec = httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	cd := rec.Header().Get("Content-Disposition")
	if rec.Code != 200 || !strings.Contains(cd, "listen.mp3") || !strings.Contains(cd, "attachment") {
		t.Fatalf("attachment %d %q", rec.Code, cd)
	}
}

func TestForumAdminCannotEditOthers(t *testing.T) {
	sqldb, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqldb.Close() })
	if err := db.Migrate(sqldb, "../db/migrations"); err != nil {
		t.Fatal(err)
	}
	alice, err := auth.UpsertDiscord(sqldb, "d1", "alice", "Alice", "", auth.RoleFriend, false)
	if err != nil {
		t.Fatal(err)
	}
	admin, err := auth.UpsertDiscord(sqldb, "d2", "admin", "Admin", "", auth.RoleAdmin, false)
	if err != nil {
		t.Fatal(err)
	}
	tok, err := auth.CreateSession(sqldb, admin.ID)
	if err != nil {
		t.Fatal(err)
	}
	sec, err := forum.GetSection(sqldb, "general")
	if err != nil {
		t.Fatal(err)
	}
	th, err := forum.CreateThread(sqldb, sec.ID, alice.ID, "Hi", "body", nil)
	if err != nil {
		t.Fatal(err)
	}
	posts, err := forum.ListPosts(sqldb, th.ID)
	if err != nil || len(posts) != 1 {
		t.Fatal(err)
	}
	s := New(config.Config{WebDir: filepath.Join("..", "..", "web")}, sqldb, nil)
	req := httptest.NewRequest(http.MethodPost, "/forum/"+sec.Slug+"/"+th.Slug+"/posts/"+posts[0].ID, strings.NewReader("body=hacked"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: tok})
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("admin edit %d %s", rec.Code, rec.Body.String())
	}
	got, err := forum.GetPost(sqldb, posts[0].ID)
	if err != nil || got.BodyRaw != "body" {
		t.Fatalf("body changed %q %v", got.BodyRaw, err)
	}
}
