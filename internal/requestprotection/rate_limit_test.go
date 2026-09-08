package requestprotection

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestRateLimiterKeepsProfileBudgetsSeparate(t *testing.T) {
	limiter := testRateLimiter(RateLimits{Search: 1, Fragment: 2, Detail: 3, EntryCapacity: 4}, time.Now)

	tests := []struct {
		profile Profile
		want    []bool
	}{
		{profile: ProfileSearch, want: []bool{true, false}},
		{profile: ProfileFragment, want: []bool{true, true, false}},
		{profile: ProfileDetail, want: []bool{true, true, true, false}},
	}

	for _, test := range tests {
		t.Run(string(test.profile), func(t *testing.T) {
			for index, want := range test.want {
				if got := limiter.AllowResolved("198.51.100.7", true, test.profile); got != want {
					t.Fatalf("request %d allowed = %t; want %t", index+1, got, want)
				}
			}
		})
	}
}

func TestRateLimiterExpiresWindows(t *testing.T) {
	now := time.Date(2026, time.September, 8, 12, 0, 0, 0, time.UTC)
	limiter := testRateLimiter(RateLimits{Search: 1, Fragment: 1, Detail: 1, EntryCapacity: 2}, func() time.Time { return now })

	if !limiter.AllowResolved("198.51.100.7", true, ProfileSearch) {
		t.Fatal("first request should be allowed")
	}
	if limiter.AllowResolved("198.51.100.7", true, ProfileSearch) {
		t.Fatal("request within exhausted window should be rejected")
	}

	now = now.Add(clientRateWindow)
	if !limiter.AllowResolved("198.51.100.7", true, ProfileSearch) {
		t.Fatal("request at window expiry should be allowed")
	}
}

func TestRateLimiterFailsClosedWithoutResolvedIdentity(t *testing.T) {
	limiter := testRateLimiter(RateLimits{Search: 1, Fragment: 1, Detail: 1, EntryCapacity: 2}, time.Now)

	if limiter.AllowResolved("visitor-supplied", false, ProfileSearch) {
		t.Fatal("missing resolved identity should fail closed")
	}
	if len(limiter.clients) != 0 {
		t.Fatalf("missing identity created %d limiter entries", len(limiter.clients))
	}
	if !limiter.AllowResolved("", false, ProfileExempt) {
		t.Fatal("exempt profile should not require an identity")
	}
}

func TestRateLimiterEvictsOldestEntryAtFixedCapacity(t *testing.T) {
	limiter := testRateLimiter(RateLimits{Search: 2, Fragment: 2, Detail: 2, EntryCapacity: 2}, time.Now)
	oldest := limiter.privateIdentity("198.51.100.1")
	second := limiter.privateIdentity("198.51.100.2")
	newest := limiter.privateIdentity("198.51.100.3")

	limiter.AllowResolved("198.51.100.1", true, ProfileSearch)
	limiter.AllowResolved("198.51.100.2", true, ProfileSearch)
	limiter.AllowResolved("198.51.100.3", true, ProfileSearch)

	if len(limiter.clients) != 2 {
		t.Fatalf("retained entries = %d; want 2", len(limiter.clients))
	}
	if _, found := limiter.clients[oldest]; found {
		t.Fatal("oldest entry was not evicted")
	}
	if _, found := limiter.clients[second]; !found {
		t.Fatal("second entry was unexpectedly evicted")
	}
	if _, found := limiter.clients[newest]; !found {
		t.Fatal("new entry was not retained")
	}
}

func TestRateLimiterBoundsRotatingAddresses(t *testing.T) {
	limiter := testRateLimiter(RateLimits{Search: 1, Fragment: 1, Detail: 1, EntryCapacity: 2}, time.Now)

	if !limiter.AllowResolved("198.51.100.1", true, ProfileSearch) || limiter.AllowResolved("198.51.100.1", true, ProfileSearch) {
		t.Fatal("first address did not receive an independent one-request budget")
	}
	if !limiter.AllowResolved("198.51.100.2", true, ProfileSearch) || !limiter.AllowResolved("198.51.100.3", true, ProfileSearch) {
		t.Fatal("rotated addresses should receive independent budgets")
	}
	if len(limiter.clients) != 2 {
		t.Fatalf("rotating addresses retained %d entries; want bounded capacity 2", len(limiter.clients))
	}
	if !limiter.AllowResolved("198.51.100.1", true, ProfileSearch) {
		t.Fatal("evicted address should begin a new window when it returns")
	}
	if len(limiter.clients) != 2 {
		t.Fatalf("returning address retained %d entries; want bounded capacity 2", len(limiter.clients))
	}
}

func TestRateLimiterRetainsOnlyProcessPrivateIdentity(t *testing.T) {
	const identity = "198.51.100.7"
	limits := RateLimits{Search: 1, Fragment: 1, Detail: 1, EntryCapacity: 2}
	first := testRateLimiter(limits, time.Now)
	secondKey := [sha256.Size]byte{2}
	second := newRateLimiter(limits, secondKey, time.Now)

	first.AllowResolved(identity, true, ProfileSearch)
	second.AllowResolved(identity, true, ProfileSearch)

	firstIdentity := first.privateIdentity(identity)
	secondIdentity := second.privateIdentity(identity)
	if firstIdentity == secondIdentity {
		t.Fatal("independent process keys produced the same private identity")
	}
	if _, found := first.clients[firstIdentity]; !found {
		t.Fatal("private identity was not retained")
	}
	for _, formatted := range []string{fmt.Sprint(first), fmt.Sprintf("%+v", first), fmt.Sprintf("%#v", first)} {
		if strings.Contains(formatted, identity) {
			t.Fatalf("formatted limiter exposed raw identity: %q", formatted)
		}
	}
}

func TestNewRateLimiterRejectsInvalidLimits(t *testing.T) {
	for _, limits := range []RateLimits{
		{},
		{Search: 0, Fragment: 1, Detail: 1, EntryCapacity: 1},
		{Search: 1, Fragment: 0, Detail: 1, EntryCapacity: 1},
		{Search: 1, Fragment: 1, Detail: 0, EntryCapacity: 1},
		{Search: 1, Fragment: 1, Detail: 1, EntryCapacity: 0},
	} {
		if _, err := NewRateLimiter(limits); err == nil {
			t.Fatalf("NewRateLimiter(%+v) succeeded; want error", limits)
		}
	}
}

func testRateLimiter(limits RateLimits, now func() time.Time) *RateLimiter {
	key := [sha256.Size]byte{1}
	return newRateLimiter(limits, key, now)
}
