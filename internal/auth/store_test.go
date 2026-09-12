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
	if _, err := LinkDiscord(sqldb, y.ID, "d9", "nine", "Nine", "", RoleMember); err != nil {
		t.Fatal(err)
	}
}

func TestMergeUnlinkDelete(t *testing.T) {
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
	if err := MergeUsers(sqldb, d.ID, y.ID); err != nil {
		t.Fatal(err)
	}
	got, err := GetUser(sqldb, d.ID)
	if err != nil || got.DiscordID != "d1" || got.YouTubeChannelID != "ch1" || got.Role != RoleMember {
		t.Fatalf("%+v %v", got, err)
	}
	if _, err := GetUser(sqldb, y.ID); err == nil {
		t.Fatal("donor should be gone")
	}
	d2, err := UpsertDiscord(sqldb, "d2", "two", "Two", "", RoleMember)
	if err != nil {
		t.Fatal(err)
	}
	if err := MergeUsers(sqldb, d.ID, d2.ID); err == nil {
		t.Fatal("both Discord should fail")
	}
	if err := UnlinkYouTube(sqldb, d.ID); err != nil {
		t.Fatal(err)
	}
	cleared, err := GetUser(sqldb, d.ID)
	if err != nil || cleared.YouTubeChannelID != "" {
		t.Fatalf("unlink %+v %v", cleared, err)
	}
	if err := DeleteUser(sqldb, d.ID, d.ID); err != ErrDeleteSelf {
		t.Fatalf("self %v", err)
	}
	if err := DeleteUser(sqldb, d2.ID, d.ID); err != nil {
		t.Fatal(err)
	}
}
