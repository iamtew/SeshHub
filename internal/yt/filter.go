package yt

import (
	"database/sql"
	"encoding/json"
	"regexp"
	"strings"
)

const MaxRules = 8
const maxValue = 80

type Rule struct {
	Field string `json:"field"`
	Op    string `json:"op"`
	Value string `json:"value"`
}

type Spec struct {
	Rules []Rule   `json:"rules,omitempty"`
	Kinds []string `json:"kinds,omitempty"`
}

func ParseSpec(raw string) Spec {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Spec{}
	}
	if strings.HasPrefix(raw, "{") {
		var s Spec
		if json.Unmarshal([]byte(raw), &s) != nil {
			return Spec{}
		}
		return NormalizeSpec(s)
	}
	var rules []Rule
	if json.Unmarshal([]byte(raw), &rules) != nil {
		return Spec{}
	}
	return NormalizeSpec(Spec{Rules: rules})
}

func EncodeSpec(s Spec) string {
	s = NormalizeSpec(s)
	if len(s.Rules) == 0 && len(s.Kinds) == 0 {
		return ""
	}
	b, err := json.Marshal(s)
	if err != nil {
		return ""
	}
	return string(b)
}

func FormRules(fields, ops, values []string) []Rule {
	n := len(fields)
	if len(ops) > n {
		n = len(ops)
	}
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
		if i < len(ops) {
			r.Op = strings.TrimSpace(ops[i])
		}
		if i < len(values) {
			r.Value = strings.TrimSpace(values[i])
		}
		out = append(out, r)
	}
	return out
}

func FormSpec(fields, ops, values, kinds []string) Spec {
	return Spec{Rules: FormRules(fields, ops, values), Kinds: kinds}
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
		r.Op = strings.ToLower(strings.TrimSpace(r.Op))
		r.Value = strings.TrimSpace(r.Value)
		if r.Value == "" {
			continue
		}
		if len(r.Value) > maxValue {
			r.Value = r.Value[:maxValue]
		}
		switch r.Field {
		case "title", "tags", "category":
		default:
			continue
		}
		switch r.Op {
		case "", "contains":
			r.Op = "contains"
		case "not_contains", "prefix", "suffix", "regexp":
		default:
			continue
		}
		out = append(out, r)
	}
	return out
}

func NormalizeSpec(s Spec) Spec {
	s.Rules = Normalize(s.Rules)
	var kinds []string
	seen := map[string]bool{}
	for _, k := range s.Kinds {
		k = strings.ToLower(strings.TrimSpace(k))
		switch k {
		case "video", "short", "live", "premiere":
			if !seen[k] {
				seen[k] = true
				kinds = append(kinds, k)
			}
		}
	}
	s.Kinds = kinds
	return s
}

func Match(v Video, spec Spec) bool {
	if len(spec.Kinds) > 0 {
		kind := UploadType(v)
		ok := false
		for _, k := range spec.Kinds {
			if k == kind {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	for _, r := range spec.Rules {
		if !matchOne(v, r) {
			return false
		}
	}
	return true
}

func matchOne(v Video, r Rule) bool {
	raw := ""
	switch r.Field {
	case "title":
		raw = v.Title
	case "tags":
		raw = v.Tags
	case "category":
		raw = v.Category
	default:
		return false
	}
	if r.Op == "regexp" {
		re, err := regexp.Compile(r.Value)
		if err != nil {
			return false
		}
		return re.MatchString(raw)
	}
	hay, needle := strings.ToLower(raw), strings.ToLower(r.Value)
	switch r.Op {
	case "not_contains":
		return !strings.Contains(hay, needle)
	case "prefix":
		return strings.HasPrefix(hay, needle)
	case "suffix":
		return strings.HasSuffix(hay, needle)
	default:
		return strings.Contains(hay, needle)
	}
}

func Filter(list []Video, spec Spec) []Video {
	if len(spec.Rules) == 0 && len(spec.Kinds) == 0 {
		return list
	}
	var out []Video
	for _, v := range list {
		if Match(v, spec) {
			out = append(out, v)
		}
	}
	return out
}

func OwnerFilters(db *sql.DB) (map[string]Spec, error) {
	rows, err := db.Query(`SELECT youtube_channel_id, IFNULL(video_filter,'') FROM users WHERE IFNULL(youtube_channel_id,'') != ''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]Spec{}
	for rows.Next() {
		var ch, raw string
		if err := rows.Scan(&ch, &raw); err != nil {
			return nil, err
		}
		out[ch] = ParseSpec(raw)
	}
	return out, rows.Err()
}

func FilterOwned(list []Video, byChannel map[string]Spec) []Video {
	if len(byChannel) == 0 {
		return list
	}
	var out []Video
	for _, v := range list {
		if Match(v, byChannel[v.ChannelID]) {
			out = append(out, v)
		}
	}
	return out
}
