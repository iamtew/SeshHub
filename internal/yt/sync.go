package yt

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const apiDefault = "https://www.googleapis.com/youtube/v3"

var runMu sync.Mutex

type Video struct {
	ID          string
	ChannelID   string
	Title       string
	Description string
	PublishedAt string
	Thumb       string
	Duration    int
	Views       int
	Likes       int
	Tags        string
	Category    string
}

type Log struct {
	ID        string
	Status    string
	Fetched   int
	Inserted  int
	Updated   int
	Error     string
	Started   string
	Completed string
}

func DurationSeconds(iso string) int {
	iso = strings.TrimPrefix(iso, "PT")
	var h, m, s int
	var n strings.Builder
	for _, r := range iso {
		if r >= '0' && r <= '9' {
			n.WriteRune(r)
			continue
		}
		v, _ := strconv.Atoi(n.String())
		n.Reset()
		switch r {
		case 'H':
			h = v
		case 'M':
			m = v
		case 'S':
			s = v
		}
	}
	return h*3600 + m*60 + s
}

func Classify(title, desc string) string {
	s := strings.ToLower(title + " " + desc)
	switch {
	case strings.Contains(s, "#fulllength") || strings.Contains(s, "full length"):
		return "part"
	case strings.Contains(s, "#contest"):
		return "contest"
	case strings.Contains(s, "#short"):
		return "short"
	default:
		return "session"
	}
}

func thumbURL(thumbs map[string]struct {
	URL string `json:"url"`
}) string {
	for _, k := range []string{"maxres", "standard", "high", "medium", "default"} {
		if t, ok := thumbs[k]; ok && t.URL != "" {
			return t.URL
		}
	}
	return ""
}

func newID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

type Client struct {
	Key, Channel, Base string
	HTTP               *http.Client
}

func (c Client) base() string {
	if c.Base != "" {
		return strings.TrimRight(c.Base, "/")
	}
	return apiDefault
}

func (c Client) http() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 20 * time.Second}
}

func (c Client) get(ctx context.Context, path string, q url.Values) ([]byte, error) {
	q.Set("key", c.Key)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base()+path+"?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("youtube %s: %s", resp.Status, b)
	}
	return b, nil
}

type Result struct {
	Fetched, Inserted, Updated int
}

// ponytail: cap at 20 playlist pages (1000 videos); raise if the channel outgrows that.
func (c Client) Sync(ctx context.Context, db *sql.DB) (Result, error) {
	if !runMu.TryLock() {
		return Result{}, fmt.Errorf("sync already running")
	}
	defer runMu.Unlock()

	logID := newID()
	_, _ = db.Exec(`INSERT INTO sync_logs (id, service, status) VALUES (?, 'youtube', 'running')`, logID)
	res, err := c.sync(ctx, db)
	st, msg := "success", ""
	if err != nil {
		st, msg = "error", err.Error()
	}
	_, _ = db.Exec(`UPDATE sync_logs SET status=?, videos_fetched=?, videos_inserted=?, videos_updated=?, error_message=?, completed_at=CURRENT_TIMESTAMP WHERE id=?`,
		st, res.Fetched, res.Inserted, res.Updated, nullEmpty(msg), logID)
	return res, err
}

func nullEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func (c Client) sync(ctx context.Context, db *sql.DB) (Result, error) {
	var out Result
	b, err := c.get(ctx, "/channels", url.Values{"part": {"contentDetails"}, "id": {c.Channel}})
	if err != nil {
		return out, err
	}
	var ch struct {
		Items []struct {
			ContentDetails struct {
				RelatedPlaylists struct {
					Uploads string `json:"uploads"`
				} `json:"relatedPlaylists"`
			} `json:"contentDetails"`
		} `json:"items"`
	}
	if err := json.Unmarshal(b, &ch); err != nil || len(ch.Items) == 0 {
		return out, fmt.Errorf("no uploads playlist")
	}
	uploads := ch.Items[0].ContentDetails.RelatedPlaylists.Uploads
	var ids []string
	token := ""
	for page := 0; page < 20; page++ {
		q := url.Values{"part": {"contentDetails"}, "playlistId": {uploads}, "maxResults": {"50"}}
		if token != "" {
			q.Set("pageToken", token)
		}
		pb, err := c.get(ctx, "/playlistItems", q)
		if err != nil {
			return out, err
		}
		var pl struct {
			NextPageToken string `json:"nextPageToken"`
			Items         []struct {
				ContentDetails struct {
					VideoID string `json:"videoId"`
				} `json:"contentDetails"`
			} `json:"items"`
		}
		if err := json.Unmarshal(pb, &pl); err != nil {
			return out, err
		}
		for _, it := range pl.Items {
			if it.ContentDetails.VideoID != "" {
				ids = append(ids, it.ContentDetails.VideoID)
			}
		}
		if pl.NextPageToken == "" {
			break
		}
		token = pl.NextPageToken
	}
	out.Fetched = len(ids)
	if len(ids) == 0 {
		return out, nil
	}
	for i := 0; i < len(ids); i += 50 {
		end := i + 50
		if end > len(ids) {
			end = len(ids)
		}
		if err := c.upsertChunk(ctx, db, ids[i:end], &out); err != nil {
			return out, err
		}
	}
	return out, nil
}

func (c Client) upsertChunk(ctx context.Context, db *sql.DB, ids []string, out *Result) error {
	vb, err := c.get(ctx, "/videos", url.Values{"part": {"snippet,contentDetails,statistics"}, "id": {strings.Join(ids, ",")}})
	if err != nil {
		return err
	}
	var vs struct {
		Items []struct {
			ID      string `json:"id"`
			Snippet struct {
				Title       string `json:"title"`
				Description string `json:"description"`
				PublishedAt string `json:"publishedAt"`
				Tags        []string `json:"tags"`
				Thumbnails  map[string]struct {
					URL string `json:"url"`
				} `json:"thumbnails"`
			} `json:"snippet"`
			ContentDetails struct {
				Duration string `json:"duration"`
			} `json:"contentDetails"`
			Statistics struct {
				ViewCount string `json:"viewCount"`
				LikeCount string `json:"likeCount"`
			} `json:"statistics"`
		} `json:"items"`
	}
	if err := json.Unmarshal(vb, &vs); err != nil {
		return err
	}
	for _, it := range vs.Items {
		views, _ := strconv.Atoi(it.Statistics.ViewCount)
		likes, _ := strconv.Atoi(it.Statistics.LikeCount)
		v := Video{
			ID: it.ID, ChannelID: c.Channel, Title: it.Snippet.Title, Description: it.Snippet.Description,
			PublishedAt: it.Snippet.PublishedAt, Thumb: thumbURL(it.Snippet.Thumbnails),
			Duration: DurationSeconds(it.ContentDetails.Duration), Views: views, Likes: likes,
			Tags: strings.Join(it.Snippet.Tags, ", "), Category: Classify(it.Snippet.Title, it.Snippet.Description),
		}
		ins, err := upsert(db, v)
		if err != nil {
			return err
		}
		if ins {
			out.Inserted++
		} else {
			out.Updated++
		}
	}
	return nil
}

func upsert(db *sql.DB, v Video) (inserted bool, err error) {
	var exists int
	_ = db.QueryRow(`SELECT 1 FROM youtube_videos WHERE id = ?`, v.ID).Scan(&exists)
	if exists == 1 {
		_, err = db.Exec(`UPDATE youtube_videos SET title=?, description=?, published_at=?, thumbnail_url=?, duration_seconds=?, view_count=?, like_count=?, tags=?, category=?, synced_at=CURRENT_TIMESTAMP WHERE id=?`,
			v.Title, v.Description, v.PublishedAt, v.Thumb, v.Duration, v.Views, v.Likes, nullEmpty(v.Tags), v.Category, v.ID)
		return false, err
	}
	_, err = db.Exec(`INSERT INTO youtube_videos (id, channel_id, title, description, published_at, thumbnail_url, duration_seconds, view_count, like_count, tags, category) VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.ChannelID, v.Title, nullEmpty(v.Description), v.PublishedAt, v.Thumb, v.Duration, v.Views, v.Likes, nullEmpty(v.Tags), v.Category)
	return true, err
}

func ListPublic(db *sql.DB) ([]Video, error) {
	rows, err := db.Query(`SELECT id, channel_id, title, IFNULL(description,''), published_at, thumbnail_url, duration_seconds, view_count, like_count, IFNULL(tags,''), IFNULL(category,'') FROM youtube_videos WHERE is_hidden = 0 ORDER BY published_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Video
	for rows.Next() {
		var v Video
		if err := rows.Scan(&v.ID, &v.ChannelID, &v.Title, &v.Description, &v.PublishedAt, &v.Thumb, &v.Duration, &v.Views, &v.Likes, &v.Tags, &v.Category); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func Get(db *sql.DB, id string) (Video, error) {
	var v Video
	if id == "" {
		return v, sql.ErrNoRows
	}
	err := db.QueryRow(`SELECT id, channel_id, title, IFNULL(description,''), published_at, thumbnail_url, duration_seconds, view_count, like_count, IFNULL(tags,''), IFNULL(category,'') FROM youtube_videos WHERE id = ? AND is_hidden = 0`, id).
		Scan(&v.ID, &v.ChannelID, &v.Title, &v.Description, &v.PublishedAt, &v.Thumb, &v.Duration, &v.Views, &v.Likes, &v.Tags, &v.Category)
	return v, err
}

func RecentLogs(db *sql.DB, n int) ([]Log, error) {
	rows, err := db.Query(`SELECT id, status, videos_fetched, videos_inserted, videos_updated, IFNULL(error_message,''), started_at, IFNULL(completed_at,'') FROM sync_logs WHERE service='youtube' ORDER BY started_at DESC LIMIT ?`, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Log
	for rows.Next() {
		var l Log
		if err := rows.Scan(&l.ID, &l.Status, &l.Fetched, &l.Inserted, &l.Updated, &l.Error, &l.Started, &l.Completed); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func Loop(ctx context.Context, db *sql.DB, c Client, every time.Duration) {
	if c.Key == "" || c.Channel == "" {
		return
	}
	if every <= 0 {
		every = time.Hour
	}
	run := func() {
		if _, err := c.Sync(ctx, db); err != nil {
			// logged in sync_logs
		}
	}
	run()
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			run()
		}
	}
}
