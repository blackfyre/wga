package requestprotection

import (
	"context"
	"fmt"
	"sync"
)

// CapacityGate bounds concurrent protected reads without queuing callers.
type CapacityGate struct {
	slots chan struct{}
}

// CapacityLease owns one gate slot. Release is safe to call repeatedly and
// concurrently; the slot is returned exactly once.
type CapacityLease struct {
	gate *CapacityGate
	once *sync.Once
}

// NewCapacityGate creates a non-blocking gate with the configured capacity.
func NewCapacityGate(capacity int) (*CapacityGate, error) {
	if capacity <= 0 {
		return nil, fmt.Errorf("protected-read capacity must be positive")
	}

	return &CapacityGate{slots: make(chan struct{}, capacity)}, nil
}

// TryAcquire immediately acquires an available slot. It returns false rather
// than waiting when the context is cancelled or the gate is exhausted.
func (g *CapacityGate) TryAcquire(ctx context.Context) (*CapacityLease, bool) {
	if g == nil || ctx == nil || ctx.Err() != nil {
		return nil, false
	}

	select {
	case g.slots <- struct{}{}:
		lease := &CapacityLease{gate: g, once: &sync.Once{}}
		if ctx.Err() != nil {
			lease.Release()
			return nil, false
		}
		return lease, true
	default:
		return nil, false
	}
}

// InUse returns the number of slots currently held.
func (g *CapacityGate) InUse() int {
	if g == nil {
		return 0
	}
	return len(g.slots)
}

// Capacity returns the configured slot limit.
func (g *CapacityGate) Capacity() int {
	if g == nil {
		return 0
	}
	return cap(g.slots)
}

// Release returns the lease's slot exactly once.
func (l *CapacityLease) Release() {
	if l == nil || l.gate == nil || l.once == nil {
		return
	}

	l.once.Do(func() {
		<-l.gate.slots
	})
}
