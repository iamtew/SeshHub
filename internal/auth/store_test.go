package auth

import (
	"testing"

	"seshhub/internal/db"
)

func TestLinkAccounts(t *testing.T) {
	sqldb, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqldb.Close() })
	if err := db.Migrate(sqldb, "../db/migrations"); err != nil {
		t.Fatal(err)
	}
	d, err := UpsertDiscord(sqldb, "d1", "disc", "Disc", "", RoleMember)
	if err != nil {
		t.Fatal(err)
	}
	y, err := UpsertYouTube(sqldb, "ch1", "Chan", "", "refresh")
	if err != nil {
		t.Fatal(err)
	}
	if y.Role != RolePending {
		t.Fatal(y.Role)
	}
	if _, err := LinkYouTube(sqldb, d.ID, "ch2", "Other", "", "r2"); err != nil {
		t.Fatal(err)
	}
	got, err := GetUser(sqldb, d.ID)
	if err != nil || got.YouTubeChannelID != "ch2" {
		t.Fatalf("%+v %v", got, err)
	}
	if _, err := LinkYouTube(sqldb, d.ID, "ch1", "Chan", "", ""); err != ErrTaken {
		t.Fatalf("want taken, %v", err)
	}
	if _, err := LinkDiscord(sqldb, y.ID, "d1", "disc", "Disc", "", RoleMember); err != ErrTaken {
		t.Fatalf("want taken, %v", err)
	}
	linked, err := LinkDiscord(sqldb, y.ID, "d9", "nine", "Nine", "", RoleMember)
	if err != nil || linked.DiscordID != "d9" || linked.Role != RoleMember {
		t.Fatalf("%+v %v", linked, err)
	}
}
