package spot

import (
	"strings"
	"testing"

	"seshhub/internal/db"
)

func TestFlattenFill(t *testing.T) {
	vals, err := FlattenJSON([]byte(`{
		"episode_short":"EP20","name":"Creature Park","episode":20,
		"listeners":[{"name":"Spot Challenge","playlist_id":"PLXYa4OOkYczs"}]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if vals["episode_short"] != "EP20" || vals["episode"] != "20" || vals["listeners.0.playlist_id"] != "PLXYa4OOkYczs" {
		t.Fatalf("%v", vals)
	}
	got := Fill("**{{episode_short}}** {{missing}} {{listeners.0.name}}", vals)
	if got != "**EP20**  Spot Challenge" {
		t.Fatal(got)
	}
	items := Items(vals)
	if len(items) != 5 || !strings.HasPrefix(items[0].Placeholder, "{{") {
		t.Fatalf("%+v", items)
	}

	sqldb, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqldb.Close() })
	if err := db.Migrate(sqldb, "../db/migrations"); err != nil {
		t.Fatal(err)
	}
	if err := Save(sqldb, Config{ContentRaw: "hi {{name}}"}); err != nil {
		t.Fatal(err)
	}
	gotCfg, err := Get(sqldb)
	if err != nil || gotCfg.ContentRaw != "hi {{name}}" {
		t.Fatalf("%+v %v", gotCfg, err)
	}
}
