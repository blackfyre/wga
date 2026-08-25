package postcards

import (
	"testing"
	"time"
)

func TestSubmissionLimiterIsPerInstanceBoundedAndResets(t *testing.T) {
	now := time.Unix(1000, 0)
	limiter := newSubmissionLimiter(3, 10*time.Minute)
	for range 3 {
		if !limiter.allow("visitor", now) {
			t.Fatal("allowed request rejected")
		}
	}
	if limiter.allow("visitor", now) {
		t.Fatal("fourth request accepted")
	}
	if !newSubmissionLimiter(3, 10*time.Minute).allow("visitor", now) {
		t.Fatal("limiter state leaked between registrations")
	}
	if !limiter.allow("visitor", now.Add(10*time.Minute)) {
		t.Fatal("window did not reset")
	}

	limited := newSubmissionLimiter(1, time.Hour)
	limited.maxKeys = 2
	if !limited.allow("one", now) || !limited.allow("two", now) {
		t.Fatal("bounded keys rejected early")
	}
	if limited.allow("three", now) {
		t.Fatal("bounded limiter accepted unbounded visitor state")
	}
}
