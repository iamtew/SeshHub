package yt

import (
	"context"
	"database/sql"
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

var userMu sync.Map

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

type Client struct {
	Token, Channel, Base string
	HTTP                 *http.Client
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
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base()+path+"?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
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

func lockUser(id string) (*sync.Mutex, bool) {
	v, _ := userMu.LoadOrStore(id, &sync.Mutex{})
	mu := v.(*sync.Mutex)
	if !mu.TryLock() {
		return nil, false
	}
	return mu, true
}

// ponytail: one playlist page (50 videos) per skater; page further if a channel outgrows that.
func (c Client) Sync(ctx context.Context, db *sql.DB) (Result, error) {
	key := c.Channel
	if key == "" {
		key = "sync"
	}
	mu, ok := lockUser(key)
	if !ok {
		return Result{}, fmt.Errorf("sync already running")
	}
	defer mu.Unlock()
	return c.sync(ctx, db)
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
	q := url.Values{"part": {"contentDetails"}, "playlistId": {uploads}, "maxResults": {"50"}}
	pb, err := c.get(ctx, "/playlistItems", q)
	if err != nil {
		return out, err
	}
	var pl struct {
		Items []struct {
			ContentDetails struct {
				VideoID string `json:"videoId"`
			} `json:"contentDetails"`
		} `json:"items"`
	}
	if err := json.Unmarshal(pb, &pl); err != nil {
		return out, err
	}
	var ids []string
	for _, it := range pl.Items {
		if it.ContentDetails.VideoID != "" {
			ids = append(ids, it.ContentDetails.VideoID)
		}
	}
	out.Fetched = len(ids)
	if len(ids) == 0 {
		return out, nil
	}
	return out, c.upsertChunk(ctx, db, ids, &out)
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
				Title       string   `json:"title"`
				Description string   `json:"description"`
				PublishedAt string   `json:"publishedAt"`
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
	return list(db, "", 0, 0)
}

func ListPublicPage(db *sql.DB, limit, offset int) ([]Video, error) {
	return list(db, "", limit, offset)
}

func CountPublic(db *sql.DB) (int, error) {
	var n int
	err := db.QueryRow(`SELECT COUNT(*) FROM youtube_videos WHERE is_hidden = 0`).Scan(&n)
	return n, err
}

func ListByChannel(db *sql.DB, channelID string, limit int) ([]Video, error) {
	if channelID == "" {
		return nil, nil
	}
	return list(db, channelID, limit, 0)
}

func list(db *sql.DB, channelID string, limit, offset int) ([]Video, error) {
	q := `SELECT id, channel_id, title, IFNULL(description,''), published_at, thumbnail_url, duration_seconds, view_count, like_count, IFNULL(tags,''), IFNULL(category,'') FROM youtube_videos WHERE is_hidden = 0`
	var args []any
	if channelID != "" {
		q += ` AND channel_id = ?`
		args = append(args, channelID)
	}
	q += ` ORDER BY published_at DESC`
	if limit > 0 {
		q += ` LIMIT ?`
		args = append(args, limit)
		if offset > 0 {
			q += ` OFFSET ?`
			args = append(args, offset)
		}
	}
	rows, err := db.Query(q, args...)
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

func GetMany(db *sql.DB, ids []string) map[string]Video {
	seen := map[string]bool{}
	var uniq []string
	for _, id := range ids {
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		uniq = append(uniq, id)
	}
	if len(uniq) == 0 {
		return nil
	}
	q := `SELECT id, channel_id, title, IFNULL(description,''), published_at, thumbnail_url, duration_seconds, view_count, like_count, IFNULL(tags,''), IFNULL(category,'') FROM youtube_videos WHERE is_hidden = 0 AND id IN (` + strings.Repeat("?,", len(uniq)-1) + `?)`
	args := make([]any, len(uniq))
	for i, id := range uniq {
		args[i] = id
	}
	rows, err := db.Query(q, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	out := map[string]Video{}
	for rows.Next() {
		var v Video
		if err := rows.Scan(&v.ID, &v.ChannelID, &v.Title, &v.Description, &v.PublishedAt, &v.Thumb, &v.Duration, &v.Views, &v.Likes, &v.Tags, &v.Category); err != nil {
			return out
		}
		out[v.ID] = v
	}
	return out
}

func staleSync(s string) bool {
	if s == "" {
		return true
	}
	t, err := time.ParseInLocation("2006-01-02 15:04:05", s, time.UTC)
	if err != nil {
		t, err = time.Parse(time.RFC3339, s)
		if err != nil {
			return true
		}
	}
	return time.Since(t.UTC()) > time.Hour
}

// RefreshAccess is set from auth to avoid an import cycle.
var RefreshAccess = func(clientID, clientSecret, refresh string) (access, newRefresh string, err error) {
	return "", "", fmt.Errorf("refresh not configured")
}

func MaybeSyncUser(ctx context.Context, db *sql.DB, userID, clientID, clientSecret string) error {
	if userID == "" || clientID == "" || clientSecret == "" {
		return nil
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM skater_profiles WHERE user_id = ?`, userID).Scan(&n); err != nil || n == 0 {
		return err
	}
	var channel, refresh, synced string
	err := db.QueryRow(`SELECT IFNULL(youtube_channel_id,''), IFNULL(youtube_refresh_token,''), IFNULL(youtube_synced_at,'') FROM users WHERE id = ?`, userID).
		Scan(&channel, &refresh, &synced)
	if err != nil || channel == "" || refresh == "" || !staleSync(synced) {
		return err
	}
	access, newRefresh, err := RefreshAccess(clientID, clientSecret, refresh)
	if err != nil {
		return err
	}
	if newRefresh != "" && newRefresh != refresh {
		_, _ = db.Exec(`UPDATE users SET youtube_refresh_token=? WHERE id=?`, newRefresh, userID)
	}
	c := Client{Token: access, Channel: channel}
	if _, err := c.Sync(ctx, db); err != nil {
		return err
	}
	_, err = db.Exec(`UPDATE users SET youtube_synced_at=CURRENT_TIMESTAMP WHERE id=?`, userID)
	return err
}

func SyncWithToken(ctx context.Context, db *sql.DB, userID, access, channel string) error {
	if access == "" || channel == "" {
		return nil
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM skater_profiles WHERE user_id = ?`, userID).Scan(&n); err != nil || n == 0 {
		return err
	}
	c := Client{Token: access, Channel: channel}
	if _, err := c.Sync(ctx, db); err != nil {
		return err
	}
	_, err := db.Exec(`UPDATE users SET youtube_synced_at=CURRENT_TIMESTAMP WHERE id=?`, userID)
	return err
}
