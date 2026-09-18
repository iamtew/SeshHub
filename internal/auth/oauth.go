package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const oauthCookie = "seshhub_oauth"

func PKCE() (verifier, challenge string) {
	verifier = base64.RawURLEncoding.EncodeToString([]byte(RandomHex(32)))[:43]
	sum := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge
}

func DiscordAuthorizeURL(clientID, redirect, state, challenge string) string {
	q := url.Values{
		"client_id":             {clientID},
		"response_type":         {"code"},
		"redirect_uri":          {redirect},
		"scope":                 {"identify guilds.members.read"},
		"state":                 {state},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
	}
	return "https://discord.com/api/oauth2/authorize?" + q.Encode()
}

func YouTubeAuthorizeURL(clientID, redirect, state, challenge string) string {
	q := url.Values{
		"client_id":             {clientID},
		"response_type":         {"code"},
		"redirect_uri":          {redirect},
		"scope":                 {"https://www.googleapis.com/auth/youtube.readonly"},
		"state":                 {state},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
		"access_type":           {"offline"},
		"prompt":                {"consent"},
	}
	return "https://accounts.google.com/o/oauth2/v2/auth?" + q.Encode()
}

type discordUser struct {
	ID         string `json:"id"`
	Username   string `json:"username"`
	GlobalName string `json:"global_name"`
	Avatar     string `json:"avatar"`
}

type discordMember struct {
	Roles []string `json:"roles"`
}

func discordAvatar(id, hash string) string {
	if hash == "" {
		return ""
	}
	ext := ".png"
	if strings.HasPrefix(hash, "a_") {
		ext = ".gif"
	}
	return "https://cdn.discordapp.com/avatars/" + id + "/" + hash + ext + "?size=128"
}

func tokenPOST(endpoint string, form url.Values) (access, refresh string, err error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.PostForm(endpoint, form)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return "", "", fmt.Errorf("token exchange %s: %s", resp.Status, body)
	}
	var out struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", "", err
	}
	if out.AccessToken == "" {
		return "", "", fmt.Errorf("no access_token")
	}
	return out.AccessToken, out.RefreshToken, nil
}

func bearerGET(url, token string) ([]byte, int, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return b, resp.StatusCode, nil
}

func ExchangeDiscord(clientID, clientSecret, redirect, code, verifier string) (string, error) {
	access, _, err := tokenPOST("https://discord.com/api/oauth2/token", url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirect},
		"code_verifier": {verifier},
	})
	return access, err
}

func FetchDiscord(token, guildID string) (discordID, username, display, avatar string, inGuild bool, roles []string, err error) {
	b, status, err := bearerGET("https://discord.com/api/users/@me", token)
	if err != nil {
		return
	}
	if status >= 300 {
		err = fmt.Errorf("discord me: %s", b)
		return
	}
	var u discordUser
	if err = json.Unmarshal(b, &u); err != nil {
		return
	}
	discordID, username = u.ID, u.Username
	display = u.GlobalName
	if display == "" {
		display = u.Username
	}
	avatar = discordAvatar(u.ID, u.Avatar)
	if guildID == "" {
		return
	}
	mb, mstatus, merr := bearerGET("https://discord.com/api/users/@me/guilds/"+guildID+"/member", token)
	if merr != nil {
		err = merr
		return
	}
	if mstatus == http.StatusNotFound {
		return
	}
	if mstatus >= 300 {
		err = fmt.Errorf("discord member: %s", mb)
		return
	}
	var m discordMember
	if err = json.Unmarshal(mb, &m); err != nil {
		return
	}
	inGuild, roles = true, m.Roles
	return
}

func ExchangeGoogle(clientID, clientSecret, redirect, code, verifier string) (access, refresh string, err error) {
	return tokenPOST("https://oauth2.googleapis.com/token", url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirect},
		"code_verifier": {verifier},
	})
}

func RefreshGoogle(clientID, clientSecret, refresh string) (access, newRefresh string, err error) {
	return tokenPOST("https://oauth2.googleapis.com/token", url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"grant_type":    {"refresh_token"},
		"refresh_token": {refresh},
	})
}

func FetchYouTubeChannel(token string) (channelID, title, avatar string, err error) {
	b, status, err := bearerGET("https://www.googleapis.com/youtube/v3/channels?part=snippet&mine=true", token)
	if err != nil {
		return
	}
	if status >= 300 {
		err = fmt.Errorf("youtube channels: %s", b)
		return
	}
	var out struct {
		Items []struct {
			ID      string `json:"id"`
			Snippet struct {
				Title      string `json:"title"`
				Thumbnails struct {
					Default struct {
						URL string `json:"url"`
					} `json:"default"`
				} `json:"thumbnails"`
			} `json:"snippet"`
		} `json:"items"`
	}
	if err = json.Unmarshal(b, &out); err != nil {
		return
	}
	if len(out.Items) == 0 {
		err = fmt.Errorf("no youtube channel")
		return
	}
	it := out.Items[0]
	return it.ID, it.Snippet.Title, it.Snippet.Thumbnails.Default.URL, nil
}

func ParseOAuthCookie(v string) (provider, state, verifier, next string, ok bool) {
	parts := strings.SplitN(v, ":", 4)
	if len(parts) < 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", "", "", false
	}
	if len(parts) == 4 {
		next = parts[3]
	}
	return parts[0], parts[1], parts[2], next, true
}

func FormatOAuthCookie(provider, state, verifier, next string) string {
	s := provider + ":" + state + ":" + verifier
	if next != "" {
		s += ":" + next
	}
	return s
}

func OAuthCookieName() string { return oauthCookie }
