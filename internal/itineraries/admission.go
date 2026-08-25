package itineraries

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

// AdmissionKind discriminates the two server-side admission budgets keyed by
// trusted client identity.
type AdmissionKind uint8

const (
	// AdmissionDraft bounds new-draft creation per trusted identity.
	AdmissionDraft AdmissionKind = iota + 1
	// AdmissionPublish bounds successful publication per trusted identity.
	AdmissionPublish
)

const (
	// admissionWindow is the rolling window shared by both budgets.
	admissionWindow = time.Hour
	// admissionDraftBudget is the maximum number of new drafts one trusted
	// identity may create within admissionWindow.
	admissionDraftBudget = 3
	// admissionPublishBudget is the maximum number of successful publications
	// one trusted identity may perform within admissionWindow.
	admissionPublishBudget = 3
	// admissionMaxKeys bounds the in-memory key space. When full, expired
	// entries are evicted first, then arbitrary entries, so memory use stays
	// bounded under synthetic-identity churn.
	admissionMaxKeys = 8192
)

// admissionWindowState records the in-window counters for one identity hash.
type admissionWindowState struct {
	start     time.Time
	drafts    int
	publishes int
}

// AdmissionLimiter is a bounded, privacy-preserving in-memory rate limiter. It
// stores only the SHA-256 hash of the trusted client identity, never the raw
// identity. A single mutex serialises admission and release so the budget is
// enforced atomically under concurrency.
type AdmissionLimiter struct {
	mu   sync.Mutex
	keys map[string]*admissionWindowState
	now  func() time.Time
}

// NewAdmissionLimiter returns a limiter with the fixed one-hour window and
// per-kind budgets described by the admission* constants.
func NewAdmissionLimiter() *AdmissionLimiter {
	return &AdmissionLimiter{
		keys: make(map[string]*admissionWindowState),
		now:  time.Now,
	}
}

// Admit records an admission for identity and reports whether the one-hour
// budget for kind still has capacity. It returns false when the budget is
// exhausted. Identity is hashed before it touches the limiter.
func (l *AdmissionLimiter) Admit(identity string, kind AdmissionKind) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.sweepLocked()

	key := hashIdentity(identity)
	window := l.keys[key]
	if window == nil {
		window = &admissionWindowState{start: l.now()}
		l.keys[key] = window
		l.evictLocked()
	}

	if !l.capacityLocked(window, kind) {
		return false
	}

	l.chargeLocked(window, kind)
	return true
}

// Release undoes a previously successful Admit. It is used for reservations
// that must not be consumed when the guarded operation subsequently fails, so
// validation and persistence failures do not permanently spend a budget slot.
func (l *AdmissionLimiter) Release(identity string, kind AdmissionKind) {
	l.mu.Lock()
	defer l.mu.Unlock()

	window := l.keys[hashIdentity(identity)]
	if window == nil {
		return
	}

	switch kind {
	case AdmissionDraft:
		if window.drafts > 0 {
			window.drafts--
		}
	case AdmissionPublish:
		if window.publishes > 0 {
			window.publishes--
		}
	}
}

// Reset clears all recorded admissions. It exists for tests and for explicit
// lifecycle control by the serial integration owner.
func (l *AdmissionLimiter) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.keys = make(map[string]*admissionWindowState)
}

func (l *AdmissionLimiter) capacityLocked(window *admissionWindowState, kind AdmissionKind) bool {
	switch kind {
	case AdmissionDraft:
		return window.drafts < admissionDraftBudget
	case AdmissionPublish:
		return window.publishes < admissionPublishBudget
	default:
		return false
	}
}

func (l *AdmissionLimiter) chargeLocked(window *admissionWindowState, kind AdmissionKind) {
	switch kind {
	case AdmissionDraft:
		window.drafts++
	case AdmissionPublish:
		window.publishes++
	}
}

// sweepLocked removes every window whose start lies outside the rolling
// one-hour window, so expired identities stop consuming key storage.
func (l *AdmissionLimiter) sweepLocked() {
	now := l.now()
	for key, window := range l.keys {
		if now.Sub(window.start) >= admissionWindow {
			delete(l.keys, key)
		}
	}
}

// evictLocked keeps the key space bounded once a new identity is added.
// sweepLocked already removed expired entries; any remaining overflow is
// evicted arbitrarily.
func (l *AdmissionLimiter) evictLocked() {
	for len(l.keys) > admissionMaxKeys {
		for key := range l.keys {
			delete(l.keys, key)
			break
		}
	}
}

// hashIdentity derives the storage key from the trusted client identity so the
// raw identity never enters the limiter.
func hashIdentity(identity string) string {
	sum := sha256.Sum256([]byte(identity))
	return hex.EncodeToString(sum[:])
}
