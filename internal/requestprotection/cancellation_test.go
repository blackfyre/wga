package requestprotection

import (
	"context"
	"errors"
	"testing"
)

func TestCheckpointPreservesContextError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := Checkpoint(ctx, "artists.search.records"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Checkpoint() error = %v, want context.Canceled", err)
	}
	if err := Checkpoint(context.Background(), "artists.search.records"); err != nil {
		t.Fatalf("active checkpoint error = %v", err)
	}
}
