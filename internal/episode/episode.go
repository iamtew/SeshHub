package episode

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"seshhub/internal/spot"
)

const ChallengeChannel = "sesh-sofa-spot-challenge"

type Row struct {
	Kind  string `json:"kind"`
	Label string `json:"label"`
	URL   string `json:"url"`
}

type Episode struct {
	Number              int
	Title               string
	Heading             string
	SubmissionsCount    *int
	WinnerName          string
	WinnerVideoID       string
	ChallengePlaylistID string
	Note                string
	Rows                []Row
}

func ParseVideoID(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if !strings.Contains(s, "://") && !strings.Contains(s, "/") && !strings.Contains(s, "?") && len(s) == 11 {
		return s
	}
	u, err := url.Parse(s)
	if err != nil {
		return ""
	}
	host := strings.ToLower(u.Host)
	if host == "youtu.be" || host == "www.youtu.be" {
		return strings.Trim(u.Path, "/")
	}
	if v := u.Query().Get("v"); v != "" {
		return v
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	for i, p := range parts {
		if (p == "embed" || p == "shorts" || p == "live" || p == "v") && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

func ParsePlaylistID(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	u, err := url.Parse(s)
	if err == nil {
		if id := u.Query().Get("list"); id != "" {
			return id
		}
	}
	if strings.HasPrefix(s, "PL") || strings.HasPrefix(s, "UU") {
		return s
	}
	return ""
}

func ThumbURL(videoID string) string {
	if videoID == "" {
		return ""
	}
	return "https://img.youtube.com/vi/" + videoID + "/hqdefault.jpg"
}

func WatchURL(videoID string) string {
	if videoID == "" {
		return ""
	}
	return "https://www.youtube.com/watch?v=" + videoID
}

func PlaylistURL(id string) string {
	if id == "" {
		return ""
	}
	return "https://www.youtube.com/playlist?list=" + id
}

func (e Episode) Anchor() string { return fmt.Sprintf("ep%d", e.Number) }

func (e Episode) H2() string {
	if h := strings.TrimSpace(e.Heading); h != "" {
		return h
	}
	if t := strings.TrimSpace(e.Title); t != "" {
		return fmt.Sprintf("EP%d %s", e.Number, t)
	}
	return fmt.Sprintf("EP%d", e.Number)
}

func (e Episode) ShowWinner() bool { return strings.TrimSpace(e.WinnerName) != "" }

func (e Episode) WinnerWatch() string {
	if e.WinnerVideoID != "" {
		return WatchURL(e.WinnerVideoID)
	}
	return ""
}

func (e Episode) VisibleRows() []Row {
	var out []Row
	hasWinner := false
	for _, r := range e.Rows {
		r.URL = strings.TrimSpace(r.URL)
		switch r.Kind {
		case "winner":
			if !e.ShowWinner() {
				continue
			}
			hasWinner = true
			if r.URL == "" {
				r.URL = e.WinnerWatch()
			}
			r.Label = "Winner: " + e.WinnerName
		case "trick":
			if r.URL == "" {
				continue
			}
		default:
			if r.URL == "" {
				continue
			}
		}
		if strings.TrimSpace(r.Label) == "" {
			r.Label = r.Kind
		}
		out = append(out, r)
	}
	if e.ShowWinner() && !hasWinner {
		out = append(out, Row{Kind: "winner", Label: "Winner: " + e.WinnerName, URL: e.WinnerWatch()})
	}
	return out
}

func (r Row) Thumb() string {
	if r.Kind != "full" && r.Kind != "winner" {
		return ""
	}
	return ThumbURL(ParseVideoID(r.URL))
}

func (r Row) Emoji() string {
	switch r.Kind {
	case "full":
		return "📺"
	case "playlist":
		return "📼"
	case "winner":
		return "🏆"
	case "trick":
		return "🛹"
	}
	return ""
}

func List(db *sql.DB) ([]Episode, error) {
	rows, err := db.Query(`SELECT number, title, heading, submissions_count, winner_name, winner_video_id, challenge_playlist_id, note, rows FROM episodes ORDER BY number DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Episode
	for rows.Next() {
		e, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func Get(db *sql.DB, n int) (Episode, error) {
	row := db.QueryRow(`SELECT number, title, heading, submissions_count, winner_name, winner_video_id, challenge_playlist_id, note, rows FROM episodes WHERE number=?`, n)
	return scan(row)
}

type scannable interface {
	Scan(dest ...any) error
}

func scan(s scannable) (Episode, error) {
	var e Episode
	var count sql.NullInt64
	var raw string
	err := s.Scan(&e.Number, &e.Title, &e.Heading, &count, &e.WinnerName, &e.WinnerVideoID, &e.ChallengePlaylistID, &e.Note, &raw)
	if err != nil {
		return e, err
	}
	if count.Valid {
		n := int(count.Int64)
		e.SubmissionsCount = &n
	}
	if strings.TrimSpace(raw) != "" {
		_ = json.Unmarshal([]byte(raw), &e.Rows)
	}
	return e, nil
}

func Save(db *sql.DB, e Episode) error {
	if e.Number <= 0 {
		return fmt.Errorf("episode number required")
	}
	e.syncWinnerRow()
	b, err := json.Marshal(e.Rows)
	if err != nil {
		return err
	}
	var count any
	if e.SubmissionsCount != nil {
		count = *e.SubmissionsCount
	}
	_, err = db.Exec(`INSERT INTO episodes (number, title, heading, submissions_count, winner_name, winner_video_id, challenge_playlist_id, note, rows)
		VALUES (?,?,?,?,?,?,?,?,?)
		ON CONFLICT(number) DO UPDATE SET title=excluded.title, heading=excluded.heading, submissions_count=excluded.submissions_count,
			winner_name=excluded.winner_name, winner_video_id=excluded.winner_video_id, challenge_playlist_id=excluded.challenge_playlist_id,
			note=excluded.note, rows=excluded.rows`,
		e.Number, e.Title, e.Heading, count, e.WinnerName, e.WinnerVideoID, e.ChallengePlaylistID, e.Note, string(b))
	return err
}

func Delete(db *sql.DB, n int) error {
	_, err := db.Exec(`DELETE FROM episodes WHERE number=?`, n)
	return err
}

func (e *Episode) syncWinnerRow() {
	watch := e.WinnerWatch()
	if watch == "" && e.WinnerName == "" {
		return
	}
	for i, r := range e.Rows {
		if r.Kind == "winner" {
			if watch != "" {
				e.Rows[i].URL = watch
			}
			if strings.TrimSpace(e.Rows[i].Label) == "" {
				e.Rows[i].Label = "Winner"
			}
			return
		}
	}
	if watch != "" {
		e.Rows = append(e.Rows, Row{Kind: "winner", Label: "Winner", URL: watch})
	}
}

func FromShow(sh spot.Show) Episode {
	e := Episode{Number: sh.Episode, Title: strings.TrimSpace(sh.Name)}
	for _, lis := range sh.Listeners {
		if lis.Kind != "content" || lis.PlaylistID == "" {
			continue
		}
		label := strings.TrimSpace(lis.Name)
		if label == "" {
			label = "Playlist"
		}
		e.Rows = append(e.Rows, Row{Kind: "playlist", Label: label, URL: PlaylistURL(lis.PlaylistID)})
		if lis.Channel == ChallengeChannel {
			e.ChallengePlaylistID = lis.PlaylistID
		}
	}
	return e
}

func EnsureCurrent(db *sql.DB) {
	sh, ok := spot.Current()
	if !ok {
		return
	}
	live := FromShow(sh)
	if live.Number <= 0 {
		return
	}
	existing, err := Get(db, live.Number)
	if err == sql.ErrNoRows {
		_ = Save(db, live)
		return
	}
	if err != nil {
		return
	}
	changed := false
	if existing.ChallengePlaylistID == "" && live.ChallengePlaylistID != "" {
		existing.ChallengePlaylistID = live.ChallengePlaylistID
		changed = true
	}
	if existing.Title == "" && live.Title != "" {
		existing.Title = live.Title
		changed = true
	}
	if !hasPlaylistURL(existing) {
		existing.Rows = append(existing.Rows, live.Rows...)
		changed = true
	}
	if changed {
		_ = Save(db, existing)
	}
}

func hasPlaylistURL(e Episode) bool {
	for _, r := range e.Rows {
		if r.Kind == "playlist" && strings.TrimSpace(r.URL) != "" {
			return true
		}
	}
	return false
}

func ParseCount(s string) *int {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return nil
	}
	return &n
}
