package twitch

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"seshhub/internal/db"
)

func TestNormalize(t *testing.T) {
	got, err := Normalize("  https://www.twitch.tv/TeW020  ")
	if err != nil || got != "tew020" {
		t.Fatalf("%q %v", got, err)
	}
	if _, err := Normalize("ab"); err == nil {
		t.Fatal("short")
	}
	if _, err := Normalize("bad name"); err == nil {
		t.Fatal("space")
	}
	empty, err := Normalize("")
	if err != nil || empty != "" {
		t.Fatalf("empty %q %v", empty, err)
	}
}

func TestParseStreams(t *testing.T) {
	body := []byte(`{"data":[{"user_login":"TeW020","title":"sesh","started_at":"2026-09-23T12:00:00Z","type":"live"}]}`)
	got, err := parseStreams(body)
	if err != nil {
		t.Fatal(err)
	}
	s, ok := got["tew020"]
	if !ok || s.Title != "sesh" || !s.Started.Equal(time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("%+v", got)
	}
}

func TestParseUser(t *testing.T) {
	ch, err := parseUser([]byte(`{"data":[{"login":"TeW020","display_name":"tew020","description":"sesh","created_at":"2012-01-15T12:00:00Z"}]}`))
	if err != nil || ch.Login != "tew020" || ch.DisplayName != "tew020" || ch.Blurb() != "sesh" || ch.CreatedOn() != "15 Jan 2012" {
		t.Fatalf("%+v %v", ch, err)
	}
	empty, err := parseUser([]byte(`{"data":[]}`))
	if err != nil || empty.Login != "" {
		t.Fatalf("missing %+v %v", empty, err)
	}
}

func TestRender(t *testing.T) {
	msg := Render("", Live{Name: "Ada", Login: "ada", Title: "sesh", Started: time.Now().Add(-90 * time.Minute)})
	if !strings.Contains(msg, "Ada is live on Twitch: sesh") || !strings.Contains(msg, "https://www.twitch.tv/ada") {
		t.Fatalf("%q", msg)
	}
	custom := Render("{name} @ {twitch} {uptime} {url}", Live{Name: "Ada", Login: "ada", Started: time.Now().Add(-5 * time.Minute)})
	if custom != "Ada @ ada 5m https://www.twitch.tv/ada" {
		t.Fatalf("%q", custom)
	}
}

func TestPollWentLiveOnce(t *testing.T) {
	sqldb, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqldb.Close() })
	if err := db.Migrate(sqldb, "../db/migrations"); err != nil {
		t.Fatal(err)
	}
	if _, err := sqldb.Exec(`INSERT INTO skater_profiles (id, slug, skater_name, real_name, twitch_login) VALUES ('p','ada','ada','Ada','adalive')`); err != nil {
		t.Fatal(err)
	}
	live := true
	title := "first"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || strings.Contains(r.URL.Path, "token") || r.Method == http.MethodPost {
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "t", "expires_in": 3600})
			return
		}
		if !live {
			io.WriteString(w, `{"data":[]}`)
			return
		}
		io.WriteString(w, `{"data":[{"user_login":"adalive","title":"`+title+`","started_at":"2026-09-23T12:00:00Z","type":"live"}]}`)
	}))
	t.Cleanup(srv.Close)
	c := New("id", "secret")
	c.HTTP = srv.Client()
	c.API, c.Auth = srv.URL, srv.URL
	ctx := context.Background()
	went, err := Poll(ctx, sqldb, c)
	if err != nil || len(went) != 1 || went[0].Name != "Ada" || went[0].Title != "first" || went[0].Login != "adalive" {
		t.Fatalf("first %+v %v", went, err)
	}
	title = "second"
	went, err = Poll(ctx, sqldb, c)
	if err != nil || len(went) != 0 {
		t.Fatalf("still live %+v %v", went, err)
	}
	var gotTitle, gotStart string
	if err := sqldb.QueryRow(`SELECT twitch_title, twitch_started_at FROM skater_profiles WHERE id='p'`).Scan(&gotTitle, &gotStart); err != nil || gotTitle != "second" || gotStart == "" {
		t.Fatalf("snapshot %q %q %v", gotTitle, gotStart, err)
	}
	live = false
	went, err = Poll(ctx, sqldb, c)
	if err != nil || len(went) != 0 {
		t.Fatalf("offline %+v %v", went, err)
	}
	if err := sqldb.QueryRow(`SELECT twitch_title, IFNULL(twitch_started_at,'') FROM skater_profiles WHERE id='p'`).Scan(&gotTitle, &gotStart); err != nil || gotTitle != "" || gotStart != "" {
		t.Fatalf("cleared %q %q %v", gotTitle, gotStart, err)
	}
	live, title = true, "again"
	went, err = Poll(ctx, sqldb, c)
	if err != nil || len(went) != 1 || went[0].Title != "again" {
		t.Fatalf("re-live %+v %v", went, err)
	}
}
