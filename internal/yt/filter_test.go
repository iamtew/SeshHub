package yt

import "testing"

func TestMatchAndFilter(t *testing.T) {
	v := Video{Title: "Wheel Session EP01", Tags: "street,clip", Category: "session", Duration: 400}
	if !Match(v, Spec{}) {
		t.Fatal("empty spec")
	}
	if !Match(v, Spec{Rules: []Rule{{Field: "title", Op: "contains", Value: "EP"}}}) {
		t.Fatal("contains EP")
	}
	if Match(v, Spec{Rules: []Rule{{Field: "title", Op: "contains", Value: "EP"}}}) == Match(Video{Title: "Night Jam"}, Spec{Rules: []Rule{{Field: "title", Op: "contains", Value: "EP"}}}) {
		t.Fatal("contains EP must miss Night Jam")
	}
	if Match(Video{Title: "Night Jam"}, Spec{Rules: []Rule{{Field: "title", Op: "contains", Value: "EP"}}}) {
		t.Fatal("title miss")
	}
	if Match(v, Spec{Rules: []Rule{{Field: "title", Op: "not_contains", Value: "EP"}}}) {
		t.Fatal("not contains")
	}
	if !Match(v, Spec{Rules: []Rule{{Field: "title", Op: "prefix", Value: "wheel"}}}) {
		t.Fatal("prefix")
	}
	if !Match(v, Spec{Rules: []Rule{{Field: "title", Op: "suffix", Value: "01"}}}) {
		t.Fatal("suffix")
	}
	if !Match(v, Spec{Rules: []Rule{{Field: "title", Op: "regexp", Value: `EP\d+`}}}) {
		t.Fatal("regexp")
	}
	if Match(v, Spec{Rules: []Rule{{Field: "title", Op: "regexp", Value: `^EP`}}}) {
		t.Fatal("regexp miss")
	}
	if !Match(v, Spec{Rules: []Rule{{Field: "title", Value: "wheel"}, {Field: "category", Op: "contains", Value: "sess"}}}) {
		t.Fatal("and")
	}
	if !Match(v, Spec{Rules: []Rule{{Field: "tags", Value: "STREET"}}}) {
		t.Fatal("tags")
	}
	if ParseSpec("not json").Rules != nil {
		t.Fatal("bad json")
	}
	got := ParseSpec(`[{"field":"title","value":"  x  "},{"field":"channel","value":"y"}]`)
	if len(got.Rules) != 1 || got.Rules[0].Field != "title" || got.Rules[0].Op != "contains" || got.Rules[0].Value != "x" {
		t.Fatalf("%#v", got)
	}
	round := ParseSpec(EncodeSpec(Spec{Rules: []Rule{{Field: "title", Op: "contains", Value: "EP"}}, Kinds: []string{"video"}}))
	if len(round.Rules) != 1 || round.Rules[0].Value != "EP" || len(round.Kinds) != 1 || round.Kinds[0] != "video" {
		t.Fatalf("round %#v", round)
	}
	if EncodeSpec(Spec{}) != "" {
		t.Fatal("encode empty")
	}
	list := []Video{v, {Title: "Other"}}
	keep := Filter(list, Spec{Rules: []Rule{{Field: "title", Op: "contains", Value: "EP"}}})
	if len(keep) != 1 || keep[0].Title != v.Title {
		t.Fatalf("filter %#v", keep)
	}
	owned := []Video{
		{ChannelID: "a", Title: "Wheel Session EP01"},
		{ChannelID: "a", Title: "Other"},
		{ChannelID: "b", Title: "Other"},
	}
	shown := FilterOwned(owned, map[string]Spec{"a": {Rules: []Rule{{Field: "title", Op: "contains", Value: "EP"}}}})
	if len(shown) != 2 || shown[0].Title != "Wheel Session EP01" || shown[1].ChannelID != "b" {
		t.Fatalf("owned %#v", shown)
	}
	if n := len(Normalize(append(FormRules([]string{"title", "title"}, []string{"contains", "contains"}, []string{"a", ""}), Rule{Field: "channel", Value: "x"}))); n != 1 {
		t.Fatalf("normalize %d", n)
	}
	short := Video{Title: "clip", Duration: 40}
	live := Video{Title: "now", Live: "live", Duration: 20}
	pre := Video{Title: "soon", Live: "upcoming"}
	if UploadType(v) != "video" || UploadType(short) != "short" || UploadType(live) != "live" || UploadType(pre) != "premiere" {
		t.Fatal("kinds")
	}
	if Match(short, Spec{Kinds: []string{"video"}}) || !Match(v, Spec{Kinds: []string{"video"}}) {
		t.Fatal("kind filter")
	}
}
