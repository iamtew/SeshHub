package bot

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"testing"

	"github.com/bwmarrin/discordgo"

	"seshhub/internal/auth"
	"seshhub/internal/db"
	"seshhub/internal/skater"
)

func TestAnnounceGate(t *testing.T) {
	sqldb, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqldb.Close() })
	if err := db.Migrate(sqldb, "../db/migrations"); err != nil {
		t.Fatal(err)
	}
	b := &Bot{db: sqldb}
	var got []string
	b.post = func(ch, content string) error {
		got = append(got, ch+" "+content)
		return nil
	}
	b.Announce("news", "News: Hello")
	if len(got) != 0 {
		t.Fatalf("off should not send, got %v", got)
	}
	if err := b.SaveSettings(Settings{ChannelID: "123", News: true}); err != nil {
		t.Fatal(err)
	}
	b.Announce("news", "News: Hello")
	b.Announce("forum", "Forum: nope")
	if len(got) != 1 || got[0] != "123 News: Hello" {
		t.Fatalf("%v", got)
	}
	if act := b.Activity(); len(act) != 1 || act[0].Kind != "news" || act[0].Text != "News: Hello" || act[0].Err != "" {
		t.Fatalf("%+v", act)
	}
}

func TestPostable(t *testing.T) {
	guild := &discordgo.Guild{
		ID: "g",
		Roles: []*discordgo.Role{
			{ID: "g", Permissions: discordgo.PermissionViewChannel},
			{ID: "r", Permissions: discordgo.PermissionViewChannel | discordgo.PermissionSendMessages},
		},
	}
	raw := []*discordgo.Channel{
		{ID: "cat", Name: "Sesh", Type: discordgo.ChannelTypeGuildCategory, Position: 1},
		{ID: "open", Name: "general", Type: discordgo.ChannelTypeGuildText, Position: 0},
		{ID: "news", Name: "announcements", Type: discordgo.ChannelTypeGuildNews, ParentID: "cat", Position: 2},
		{ID: "shut", Name: "mods", Type: discordgo.ChannelTypeGuildText, ParentID: "cat", Position: 1, PermissionOverwrites: []*discordgo.PermissionOverwrite{
			{ID: "r", Type: discordgo.PermissionOverwriteTypeRole, Deny: discordgo.PermissionSendMessages},
		}},
		{ID: "voice", Name: "talk", Type: discordgo.ChannelTypeGuildVoice},
	}
	got := postable(guild, raw, "bot", []string{"r"})
	if len(got) != 2 || got[0].Name != "#general" || got[1].Name != "Sesh / #announcements" {
		t.Fatalf("%+v", got)
	}
}

func TestHears(t *testing.T) {
	person := &discordgo.User{ID: "u"}
	botUser := &discordgo.User{ID: "bot"}
	home := &discordgo.Message{ChannelID: "home", Author: person}
	if !hears("home", "bot", home) {
		t.Fatal("home channel")
	}
	other := &discordgo.Message{ChannelID: "elsewhere", Author: person}
	if hears("home", "bot", other) || hears("", "bot", home) {
		t.Fatal("other channels and an unset home stay quiet")
	}
	mentioned := &discordgo.Message{ChannelID: "elsewhere", Author: person, Mentions: []*discordgo.User{botUser}}
	if !hears("home", "bot", mentioned) {
		t.Fatal("mention")
	}
	reply := &discordgo.Message{ChannelID: "elsewhere", Author: person, ReferencedMessage: &discordgo.Message{Author: botUser}}
	if !hears("home", "bot", reply) {
		t.Fatal("reply")
	}
	fromBot := &discordgo.Message{ChannelID: "home", Author: &discordgo.User{ID: "bot", Bot: true}}
	if hears("home", "bot", fromBot) {
		t.Fatal("bot messages")
	}
}

func tinyPNG() []byte {
	src := image.NewRGBA(image.Rect(0, 0, 20, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 20; x++ {
			src.Set(x, y, color.RGBA{B: 200, A: 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, src)
	return buf.Bytes()
}

func galleryBot(t *testing.T) *Bot {
	t.Helper()
	sqldb, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqldb.Close() })
	if err := db.Migrate(sqldb, "../db/migrations"); err != nil {
		t.Fatal(err)
	}
	t.Chdir(t.TempDir())
	if _, err := sqldb.Exec(`INSERT INTO users (id, username, display_name, role, discord_id) VALUES ('uid','bob','Bob','skater','u')`); err != nil {
		t.Fatal(err)
	}
	if _, err := sqldb.Exec(`INSERT INTO skater_profiles (id, user_id, slug, skater_name) VALUES ('p','uid','bob','Bob')`); err != nil {
		t.Fatal(err)
	}
	b := &Bot{db: sqldb, guildID: "g", selfID: "bot"}
	if err := b.SaveSettings(Settings{HomeChannelID: "home"}); err != nil {
		t.Fatal(err)
	}
	return b
}

func mentionMsg(id, ch string, atts []*discordgo.MessageAttachment) *discordgo.MessageCreate {
	return &discordgo.MessageCreate{Message: &discordgo.Message{
		ID: id, ChannelID: ch, GuildID: "g",
		Author:      &discordgo.User{ID: "u", Username: "bob"},
		Mentions:    []*discordgo.User{{ID: "bot"}},
		Attachments: atts,
	}}
}

func pngAtt(id, name string) *discordgo.MessageAttachment {
	return &discordgo.MessageAttachment{
		ID: id, Filename: name, ContentType: "image/png",
		URL: "https://cdn.discordapp.com/attachments/1/2/" + name,
	}
}

func TestGalleryMentionRequired(t *testing.T) {
	b := galleryBot(t)
	var got []string
	b.get = func(u string) (io.ReadCloser, error) {
		got = append(got, u)
		return io.NopCloser(bytes.NewReader(tinyPNG())), nil
	}
	b.onMessage(nil, &discordgo.MessageCreate{Message: &discordgo.Message{
		ID: "m", ChannelID: "home", GuildID: "g",
		Author:      &discordgo.User{ID: "u", Username: "bob"},
		Attachments: []*discordgo.MessageAttachment{pngAtt("a", "a.png")},
	}})
	if len(got) != 0 {
		t.Fatal("no mention")
	}
}

func TestGalleryMentionPhotos(t *testing.T) {
	b := galleryBot(t)
	looked := false
	b.history = func(string, string, int) ([]*discordgo.Message, error) {
		looked = true
		return nil, nil
	}
	var got []string
	b.get = func(u string) (io.ReadCloser, error) {
		got = append(got, u)
		return io.NopCloser(bytes.NewReader(tinyPNG())), nil
	}
	var emoji string
	b.react = func(ch, id, e string) error {
		if ch != "home" || id != "m1" {
			t.Fatalf("%s %s", ch, id)
		}
		emoji = e
		return nil
	}
	b.onMessage(nil, mentionMsg("m1", "home", []*discordgo.MessageAttachment{
		pngAtt("a", "a.png"),
		{ID: "g", Filename: "x.gif", ContentType: "image/gif", URL: "https://cdn.discordapp.com/attachments/1/2/x.gif"},
		pngAtt("b", "b.png"),
	}))
	if looked {
		t.Fatal("lookback")
	}
	if len(got) != 2 {
		t.Fatalf("gets %v", got)
	}
	if emoji != frameEmoji {
		t.Fatalf("emoji %q", emoji)
	}
	list, err := skater.ListByProfile(b.db, "p")
	if err != nil || len(list) != 2 {
		t.Fatalf("photos %d %v", len(list), err)
	}
}

func TestGalleryLookbackNearest(t *testing.T) {
	b := galleryBot(t)
	b.history = func(ch, before string, limit int) ([]*discordgo.Message, error) {
		if ch != "home" || before != "m1" || limit != 50 {
			t.Fatalf("%s %s %d", ch, before, limit)
		}
		return []*discordgo.Message{
			{ID: "p1", Author: &discordgo.User{ID: "other"}, Attachments: []*discordgo.MessageAttachment{pngAtt("x", "x.png")}},
			{ID: "p2", Author: &discordgo.User{ID: "u"}, Attachments: []*discordgo.MessageAttachment{
				{ID: "g", Filename: "x.gif", ContentType: "image/gif", URL: "https://cdn.discordapp.com/attachments/1/2/x.gif"},
			}},
			{ID: "p3", Author: &discordgo.User{ID: "u"}, Attachments: []*discordgo.MessageAttachment{pngAtt("keep", "keep.png")}},
			{ID: "p4", Author: &discordgo.User{ID: "u"}, Attachments: []*discordgo.MessageAttachment{pngAtt("old", "old.png")}},
		}, nil
	}
	var got []string
	b.get = func(u string) (io.ReadCloser, error) {
		got = append(got, u)
		return io.NopCloser(bytes.NewReader(tinyPNG())), nil
	}
	var emoji, reactID string
	b.react = func(ch, id, e string) error {
		if ch != "home" {
			t.Fatalf("ch %s", ch)
		}
		reactID, emoji = id, e
		return nil
	}
	b.onMessage(nil, mentionMsg("m1", "home", nil))
	if len(got) != 1 || got[0] != "https://cdn.discordapp.com/attachments/1/2/keep.png" {
		t.Fatalf("nearest %v", got)
	}
	if reactID != "p3" || emoji != frameEmoji {
		t.Fatalf("react %s %q", reactID, emoji)
	}
	list, err := skater.ListByProfile(b.db, "p")
	if err != nil || len(list) != 1 {
		t.Fatalf("photos %d %v", len(list), err)
	}
}

func TestGalleryShrug(t *testing.T) {
	b := galleryBot(t)
	b.history = func(string, string, int) ([]*discordgo.Message, error) {
		return []*discordgo.Message{
			{ID: "p1", Author: &discordgo.User{ID: "u"}, Attachments: []*discordgo.MessageAttachment{
				{ID: "g", Filename: "x.gif", ContentType: "image/gif", URL: "https://cdn.discordapp.com/attachments/1/2/x.gif"},
			}},
		}, nil
	}
	b.get = func(string) (io.ReadCloser, error) {
		t.Fatal("get")
		return nil, nil
	}
	var emoji string
	b.react = func(ch, id, e string) error {
		if ch != "home" || id != "m1" {
			t.Fatalf("%s %s", ch, id)
		}
		emoji = e
		return nil
	}
	b.onMessage(nil, mentionMsg("m1", "home", nil))
	if emoji != shrugEmoji {
		t.Fatalf("emoji %q", emoji)
	}
}

func TestGalleryPendingQuiet(t *testing.T) {
	b := galleryBot(t)
	if _, err := b.db.Exec(`UPDATE users SET role=? WHERE id='uid'`, auth.RolePending); err != nil {
		t.Fatal(err)
	}
	reacted := false
	b.react = func(string, string, string) error {
		reacted = true
		return nil
	}
	b.get = func(string) (io.ReadCloser, error) {
		t.Fatal("get")
		return nil, nil
	}
	b.onMessage(nil, mentionMsg("m1", "home", []*discordgo.MessageAttachment{pngAtt("a", "a.png")}))
	if reacted {
		t.Fatal("shrug")
	}
}

func TestGalleryCDNHost(t *testing.T) {
	b := galleryBot(t)
	b.get = func(string) (io.ReadCloser, error) {
		t.Fatal("get")
		return nil, nil
	}
	b.onMessage(nil, mentionMsg("m1", "home", []*discordgo.MessageAttachment{{
		ID: "a", Filename: "a.png", ContentType: "image/png",
		URL: "https://evil.example/a.png",
	}}))
	list, err := skater.ListByProfile(b.db, "p")
	if err != nil || len(list) != 0 {
		t.Fatalf("photos %d %v", len(list), err)
	}
}

func TestGalleryFilesWritten(t *testing.T) {
	b := galleryBot(t)
	b.get = func(string) (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(tinyPNG())), nil
	}
	b.onMessage(nil, mentionMsg("m1", "elsewhere", []*discordgo.MessageAttachment{pngAtt("a", "a.png")}))
	list, err := skater.ListByProfile(b.db, "p")
	if err != nil || len(list) != 1 {
		t.Fatalf("photos %d %v", len(list), err)
	}
	if _, err := os.Stat(skater.GalleryPath(list[0].ID)); err != nil {
		t.Fatal(err)
	}
}
