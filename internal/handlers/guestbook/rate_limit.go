package guestbook

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

const (
	guestbookSubmissionLimit  = 3
	guestbookRateLimitEntries = 1024
	guestbookRateLimitWindow  = time.Hour
)

type rateLimitWindow struct {
	count     int
	startedAt time.Time
}

type submissionRateLimiter struct {
	mu      sync.Mutex
	windows map[string]rateLimitWindow
	now     func() time.Time
}

func newSubmissionRateLimiter(now func() time.Time) *submissionRateLimiter {
	return &submissionRateLimiter{
		windows: make(map[string]rateLimitWindow),
		now:     now,
	}
}

func (l *submissionRateLimiter) allow(remoteAddress string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	for key, window := range l.windows {
		if now.Sub(window.startedAt) >= guestbookRateLimitWindow {
			delete(l.windows, key)
		}
	}

	key := privateRateLimitKey(remoteAddress)
	window, found := l.windows[key]
	if !found {
		if len(l.windows) >= guestbookRateLimitEntries {
			l.removeOldest()
		}
		l.windows[key] = rateLimitWindow{count: 1, startedAt: now}
		return true
	}

	if window.count >= guestbookSubmissionLimit {
		return false
	}

	window.count++
	l.windows[key] = window
	return true
}

func (l *submissionRateLimiter) removeOldest() {
	var oldestKey string
	var oldestTime time.Time
	for key, window := range l.windows {
		if oldestKey == "" || window.startedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = window.startedAt
		}
	}
	delete(l.windows, oldestKey)
}

func privateRateLimitKey(remoteAddress string) string {
	sum := sha256.Sum256([]byte(remoteAddress))
	return hex.EncodeToString(sum[:])
}
