package article

import (
	"bytes"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
)

func Render(raw string) string {
	var buf bytes.Buffer
	if err := goldmark.Convert([]byte(raw), &buf); err != nil {
		return bluemonday.UGCPolicy().Sanitize(raw)
	}
	return bluemonday.UGCPolicy().Sanitize(buf.String())
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
