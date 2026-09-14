package page

import (
	"testing"

	"seshhub/internal/db"
)

func TestReservedAndSave(t *testing.T) {
	if !Reserved("admin") || !Reserved("spot") || !Reserved("episodes") || Reserved("about") {
		t.Fatal("reserved")
	}
	sqldb, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqldb.Close() })
	if err := db.Migrate(sqldb, "../db/migrations"); err != nil {
		t.Fatal(err)
	}
	_, err = Save(sqldb, Page{Title: "About", ContentRaw: "hi **x**", Published: true})
	if err != nil {
		t.Fatal(err)
	}
	p, err := Get(sqldb, "slug", "about")
	if err != nil || !p.Published || p.ContentHTML == "" {
		t.Fatalf("%+v %v", p, err)
	}
	if _, err := Save(sqldb, Page{Title: "Team", Slug: "team", ContentRaw: "no"}); err == nil {
		t.Fatal("expected reserved")
	}
}
