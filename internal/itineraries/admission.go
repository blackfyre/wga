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
	// admissionDraftBudget is the default maximum number of new drafts one
	// trusted identity may create within admissionWindow.
	admissionDraftBudget = 3
	// admissionPublishBudget is the default maximum number of successful
	// publications one trusted identity may perform within admissionWindow.
	admissionPublishBudget = 3
	// admissionMaxKeys bounds the in-memory key space. When full, expired
	// entries are evicted first, then arbitrary entries, so memory use stays
	// bounded under synthetic-identity churn.
	admissionMaxKeys = 8192
)

// admissionWindowState records the in-window admission times for one identity
// hash, oldest first. Each slice holds at most its kind's budget, so a budget is
// enforced over every trailing admissionWindow rather than a fixed bucket.
type admissionWindowState struct {
	drafts    []time.Time
	publishes []time.Time
}

// slot returns the admission times recorded for kind, or nil for an unknown
// kind.
func (w *admissionWindowState) slot(kind AdmissionKind) *[]time.Time {
	switch kind {
	case AdmissionDraft:
		return &w.drafts
	case AdmissionPublish:
		return &w.publishes
	default:
		return nil
	}
}

// prune drops admissions that have left the trailing window at now.
func (w *admissionWindowState) prune(now time.Time) {
	for _, times := range []*[]time.Time{&w.drafts, &w.publishes} {
		keep := 0
		for keep < len(*times) && now.Sub((*times)[keep]) >= admissionWindow {
			keep++
		}
		*times = (*times)[keep:]
	}
}

// empty reports whether no admission remains in the window.
func (w *admissionWindowState) empty() bool {
	return len(w.drafts) == 0 && len(w.publishes) == 0
}

// AdmissionBudgets holds the per-identity budgets for the rolling admission
// window. A non-positive budget selects the production default of 3.
type AdmissionBudgets struct {
	// Drafts is the maximum number of new drafts per identity per window.
	Drafts int
	// Publishes is the maximum number of successful publications per identity
	// per window.
	Publishes int
}

// normalized replaces non-positive budgets with the production defaults.
func (b AdmissionBudgets) normalized() AdmissionBudgets {
	if b.Drafts <= 0 {
		b.Drafts = admissionDraftBudget
	}
	if b.Publishes <= 0 {
		b.Publishes = admissionPublishBudget
	}
	return b
}

// AdmissionLimiter is a bounded, privacy-preserving in-memory rate limiter. It
// stores only the SHA-256 hash of the trusted client identity, never the raw
// identity. A single mutex serialises admission and release so the budget is
// enforced atomically under concurrency.
type AdmissionLimiter struct {
	mu      sync.Mutex
	keys    map[string]*admissionWindowState
	now     func() time.Time
	budgets AdmissionBudgets
}

// NewAdmissionLimiter returns a limiter with the rolling one-hour window and the
// given per-kind budgets. Non-positive budgets fall back to the production
// defaults described by the admission* constants.
func NewAdmissionLimiter(budgets AdmissionBudgets) *AdmissionLimiter {
	return &AdmissionLimiter{
		keys:    make(map[string]*admissionWindowState),
		now:     time.Now,
		budgets: budgets.normalized(),
	}
}

// Admit records an admission for identity and reports whether the one-hour
// budget for kind still has capacity. It returns false when the budget is
// exhausted. Identity is hashed before it touches the limiter.
func (l *AdmissionLimiter) Admit(identity string, kind AdmissionKind) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.sweepLocked()

	budget := l.budgetFor(kind)
	if budget <= 0 {
		return false
	}

	key := hashIdentity(identity)
	window := l.keys[key]
	if window == nil {
		window = &admissionWindowState{}
		l.keys[key] = window
		l.evictLocked()
	}

	times := window.slot(kind)
	if len(*times) >= budget {
		return false
	}

	*times = append(*times, l.now())
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

	times := window.slot(kind)
	if times == nil || len(*times) == 0 {
		return
	}
	// The reservation being released is the most recent admission.
	*times = (*times)[:len(*times)-1]
}

// Reset clears all recorded admissions. It exists for tests and for explicit
// lifecycle control by the serial integration owner.
func (l *AdmissionLimiter) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.keys = make(map[string]*admissionWindowState)
}

func (l *AdmissionLimiter) budgetFor(kind AdmissionKind) int {
	switch kind {
	case AdmissionDraft:
		return l.budgets.Drafts
	case AdmissionPublish:
		return l.budgets.Publishes
	default:
		return 0
	}
}

// sweepLocked drops admissions older than the trailing one-hour window and
// removes identities with none left, so expired identities stop consuming key
// storage.
func (l *AdmissionLimiter) sweepLocked() {
	now := l.now()
	for key, window := range l.keys {
		window.prune(now)
		if window.empty() {
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
