package twitch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
	"unicode"
)

const (
	helixAPI = "https://api.twitch.tv"
	tokenURL = "https://id.twitch.tv/oauth2/token"
	maxLogin = 25
	minLogin = 4
)

type Client struct {
	id, secret string
	HTTP       *http.Client
	API, Auth  string

	mu    sync.Mutex
	token string
	until time.Time
}

type Stream struct {
	Login   string
	Title   string
	Game    string
	Started time.Time
}

type Channel struct {
	Login       string
	DisplayName string
	Description string
	Created     time.Time
}

func (ch Channel) CreatedOn() string {
	if ch.Created.IsZero() {
		return ""
	}
	return ch.Created.UTC().Format("2 Jan 2006")
}

func (ch Channel) Blurb() string {
	s := strings.TrimSpace(ch.Description)
	rs := []rune(s)
	if len(rs) > 280 {
		return string(rs[:280]) + "…"
	}
	return s
}

func New(id, secret string) *Client {
	return &Client{
		id: strings.TrimSpace(id), secret: strings.TrimSpace(secret),
		HTTP: &http.Client{Timeout: 15 * time.Second},
		API:  helixAPI, Auth: tokenURL,
	}
}

func (c *Client) Configured() bool {
	return c != nil && c.id != "" && c.secret != ""
}

func Normalize(s string) (string, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if strings.HasPrefix(s, "@") {
		s = s[1:]
	}
	if i := strings.LastIndex(s, "/"); i >= 0 {
		s = s[i+1:]
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil
	}
	if len(s) < minLogin || len(s) > maxLogin {
		return "", fmt.Errorf("twitch name must be 4–25 characters")
	}
	for _, r := range s {
		if r != '_' && !unicode.IsDigit(r) && (r < 'a' || r > 'z') {
			return "", fmt.Errorf("twitch name is letters, digits, and underscore")
		}
	}
	return s, nil
}

func ChannelURL(login string) string {
	login = strings.ToLower(strings.TrimSpace(login))
	if login == "" {
		return ""
	}
	return "https://www.twitch.tv/" + login
}

func (c *Client) UserExists(ctx context.Context, login string) (bool, error) {
	ch, err := c.User(ctx, login)
	return ch.Login != "", err
}

func (c *Client) User(ctx context.Context, login string) (Channel, error) {
	login, err := Normalize(login)
	if err != nil || login == "" {
		return Channel{}, err
	}
	body, err := c.helix(ctx, "/helix/users", url.Values{"login": {login}})
	if err != nil {
		return Channel{}, err
	}
	return parseUser(body)
}

func parseUser(body []byte) (Channel, error) {
	var raw struct {
		Data []struct {
			Login       string `json:"login"`
			DisplayName string `json:"display_name"`
			Description string `json:"description"`
			CreatedAt   string `json:"created_at"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return Channel{}, err
	}
	if len(raw.Data) == 0 {
		return Channel{}, nil
	}
	it := raw.Data[0]
	ch := Channel{
		Login:       strings.ToLower(it.Login),
		DisplayName: it.DisplayName,
		Description: it.Description,
	}
	if t, err := time.Parse(time.RFC3339, it.CreatedAt); err == nil {
		ch.Created = t
	}
	return ch, nil
}

func (c *Client) Streams(ctx context.Context, logins []string) (map[string]Stream, error) {
	out := map[string]Stream{}
	for i := 0; i < len(logins); i += 100 {
		end := i + 100
		if end > len(logins) {
			end = len(logins)
		}
		chunk, err := c.streams(ctx, logins[i:end])
		if err != nil {
			return out, err
		}
		for k, v := range chunk {
			out[k] = v
		}
	}
	return out, nil
}

func parseStreams(body []byte) (map[string]Stream, error) {
	var raw struct {
		Data []struct {
			UserLogin string `json:"user_login"`
			Title     string `json:"title"`
			GameName  string `json:"game_name"`
			StartedAt string `json:"started_at"`
			Type      string `json:"type"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	out := map[string]Stream{}
	for _, it := range raw.Data {
		if it.Type != "" && it.Type != "live" {
			continue
		}
		login := strings.ToLower(it.UserLogin)
		started, err := time.Parse(time.RFC3339, it.StartedAt)
		if err != nil {
			continue
		}
		out[login] = Stream{Login: login, Title: it.Title, Game: it.GameName, Started: started}
	}
	return out, nil
}

func (c *Client) streams(ctx context.Context, logins []string) (map[string]Stream, error) {
	q := url.Values{}
	for _, l := range logins {
		if l != "" {
			q.Add("user_login", l)
		}
	}
	if len(q) == 0 {
		return map[string]Stream{}, nil
	}
	body, err := c.helix(ctx, "/helix/streams", q)
	if err != nil {
		return nil, err
	}
	return parseStreams(body)
}

func (c *Client) helix(ctx context.Context, path string, q url.Values) ([]byte, error) {
	token, err := c.accessToken(ctx)
	if err != nil {
		return nil, err
	}
	u := strings.TrimRight(c.API, "/") + path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Client-ID", c.id)
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := c.http().Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	b, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("twitch %s: %s", res.Status, bytes.TrimSpace(b))
	}
	return b, nil
}

func (c *Client) accessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && time.Now().Before(c.until) {
		return c.token, nil
	}
	form := url.Values{
		"client_id":     {c.id},
		"client_secret": {c.secret},
		"grant_type":    {"client_credentials"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Auth, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := c.http().Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	b, err := io.ReadAll(io.LimitReader(res.Body, 1<<16))
	if err != nil {
		return "", err
	}
	if res.StatusCode >= 300 {
		return "", fmt.Errorf("twitch token: %s", res.Status)
	}
	var tok struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(b, &tok); err != nil {
		return "", err
	}
	if tok.AccessToken == "" {
		return "", fmt.Errorf("twitch token: empty")
	}
	ttl := time.Duration(tok.ExpiresIn) * time.Second
	if ttl <= 0 {
		ttl = time.Hour
	}
	if ttl > 2*time.Minute {
		ttl -= time.Minute
	}
	c.token, c.until = tok.AccessToken, time.Now().Add(ttl)
	return c.token, nil
}

func (c *Client) http() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return http.DefaultClient
}
