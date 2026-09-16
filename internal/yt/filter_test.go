package yt

import "testing"

func TestMatchAndFilter(t *testing.T) {
	v := Video{Title: "Wheel Session", ChannelTitle: "Sofa TV", Tags: "street,clip", Category: "session"}
	if !Match(v, nil) {
		t.Fatal("empty rules")
	}
	if !Match(v, []Rule{{Field: "title", Value: "wheel"}}) {
		t.Fatal("title")
	}
	if Match(v, []Rule{{Field: "title", Value: "contest"}}) {
		t.Fatal("title miss")
	}
	if !Match(v, []Rule{{Field: "title", Value: "wheel"}, {Field: "category", Value: "session"}}) {
		t.Fatal("and")
	}
	if Match(v, []Rule{{Field: "title", Value: "wheel"}, {Field: "category", Value: "short"}}) {
		t.Fatal("and miss")
	}
	if !Match(v, []Rule{{Field: "channel", Value: "sofa"}}) {
		t.Fatal("channel")
	}
	if !Match(v, []Rule{{Field: "tags", Value: "STREET"}}) {
		t.Fatal("tags")
	}
	if !Match(v, []Rule{{Field: "category", Value: "SESSION"}}) {
		t.Fatal("category fold")
	}
	if len(ParseRules("not json")) != 0 {
		t.Fatal("bad json")
	}
	got := ParseRules(`[{"field":"title","value":"  x  "},{"field":"nope","value":"y"}]`)
	if len(got) != 1 || got[0].Field != "title" || got[0].Value != "x" {
		t.Fatalf("%#v", got)
	}
	if EncodeRules(nil) != "" {
		t.Fatal("encode empty")
	}
	list := []Video{v, {Title: "Other"}}
	keep := Filter(list, []Rule{{Field: "title", Value: "wheel"}})
	if len(keep) != 1 || keep[0].Title != v.Title {
		t.Fatalf("filter %#v", keep)
	}
	owned := []Video{
		{ChannelID: "a", Title: "Wheel Session"},
		{ChannelID: "a", Title: "Other"},
		{ChannelID: "b", Title: "Other"},
	}
	shown := FilterOwned(owned, map[string][]Rule{"a": {{Field: "title", Value: "wheel"}}})
	if len(shown) != 2 || shown[0].Title != "Wheel Session" || shown[1].ChannelID != "b" {
		t.Fatalf("owned %#v", shown)
	}
	if n := len(Normalize(append(FormRules([]string{"title", "title"}, []string{"a", ""}), Rule{Field: "category", Value: "nope"}))); n != 1 {
		t.Fatalf("normalize %d", n)
	}
}
