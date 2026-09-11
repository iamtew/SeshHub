package db

import "testing"

func TestMigrateCreatesTables(t *testing.T) {
	sqldb, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqldb.Close() })
	if err := Migrate(sqldb, "migrations"); err != nil {
		t.Fatal(err)
	}
	tables := []string{"users", "sessions", "skater_profiles", "articles", "pages", "youtube_videos", "sync_logs"}
	for _, name := range tables {
		var n int
		err := sqldb.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, name).Scan(&n)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if n != 1 {
			t.Fatalf("missing table %s", name)
		}
	}
}
