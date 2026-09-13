package spot

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"sync"
	"time"
)

const rowID = 1

// ponytail: in-memory TTL; persist last-good if Subotto downtime becomes a problem.
const cacheTTL = 60 * time.Second

var EpisodeURL = "https://subotto.seshsofa.nl/api/get/episode/sesh-sofa"

var httpc = &http.Client{Timeout: 3 * time.Second}

var slot = regexp.MustCompile(`\{\{([a-zA-Z0-9_.]+)\}\}`)

type Config struct {
	ContentRaw string
}

type Item struct {
	Key, Placeholder, Val string
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
	cached  map[string]string
	cacheAt time.Time
	cacheOK bool
)

func Latest() (map[string]string, bool) {
	cacheMu.Lock()
	if cacheOK && time.Since(cacheAt) < cacheTTL {
		vals := cached
		cacheMu.Unlock()
		return vals, true
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
	if err != nil {
		return nil, false
	}
	vals, err := FlattenJSON(body)
	if err != nil || len(vals) == 0 {
		return nil, false
	}
	cacheMu.Lock()
	cached, cacheAt, cacheOK = vals, time.Now(), true
	cacheMu.Unlock()
	return vals, true
}
