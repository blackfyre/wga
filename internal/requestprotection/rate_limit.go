package requestprotection

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"sync"
	"time"
)

const clientRateWindow = time.Minute

// RateLimits contains the independent per-minute limits and the maximum number
// of private client identities retained by one limiter.
type RateLimits struct {
	Search        int
	Fragment      int
	Detail        int
	EntryCapacity int
}

type privateIdentity [sha256.Size]byte

type rateWindow struct {
	startedAt time.Time
	count     int
}

type clientRateState struct {
	createdOrder uint64
	windows      map[Profile]rateWindow
}

// RateLimiter provides bounded, privacy-preserving fixed-window accounting for
// protected public-read profiles. It retains only process-private keyed
// identity digests and never stores the resolved raw identity.
type RateLimiter struct {
	mu        sync.Mutex
	limits    RateLimits
	key       [sha256.Size]byte
	clients   map[privateIdentity]*clientRateState
	nextOrder uint64
	now       func() time.Time
}

// NewRateLimiter creates an independent limiter with a random process-private
// digest key.
func NewRateLimiter(limits RateLimits) (*RateLimiter, error) {
	if err := validateRateLimits(limits); err != nil {
		return nil, err
	}

	var key [sha256.Size]byte
	if _, err := rand.Read(key[:]); err != nil {
		return nil, fmt.Errorf("create private identity key: %w", err)
	}

	return newRateLimiter(limits, key, time.Now), nil
}

func newRateLimiter(limits RateLimits, key [sha256.Size]byte, now func() time.Time) *RateLimiter {
	return &RateLimiter{
		limits:  limits,
		key:     key,
		clients: make(map[privateIdentity]*clientRateState, limits.EntryCapacity),
		now:     now,
	}
}

// AllowResolved records one request for a protected profile and reports whether
// its per-client budget remains available. Failed trusted resolution fails
// closed without creating limiter state, even if a caller supplies a value.
func (l *RateLimiter) AllowResolved(identity string, resolved bool, profile Profile) bool {
	if !profile.Protected() {
		return true
	}
	if l == nil || !resolved || identity == "" {
		return false
	}

	limit := l.limit(profile)
	if limit <= 0 {
		return false
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.removeExpired(now)

	key := l.privateIdentity(identity)
	state := l.clients[key]
	if state == nil {
		if len(l.clients) >= l.limits.EntryCapacity {
			l.removeOldest()
		}
		l.nextOrder++
		state = &clientRateState{
			createdOrder: l.nextOrder,
			windows:      make(map[Profile]rateWindow, 3),
		}
		l.clients[key] = state
	}

	window := state.windows[profile]
	if window.startedAt.IsZero() || now.Sub(window.startedAt) >= clientRateWindow {
		window = rateWindow{startedAt: now}
	}
	if window.count >= limit {
		return false
	}

	window.count++
	state.windows[profile] = window
	return true
}

// String prevents accidental formatting of private limiter state.
func (*RateLimiter) String() string {
	return "[redacted]"
}

// GoString prevents accidental Go-syntax formatting of private limiter state.
func (*RateLimiter) GoString() string {
	return "requestprotection.RateLimiter([redacted])"
}

func validateRateLimits(limits RateLimits) error {
	if limits.Search <= 0 || limits.Fragment <= 0 || limits.Detail <= 0 || limits.EntryCapacity <= 0 {
		return fmt.Errorf("rate limits and entry capacity must be positive")
	}
	return nil
}

func (l *RateLimiter) limit(profile Profile) int {
	switch profile {
	case ProfileSearch:
		return l.limits.Search
	case ProfileFragment:
		return l.limits.Fragment
	case ProfileDetail:
		return l.limits.Detail
	default:
		return 0
	}
}

func (l *RateLimiter) privateIdentity(identity string) privateIdentity {
	digest := hmac.New(sha256.New, l.key[:])
	_, _ = digest.Write([]byte(identity))
	var key privateIdentity
	copy(key[:], digest.Sum(nil))
	return key
}

func (l *RateLimiter) removeExpired(now time.Time) {
	for key, state := range l.clients {
		for profile, window := range state.windows {
			if now.Sub(window.startedAt) >= clientRateWindow {
				delete(state.windows, profile)
			}
		}
		if len(state.windows) == 0 {
			delete(l.clients, key)
		}
	}
}

func (l *RateLimiter) removeOldest() {
	var oldestKey privateIdentity
	var oldestOrder uint64
	found := false
	for key, state := range l.clients {
		if !found || state.createdOrder < oldestOrder {
			oldestKey = key
			oldestOrder = state.createdOrder
			found = true
		}
	}
	if found {
		delete(l.clients, oldestKey)
	}
}
