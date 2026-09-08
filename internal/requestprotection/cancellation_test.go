package requestprotection

import (
	"context"
	"errors"
	"testing"
)

func TestCheckpointPreservesCancellationCause(t *testing.T) {
	cause := errors.New("caller stopped request")
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(cause)
	if err := Checkpoint(ctx, "artists.search.records"); !errors.Is(err, cause) {
		t.Fatalf("Checkpoint() error = %v, want original cause", err)
	}
	if err := Checkpoint(context.Background(), "artists.search.records"); err != nil {
		t.Fatalf("active checkpoint error = %v", err)
	}
}
