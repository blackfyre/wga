package postcards

import (
	"testing"
	"time"
)

func TestSubmissionLimiterRejectsRequestsBeyondSuccessfulSendAllowance(t *testing.T) {
	limiter := newSubmissionLimiter(1, 10*time.Minute)
	now := time.Now()
	if !limiter.allow("visitor", now) {
		t.Fatal("first successful send was rejected")
	}
	if limiter.allow("visitor", now.Add(time.Second)) {
		t.Fatal("second successful send was not throttled")
	}
}
