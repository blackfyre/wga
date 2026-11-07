package postcards

import (
	"sync"
	"time"
)

const maxSubmissionVisitorWindows = 10000

type submissionWindow struct {
	started time.Time
	count   int
}

// submissionLimiter is process-local by design: it avoids persisting visitor IP addresses.
type submissionLimiter struct {
	mu       sync.Mutex
	maximum  int
	duration time.Duration
	windows  map[string]submissionWindow
	maxKeys  int
}

func newSubmissionLimiter(maximum int, duration time.Duration) *submissionLimiter {
	return &submissionLimiter{maximum: maximum, duration: duration, windows: make(map[string]submissionWindow), maxKeys: maxSubmissionVisitorWindows}
}

// submissionAdmission identifies the window that admitted one submission, so
// a refund can return the admission to that window and no other.
type submissionAdmission struct {
	visitorKey string
	started    time.Time
}

func (l *submissionLimiter) allow(visitorKey string, now time.Time) bool {
	_, ok := l.admit(visitorKey, now)
	return ok
}

// admit records one submission in the visitor's current window and returns
// the admission for a later release.
func (l *submissionLimiter) admit(visitorKey string, now time.Time) (submissionAdmission, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	for key, window := range l.windows {
		if !now.Before(window.started.Add(l.duration)) {
			delete(l.windows, key)
		}
	}
	window := l.windows[visitorKey]
	if window.started.IsZero() && len(l.windows) >= l.maxKeys {
		return submissionAdmission{}, false
	}
	if window.started.IsZero() {
		window.started = now
	}
	if window.count >= l.maximum {
		return submissionAdmission{}, false
	}
	window.count++
	l.windows[visitorKey] = window
	return submissionAdmission{visitorKey: visitorKey, started: window.started}, true
}

// release returns an admission whose submission did not queue a postcard, so
// only successful sends consume the allowance. It refunds only the window
// that granted the admission: once that window has rolled over, the refund is
// a no-op and cannot inflate a later window's allowance.
func (l *submissionLimiter) release(admission submissionAdmission) {
	l.mu.Lock()
	defer l.mu.Unlock()

	window, ok := l.windows[admission.visitorKey]
	if !ok || admission.started.IsZero() || !window.started.Equal(admission.started) || window.count == 0 {
		return
	}
	window.count--
	l.windows[admission.visitorKey] = window
}
