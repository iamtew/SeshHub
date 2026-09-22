package bot

import (
	"testing"

	"github.com/bwmarrin/discordgo"

	"seshhub/internal/db"
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
