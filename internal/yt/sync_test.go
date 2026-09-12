package yt

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"seshhub/internal/db"
)

func TestDurationAndClassify(t *testing.T) {
	if DurationSeconds("PT1H2M3S") != 3723 {
		t.Fatal("duration")
	}
	if Classify("Street #FullLength", "") != "part" {
		t.Fatal("classify")
	}
	if Classify("SOTW", "a #short clip") != "short" {
		t.Fatal("short")
	}
}

func TestSyncUpsertBearer(t *testing.T) {
	sqldb, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqldb.Close() })
	if err := db.Migrate(sqldb, "../db/migrations"); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/channels", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			http.Error(w, "no bearer", 401)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []any{map[string]any{"contentDetails": map[string]any{"relatedPlaylists": map[string]any{"uploads": "UU1"}}}},
		})
	})
	mux.HandleFunc("/playlistItems", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"nextPageToken": "p2",
			"items":         []any{map[string]any{"contentDetails": map[string]any{"videoId": "vid1"}}},
		})
	})
	mux.HandleFunc("/videos", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"items": []any{map[string]any{
				"id": "vid1",
				"snippet": map[string]any{
					"title": "Session", "description": "x", "publishedAt": "2026-01-01T00:00:00Z",
					"thumbnails": map[string]any{"high": map[string]any{"url": "http://t/i.jpg"}},
				},
				"contentDetails": map[string]any{"duration": "PT4M13S"},
				"statistics":     map[string]any{"viewCount": "10", "likeCount": "2"},
			}},
		})
	})
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	c := Client{Token: "tok", Channel: "ch", Base: ts.URL, HTTP: ts.Client()}
	res, err := c.Sync(context.Background(), sqldb)
	if err != nil {
		t.Fatal(err)
	}
	if res.Inserted != 1 || res.Fetched != 1 {
		t.Fatalf("%+v", res)
	}
	res, err = c.Sync(context.Background(), sqldb)
	if err != nil || res.Updated != 1 || res.Fetched != 1 {
		t.Fatalf("update %+v %v", res, err)
	}
	list, err := ListByChannel(sqldb, "ch", 6)
	if err != nil || len(list) != 1 || list[0].Duration != 253 {
		t.Fatalf("list %v %#v", err, list)
	}
}

func TestMaybeSyncUserNeedsProfile(t *testing.T) {
	sqldb, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqldb.Close() })
	if err := db.Migrate(sqldb, "../db/migrations"); err != nil {
		t.Fatal(err)
	}
	_, err = sqldb.Exec(`INSERT INTO users (id, username, display_name, role, youtube_channel_id, youtube_refresh_token) VALUES ('u','u','U','skater','ch','rt')`)
	if err != nil {
		t.Fatal(err)
	}
	called := false
	RefreshAccess = func(string, string, string) (string, string, error) {
		called = true
		return "tok", "", nil
	}
	t.Cleanup(func() {
		RefreshAccess = func(string, string, string) (string, string, error) {
			return "", "", nil
		}
	})
	if err := MaybeSyncUser(context.Background(), sqldb, "u", "cid", "sec"); err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("sync without roster profile")
	}
}
