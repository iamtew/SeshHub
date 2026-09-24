package twitch

import "testing"

func TestAuditionToggleExpires(t *testing.T) {
	t.Cleanup(func() {
		auditionUntil.Store(0)
		auditionStart.Store(0)
	})
	if On() {
		t.Fatal("start off")
	}
	if !Toggle() || !On() || Started().IsZero() || RemainingLabel() == "" {
		t.Fatalf("on remaining=%q started=%v", RemainingLabel(), Started())
	}
	if Toggle() || On() || RemainingLabel() != "" {
		t.Fatal("toggle off")
	}
	if !Toggle() {
		t.Fatal("on again")
	}
	auditionUntil.Store(1)
	if On() || RemainingLabel() != "" {
		t.Fatal("expired")
	}
}
