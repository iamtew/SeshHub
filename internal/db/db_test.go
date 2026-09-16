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
	if err := Migrate(sqldb, "migrations"); err != nil {
		t.Fatal(err)
	}
	tables := []string{"users", "sessions", "skater_profiles", "skater_slug_redirects", "articles", "pages", "youtube_videos", "sync_logs", "access_requests"}
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

func TestMigrateMemberToFriend(t *testing.T) {
	sqldb, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqldb.Close() })
	if err := Migrate(sqldb, "migrations"); err != nil {
		t.Fatal(err)
	}
	_, err = sqldb.Exec(`INSERT INTO users (id, username, display_name, role) VALUES
		('m1','mem','Mem','member'),
		('s1','sk','Sk','skater'),
		('a1','ad','Ad','admin')`)
	if err != nil {
		t.Fatal(err)
	}
	if err := Migrate(sqldb, "migrations"); err != nil {
		t.Fatal(err)
	}
	var mem, sk, ad string
	if err := sqldb.QueryRow(`SELECT role FROM users WHERE id='m1'`).Scan(&mem); err != nil || mem != "friend" {
		t.Fatalf("member→friend %q %v", mem, err)
	}
	if err := sqldb.QueryRow(`SELECT role FROM users WHERE id='s1'`).Scan(&sk); err != nil || sk != "skater" {
		t.Fatalf("skater %q %v", sk, err)
	}
	if err := sqldb.QueryRow(`SELECT role FROM users WHERE id='a1'`).Scan(&ad); err != nil || ad != "admin" {
		t.Fatalf("admin %q %v", ad, err)
	}
	var n int
	if err := sqldb.QueryRow(`SELECT COUNT(*) FROM skater_profiles WHERE user_id='m1'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("friend profile %d %v", n, err)
	}
	if err := sqldb.QueryRow(`SELECT COUNT(*) FROM skater_profiles WHERE user_id IN ('s1','a1','deleted-user')`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("no extra profiles %d %v", n, err)
	}
	var tomb string
	if err := sqldb.QueryRow(`SELECT role FROM users WHERE id='deleted-user'`).Scan(&tomb); err != nil || tomb != "member" {
		t.Fatalf("tombstone %q %v", tomb, err)
	}
}
