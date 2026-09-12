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
	if !CanEdit("admin", "u1", "u1") || !CanEdit("skater", "u1", "u1") {
		t.Fatal("own")
	}
	if CanEdit("admin", "u1", "u2") || CanEdit("skater", "u1", "u2") || CanEdit("member", "u1", "u1") {
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

func TestEnsureForUser(t *testing.T) {
	sqldb, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqldb.Close() })
	if err := db.Migrate(sqldb, "../db/migrations"); err != nil {
		t.Fatal(err)
	}
	_, err = sqldb.Exec(`INSERT INTO users (id, username, display_name, role) VALUES ('u1','disco','Disco','skater')`)
	if err != nil {
		t.Fatal(err)
	}
	a, err := EnsureForUser(sqldb, "u1", "disco")
	if err != nil || a.UserID != "u1" || a.SkaterName != "disco" || a.Slug != "disco" {
		t.Fatalf("insert %+v %v", a, err)
	}
	a.RealName = "Show Name"
	if _, err := Save(sqldb, a); err != nil {
		t.Fatal(err)
	}
	b, err := EnsureForUser(sqldb, "u1", "newname")
	if err != nil {
		t.Fatal(err)
	}
	if b.ID != a.ID || b.SkaterName != "newname" || b.RealName != "Show Name" || b.PublicName() != "Show Name" {
		t.Fatalf("update %+v", b)
	}
	c, err := EnsureForUser(sqldb, "u1", "newname")
	if err != nil || c.ID != a.ID {
		t.Fatalf("idempotent %+v %v", c, err)
	}
	team, err := ListTeam(sqldb)
	if err != nil || len(team) != 1 {
		t.Fatalf("team %v %d", err, len(team))
	}
	if _, err := Save(sqldb, Profile{SkaterName: "orphan"}); err != nil {
		t.Fatal(err)
	}
	team, err = ListTeam(sqldb)
	all, err2 := List(sqldb)
	if err != nil || err2 != nil || len(team) != 1 || len(all) != 2 {
		t.Fatalf("linked vs all team=%d all=%d %v %v", len(team), len(all), err, err2)
	}
}
