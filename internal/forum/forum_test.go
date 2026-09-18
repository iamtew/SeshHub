package forum

import (
	"database/sql"
	"testing"

	"seshhub/internal/auth"
	"seshhub/internal/db"
)

func TestCreateUnreadReply(t *testing.T) {
	sqldb, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqldb.Close() })
	if err := db.Migrate(sqldb, "../db/migrations"); err != nil {
		t.Fatal(err)
	}
	alice, err := auth.UpsertDiscord(sqldb, "d1", "alice", "Alice", "", auth.RoleFriend, false)
	if err != nil {
		t.Fatal(err)
	}
	bob, err := auth.UpsertDiscord(sqldb, "d2", "bob", "Bob", "", auth.RoleFriend, false)
	if err != nil {
		t.Fatal(err)
	}
	sec, err := GetSection(sqldb, "general")
	if err != nil {
		t.Fatal(err)
	}
	th, err := CreateThread(sqldb, sec.ID, alice.ID, "Hello", "first body")
	if err != nil {
		t.Fatal(err)
	}
	n, err := UnreadCount(sqldb, alice.ID)
	if err != nil || n != 0 {
		t.Fatalf("own post unread %d %v", n, err)
	}
	n, err = UnreadCount(sqldb, bob.ID)
	if err != nil || n != 1 {
		t.Fatalf("bob unread %d %v", n, err)
	}
	if err := MarkRead(sqldb, bob.ID, th.ID); err != nil {
		t.Fatal(err)
	}
	n, err = UnreadCount(sqldb, bob.ID)
	if err != nil || n != 0 {
		t.Fatalf("after read %d %v", n, err)
	}
	if err := Reply(sqldb, th.ID, bob.ID, "reply"); err != nil {
		t.Fatal(err)
	}
	got, err := GetThread(sqldb, sec.ID, th.Slug)
	if err != nil {
		t.Fatal(err)
	}
	if got.ReplyCount != 1 {
		t.Fatalf("replies %d", got.ReplyCount)
	}
	n, err = UnreadCount(sqldb, alice.ID)
	if err != nil || n != 1 {
		t.Fatalf("alice after reply %d %v", n, err)
	}
	if err := DeleteSection(sqldb, sec.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := GetSection(sqldb, "general"); err != sql.ErrNoRows {
		t.Fatalf("section lingered %v", err)
	}
}
