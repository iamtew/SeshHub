package article

import (
	"bytes"
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark-emoji"
	"github.com/yuin/goldmark/extension"
	gmhtml "github.com/yuin/goldmark/renderer/html"
)

// goldmark + bluemonday, not marked.js. WithUnsafe lets <img> through; the sanitizer is the XSS gate.
// ponytail: UGCPolicy strips relative src unless AllowRelativeURLs; /static/... images need it.
var md = goldmark.New(
	goldmark.WithExtensions(extension.Table, extension.Strikethrough, emoji.Emoji),
	goldmark.WithRendererOptions(gmhtml.WithUnsafe(), gmhtml.WithHardWraps()),
)

var policy = func() *bluemonday.Policy {
	p := bluemonday.UGCPolicy()
	p.AllowRelativeURLs(true)
	p.AllowElements("time")
	p.AllowAttrs("datetime", "class", "data-fmt").OnElements("time")
	return p
}()

var dateTag = regexp.MustCompile(`\[date_(count|local|24h|12h):([^\]]+)\]`)

func Render(raw string) string {
	raw = expandDateTags(raw)
	var buf bytes.Buffer
	if err := md.Convert([]byte(raw), &buf); err != nil {
		return policy.Sanitize(raw)
	}
	return policy.Sanitize(buf.String())
}

func expandDateTags(raw string) string {
	return dateTag.ReplaceAllStringFunc(raw, func(m string) string {
		p := dateTag.FindStringSubmatch(m)
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(p[2]))
		if err != nil {
			return m
		}
		iso := html.EscapeString(t.Format(time.RFC3339))
		fallback := html.EscapeString(t.Format("2 Jan 2006 15:04 MST"))
		return fmt.Sprintf(`<time class="md-date" data-fmt="%s" datetime="%s">%s</time>`, p[1], iso, fallback)
	})
}

func Excerpt(raw, given string) string {
	given = strings.TrimSpace(given)
	if given != "" {
		return given
	}
	plain := strings.Join(strings.Fields(raw), " ")
	runes := []rune(plain)
	if len(runes) <= 180 {
		return plain
	}
	return string(runes[:180]) + "…"
}

func ReadingMinutes(raw string) int {
	n := len(strings.Fields(raw))
	m := (n + 199) / 200
	if m < 1 {
		return 1
	}
	return m
}
