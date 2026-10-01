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
	drafts    []admissionEntry
	publishes []admissionEntry
}

// admissionEntry is one admission inside the window. The sequence number lets
// a reservation release exactly its own entry even when operations for the
// same identity overlap and finish out of order.
type admissionEntry struct {
	at  time.Time
	seq uint64
}

// slot returns the admissions recorded for kind, or nil for an unknown kind.
func (w *admissionWindowState) slot(kind AdmissionKind) *[]admissionEntry {
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
	for _, times := range []*[]admissionEntry{&w.drafts, &w.publishes} {
		keep := 0
		for keep < len(*times) && now.Sub((*times)[keep].at) >= admissionWindow {
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
	seq     uint64
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

// AdmissionReservation identifies one successful admission so that a guarded
// operation that fails afterwards can release exactly that admission. The zero
// value reserves nothing and its Release is a no-op.
type AdmissionReservation struct {
	limiter *AdmissionLimiter
	key     string
	kind    AdmissionKind
	seq     uint64
}

// Release undoes the reserved admission, so validation and persistence
// failures do not spend a budget slot. It removes only this reservation's
// entry, is idempotent, and does nothing once the entry has left the window.
func (r AdmissionReservation) Release() {
	if r.limiter == nil {
		return
	}
	r.limiter.release(r)
}

// Admit records an admission for identity and reports whether the one-hour
// budget for kind still has capacity. The admission is kept; use Reserve when
// a later failure must be able to release it.
func (l *AdmissionLimiter) Admit(identity string, kind AdmissionKind) bool {
	_, ok := l.Reserve(identity, kind)
	return ok
}

// Reserve records an admission for identity and returns its reservation, or
// false when the one-hour budget for kind is exhausted. Identity is hashed
// before it touches the limiter.
func (l *AdmissionLimiter) Reserve(identity string, kind AdmissionKind) (AdmissionReservation, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.sweepLocked()

	budget := l.budgetFor(kind)
	if budget <= 0 {
		return AdmissionReservation{}, false
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
		return AdmissionReservation{}, false
	}

	l.seq++
	*times = append(*times, admissionEntry{at: l.now(), seq: l.seq})
	return AdmissionReservation{limiter: l, key: key, kind: kind, seq: l.seq}, true
}

func (l *AdmissionLimiter) release(r AdmissionReservation) {
	l.mu.Lock()
	defer l.mu.Unlock()

	window := l.keys[r.key]
	if window == nil {
		return
	}

	times := window.slot(r.kind)
	if times == nil {
		return
	}
	for index, entry := range *times {
		if entry.seq == r.seq {
			*times = append((*times)[:index], (*times)[index+1:]...)
			break
		}
	}
	if window.empty() {
		delete(l.keys, r.key)
	}
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
