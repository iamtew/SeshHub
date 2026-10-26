package spot

import (
	"encoding/json"
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

	var sh Show
	if err := json.Unmarshal([]byte(`{"episode":20,"name":"Creature Park","listeners":[{"kind":"content","name":"Spot","channel":"sesh-sofa-spot-challenge","playlist_id":"PLX"}]}`), &sh); err != nil {
		t.Fatal(err)
	}
	if sh.Episode != 20 || sh.Listeners[0].Channel != "sesh-sofa-spot-challenge" {
		t.Fatalf("%+v", sh)
	}
}

func TestImagesSplit(t *testing.T) {
	old := EpisodeURL
	EpisodeURL = "https://subotto.seshsofa.nl/api/get/episode/sesh-sofa"
	t.Cleanup(func() { EpisodeURL = old })
	urls := Images([]byte(`{"spot_images":["https://a.example/1.jpg","/media/2.jpg","spots/3.jpg","ftp://no","",{"url":"https://c.example/4.jpg"},{"url":"/media/5.jpg"}]}`))
	want := []string{
		"https://a.example/1.jpg",
		"https://subotto.seshsofa.nl/media/2.jpg",
		"https://subotto.seshsofa.nl/spots/3.jpg",
		"https://c.example/4.jpg",
		"https://subotto.seshsofa.nl/media/5.jpg",
	}
	if len(urls) != len(want) {
		t.Fatal(urls)
	}
	for i := range want {
		if urls[i] != want[i] {
			t.Fatal(urls)
		}
	}
	if n := len(Images([]byte(`{"spot_images":[]}`))); n != 0 {
		t.Fatal(n)
	}
	if Images(nil) != nil {
		t.Fatal("bad json")
	}
	b, a, ok := SplitImages("hello {{spot_images}} world {{spot_images}}")
	if !ok || b != "hello " || a != " world {{spot_images}}" {
		t.Fatalf("%q %q %v", b, a, ok)
	}
	if _, _, ok := SplitImages("no hole"); ok {
		t.Fatal("wanted no hole")
	}
}
