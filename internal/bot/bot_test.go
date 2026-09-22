package bot

import (
	"testing"

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
