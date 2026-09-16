package yt

import (
	"database/sql"
	"encoding/json"
	"net/url"
	"strings"
)

const MaxRules = 8
const maxValue = 80

type Rule struct {
	Field string `json:"field"`
	Value string `json:"value"`
}

func ParseRules(raw string) []Rule {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var in []Rule
	if json.Unmarshal([]byte(raw), &in) != nil {
		return nil
	}
	return Normalize(in)
}

func EncodeRules(rules []Rule) string {
	rules = Normalize(rules)
	if len(rules) == 0 {
		return ""
	}
	b, err := json.Marshal(rules)
	if err != nil {
		return ""
	}
	return string(b)
}

func FormRules(fields, values []string) []Rule {
	n := len(fields)
	if len(values) > n {
		n = len(values)
	}
	if n > MaxRules {
		n = MaxRules
	}
	out := make([]Rule, 0, n)
	for i := 0; i < n; i++ {
		r := Rule{}
		if i < len(fields) {
			r.Field = strings.TrimSpace(fields[i])
		}
		if i < len(values) {
			r.Value = strings.TrimSpace(values[i])
		}
		out = append(out, r)
	}
	return out
}

func WithBlank(rows []Rule) []Rule {
	if len(rows) >= MaxRules {
		return rows
	}
	out := make([]Rule, len(rows)+1)
	copy(out, rows)
	return out
}

func Normalize(in []Rule) []Rule {
	var out []Rule
	for _, r := range in {
		if len(out) >= MaxRules {
			break
		}
		r.Field = strings.ToLower(strings.TrimSpace(r.Field))
		r.Value = strings.TrimSpace(r.Value)
		if r.Value == "" {
			continue
		}
		if len(r.Value) > maxValue {
			r.Value = r.Value[:maxValue]
		}
		switch r.Field {
		case "title", "channel", "tags":
			out = append(out, r)
		case "category":
			r.Value = strings.ToLower(r.Value)
			switch r.Value {
			case "session", "short", "contest", "part":
				out = append(out, r)
			}
		}
	}
	return out
}

func Match(v Video, rules []Rule) bool {
	for _, r := range rules {
		if !matchOne(v, r) {
			return false
		}
	}
	return true
}

func matchOne(v Video, r Rule) bool {
	switch r.Field {
	case "title":
		return strings.Contains(strings.ToLower(v.Title), strings.ToLower(r.Value))
	case "channel":
		return strings.Contains(strings.ToLower(v.ChannelTitle), strings.ToLower(r.Value))
	case "tags":
		return strings.Contains(strings.ToLower(v.Tags), strings.ToLower(r.Value))
	case "category":
		return strings.EqualFold(v.Category, r.Value)
	default:
		return true
	}
}

func Filter(list []Video, rules []Rule) []Video {
	if len(rules) == 0 {
		return list
	}
	var out []Video
	for _, v := range list {
		if Match(v, rules) {
			out = append(out, v)
		}
	}
	return out
}

func Query(rules []Rule) string {
	q := url.Values{}
	q.Set("test", "1")
	for _, r := range Normalize(rules) {
		q.Add("field", r.Field)
		q.Add("value", r.Value)
	}
	return "&" + q.Encode()
}

func GetFilter(db *sql.DB) (string, error) {
	var s sql.NullString
	err := db.QueryRow(`SELECT rules FROM site_video_filter WHERE id=1`).Scan(&s)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return s.String, err
}

func SetFilter(db *sql.DB, raw string) error {
	// ponytail: one site row, last save wins; per-editor history if two people fight over it
	var arg any
	if raw != "" {
		arg = raw
	}
	_, err := db.Exec(`INSERT INTO site_video_filter (id, rules) VALUES (1, ?) ON CONFLICT(id) DO UPDATE SET rules=excluded.rules`, arg)
	return err
}
