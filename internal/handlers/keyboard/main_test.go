package keyboard

import (
	"testing"
	"time"
)

func TestValidQueryRequiresTwoRunes(t *testing.T) {
	if validQuery("a") {
		t.Fatal("one rune query must be rejected")
	}
	if !validQuery("éx") {
		t.Fatal("two rune query must be accepted")
	}
}

func TestRequestLimiterResetsAfterOneMinute(t *testing.T) {
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	limiter := newRequestLimiter()
	limiter.now = func() time.Time { return now }

	for range requestsPerMinute {
		if !limiter.allow("127.0.0.1") {
			t.Fatal("request within limit was rejected")
		}
	}
	if limiter.allow("127.0.0.1") {
		t.Fatal("request above limit was accepted")
	}

	now = now.Add(time.Minute)
	if !limiter.allow("127.0.0.1") {
		t.Fatal("request after rate window reset was rejected")
	}
}
