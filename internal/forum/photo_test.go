package forum

import (
	"bytes"
	"testing"

	"seshhub/internal/auth"
	"seshhub/internal/db"
)

func TestSniffAndAttach(t *testing.T) {
	k, e, _, err := Sniff(tinyPNG())
	if err != nil || k != "image" || e != "png" {
		t.Fatalf("png %s %s %v", k, e, err)
	}
	mp3 := append([]byte("ID3"), make([]byte, 20)...)
	k, e, _, err = Sniff(mp3)
	if err != nil || k != "audio" || e != "mp3" {
		t.Fatalf("mp3 %s %s %v", k, e, err)
	}
	if _, _, _, err := Sniff([]byte("<html>hello!!")); err == nil {
		t.Fatal("html")
	}
	if _, _, _, err := Sniff([]byte("<svg xmlns='x")); err == nil {
		t.Fatal("svg")
	}

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
	sec, err := GetSection(sqldb, "general")
	if err != nil {
		t.Fatal(err)
	}
	th, err := CreateThread(sqldb, sec.ID, alice.ID, "Files", "body", []FileIn{{R: bytes.NewReader(tinyPNG())}})
	if err != nil {
		t.Fatal(err)
	}
	posts, err := ListPosts(sqldb, th.ID)
	if err != nil || len(posts) != 1 || len(posts[0].Photos) != 1 {
		t.Fatalf("png post %+v %v", posts, err)
	}
	png := posts[0].Photos[0]
	if png.Kind != "image" || png.Ext != "jpg" || !png.IsImage() {
		t.Fatalf("png saved %+v", png)
	}
	t.Cleanup(func() { RemovePhotos([]string{png.ID + "." + png.Ext}) })

	if err := Reply(sqldb, th.ID, alice.ID, "listen", "", []FileIn{{R: bytes.NewReader(mp3)}}); err != nil {
		t.Fatal(err)
	}
	posts, err = ListPosts(sqldb, th.ID)
	if err != nil || len(posts) != 2 || len(posts[1].Photos) != 1 {
		t.Fatalf("audio post %+v %v", posts, err)
	}
	au := posts[1].Photos[0]
	if au.Kind != "audio" || au.Ext != "mp3" || !au.IsAudio() {
		t.Fatalf("audio saved %+v", au)
	}
	t.Cleanup(func() { RemovePhotos([]string{au.ID + "." + au.Ext}) })

	if err := Reply(sqldb, th.ID, alice.ID, "nope", "", []FileIn{{R: bytes.NewReader([]byte("<html>hello!!"))}}); err == nil {
		t.Fatal("html attach")
	}
}
