package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"seshhub/internal/auth"
	"seshhub/internal/bot"
	"seshhub/internal/config"
	"seshhub/internal/db"
)

func TestAdminBotPage(t *testing.T) {
	sqldb, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqldb.Close() })
	if err := db.Migrate(sqldb, "../db/migrations"); err != nil {
		t.Fatal(err)
	}
	b := bot.Open(sqldb, "", "")
	s := New(config.Config{WebDir: filepath.Join("..", "..", "web")}, sqldb, b)
	admin, err := auth.UpsertDiscord(sqldb, "admin1", "ada", "Ada", "", auth.RoleAdmin, false)
	if err != nil {
		t.Fatal(err)
	}
	tok, err := auth.CreateSession(sqldb, admin.ID)
	if err != nil {
		t.Fatal(err)
	}
	get := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/admin/bot", nil)
		req.AddCookie(&http.Cookie{Name: auth.CookieName, Value: tok})
		rec := httptest.NewRecorder()
		s.ServeHTTP(rec, req)
		return rec
	}
	rec := get()
	if rec.Code != http.StatusOK {
		t.Fatalf("get %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"Discord bot", "missing", "not configured", "New published news", "New forum thread", "New access request"} {
		if !strings.Contains(body, want) {
			t.Fatalf("page missing %q", want)
		}
	}
	form := url.Values{"channel_id": {"123456789012345678"}, "news": {"1"}}
	post := httptest.NewRequest(http.MethodPost, "/admin/bot", strings.NewReader(form.Encode()))
	post.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	post.AddCookie(&http.Cookie{Name: auth.CookieName, Value: tok})
	rec = httptest.NewRecorder()
	s.ServeHTTP(rec, post)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("post %d", rec.Code)
	}
	settings, err := b.Settings()
	if err != nil || settings.ChannelID != "123456789012345678" || !settings.News || settings.Forum || settings.Access {
		t.Fatalf("%+v %v", settings, err)
	}
	if body = get().Body.String(); !strings.Contains(body, `name="news" value="1" checked`) {
		t.Fatal("news toggle did not stay on")
	}
	bad := httptest.NewRequest(http.MethodPost, "/admin/bot", strings.NewReader("channel_id=nope"))
	bad.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	bad.AddCookie(&http.Cookie{Name: auth.CookieName, Value: tok})
	rec = httptest.NewRecorder()
	s.ServeHTTP(rec, bad)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bad channel %d", rec.Code)
	}
}

func TestChannelID(t *testing.T) {
	if !channelID("") || !channelID("12345") {
		t.Fatal("empty and a snowflake are ok")
	}
	if channelID("12") || channelID("abc") || channelID("1234x") {
		t.Fatal("short or non-digit channel id")
	}
}
