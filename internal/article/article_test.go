package article

import (
	"strings"
	"testing"

	"seshhub/internal/db"
)

func TestRenderDateTags(t *testing.T) {
	html := Render("- [date_24h:2026-09-20T20:00:00+02:00]")
	if !strings.Contains(html, `data-fmt="24h"`) || !strings.Contains(html, `datetime="2026-09-20T20:00:00+02:00"`) {
		t.Fatalf("date tag: %s", html)
	}
	if strings.Contains(Render("[date_count:nope]"), `data-fmt="count"`) {
		t.Fatal("bad timestamp should stay put")
	}
}

func TestRenderImages(t *testing.T) {
	md := Render("![x](https://example.com/a.png)")
	if !strings.Contains(md, `src="https://example.com/a.png"`) {
		t.Fatalf("https image: %s", md)
	}
	rel := Render("![x](/static/img/seshsofa.png)")
	if !strings.Contains(rel, `src="/static/img/seshsofa.png"`) {
		t.Fatalf("relative image: %s", rel)
	}
	raw := Render(`<img src="https://example.com/a.png" alt="x">`)
	if !strings.Contains(raw, `src="https://example.com/a.png"`) {
		t.Fatalf("html image: %s", raw)
	}
}

func TestRenderTable(t *testing.T) {
	html := Render("| A | B |\n| --- | --- |\n| 1 | 2 |")
	if !strings.Contains(html, "<table") || !strings.Contains(html, "<td>") {
		t.Fatalf("gfm table: %s", html)
	}
}

func TestRenderHardWrapsAndStrike(t *testing.T) {
	html := Render("line1\nline2")
	if !strings.Contains(html, "<br") {
		t.Fatalf("hard wrap: %s", html)
	}
	html = Render("~~x~~")
	if !strings.Contains(html, "<del>") && !strings.Contains(html, "<s>") {
		t.Fatalf("strike: %s", html)
	}
}

func TestRenderEmojiShortcode(t *testing.T) {
	html := Render(":smile:")
	if !strings.Contains(html, "😄") && !strings.Contains(html, "😀") && !strings.Contains(html, "emoji") {
		t.Fatalf("shortcode: %s", html)
	}
}

func TestRenderStripsScript(t *testing.T) {
	html := Render("hi <script>alert(1)</script> **x**")
	if strings.Contains(html, "<script") {
		t.Fatalf("unsanitized: %s", html)
	}
	if !strings.Contains(html, "<strong>") && !strings.Contains(html, "<b>") {
		t.Fatalf("markdown missing: %s", html)
	}
}

func TestSavePublish(t *testing.T) {
	sqldb, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqldb.Close() })
	if err := db.Migrate(sqldb, "../db/migrations"); err != nil {
		t.Fatal(err)
	}
	uid := "user1"
	_, err = sqldb.Exec(`INSERT INTO users (id, username, display_name, role) VALUES (?,?,?,?)`, uid, "a", "Ada", "admin")
	if err != nil {
		t.Fatal(err)
	}
	a, err := Save(sqldb, Article{Title: "Hello", ContentRaw: "body **here**", AuthorID: uid, Status: "draft"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if a.Status != "draft" || a.Slug != "hello" {
		t.Fatalf("%+v", a)
	}
	a.Status = "published"
	a, err = Save(sqldb, a, true)
	if err != nil || a.PublishedAt == "" {
		t.Fatal(err, a.PublishedAt)
	}
	pub, err := ListPublished(sqldb, false)
	if err != nil || len(pub) != 1 {
		t.Fatalf("%v %d", err, len(pub))
	}
	a.Visibility = "internal"
	a, err = Save(sqldb, a, true)
	if err != nil || a.Visibility != "internal" {
		t.Fatal(err, a.Visibility)
	}
	guest, err := ListPublished(sqldb, false)
	if err != nil || len(guest) != 0 {
		t.Fatalf("guest saw internal: %v %d", err, len(guest))
	}
	hub, err := ListPublished(sqldb, true)
	if err != nil || len(hub) != 1 {
		t.Fatalf("logged-in miss: %v %d", err, len(hub))
	}
	if Visible(a, false, false) || !Visible(a, true, false) {
		t.Fatal("Visible")
	}
}
