package requestprotection

import (
	"context"
	"errors"
	"testing"
)

func TestCheckpointPreservesContextError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	events := []CancellationEvent{}
	ctx = WithCancellationObserver(ctx, ProfileSearch, func(event CancellationEvent) {
		events = append(events, event)
	})
	cancel()
	if err := Checkpoint(ctx, "artists.search.records"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Checkpoint() error = %v, want context.Canceled", err)
	}
	if len(events) != 1 || events[0].Profile != ProfileSearch || events[0].Stage != "artists.search.records" {
		t.Fatalf("cancellation events = %+v", events)
	}
	if err := Checkpoint(context.Background(), "artists.search.records"); err != nil {
		t.Fatalf("active checkpoint error = %v", err)
	}
}

func TestCheckpointDoesNotObserveActiveContext(t *testing.T) {
	called := false
	ctx := WithCancellationObserver(context.Background(), ProfileDetail, func(CancellationEvent) {
		called = true
	})
	if err := Checkpoint(ctx, "artist.detail.lookup"); err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("active checkpoint emitted cancellation telemetry")
	}
}
