package yt

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type PlaylistItem struct {
	VideoID, Title, ChannelTitle string
}

func (c Client) PlaylistItems(ctx context.Context, playlistID string) ([]PlaylistItem, error) {
	if playlistID == "" {
		return nil, nil
	}
	q := url.Values{"part": {"snippet"}, "playlistId": {playlistID}, "maxResults": {"50"}}
	b, err := c.get(ctx, "/playlistItems", q)
	if err != nil {
		return nil, err
	}
	var pl struct {
		Items []struct {
			Snippet struct {
				Title                  string `json:"title"`
				ChannelTitle           string `json:"channelTitle"`
				VideoOwnerChannelTitle string `json:"videoOwnerChannelTitle"`
				ResourceID             struct {
					VideoID string `json:"videoId"`
				} `json:"resourceId"`
			} `json:"snippet"`
		} `json:"items"`
	}
	if err := json.Unmarshal(b, &pl); err != nil {
		return nil, err
	}
	var out []PlaylistItem
	for _, it := range pl.Items {
		id := it.Snippet.ResourceID.VideoID
		if id == "" {
			continue
		}
		ch := it.Snippet.VideoOwnerChannelTitle
		if ch == "" {
			ch = it.Snippet.ChannelTitle
		}
		out = append(out, PlaylistItem{VideoID: id, Title: it.Snippet.Title, ChannelTitle: ch})
	}
	return out, nil
}

func ChannelOf(items []PlaylistItem, videoID string) string {
	for _, it := range items {
		if it.VideoID == videoID {
			return it.ChannelTitle
		}
	}
	return ""
}

func OEmbedAuthor(ctx context.Context, videoID string) string {
	if videoID == "" {
		return ""
	}
	u := "https://www.youtube.com/oembed?format=json&url=" + url.QueryEscape("https://www.youtube.com/watch?v="+videoID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return ""
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return ""
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return ""
	}
	var o struct {
		AuthorName string `json:"author_name"`
	}
	b, _ := io.ReadAll(io.LimitReader(res.Body, 1<<16))
	if json.Unmarshal(b, &o) != nil {
		return ""
	}
	return strings.TrimSpace(o.AuthorName)
}

func AccessToken(db *sql.DB, userID, clientID, clientSecret string) (string, error) {
	if userID == "" || clientID == "" || clientSecret == "" {
		return "", nil
	}
	var refresh string
	err := db.QueryRow(`SELECT IFNULL(youtube_refresh_token,'') FROM users WHERE id=?`, userID).Scan(&refresh)
	if err != nil || refresh == "" {
		return "", err
	}
	access, newRefresh, err := RefreshAccess(clientID, clientSecret, refresh)
	if err != nil {
		return "", err
	}
	if newRefresh != "" && newRefresh != refresh {
		_, _ = db.Exec(`UPDATE users SET youtube_refresh_token=? WHERE id=?`, newRefresh, userID)
	}
	return access, nil
}
