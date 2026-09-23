package skater

import "testing"

func TestParseLinkRejects(t *testing.T) {
	for _, s := range []string{"javascript:alert(1)", "ftp://x.com", "not a url", "https://user:pass@x.com"} {
		if _, ok := parseLink(s); ok {
			t.Fatalf("accepted %q", s)
		}
	}
}

func TestParseLinkBrands(t *testing.T) {
	l, ok := parseLink("https://www.instagram.com/foo")
	if !ok || l.Icon != "fa-brands fa-instagram" || l.Label != "Instagram" {
		t.Fatalf("instagram: %+v %v", l, ok)
	}
	l, ok = parseLink("https://youtu.be/abc")
	if !ok || l.Icon != "fa-brands fa-youtube" {
		t.Fatalf("youtube: %+v %v", l, ok)
	}
	l, ok = parseLink("https://example.com/x")
	if !ok || l.Icon != "fa-solid fa-link" || l.Label != "example.com" {
		t.Fatalf("unknown: %+v %v", l, ok)
	}
}

func TestJoinLinksCapAndSkip(t *testing.T) {
	got := JoinLinks([]string{"https://x.com/a", "javascript:x", "https://x.com/a", ""})
	if got != "https://x.com/a" {
		t.Fatalf("got %q", got)
	}
	var many []string
	for i := 0; i < 15; i++ {
		many = append(many, "https://example.com/"+string(rune('a'+i)))
	}
	if n := len(Lines(JoinLinks(many))); n != MaxLinks {
		t.Fatalf("cap %d", n)
	}
}

func TestProfileLinksYouTubeFirst(t *testing.T) {
	links := ProfileLinks("UC1", "https://www.youtube.com/watch?v=x\nhttps://twitch.tv/n")
	if len(links) != 2 || links[0].Icon != "fa-brands fa-youtube" || links[1].Icon != "fa-brands fa-twitch" {
		t.Fatalf("%+v", links)
	}
}
