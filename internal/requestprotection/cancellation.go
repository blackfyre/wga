package requestprotection

import "context"

// Checkpoint stops a multi-stage protected workflow before its next material
// stage. The stage is part of the shared boundary so task-specific telemetry can
// observe it without changing workflow call sites.
func Checkpoint(ctx context.Context, stage string) error {
	_ = stage
	if ctx == nil || ctx.Err() == nil {
		return nil
	}
	if cause := context.Cause(ctx); cause != nil {
		return cause
	}
	return ctx.Err()
}
