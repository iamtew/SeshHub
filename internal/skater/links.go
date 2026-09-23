package skater

import (
	"net/url"
	"strings"

	"seshhub/internal/twitch"
)

const MaxLinks = 10

type Link struct {
	URL   string
	Label string
	Icon  string
}

func JoinLinks(raw []string) string {
	var out []string
	seen := map[string]bool{}
	for _, s := range raw {
		l, ok := parseLink(s)
		if !ok || seen[l.URL] {
			continue
		}
		seen[l.URL] = true
		out = append(out, l.URL)
		if len(out) >= MaxLinks {
			break
		}
	}
	return strings.Join(out, "\n")
}

func ParseLinks(s string) []Link {
	var out []Link
	for _, line := range Lines(s) {
		if l, ok := parseLink(line); ok {
			out = append(out, l)
		}
	}
	return out
}

func ChannelLink(channelID string) (Link, bool) {
	id := strings.TrimSpace(channelID)
	if id == "" {
		return Link{}, false
	}
	return Link{URL: "https://www.youtube.com/channel/" + id, Label: "YouTube", Icon: "fa-brands fa-youtube"}, true
}

func ProfileLinks(channelID, social, twitchLogin string) []Link {
	var out []Link
	yt, hasYT := ChannelLink(channelID)
	if hasYT {
		out = append(out, yt)
	}
	tw, hasTW := twitchLink(twitchLogin)
	if hasTW {
		out = append(out, tw)
	}
	for _, l := range ParseLinks(social) {
		if hasYT && l.Icon == "fa-brands fa-youtube" {
			continue
		}
		if hasTW && l.Icon == "fa-brands fa-twitch" {
			continue
		}
		out = append(out, l)
	}
	return out
}

func twitchLink(login string) (Link, bool) {
	u := twitch.ChannelURL(login)
	if u == "" {
		return Link{}, false
	}
	return Link{URL: u, Label: "Twitch", Icon: "fa-brands fa-twitch"}, true
}

func parseLink(s string) (Link, bool) {
	s = strings.TrimSpace(s)
	if s == "" || len(s) > 2048 {
		return Link{}, false
	}
	u, err := url.Parse(s)
	if err != nil || u.Host == "" || u.User != nil {
		return Link{}, false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return Link{}, false
	}
	u.Fragment = ""
	icon, label := brand(u.Hostname())
	return Link{URL: u.String(), Label: label, Icon: icon}, true
}

func brand(host string) (icon, label string) {
	host = strings.ToLower(strings.TrimPrefix(host, "www."))
	switch {
	case host == "youtu.be" || strings.HasSuffix(host, ".youtube.com") || host == "youtube.com":
		return "fa-brands fa-youtube", "YouTube"
	case host == "instagram.com" || strings.HasSuffix(host, ".instagram.com"):
		return "fa-brands fa-instagram", "Instagram"
	case host == "twitch.tv" || strings.HasSuffix(host, ".twitch.tv"):
		return "fa-brands fa-twitch", "Twitch"
	case host == "discord.com" || host == "discord.gg" || strings.HasSuffix(host, ".discord.com"):
		return "fa-brands fa-discord", "Discord"
	case host == "tiktok.com" || strings.HasSuffix(host, ".tiktok.com"):
		return "fa-brands fa-tiktok", "TikTok"
	case host == "x.com" || host == "twitter.com" || strings.HasSuffix(host, ".x.com") || strings.HasSuffix(host, ".twitter.com"):
		return "fa-brands fa-x-twitter", "X"
	case host == "facebook.com" || host == "fb.com" || strings.HasSuffix(host, ".facebook.com"):
		return "fa-brands fa-facebook", "Facebook"
	case host == "github.com" || strings.HasSuffix(host, ".github.com"):
		return "fa-brands fa-github", "GitHub"
	case host == "linkedin.com" || strings.HasSuffix(host, ".linkedin.com"):
		return "fa-brands fa-linkedin", "LinkedIn"
	case host == "reddit.com" || strings.HasSuffix(host, ".reddit.com"):
		return "fa-brands fa-reddit", "Reddit"
	case host == "spotify.com" || strings.HasSuffix(host, ".spotify.com"):
		return "fa-brands fa-spotify", "Spotify"
	case host == "soundcloud.com" || strings.HasSuffix(host, ".soundcloud.com"):
		return "fa-brands fa-soundcloud", "SoundCloud"
	case host == "patreon.com" || strings.HasSuffix(host, ".patreon.com"):
		return "fa-brands fa-patreon", "Patreon"
	case host == "vimeo.com" || strings.HasSuffix(host, ".vimeo.com"):
		return "fa-brands fa-vimeo", "Vimeo"
	case host == "snapchat.com" || strings.HasSuffix(host, ".snapchat.com"):
		return "fa-brands fa-snapchat", "Snapchat"
	case host == "t.me" || host == "telegram.org" || strings.HasSuffix(host, ".telegram.org"):
		return "fa-brands fa-telegram", "Telegram"
	case host == "pinterest.com" || strings.HasSuffix(host, ".pinterest.com"):
		return "fa-brands fa-pinterest", "Pinterest"
	case host == "tumblr.com" || strings.HasSuffix(host, ".tumblr.com"):
		return "fa-brands fa-tumblr", "Tumblr"
	case host == "bandcamp.com" || strings.HasSuffix(host, ".bandcamp.com"):
		return "fa-brands fa-bandcamp", "Bandcamp"
	case host == "steampowered.com" || host == "steamcommunity.com" || strings.HasSuffix(host, ".steampowered.com") || strings.HasSuffix(host, ".steamcommunity.com"):
		return "fa-brands fa-steam", "Steam"
	default:
		return "fa-solid fa-link", host
	}
}
