package twitch

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const decapiAPI = "https://decapi.me"

func LastTitles(ctx context.Context, c *Client, logins []string) map[string]string {
	out := map[string]string{}
	if c == nil {
		c = New("", "")
	}
	var missing []string
	for _, l := range logins {
		l, _ = Normalize(l)
		if l != "" {
			missing = append(missing, l)
		}
	}
	if c.Configured() {
		if ids, err := c.userIDs(ctx, missing); err == nil && len(ids) > 0 {
			if titles, err := c.channelTitles(ctx, ids); err == nil {
				for login, title := range titles {
					if title != "" {
						out[login] = title
					}
				}
			}
		}
	}
	for _, login := range missing {
		if out[login] != "" {
			continue
		}
		if t := c.decapiTitle(ctx, login); t != "" {
			out[login] = t
		}
	}
	return out
}

func setTitle(db *sql.DB, id, title string) error {
	_, err := db.Exec(`UPDATE skater_profiles SET twitch_title=? WHERE id=?`, title, id)
	return err
}

func (c *Client) userIDs(ctx context.Context, logins []string) (map[string]string, error) {
	out := map[string]string{}
	q := url.Values{}
	for _, l := range logins {
		q.Add("login", l)
	}
	if len(q) == 0 {
		return out, nil
	}
	body, err := c.helix(ctx, "/helix/users", q)
	if err != nil {
		return nil, err
	}
	var raw struct {
		Data []struct {
			ID    string `json:"id"`
			Login string `json:"login"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	for _, it := range raw.Data {
		login := strings.ToLower(it.Login)
		if login != "" && it.ID != "" {
			out[login] = it.ID
		}
	}
	return out, nil
}

func (c *Client) channelTitles(ctx context.Context, loginIDs map[string]string) (map[string]string, error) {
	out := map[string]string{}
	idLogin := map[string]string{}
	q := url.Values{}
	for login, id := range loginIDs {
		if id == "" {
			continue
		}
		q.Add("broadcaster_id", id)
		idLogin[id] = login
	}
	if len(q) == 0 {
		return out, nil
	}
	body, err := c.helix(ctx, "/helix/channels", q)
	if err != nil {
		return nil, err
	}
	var raw struct {
		Data []struct {
			ID    string `json:"broadcaster_id"`
			Login string `json:"broadcaster_login"`
			Title string `json:"title"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	for _, it := range raw.Data {
		login := strings.ToLower(it.Login)
		if login == "" {
			login = idLogin[it.ID]
		}
		title := strings.TrimSpace(it.Title)
		if login != "" && title != "" {
			out[login] = title
		}
	}
	return out, nil
}

func (c *Client) decapiTitle(ctx context.Context, login string) string {
	base := c.Decapi
	if base == "" {
		base = decapiAPI
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(base, "/")+"/twitch/title/"+url.PathEscape(login), nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", "SeshHub (https://hub.seshsofa.nl)")
	res, err := c.http().Do(req)
	if err != nil {
		return ""
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return ""
	}
	b, err := io.ReadAll(io.LimitReader(res.Body, 1<<12))
	if err != nil {
		return ""
	}
	s := strings.TrimSpace(string(b))
	if s == "" || strings.HasPrefix(strings.ToLower(s), "error") {
		return ""
	}
	return s
}
