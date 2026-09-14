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
	d, err := UpsertDiscord(sqldb, "d1", "disc", "Disc", "", RoleMember, false)
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
	if _, err := LinkDiscord(sqldb, y.ID, "d1", "disc", "Disc", "", RoleMember, false); err != ErrTaken {
		t.Fatalf("want taken, %v", err)
	}
	if _, err := LinkDiscord(sqldb, y.ID, "d9", "nine", "Nine", "", RoleMember, false); err != nil {
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
	d, err := UpsertDiscord(sqldb, "d1", "disc", "Disc", "", RoleMember, false)
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
	d2, err := UpsertDiscord(sqldb, "d2", "two", "Two", "", RoleMember, false)
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

func TestHostFlag(t *testing.T) {
	sqldb, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqldb.Close() })
	if err := db.Migrate(sqldb, "../db/migrations"); err != nil {
		t.Fatal(err)
	}
	h, err := UpsertDiscord(sqldb, "h1", "host", "Host", "", RoleMember, true)
	if err != nil || !h.Host {
		t.Fatalf("insert %+v %v", h, err)
	}
	h, err = UpsertDiscord(sqldb, "h1", "host", "Host", "", RoleMember, false)
	if err != nil || h.Host {
		t.Fatalf("login without role must clear host %+v %v", h, err)
	}
	h, err = UpsertDiscord(sqldb, "h1", "host", "Host", "", RoleMember, true)
	if err != nil || !h.Host {
		t.Fatal(err)
	}
	y, err := UpsertYouTube(sqldb, "ch-h", "Chan", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := MergeUsers(sqldb, y.ID, h.ID); err != nil {
		t.Fatal(err)
	}
	got, err := GetUser(sqldb, y.ID)
	if err != nil || !got.Host || got.DiscordID != "h1" {
		t.Fatalf("merge host %+v %v", got, err)
	}
	if err := UnlinkDiscord(sqldb, y.ID); err != nil {
		t.Fatal(err)
	}
	got, err = GetUser(sqldb, y.ID)
	if err != nil || got.Host {
		t.Fatalf("unlink must drop host %+v %v", got, err)
	}
}
