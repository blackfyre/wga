package postcards

import (
	"context"
	"time"

	"github.com/blackfyre/wga/internal/requestprotection"
)

const (
	replacementSearchesPerVisitor = 10
	replacementSearchWindow       = 10 * time.Minute
	replacementSearchConcurrency  = 4
)

// replacementSearchGuard bounds the unavailable-artwork title search, which
// runs before CAPTCHA and send admission. It is separate from the send
// allowance: searches never consume it and never verify a CAPTCHA.
type replacementSearchGuard struct {
	perVisitor *submissionLimiter
	capacity   *requestprotection.CapacityGate
}

func newReplacementSearchGuard(perVisitor int, window time.Duration, concurrency int) (*replacementSearchGuard, error) {
	capacity, err := requestprotection.NewCapacityGate(concurrency)
	if err != nil {
		return nil, err
	}
	return &replacementSearchGuard{perVisitor: newSubmissionLimiter(perVisitor, window), capacity: capacity}, nil
}

// admit charges one search to the visitor and takes a concurrency slot. A
// nil guard admits nothing. The returned release frees the slot; a search
// rejected for capacity is refunded to the visitor's window.
func (g *replacementSearchGuard) admit(ctx context.Context, visitorKey string, now time.Time) (func(), bool) {
	if g == nil || g.perVisitor == nil || g.capacity == nil {
		return nil, false
	}
	admission, ok := g.perVisitor.admit(visitorKey, now)
	if !ok {
		return nil, false
	}
	lease, ok := g.capacity.TryAcquire(ctx)
	if !ok {
		g.perVisitor.release(admission)
		return nil, false
	}
	return lease.Release, true
}
