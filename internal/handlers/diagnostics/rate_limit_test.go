package diagnostics

import (
	"strconv"
	"testing"
	"time"
)

func newTestLimiter(t *testing.T, now *time.Time) *rateLimiter {
	t.Helper()
	limiter, err := newRateLimiter(func() time.Time { return *now })
	if err != nil {
		t.Fatal(err)
	}
	return limiter
}

func TestRateLimiterWindowExpires(t *testing.T) {
	now := time.Unix(0, 0)
	limiter := newTestLimiter(t, &now)
	for range rateLimitPerWindow {
		if !limiter.allow("203.0.113.7") {
			t.Fatal("request within budget rejected")
		}
	}
	if limiter.allow("203.0.113.7") {
		t.Fatal("request over budget allowed")
	}

	now = now.Add(rateLimitWindow)
	if !limiter.allow("203.0.113.7") {
		t.Fatal("budget did not reset after the window")
	}
}

func TestRateLimiterStaysBoundedAndPrefersExpiredEntries(t *testing.T) {
	now := time.Unix(0, 0)
	limiter := newTestLimiter(t, &now)
	limiter.allow("198.51.100.1")
	now = now.Add(rateLimitWindow)

	for i := range rateLimitEntries + 10 {
		limiter.allow("10.0." + strconv.Itoa(i/256) + "." + strconv.Itoa(i%256))
		if len(limiter.windows) > rateLimitEntries {
			t.Fatalf("limiter grew to %d entries", len(limiter.windows))
		}
	}
	if _, found := limiter.windows[limiter.privateIdentity("198.51.100.1")]; found {
		t.Fatal("expired window was retained while the table was full")
	}
}

func TestRateLimiterGroupsIPv6ByPrefix(t *testing.T) {
	now := time.Unix(0, 0)
	limiter := newTestLimiter(t, &now)
	for i := range rateLimitPerWindow {
		limiter.allow("2001:db8:1:2::" + strconv.FormatInt(int64(i+1), 16))
	}
	if limiter.allow("2001:db8:1:2:ffff::1") {
		t.Fatal("rotating addresses within one /64 escaped the limit")
	}
	if !limiter.allow("2001:db8:1:3::1") {
		t.Fatal("another /64 was limited")
	}
	if clientBucket("::ffff:203.0.113.7") != "203.0.113.7" {
		t.Fatalf("IPv4-mapped address bucket = %q", clientBucket("::ffff:203.0.113.7"))
	}
}
