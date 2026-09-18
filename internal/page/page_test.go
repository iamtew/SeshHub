package page

import (
	"database/sql"
	"testing"

	"seshhub/internal/db"
)

func TestReservedAndSave(t *testing.T) {
	if !Reserved("admin") || !Reserved("spot") || !Reserved("episodes") || !Reserved("photos") || Reserved("about") {
		t.Fatal("reserved")
	}
	if !Reserved("about/privacy") || !Reserved("about/tos") {
		t.Fatal("legal reserved")
	}
	if !Reserved("team/x") || !Reserved("friends") {
		t.Fatal("nested reserved")
	}
	sqldb, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqldb.Close() })
	if err := db.Migrate(sqldb, "../db/migrations"); err != nil {
		t.Fatal(err)
	}
	p, err := Get(sqldb, "slug", "about")
	if err != nil || !p.Published || p.Title != "About" {
		t.Fatalf("seed about %+v %v", p, err)
	}
	if _, err := Get(sqldb, "slug", "about/privacy"); err != sql.ErrNoRows {
		t.Fatalf("privacy cms gone: %v", err)
	}
	if _, err := Save(sqldb, Page{Title: "Nope", Slug: "about/privacy", ContentRaw: "no"}); err == nil {
		t.Fatal("expected privacy reserved")
	}
	nested, err := Save(sqldb, Page{Title: "Extra", Slug: "about/privacy/extra", ContentRaw: "hi **x**", Published: true})
	if err != nil || nested.Slug != "about/privacy/extra" {
		t.Fatalf("%+v %v", nested, err)
	}
	if _, err := Save(sqldb, Page{Title: "Nope", Slug: "team/x", ContentRaw: "no"}); err == nil {
		t.Fatal("expected reserved")
	}
	team, err := Get(sqldb, "slug", "team")
	if err != nil || team.ID != TeamID || team.Title != "FS Team" {
		t.Fatalf("seed team %+v %v", team, err)
	}
	saved, err := Save(sqldb, Page{ID: TeamID, Title: "Crew", Slug: "stolen", ContentRaw: "# Crew", Published: true})
	if err != nil || saved.Slug != "team" || saved.Title != "Crew" {
		t.Fatalf("lock slug %+v %v", saved, err)
	}
	if _, err := Save(sqldb, Page{Title: "Nope", Slug: "team", ContentRaw: "no"}); err == nil {
		t.Fatal("expected reserved")
	}
	if err := Delete(sqldb, TeamID); err == nil {
		t.Fatal("expected locked")
	}
	if err := Delete(sqldb, AboutID); err == nil {
		t.Fatal("expected about locked")
	}
	about, err := Save(sqldb, Page{ID: AboutID, Title: "About us", Slug: "stolen", ContentRaw: "# About us", Published: true})
	if err != nil || about.Slug != "about" {
		t.Fatalf("lock about slug %+v %v", about, err)
	}
	friends, err := Get(sqldb, "slug", "friends")
	if err != nil || friends.ID != FriendsID {
		t.Fatalf("seed friends %+v %v", friends, err)
	}
	if err := Delete(sqldb, FriendsID); err == nil {
		t.Fatal("expected friends locked")
	}
	if _, err := Save(sqldb, Page{Title: "Nope", Slug: "friends", ContentRaw: "no"}); err == nil {
		t.Fatal("expected reserved")
	}
	photos, err := Get(sqldb, "slug", "photos")
	if err != nil || photos.ID != PhotosID {
		t.Fatalf("seed photos %+v %v", photos, err)
	}
	if err := Delete(sqldb, PhotosID); err == nil {
		t.Fatal("expected photos locked")
	}
	if _, err := Save(sqldb, Page{Title: "Nope", Slug: "photos", ContentRaw: "no"}); err == nil {
		t.Fatal("expected reserved")
	}
	videos, err := Get(sqldb, "slug", "videos")
	if err != nil || videos.ID != VideosID {
		t.Fatalf("seed videos %+v %v", videos, err)
	}
	if err := Delete(sqldb, VideosID); err == nil {
		t.Fatal("expected videos locked")
	}
	secret, err := Save(sqldb, Page{Title: "Secret", Slug: "secret", ContentRaw: "nope", Published: true, Visibility: "internal"})
	if err != nil || secret.Visibility != "internal" {
		t.Fatalf("internal save %+v %v", secret, err)
	}
	if Visible(secret, false, false) || !Visible(secret, true, false) {
		t.Fatal("Visible")
	}
	if !Visible(Page{Published: false}, false, true) {
		t.Fatal("admin draft")
	}
}
