package admin

import (
	"testing"

	"seshhub/internal/db"
)

func TestStatsFrom(t *testing.T) {
	sqldb, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqldb.Close() })
	if err := db.Migrate(sqldb, "../db/migrations"); err != nil {
		t.Fatal(err)
	}
	s, err := StatsFrom(sqldb)
	if err != nil {
		t.Fatal(err)
	}
	if s.Articles != 0 || s.Videos != 0 {
		t.Fatalf("%+v", s)
	}
	_, err = sqldb.Exec(`INSERT INTO users (id, username, display_name, role) VALUES ('u','u','U','admin')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = sqldb.Exec(`INSERT INTO articles (id, slug, title, content_raw, content_html, author_id) VALUES ('a','s','t','x','<p>x</p>','u')`)
	if err != nil {
		t.Fatal(err)
	}
	s, err = StatsFrom(sqldb)
	if err != nil || s.Articles != 1 {
		t.Fatalf("%+v %v", s, err)
	}
}
