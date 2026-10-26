package spot

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

const rowID = 1

// ponytail: in-memory TTL; persist last-good if Subotto downtime becomes a problem.
const cacheTTL = 60 * time.Second

var EpisodeURL string

var httpc = &http.Client{Timeout: 3 * time.Second}

var slot = regexp.MustCompile(`\{\{([a-zA-Z0-9_.]+)\}\}`)

type Config struct {
	ContentRaw string
}

type Item struct {
	Key, Placeholder, Val string
}

type Listener struct {
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	Channel    string `json:"channel"`
	PlaylistID string `json:"playlist_id"`
}

type Show struct {
	Episode   int        `json:"episode"`
	Name      string     `json:"name"`
	Listeners []Listener `json:"listeners"`
}

func Get(db *sql.DB) (Config, error) {
	var c Config
	err := db.QueryRow(`SELECT content_raw FROM spot WHERE id=?`, rowID).Scan(&c.ContentRaw)
	return c, err
}

func Save(db *sql.DB, c Config) error {
	_, err := db.Exec(`UPDATE spot SET content_raw=? WHERE id=?`, c.ContentRaw, rowID)
	return err
}

func FlattenJSON(raw []byte) (map[string]string, error) {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, err
	}
	out := map[string]string{}
	flatten(v, "", out)
	return out, nil
}

func flatten(v any, prefix string, out map[string]string) {
	switch t := v.(type) {
	case map[string]any:
		for k, child := range t {
			key := k
			if prefix != "" {
				key = prefix + "." + k
			}
			flatten(child, key, out)
		}
	case []any:
		for i, child := range t {
			flatten(child, fmt.Sprintf("%s.%d", prefix, i), out)
		}
	case nil:
		return
	default:
		if prefix != "" {
			out[prefix] = fmt.Sprint(t)
		}
	}
}

func Fill(md string, vals map[string]string) string {
	return slot.ReplaceAllStringFunc(md, func(m string) string {
		return vals[m[2:len(m)-2]]
	})
}

const imagesHole = "{{spot_images}}"

// SplitImages cuts markdown at the first {{spot_images}} so the live page can
// drop the roster slideshow in that hole. Later copies stay for Fill (empty).
func SplitImages(md string) (before, after string, found bool) {
	i := strings.Index(md, imagesHole)
	if i < 0 {
		return md, "", false
	}
	return md[:i], md[i+len(imagesHole):], true
}

// absImage turns a Subotto image path into an absolute URL on the instance
// EpisodeURL points at. Absolute http(s) URLs pass through.
func absImage(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	ref, err := url.Parse(s)
	if err != nil {
		return ""
	}
	if ref.IsAbs() {
		if ref.Scheme == "http" || ref.Scheme == "https" {
			return ref.String()
		}
		return ""
	}
	base, err := url.Parse(EpisodeURL)
	if err != nil || base.Host == "" {
		return ""
	}
	base.Path, base.RawQuery, base.Fragment = "/", "", ""
	return base.ResolveReference(ref).String()
}

func imageURL(v any) string {
	switch t := v.(type) {
	case string:
		return absImage(t)
	case map[string]any:
		s, _ := t["url"].(string)
		return absImage(s)
	default:
		return ""
	}
}

// Images reads spot_images from Subotto JSON: URL strings or {url} objects.
func Images(raw []byte) []string {
	var wrap struct {
		Images any `json:"spot_images"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return nil
	}
	arr, ok := wrap.Images.([]any)
	if !ok {
		return nil
	}
	var out []string
	for _, v := range arr {
		if u := imageURL(v); u != "" {
			out = append(out, u)
		}
	}
	return out
}

func Items(vals map[string]string) []Item {
	keys := make([]string, 0, len(vals))
	for k := range vals {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]Item, 0, len(keys))
	for _, k := range keys {
		v := vals[k]
		if len([]rune(v)) > 80 {
			v = string([]rune(v)[:80]) + "…"
		}
		out = append(out, Item{Key: k, Placeholder: "{{" + k + "}}", Val: v})
	}
	return out
}

var (
	cacheMu sync.Mutex
	cached  []byte
	cacheAt time.Time
	cacheOK bool
)

func ClearCache() {
	cacheMu.Lock()
	cached, cacheOK = nil, false
	cacheMu.Unlock()
}

func LatestRaw() ([]byte, bool) {
	cacheMu.Lock()
	if cacheOK && time.Since(cacheAt) < cacheTTL {
		b := cached
		cacheMu.Unlock()
		return b, true
	}
	cacheMu.Unlock()

	req, err := http.NewRequest(http.MethodGet, EpisodeURL, nil)
	if err != nil {
		return nil, false
	}
	res, err := httpc.Do(req)
	if err != nil {
		return nil, false
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, false
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil || len(body) == 0 {
		return nil, false
	}
	cacheMu.Lock()
	cached, cacheAt, cacheOK = body, time.Now(), true
	cacheMu.Unlock()
	return body, true
}

func Latest() (map[string]string, bool) {
	body, ok := LatestRaw()
	if !ok {
		return nil, false
	}
	vals, err := FlattenJSON(body)
	if err != nil || len(vals) == 0 {
		return nil, false
	}
	return vals, true
}

func Current() (Show, bool) {
	body, ok := LatestRaw()
	if !ok {
		return Show{}, false
	}
	var s Show
	if err := json.Unmarshal(body, &s); err != nil || s.Episode == 0 {
		return Show{}, false
	}
	return s, true
}
