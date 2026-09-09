package requestprotection

import (
	"context"
	"errors"
)

type cancellationObserverKey struct{}

// CancellationError identifies cancellation observed at a protected workflow
// checkpoint while preserving the original context cause for errors.Is.
type CancellationError struct {
	cause error
}

func (e *CancellationError) Error() string { return e.cause.Error() }
func (e *CancellationError) Unwrap() error { return e.cause }

// IsCancellation reports whether err originated from a protected checkpoint.
func IsCancellation(err error) bool {
	var cancellation *CancellationError
	return errors.As(err, &cancellation)
}

// CancellationEvent contains only bounded, server-controlled request metadata.
type CancellationEvent struct {
	Profile Profile
	Stage   string
}

// WithCancellationObserver attaches request-scoped telemetry to the shared
// checkpoint boundary. Neither client identity nor request URL data enters the
// observer contract.
func WithCancellationObserver(ctx context.Context, profile Profile, observer func(CancellationEvent)) context.Context {
	if ctx == nil || observer == nil {
		return ctx
	}
	return context.WithValue(ctx, cancellationObserverKey{}, func(stage string) {
		observer(CancellationEvent{Profile: profile, Stage: stage})
	})
}

// Checkpoint stops a multi-stage protected workflow before its next material
// stage. The stage is part of the shared boundary so task-specific telemetry can
// observe it without changing workflow call sites.
func Checkpoint(ctx context.Context, stage string) error {
	_ = stage
	if ctx == nil || ctx.Err() == nil {
		return nil
	}
	if observer, ok := ctx.Value(cancellationObserverKey{}).(func(string)); ok {
		observer(stage)
	}
	return &CancellationError{cause: ctx.Err()}
}
