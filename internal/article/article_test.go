package article

import (
	"strings"
	"testing"

	"seshhub/internal/db"
)

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
