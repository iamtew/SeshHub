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

func TestDeleteUserErasure(t *testing.T) {
	sqldb, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqldb.Close() })
	if err := db.Migrate(sqldb, "../db/migrations"); err != nil {
		t.Fatal(err)
	}
	u, err := UpsertDiscord(sqldb, "d-erase", "erase", "Erase", "", RoleSkater, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := LinkYouTube(sqldb, u.ID, "ch-erase", "Chan", "", "refresh"); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateSession(sqldb, u.ID); err != nil {
		t.Fatal(err)
	}
	if err := RequestAccess(sqldb, u.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := sqldb.Exec(`INSERT INTO skater_profiles (id, user_id, slug, skater_name) VALUES ('p-erase', ?, 'erase', 'Erase')`, u.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := sqldb.Exec(`INSERT INTO skater_slug_redirects (slug, profile_id) VALUES ('old-erase', 'p-erase')`); err != nil {
		t.Fatal(err)
	}
	if _, err := sqldb.Exec(`INSERT INTO articles (id, slug, title, content_raw, content_html, author_id, status) VALUES ('a-erase', 'erase-post', 'Post', 'hi', 'hi', ?, 'published')`, u.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := sqldb.Exec(`INSERT INTO forum_threads (id, section_id, user_id, title, slug) VALUES ('t-erase', 'forum-general', ?, 'Hi', 'hi')`, u.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := sqldb.Exec(`INSERT INTO forum_posts (id, thread_id, user_id, body_raw, body_html, is_first_post) VALUES ('p-erase', 't-erase', ?, 'body', 'body', 1)`, u.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := sqldb.Exec(`INSERT INTO forum_thread_reads (user_id, thread_id) VALUES (?, 't-erase')`, u.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := sqldb.Exec(`INSERT INTO youtube_videos (id, channel_id, title, published_at, thumbnail_url) VALUES ('vid-erase', 'ch-erase', 'Clip', '2020-01-01', 'https://img')`); err != nil {
		t.Fatal(err)
	}
	if err := DeleteUser(sqldb, u.ID, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := GetUser(sqldb, u.ID); err == nil {
		t.Fatal("user still there")
	}
	var n int
	_ = sqldb.QueryRow(`SELECT COUNT(*) FROM skater_profiles WHERE id='p-erase'`).Scan(&n)
	if n != 0 {
		t.Fatal("profile survived")
	}
	_ = sqldb.QueryRow(`SELECT COUNT(*) FROM skater_slug_redirects WHERE profile_id='p-erase'`).Scan(&n)
	if n != 0 {
		t.Fatal("slug redirect survived")
	}
	_ = sqldb.QueryRow(`SELECT COUNT(*) FROM sessions WHERE user_id=?`, u.ID).Scan(&n)
	if n != 0 {
		t.Fatal("session survived")
	}
	_ = sqldb.QueryRow(`SELECT COUNT(*) FROM access_requests WHERE user_id=?`, u.ID).Scan(&n)
	if n != 0 {
		t.Fatal("access request survived")
	}
	_ = sqldb.QueryRow(`SELECT COUNT(*) FROM youtube_videos WHERE channel_id='ch-erase'`).Scan(&n)
	if n != 0 {
		t.Fatal("clips survived")
	}
	var author string
	if err := sqldb.QueryRow(`SELECT author_id FROM articles WHERE id='a-erase'`).Scan(&author); err != nil || author != TombstoneID {
		t.Fatalf("article author %q %v", author, err)
	}
	if err := sqldb.QueryRow(`SELECT user_id FROM forum_posts WHERE id='p-erase'`).Scan(&author); err != nil || author != TombstoneID {
		t.Fatalf("forum post user %q %v", author, err)
	}
	if err := sqldb.QueryRow(`SELECT user_id FROM forum_threads WHERE id='t-erase'`).Scan(&author); err != nil || author != TombstoneID {
		t.Fatalf("forum thread user %q %v", author, err)
	}
	_ = sqldb.QueryRow(`SELECT COUNT(*) FROM forum_thread_reads WHERE user_id=?`, u.ID).Scan(&n)
	if n != 0 {
		t.Fatal("forum reads survived")
	}
	list, err := ListUsers(sqldb)
	if err != nil {
		t.Fatal(err)
	}
	for _, got := range list {
		if got.ID == TombstoneID {
			t.Fatal("tombstone listed")
		}
	}
}
