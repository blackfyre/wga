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

func (l *submissionLimiter) allow(visitorKey string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	for key, window := range l.windows {
		if !now.Before(window.started.Add(l.duration)) {
			delete(l.windows, key)
		}
	}
	window := l.windows[visitorKey]
	if window.started.IsZero() && len(l.windows) >= l.maxKeys {
		return false
	}
	if window.started.IsZero() {
		window.started = now
	}
	if window.count >= l.maximum {
		return false
	}
	window.count++
	l.windows[visitorKey] = window
	return true
}
