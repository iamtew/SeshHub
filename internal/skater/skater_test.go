package skater

import (
	"testing"

	"seshhub/internal/db"
)

func TestSlugify(t *testing.T) {
	if got := Slugify("  Mike 'Mongo' Jones  "); got != "mike-mongo-jones" {
		t.Fatalf("got %q", got)
	}
	if Slugify("***") != "skater" {
		t.Fatal("empty slug")
	}
}

func TestCanEdit(t *testing.T) {
	if !CanEdit("admin", "x", "") {
		t.Fatal("admin")
	}
	if !CanEdit("skater", "u1", "u1") {
		t.Fatal("own")
	}
	if CanEdit("skater", "u1", "u2") || CanEdit("member", "u1", "u1") {
		t.Fatal("denied")
	}
}

func TestSaveUniqueSlug(t *testing.T) {
	sqldb, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqldb.Close() })
	if err := db.Migrate(sqldb, "../db/migrations"); err != nil {
		t.Fatal(err)
	}
	a, err := Save(sqldb, Profile{SkaterName: "Alex Flow"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := Save(sqldb, Profile{SkaterName: "Alex Flow"})
	if err != nil {
		t.Fatal(err)
	}
	if a.Slug != "alex-flow" || b.Slug != "alex-flow-2" {
		t.Fatalf("slugs %q %q", a.Slug, b.Slug)
	}
	a.FeaturedVideoID = "abc"
	a, err = Save(sqldb, a)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Get(sqldb, "slug", a.Slug)
	if err != nil || got.FeaturedVideoID != "abc" {
		t.Fatalf("featured %v %+v", err, got)
	}
	list, err := List(sqldb)
	if err != nil || len(list) != 2 {
		t.Fatalf("list %v %d", err, len(list))
	}
}
