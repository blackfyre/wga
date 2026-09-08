package requestprotection

import (
	"context"
	"sync"
	"testing"
)

func TestCapacityGateAcquiresAvailableSlot(t *testing.T) {
	gate := mustCapacityGate(t, 2)

	lease, ok := gate.TryAcquire(context.Background())
	if !ok || lease == nil {
		t.Fatal("available capacity was not acquired")
	}
	if gate.InUse() != 1 || gate.Capacity() != 2 {
		t.Fatalf("gate utilisation = %d/%d; want 1/2", gate.InUse(), gate.Capacity())
	}

	lease.Release()
	if gate.InUse() != 0 {
		t.Fatalf("gate utilisation after release = %d; want 0", gate.InUse())
	}
}

func TestCapacityGateRejectsExhaustedWithoutQueue(t *testing.T) {
	const capacity = 3
	const callers = 24
	gate := mustCapacityGate(t, capacity)
	start := make(chan struct{})
	release := make(chan struct{})
	results := make(chan bool, callers)

	var workers sync.WaitGroup
	workers.Add(callers)
	for range callers {
		go func() {
			defer workers.Done()
			<-start
			lease, ok := gate.TryAcquire(context.Background())
			results <- ok
			if ok {
				<-release
				lease.Release()
			}
		}()
	}

	close(start)
	acquired := 0
	for range callers {
		if <-results {
			acquired++
		}
	}
	if acquired != capacity {
		t.Fatalf("concurrent acquisitions = %d; want %d", acquired, capacity)
	}
	if gate.InUse() != capacity {
		t.Fatalf("gate utilisation = %d; want %d", gate.InUse(), capacity)
	}

	close(release)
	workers.Wait()
	if gate.InUse() != 0 {
		t.Fatalf("gate utilisation after worker release = %d; want 0", gate.InUse())
	}
}

func TestCapacityGateRejectsCancelledContext(t *testing.T) {
	gate := mustCapacityGate(t, 1)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if lease, ok := gate.TryAcquire(ctx); ok || lease != nil {
		t.Fatal("cancelled context acquired capacity")
	}
	if gate.InUse() != 0 {
		t.Fatalf("cancelled acquisition retained %d slots", gate.InUse())
	}
}

func TestCapacityLeaseReleasesExactlyOnceConcurrently(t *testing.T) {
	gate := mustCapacityGate(t, 1)
	lease, ok := gate.TryAcquire(context.Background())
	if !ok {
		t.Fatal("failed to acquire initial slot")
	}
	leaseCopy := *lease

	const releasers = 32
	var workers sync.WaitGroup
	workers.Add(releasers)
	for index := range releasers {
		go func() {
			defer workers.Done()
			if index%2 == 0 {
				lease.Release()
				return
			}
			leaseCopy.Release()
		}()
	}
	workers.Wait()

	if gate.InUse() != 0 {
		t.Fatalf("gate utilisation after repeated release = %d; want 0", gate.InUse())
	}
	next, ok := gate.TryAcquire(context.Background())
	if !ok {
		t.Fatal("slot was not available after idempotent release")
	}
	next.Release()
}

func TestCapacityGateRejectsInvalidInputs(t *testing.T) {
	if _, err := NewCapacityGate(0); err == nil {
		t.Fatal("zero capacity succeeded; want error")
	}
	if lease, ok := (*CapacityGate)(nil).TryAcquire(context.Background()); ok || lease != nil {
		t.Fatal("nil gate acquired capacity")
	}
	gate := mustCapacityGate(t, 1)
	if lease, ok := gate.TryAcquire(nil); ok || lease != nil {
		t.Fatal("nil context acquired capacity")
	}
	(*CapacityLease)(nil).Release()
}

func mustCapacityGate(t *testing.T, capacity int) *CapacityGate {
	t.Helper()
	gate, err := NewCapacityGate(capacity)
	if err != nil {
		t.Fatal(err)
	}
	return gate
}
