package article

import (
	"bytes"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/renderer/html"
)

// goldmark + bluemonday, not marked.js. WithUnsafe lets <img> through; the sanitizer is the XSS gate.
// ponytail: UGCPolicy strips relative src unless AllowRelativeURLs; /static/... images need it.
var md = goldmark.New(goldmark.WithRendererOptions(html.WithUnsafe()))

var policy = func() *bluemonday.Policy {
	p := bluemonday.UGCPolicy()
	p.AllowRelativeURLs(true)
	return p
}()

func Render(raw string) string {
	var buf bytes.Buffer
	if err := md.Convert([]byte(raw), &buf); err != nil {
		return policy.Sanitize(raw)
	}
	return policy.Sanitize(buf.String())
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
