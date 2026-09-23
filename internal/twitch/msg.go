package twitch

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	DefaultMsg = "{name} is live on Twitch: {title}\n{url}"
	MaxMsg     = 500
)

type Live struct {
	Name    string
	Login   string
	Title   string
	Started time.Time
}

func ClampMsg(s string) string {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) <= MaxMsg {
		return s
	}
	r := []rune(s)
	return string(r[:MaxMsg])
}

func Render(tmpl string, ev Live) string {
	tmpl = strings.TrimSpace(tmpl)
	if tmpl == "" {
		tmpl = DefaultMsg
	}
	return strings.NewReplacer(
		"{name}", ev.Name,
		"{twitch}", ev.Login,
		"{url}", ChannelURL(ev.Login),
		"{title}", ev.Title,
		"{uptime}", Uptime(ev.Started),
	).Replace(tmpl)
}

func Uptime(started time.Time) string {
	if started.IsZero() {
		return ""
	}
	d := time.Since(started)
	if d < 0 {
		d = 0
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	switch {
	case h > 0:
		return fmt.Sprintf("%dh %dm", h, m)
	case m > 0:
		return fmt.Sprintf("%dm", m)
	default:
		return "just now"
	}
}
