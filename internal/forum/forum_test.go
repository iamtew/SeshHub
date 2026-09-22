package forum

import (
	"bytes"
	"database/sql"
	"image"
	"image/color"
	"image/png"
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
	th, err := CreateThread(sqldb, sec.ID, alice.ID, "Hello", "first body", nil)
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
	if err := Reply(sqldb, th.ID, bob.ID, "reply", "", nil); err != nil {
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
	if err := db.Migrate(sqldb, "../db/migrations"); err != nil {
		t.Fatal(err)
	}
	if _, err := GetSection(sqldb, "general"); err != sql.ErrNoRows {
		t.Fatalf("migrate reseeded %v", err)
	}
}

func TestReplyParentMentionsAndPhotos(t *testing.T) {
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
	if err := CreateSection(sqldb, "Mentions", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := GetSection(sqldb, "mentions"); err != sql.ErrNoRows {
		t.Fatal("mentions slug")
	}
	th, err := CreateThread(sqldb, sec.ID, alice.ID, "Hello", "hey @bob", nil)
	if err != nil {
		t.Fatal(err)
	}
	n, err := UnreadMentions(sqldb, bob.ID)
	if err != nil || n != 1 {
		t.Fatalf("mention unread %d %v", n, err)
	}
	if err := MarkMentionsRead(sqldb, bob.ID); err != nil {
		t.Fatal(err)
	}
	n, err = UnreadMentions(sqldb, bob.ID)
	if err != nil || n != 0 {
		t.Fatalf("after mention read %d %v", n, err)
	}
	if err := Reply(sqldb, th.ID, bob.ID, "nope @missing", "", nil); err == nil {
		t.Fatal("unknown mention")
	}
	posts, err := ListPosts(sqldb, th.ID)
	if err != nil || len(posts) != 1 {
		t.Fatalf("posts %d %v", len(posts), err)
	}
	png := tinyPNG()
	files := []FileIn{{R: bytes.NewReader(png)}, {R: bytes.NewReader(png)}, {R: bytes.NewReader(png)}, {R: bytes.NewReader(png)}}
	if err := Reply(sqldb, th.ID, bob.ID, "too many", posts[0].ID, files); err == nil {
		t.Fatal("4 photos")
	}
	if err := Reply(sqldb, th.ID, bob.ID, "mid", "", nil); err != nil {
		t.Fatal(err)
	}
	posts, err = ListPosts(sqldb, th.ID)
	if err != nil || len(posts) != 2 {
		t.Fatalf("mid %d %v", len(posts), err)
	}
	mid := posts[1]
	if err := Reply(sqldb, th.ID, alice.ID, "quote you", mid.ID, []FileIn{{R: bytes.NewReader(png)}}); err != nil {
		t.Fatal(err)
	}
	posts, err = ListPosts(sqldb, th.ID)
	if err != nil || len(posts) != 3 {
		t.Fatalf("after quote %d %v", len(posts), err)
	}
	if posts[2].ParentID != mid.ID || posts[2].ParentAuthor != "Bob" {
		t.Fatalf("parent %+v", posts[2])
	}
	if len(posts[2].Photos) != 1 {
		t.Fatalf("photos %d", len(posts[2].Photos))
	}
	photoID := posts[2].Photos[0].ID
	t.Cleanup(func() { RemovePhotos([]string{photoID}) })
	if _, err := DeletePost(sqldb, mid); err != nil {
		t.Fatal(err)
	}
	posts, err = ListPosts(sqldb, th.ID)
	if err != nil || len(posts) != 2 {
		t.Fatalf("after parent delete %d %v", len(posts), err)
	}
	if posts[1].ParentID != "" {
		t.Fatalf("parent lingered %q", posts[1].ParentID)
	}
}

func tinyPNG() []byte {
	src := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			src.Set(x, y, color.RGBA{G: 200, A: 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, src)
	return buf.Bytes()
}

func TestCanEditOwnerOnly(t *testing.T) {
	if !CanEdit("a", "a") || CanEdit("admin", "a") || CanEdit("", "") {
		t.Fatal("edit")
	}
	if !CanDelete("admin", "x", "a") || !CanDelete("friend", "a", "a") || CanDelete("friend", "x", "a") {
		t.Fatal("delete")
	}
}
