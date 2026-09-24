package twitch

import (
	"fmt"
	"sync/atomic"
	"time"
)

const AuditionFor = 15 * time.Minute

// ponytail: process-global; DB if more than one binary shares a site.
var auditionUntil, auditionStart atomic.Int64

func On() bool {
	return time.Now().UnixNano() < auditionUntil.Load()
}

func Toggle() bool {
	if On() {
		auditionUntil.Store(0)
		auditionStart.Store(0)
		return false
	}
	now := time.Now()
	auditionStart.Store(now.UnixNano())
	auditionUntil.Store(now.Add(AuditionFor).UnixNano())
	return true
}

func Started() time.Time {
	n := auditionStart.Load()
	if n == 0 {
		return time.Time{}
	}
	return time.Unix(0, n)
}

func RemainingLabel() string {
	n := auditionUntil.Load()
	if n == 0 {
		return ""
	}
	d := time.Until(time.Unix(0, n))
	if d <= 0 {
		return ""
	}
	m := int(d.Minutes())
	if m < 1 {
		return "under 1m"
	}
	return fmt.Sprintf("%dm", m)
}
