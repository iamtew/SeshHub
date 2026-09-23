package yt

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
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

// ponytail: 30s steal is above the 25s caller timeout so a live run is not stolen; steal only recovers a leaked entry.
const runHold = 30 * time.Second

var errBusy = errors.New("sync already running")

var running, swept sync.Map

type runSlot struct{ at time.Time }

func tryBegin(key string) *runSlot {
	mine := &runSlot{at: time.Now()}
	v, loaded := running.LoadOrStore(key, mine)
	if !loaded {
		return mine
	}
	other := v.(*runSlot)
	if time.Since(other.at) < runHold {
		return nil
	}
	if !running.CompareAndSwap(key, other, mine) {
		return nil
	}
	return mine
}

func endRun(key string, mine *runSlot) {
	if mine != nil {
		running.CompareAndDelete(key, mine)
	}
}

type Video struct {
	ID           string
	ChannelID    string
	ChannelTitle string
	Title        string
	Description  string
	PublishedAt  string
	Thumb        string
	Duration     int
	Views        int
	Likes        int
	Comments     int
	Tags         string
	Category     string
	Live         string
	Hidden       bool
}

// ponytail: Shorts ≤180s (YouTube's 3min cap); live/upcoming from snippet.liveBroadcastContent, not a stored kind column.
func UploadType(v Video) string {
	switch strings.ToLower(v.Live) {
	case "live":
		return "live"
	case "upcoming":
		return "premiere"
	}
	if v.Duration > 0 && v.Duration <= 180 {
		return "short"
	}
	return "video"
}

func (v Video) Kind() string { return UploadType(v) }

func (v Video) StatsLine() string {
	var parts []string
	if v.ChannelTitle != "" {
		parts = append(parts, v.ChannelTitle)
	}
	if v.Views > 0 {
		parts = append(parts, strconv.Itoa(v.Views)+" views")
	}
	if v.Likes > 0 {
		parts = append(parts, strconv.Itoa(v.Likes)+" likes")
	}
	if v.Comments > 0 {
		parts = append(parts, strconv.Itoa(v.Comments)+" comments")
	}
	return strings.Join(parts, " · ")
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
	Token, Key, Channel, Base string
	HTTP                      *http.Client
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
	return c.do(ctx, path, q, true)
}

func (c Client) getPublic(ctx context.Context, path string, q url.Values) ([]byte, error) {
	return c.do(ctx, path, q, false)
}

func (c Client) do(ctx context.Context, path string, q url.Values, bearer bool) ([]byte, error) {
	if q == nil {
		q = url.Values{}
	}
	if c.Key != "" {
		q.Set("key", c.Key)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base()+path+"?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	if bearer && c.Token != "" {
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

// ponytail: one playlist page (50 videos) per skater; page further if a channel outgrows that.
func (c Client) Sync(ctx context.Context, db *sql.DB) (Result, error) {
	key := c.Channel
	if key == "" {
		key = "sync"
	}
	mine := tryBegin(key)
	if mine == nil {
		return Result{}, errBusy
	}
	defer endRun(key, mine)
	return c.sync(ctx, db)
}

func nullEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func parseUploadIDs(b []byte) (ids, denied []string, err error) {
	var pl struct {
		Items []struct {
			ContentDetails struct {
				VideoID string `json:"videoId"`
			} `json:"contentDetails"`
			Status struct {
				PrivacyStatus string `json:"privacyStatus"`
			} `json:"status"`
		} `json:"items"`
	}
	if err := json.Unmarshal(b, &pl); err != nil {
		return nil, nil, err
	}
	for _, it := range pl.Items {
		id := it.ContentDetails.VideoID
		if id == "" {
			continue
		}
		if st := strings.ToLower(it.Status.PrivacyStatus); st != "" && st != "public" {
			denied = append(denied, id)
			continue
		}
		ids = append(ids, id)
	}
	return ids, denied, nil
}

func (c Client) uploadsPlaylist(ctx context.Context, public bool) (string, error) {
	get := c.get
	if public {
		get = c.getPublic
	}
	b, err := get(ctx, "/channels", url.Values{"part": {"contentDetails"}, "id": {c.Channel}})
	if err != nil {
		return "", err
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
		return "", fmt.Errorf("no uploads playlist")
	}
	return ch.Items[0].ContentDetails.RelatedPlaylists.Uploads, nil
}

func (c Client) playlistPage(ctx context.Context, uploads string, public bool) ([]byte, error) {
	get := c.get
	if public {
		get = c.getPublic
	}
	q := url.Values{"part": {"contentDetails,status"}, "playlistId": {uploads}, "maxResults": {"50"}}
	return get(ctx, "/playlistItems", q)
}

func channelVideoIDs(db *sql.DB, channel string) ([]string, error) {
	if channel == "" {
		return nil, nil
	}
	rows, err := db.Query(`SELECT id FROM youtube_videos WHERE channel_id=?`, channel)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func mergeIDs(parts ...[]string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, p := range parts {
		for _, id := range p {
			if id == "" {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	return out
}

func pruneChannelNotIn(db *sql.DB, channel string, keep []string) error {
	if channel == "" {
		return nil
	}
	if len(keep) == 0 {
		_, err := db.Exec(`DELETE FROM youtube_videos WHERE channel_id=?`, channel)
		return err
	}
	args := make([]any, 0, 1+len(keep))
	args = append(args, channel)
	for _, id := range keep {
		args = append(args, id)
	}
	q := `DELETE FROM youtube_videos WHERE channel_id=? AND id NOT IN (` + strings.Repeat("?,", len(keep)-1) + `?)`
	_, err := db.Exec(q, args...)
	return err
}

func (c Client) upsertIDs(ctx context.Context, db *sql.DB, ids []string, out *Result) error {
	for i := 0; i < len(ids); i += 50 {
		end := i + 50
		if end > len(ids) {
			end = len(ids)
		}
		if err := c.upsertChunk(ctx, db, ids[i:end], out); err != nil {
			return err
		}
	}
	return nil
}

func (c Client) sync(ctx context.Context, db *sql.DB) (Result, error) {
	var out Result
	if c.Key != "" {
		ids, err := c.publicUploadIDs(ctx)
		if err == nil {
			out.Fetched = len(ids)
			if err := c.upsertIDs(ctx, db, ids, &out); err != nil {
				return out, err
			}
			return out, pruneChannelNotIn(db, c.Channel, ids)
		}
	}
	uploads, err := c.uploadsPlaylist(ctx, false)
	if err != nil {
		return out, err
	}
	pb, err := c.playlistPage(ctx, uploads, false)
	if err != nil {
		return out, err
	}
	ids, denied, err := parseUploadIDs(pb)
	if err != nil {
		return out, err
	}
	cached, err := channelVideoIDs(db, c.Channel)
	if err != nil {
		return out, err
	}
	if err := dropNonPublic(db, denied, nil); err != nil {
		return out, err
	}
	ids = mergeIDs(ids, cached)
	out.Fetched = len(ids)
	if len(ids) == 0 {
		return out, nil
	}
	return out, c.upsertIDs(ctx, db, ids, &out)
}

func (c Client) publicUploadIDs(ctx context.Context) ([]string, error) {
	uploads, err := c.uploadsPlaylist(ctx, true)
	if err != nil {
		return nil, err
	}
	pb, err := c.playlistPage(ctx, uploads, true)
	if err != nil {
		return nil, err
	}
	ids, _, err := parseUploadIDs(pb)
	return ids, err
}

func dropNonPublic(db *sql.DB, ids []string, public map[string]struct{}) error {
	for _, id := range ids {
		if _, ok := public[id]; ok {
			continue
		}
		if _, err := db.Exec(`DELETE FROM youtube_videos WHERE id=?`, id); err != nil {
			return err
		}
	}
	return nil
}

func (c Client) upsertChunk(ctx context.Context, db *sql.DB, ids []string, out *Result) error {
	vb, err := c.get(ctx, "/videos", url.Values{"part": {"snippet,contentDetails,statistics,status"}, "id": {strings.Join(ids, ",")}})
	if err != nil {
		return err
	}
	var vs struct {
		Items []struct {
			ID      string `json:"id"`
			Snippet struct {
				Title                string   `json:"title"`
				Description          string   `json:"description"`
				ChannelID            string   `json:"channelId"`
				ChannelTitle         string   `json:"channelTitle"`
				PublishedAt          string   `json:"publishedAt"`
				Tags                 []string `json:"tags"`
				LiveBroadcastContent string   `json:"liveBroadcastContent"`
				Thumbnails           map[string]struct {
					URL string `json:"url"`
				} `json:"thumbnails"`
			} `json:"snippet"`
			ContentDetails struct {
				Duration string `json:"duration"`
			} `json:"contentDetails"`
			Statistics struct {
				ViewCount    string `json:"viewCount"`
				LikeCount    string `json:"likeCount"`
				CommentCount string `json:"commentCount"`
			} `json:"statistics"`
			Status struct {
				PrivacyStatus string `json:"privacyStatus"`
			} `json:"status"`
		} `json:"items"`
	}
	if err := json.Unmarshal(vb, &vs); err != nil {
		return err
	}
	kept := map[string]struct{}{}
	for _, it := range vs.Items {
		if it.Status.PrivacyStatus != "public" {
			continue
		}
		kept[it.ID] = struct{}{}
		views, _ := strconv.Atoi(it.Statistics.ViewCount)
		likes, _ := strconv.Atoi(it.Statistics.LikeCount)
		comments, _ := strconv.Atoi(it.Statistics.CommentCount)
		ch := it.Snippet.ChannelID
		if ch == "" {
			ch = c.Channel
		}
		v := Video{
			ID: it.ID, ChannelID: ch, ChannelTitle: it.Snippet.ChannelTitle,
			Title: it.Snippet.Title, Description: it.Snippet.Description,
			PublishedAt: it.Snippet.PublishedAt, Thumb: thumbURL(it.Snippet.Thumbnails),
			Duration: DurationSeconds(it.ContentDetails.Duration), Views: views, Likes: likes, Comments: comments,
			Tags: strings.Join(it.Snippet.Tags, ", "), Category: Classify(it.Snippet.Title, it.Snippet.Description),
			Live: it.Snippet.LiveBroadcastContent,
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
	return dropNonPublic(db, ids, kept)
}

func upsert(db *sql.DB, v Video) (inserted bool, err error) {
	var exists int
	_ = db.QueryRow(`SELECT 1 FROM youtube_videos WHERE id = ?`, v.ID).Scan(&exists)
	if exists == 1 {
		_, err = db.Exec(`UPDATE youtube_videos SET title=?, channel_title=?, description=?, published_at=?, thumbnail_url=?, duration_seconds=?, view_count=?, like_count=?, comment_count=?, tags=?, category=?, live_broadcast=?, synced_at=CURRENT_TIMESTAMP WHERE id=?`,
			v.Title, nullEmpty(v.ChannelTitle), v.Description, v.PublishedAt, v.Thumb, v.Duration, v.Views, v.Likes, v.Comments, nullEmpty(v.Tags), v.Category, nullEmpty(v.Live), v.ID)
		return false, err
	}
	_, err = db.Exec(`INSERT INTO youtube_videos (id, channel_id, channel_title, title, description, published_at, thumbnail_url, duration_seconds, view_count, like_count, comment_count, tags, category, live_broadcast) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		v.ID, v.ChannelID, nullEmpty(v.ChannelTitle), v.Title, nullEmpty(v.Description), v.PublishedAt, v.Thumb, v.Duration, v.Views, v.Likes, v.Comments, nullEmpty(v.Tags), v.Category, nullEmpty(v.Live))
	return true, err
}

func ListPublic(db *sql.DB) ([]Video, error) {
	return list(db, "", 0, 0, false)
}

func ListPublicPage(db *sql.DB, limit, offset int) ([]Video, error) {
	return list(db, "", limit, offset, false)
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
	return list(db, channelID, limit, 0, false)
}

func ListByChannelAll(db *sql.DB, channelID string, limit int) ([]Video, error) {
	if channelID == "" {
		return nil, nil
	}
	return list(db, channelID, limit, 0, true)
}

func SetHidden(db *sql.DB, id string, hidden bool) error {
	if id == "" {
		return sql.ErrNoRows
	}
	_, err := db.Exec(`UPDATE youtube_videos SET is_hidden=? WHERE id=?`, hidden, id)
	return err
}

func list(db *sql.DB, channelID string, limit, offset int, includeHidden bool) ([]Video, error) {
	q := `SELECT id, channel_id, IFNULL(channel_title,''), title, IFNULL(description,''), published_at, thumbnail_url, duration_seconds, view_count, like_count, comment_count, IFNULL(tags,''), IFNULL(category,''), IFNULL(live_broadcast,''), is_hidden FROM youtube_videos WHERE 1=1`
	var args []any
	if !includeHidden {
		q += ` AND is_hidden = 0`
	}
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
		v, err := scanVideo(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func scanVideo(scan func(dest ...any) error) (Video, error) {
	var v Video
	var hidden int
	err := scan(&v.ID, &v.ChannelID, &v.ChannelTitle, &v.Title, &v.Description, &v.PublishedAt, &v.Thumb, &v.Duration, &v.Views, &v.Likes, &v.Comments, &v.Tags, &v.Category, &v.Live, &hidden)
	v.Hidden = hidden != 0
	return v, err
}

func Get(db *sql.DB, id string) (Video, error) {
	return get(db, id, false)
}

func GetAny(db *sql.DB, id string) (Video, error) {
	return get(db, id, true)
}

func get(db *sql.DB, id string, includeHidden bool) (Video, error) {
	var v Video
	if id == "" {
		return v, sql.ErrNoRows
	}
	q := `SELECT id, channel_id, IFNULL(channel_title,''), title, IFNULL(description,''), published_at, thumbnail_url, duration_seconds, view_count, like_count, comment_count, IFNULL(tags,''), IFNULL(category,''), IFNULL(live_broadcast,''), is_hidden FROM youtube_videos WHERE id = ?`
	if !includeHidden {
		q += ` AND is_hidden = 0`
	}
	v, err := scanVideo(db.QueryRow(q, id).Scan)
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
	q := `SELECT id, channel_id, IFNULL(channel_title,''), title, IFNULL(description,''), published_at, thumbnail_url, duration_seconds, view_count, like_count, comment_count, IFNULL(tags,''), IFNULL(category,''), IFNULL(live_broadcast,''), is_hidden FROM youtube_videos WHERE is_hidden = 0 AND id IN (` + strings.Repeat("?,", len(uniq)-1) + `?)`
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
		v, err := scanVideo(rows.Scan)
		if err != nil {
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

func MaybeSyncUser(ctx context.Context, db *sql.DB, userID, clientID, clientSecret, apiKey string) error {
	if userID == "" || clientID == "" || clientSecret == "" {
		return nil
	}
	mine := tryBegin("u:" + userID)
	if mine == nil {
		return nil
	}
	defer endRun("u:"+userID, mine)
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM skater_profiles WHERE user_id = ?`, userID).Scan(&n); err != nil || n == 0 {
		return err
	}
	var channel, refresh, synced string
	err := db.QueryRow(`SELECT IFNULL(youtube_channel_id,''), IFNULL(youtube_refresh_token,''), IFNULL(youtube_synced_at,'') FROM users WHERE id = ?`, userID).
		Scan(&channel, &refresh, &synced)
	if err != nil || channel == "" || refresh == "" {
		return err
	}
	_, did := swept.Load(channel)
	if !staleSync(synced) && did {
		return nil
	}
	access, newRefresh, err := RefreshAccess(clientID, clientSecret, refresh)
	if err != nil {
		return err
	}
	if newRefresh != "" && newRefresh != refresh {
		_, _ = db.Exec(`UPDATE users SET youtube_refresh_token=? WHERE id=?`, newRefresh, userID)
	}
	c := Client{Token: access, Key: apiKey, Channel: channel}
	if _, err := c.Sync(ctx, db); err != nil {
		if errors.Is(err, errBusy) {
			return nil
		}
		return err
	}
	swept.Store(channel, struct{}{})
	_, err = db.Exec(`UPDATE users SET youtube_synced_at=CURRENT_TIMESTAMP WHERE id=?`, userID)
	return err
}

func SyncWithToken(ctx context.Context, db *sql.DB, userID, access, channel, apiKey string) error {
	if access == "" || channel == "" {
		return nil
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM skater_profiles WHERE user_id = ?`, userID).Scan(&n); err != nil || n == 0 {
		return err
	}
	c := Client{Token: access, Key: apiKey, Channel: channel}
	if _, err := c.Sync(ctx, db); err != nil {
		if errors.Is(err, errBusy) {
			return nil
		}
		return err
	}
	swept.Store(channel, struct{}{})
	_, err := db.Exec(`UPDATE users SET youtube_synced_at=CURRENT_TIMESTAMP WHERE id=?`, userID)
	return err
}

// PollPublic refreshes title, channel name, and public counts for IDs we already store.
func PollPublic(ctx context.Context, db *sql.DB, apiKey string) (int, error) {
	if apiKey == "" {
		return 0, nil
	}
	return (Client{Key: apiKey}).Poll(ctx, db)
}

// ponytail: 50 ids per videos.list (API max); no new channels, no comment text.
func (c Client) pruneCachedToPublicPlaylists(ctx context.Context, db *sql.DB) {
	if c.Key == "" {
		return
	}
	rows, err := db.Query(`SELECT DISTINCT channel_id FROM youtube_videos WHERE IFNULL(channel_id,'') != ''`)
	if err != nil {
		return
	}
	var chans []string
	for rows.Next() {
		var ch string
		if err := rows.Scan(&ch); err != nil {
			rows.Close()
			return
		}
		chans = append(chans, ch)
	}
	rows.Close()
	for _, ch := range chans {
		cc := c
		cc.Channel = ch
		ids, err := cc.publicUploadIDs(ctx)
		if err != nil {
			continue
		}
		_ = pruneChannelNotIn(db, ch, ids)
	}
}

func (c Client) Poll(ctx context.Context, db *sql.DB) (int, error) {
	c.pruneCachedToPublicPlaylists(ctx, db)
	rows, err := db.Query(`SELECT id FROM youtube_videos WHERE is_hidden = 0`)
	if err != nil {
		return 0, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil || len(ids) == 0 {
		return 0, err
	}
	var updated int
	for i := 0; i < len(ids); i += 50 {
		end := i + 50
		if end > len(ids) {
			end = len(ids)
		}
		n, err := c.pollChunk(ctx, db, ids[i:end])
		if err != nil {
			return updated, err
		}
		updated += n
	}
	return updated, nil
}

func (c Client) pollChunk(ctx context.Context, db *sql.DB, ids []string) (int, error) {
	vb, err := c.get(ctx, "/videos", url.Values{"part": {"snippet,contentDetails,statistics,status"}, "id": {strings.Join(ids, ",")}})
	if err != nil {
		return 0, err
	}
	var vs struct {
		Items []struct {
			ID      string `json:"id"`
			Snippet struct {
				Title                string `json:"title"`
				ChannelTitle         string `json:"channelTitle"`
				LiveBroadcastContent string `json:"liveBroadcastContent"`
			} `json:"snippet"`
			ContentDetails struct {
				Duration string `json:"duration"`
			} `json:"contentDetails"`
			Statistics struct {
				ViewCount    string `json:"viewCount"`
				LikeCount    string `json:"likeCount"`
				CommentCount string `json:"commentCount"`
			} `json:"statistics"`
			Status struct {
				PrivacyStatus string `json:"privacyStatus"`
			} `json:"status"`
		} `json:"items"`
	}
	if err := json.Unmarshal(vb, &vs); err != nil {
		return 0, err
	}
	kept := map[string]struct{}{}
	n := 0
	for _, it := range vs.Items {
		if it.Status.PrivacyStatus != "public" {
			continue
		}
		kept[it.ID] = struct{}{}
		views, _ := strconv.Atoi(it.Statistics.ViewCount)
		likes, _ := strconv.Atoi(it.Statistics.LikeCount)
		comments, _ := strconv.Atoi(it.Statistics.CommentCount)
		dur := DurationSeconds(it.ContentDetails.Duration)
		res, err := db.Exec(`UPDATE youtube_videos SET title=COALESCE(NULLIF(?, ''), title), channel_title=?, duration_seconds=CASE WHEN ? > 0 THEN ? ELSE duration_seconds END, live_broadcast=COALESCE(NULLIF(?, ''), live_broadcast), view_count=?, like_count=?, comment_count=?, synced_at=CURRENT_TIMESTAMP WHERE id=?`,
			it.Snippet.Title, nullEmpty(it.Snippet.ChannelTitle), dur, dur, nullEmpty(it.Snippet.LiveBroadcastContent), views, likes, comments, it.ID)
		if err != nil {
			return n, err
		}
		if k, _ := res.RowsAffected(); k > 0 {
			n++
		}
	}
	if err := dropNonPublic(db, ids, kept); err != nil {
		return n, err
	}
	return n, nil
}
