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

func TestRequestLimiterRemovesExpiredClients(t *testing.T) {
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	limiter := newRequestLimiter()
	limiter.now = func() time.Time { return now }
	limiter.clients["expired"] = requestWindow{started: now.Add(-time.Minute)}

	if !limiter.allow("current") {
		t.Fatal("request within limit was rejected")
	}
	if _, ok := limiter.clients["expired"]; ok {
		t.Fatal("expired client was not removed")
	}
}

func TestSuggestionLimitForBoundsRequestedCapacity(t *testing.T) {
	if got := suggestionLimitFor("3"); got != 3 {
		t.Fatalf("limit = %d, want 3", got)
	}
	if got := suggestionLimitFor("0"); got != suggestionLimit {
		t.Fatalf("zero limit = %d, want %d", got, suggestionLimit)
	}
	if got := suggestionLimitFor("99"); got != suggestionLimit {
		t.Fatalf("capped limit = %d, want %d", got, suggestionLimit)
	}
}

func TestArtistLabelIncludesKnownLifespan(t *testing.T) {
	if got := artistLabel("Vermeer", 1632, 1675); got != "Vermeer · 1632–1675" {
		t.Fatalf("artist label = %q", got)
	}
}
