package episode

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"seshhub/internal/db"
	"seshhub/internal/spot"
)

func TestParseIDsAndVisible(t *testing.T) {
	if ParseVideoID("https://youtu.be/bIP51Oyfc2U") != "bIP51Oyfc2U" {
		t.Fatal("youtu.be")
	}
	if ParseVideoID("https://www.youtube.com/watch?v=jmRgohBsm-g&t=1") != "jmRgohBsm-g" {
		t.Fatal("watch")
	}
	if ParsePlaylistID("https://www.youtube.com/playlist?list=PLckiUs3F1znI") != "PLckiUs3F1znI" {
		t.Fatal("playlist")
	}
	e := Episode{WinnerName: "", Rows: []Row{{Kind: "winner", URL: "https://youtu.be/xx"}, {Kind: "trick", URL: ""}, {Kind: "full", URL: "https://youtu.be/bIP51Oyfc2U"}}}
	vis := e.VisibleRows()
	if len(vis) != 1 || vis[0].Kind != "full" {
		t.Fatalf("%+v", vis)
	}
	e.WinnerName = "Smolin"
	e.WinnerVideoID = "jmRgohBsm-g"
	vis = e.VisibleRows()
	if len(vis) != 2 || vis[0].Kind != "winner" && vis[1].Kind != "winner" {
		t.Fatalf("%+v", vis)
	}
}

func TestEnsureCurrentOnce(t *testing.T) {
	body := []byte(`{"episode":99,"name":"Test Spot","listeners":[
		{"kind":"content","name":"EP99 Challenge","channel":"sesh-sofa-spot-challenge","playlist_id":"PLchal"},
		{"kind":"content","name":"EP99 Community","channel":"sesh-sofa-content","playlist_id":"PLcom"}
	]}`)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	}))
	t.Cleanup(ts.Close)
	old := spot.EpisodeURL
	spot.EpisodeURL = ts.URL
	spot.ClearCache()
	t.Cleanup(func() { spot.EpisodeURL = old; spot.ClearCache() })

	sqldb, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqldb.Close() })
	if err := db.Migrate(sqldb, "../db/migrations"); err != nil {
		t.Fatal(err)
	}
	EnsureCurrent(sqldb)
	e, err := Get(sqldb, 99)
	if err != nil || e.Title != "Test Spot" || e.ChallengePlaylistID != "PLchal" || len(e.Rows) != 2 {
		t.Fatalf("%+v %v", e, err)
	}
	e.WinnerName = "KeepMe"
	e.Title = "Edited"
	if err := Save(sqldb, e); err != nil {
		t.Fatal(err)
	}
	EnsureCurrent(sqldb)
	e2, err := Get(sqldb, 99)
	if err != nil || e2.WinnerName != "KeepMe" || e2.Title != "Edited" {
		t.Fatalf("%+v %v", e2, err)
	}

	seed, err := Get(sqldb, 19)
	if err != nil || seed.WinnerName != "Smolin" {
		t.Fatalf("backfill %+v %v", seed, err)
	}
}

func TestFromShowJSON(t *testing.T) {
	var sh spot.Show
	if err := json.Unmarshal([]byte(`{"episode":20,"name":"Park","listeners":[{"kind":"content","channel":"sesh-sofa-spot-challenge","playlist_id":"PLX","name":"Chal"}]}`), &sh); err != nil {
		t.Fatal(err)
	}
	e := FromShow(sh)
	if e.ChallengePlaylistID != "PLX" || len(e.Rows) != 1 {
		t.Fatalf("%+v", e)
	}
}
