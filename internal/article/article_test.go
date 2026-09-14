package article

import (
	"strings"
	"testing"

	"seshhub/internal/db"
)

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
	pub, err := ListPublished(sqldb)
	if err != nil || len(pub) != 1 {
		t.Fatalf("%v %d", err, len(pub))
	}
}
