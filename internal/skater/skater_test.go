package skater

import (
	"database/sql"
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
	if !CanEdit("friend", "u1", "u1") {
		t.Fatal("friend own")
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
	custom, err := Save(sqldb, Profile{SkaterName: "keep", Slug: "mongo-link"})
	if err != nil || custom.Slug != "mongo-link" {
		t.Fatalf("custom slug %+v %v", custom, err)
	}
	custom.RealName = "Show Name"
	custom.Slug = ""
	custom, err = Save(sqldb, custom)
	if err != nil || custom.Slug != "show-name" {
		t.Fatalf("slug from display %+v %v", custom, err)
	}
	same, err := Save(sqldb, custom)
	if err != nil || same.Slug != "show-name" {
		t.Fatalf("own slug bump %+v %v", same, err)
	}
	gotOld, err := CurrentSlug(sqldb, "mongo-link")
	if err != nil || gotOld != "show-name" {
		t.Fatalf("redirect %q %v", gotOld, err)
	}
	custom.Slug = "mongo-link"
	custom, err = Save(sqldb, custom)
	if err != nil || custom.Slug != "mongo-link" {
		t.Fatalf("reclaim %+v %v", custom, err)
	}
	if _, err := CurrentSlug(sqldb, "mongo-link"); err != sql.ErrNoRows {
		t.Fatalf("live slug still redirects %v", err)
	}
	custom.Slug = "show-name"
	custom, err = Save(sqldb, custom)
	if err != nil {
		t.Fatal(err)
	}
	taken, err := Save(sqldb, Profile{SkaterName: "other", Slug: "mongo-link"})
	if err != nil || taken.Slug != "mongo-link" {
		t.Fatalf("reuse old slug %+v %v", taken, err)
	}
	if _, err := CurrentSlug(sqldb, "mongo-link"); err != sql.ErrNoRows {
		t.Fatalf("claimed slug still redirects %v", err)
	}
	custom.Slug = "mongo-link"
	custom, err = Save(sqldb, custom)
	if err != nil || custom.Slug != "mongo-link-2" {
		t.Fatalf("live collision %+v %v", custom, err)
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
	_, err = sqldb.Exec(`INSERT INTO users (id, username, display_name, role) VALUES ('u2','pal','Pal','friend')`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := EnsureForUser(sqldb, "u2", "pal"); err != nil {
		t.Fatal(err)
	}
	team, err = ListTeam(sqldb)
	friends, err3 := ListFriends(sqldb)
	if err != nil || err3 != nil || len(team) != 1 || len(friends) != 1 || friends[0].UserID != "u2" {
		t.Fatalf("roster team=%d friends=%d %v %v", len(team), len(friends), err, err3)
	}
	if friends[0].Status != "Friend" {
		t.Fatalf("friend status %q", friends[0].Status)
	}
}
