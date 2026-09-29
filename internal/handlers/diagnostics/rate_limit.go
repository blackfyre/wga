package diagnostics

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/netip"
	"sync"
	"time"
)

const (
	rateLimitPerWindow = 60
	rateLimitWindow    = time.Minute
	rateLimitEntries   = 4096
)

type rateWindow struct {
	count     int
	startedAt time.Time
}

// rateLimiter is a bounded fixed-window limiter keyed by a one-way HMAC of the
// trusted client identity, so raw client addresses are never retained.
type rateLimiter struct {
	mu      sync.Mutex
	key     []byte
	windows map[string]rateWindow
	now     func() time.Time
}

func newRateLimiter(now func() time.Time) (*rateLimiter, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	return &rateLimiter{key: key, windows: make(map[string]rateWindow), now: now}, nil
}

func (l *rateLimiter) allow(identity string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	id := l.privateIdentity(identity)
	window, found := l.windows[id]
	if found && now.Sub(window.startedAt) >= rateLimitWindow {
		found = false
	}
	if !found {
		if len(l.windows) >= rateLimitEntries {
			l.evict(now)
		}
		l.windows[id] = rateWindow{count: 1, startedAt: now}
		return true
	}
	if window.count >= rateLimitPerWindow {
		return false
	}
	window.count++
	l.windows[id] = window
	return true
}

// evict removes expired windows, or the oldest window when none has expired.
func (l *rateLimiter) evict(now time.Time) {
	oldestID := ""
	var oldest time.Time
	for id, window := range l.windows {
		if now.Sub(window.startedAt) >= rateLimitWindow {
			delete(l.windows, id)
			continue
		}
		if oldestID == "" || window.startedAt.Before(oldest) {
			oldestID, oldest = id, window.startedAt
		}
	}
	if len(l.windows) >= rateLimitEntries {
		delete(l.windows, oldestID)
	}
}

func (l *rateLimiter) privateIdentity(identity string) string {
	mac := hmac.New(sha256.New, l.key)
	mac.Write([]byte(clientBucket(identity)))
	return hex.EncodeToString(mac.Sum(nil))
}

// clientBucket groups IPv6 identities by their /64 prefix, the smallest block
// commonly delegated to one subscriber, so rotating addresses within it does
// not multiply the per-client budget. Other identities are used as given.
func clientBucket(identity string) string {
	address, err := netip.ParseAddr(identity)
	if err != nil {
		return identity
	}
	address = address.Unmap()
	if address.Is6() {
		prefix, err := address.Prefix(64)
		if err == nil {
			return prefix.String()
		}
	}
	return address.String()
}
