package web

import "testing"

func TestVideoPage(t *testing.T) {
	per, page, offset, from, to := videoPage(9, 1, 248)
	if per != 9 || page != 1 || offset != 0 || from != 1 || to != 9 {
		t.Fatalf("p1 got per=%d page=%d off=%d %d–%d", per, page, offset, from, to)
	}
	per, page, offset, from, to = videoPage(9, 28, 248)
	if per != 9 || page != 28 || offset != 243 || from != 244 || to != 248 {
		t.Fatalf("last got per=%d page=%d off=%d %d–%d", per, page, offset, from, to)
	}
	per, page, _, from, to = videoPage(99, 0, 248)
	if per != 9 || page != 1 || from != 1 || to != 9 {
		t.Fatalf("clamp got per=%d page=%d %d–%d", per, page, from, to)
	}
	per, page, _, from, to = videoPage(18, 100, 248)
	if per != 18 || page != 14 || from != 235 || to != 248 {
		t.Fatalf("oversize got per=%d page=%d %d–%d", per, page, from, to)
	}
	per, page, offset, from, to = videoPage(27, 1, 0)
	if per != 27 || page != 1 || offset != 0 || from != 0 || to != 0 {
		t.Fatalf("empty got per=%d page=%d off=%d %d–%d", per, page, offset, from, to)
	}
}
