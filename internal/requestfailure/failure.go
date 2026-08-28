// Package requestfailure holds request-scoped failure data shared by HTTP
// response adapters and observability adapters.
package requestfailure

import (
	"github.com/pocketbase/pocketbase/core"
)

const key = "wga.server_failure"

// Failure identifies the safe category and optional cause of a rendered
// server-fault response for request-scoped observability.
type Failure struct {
	Category string
	Cause    error
}

// Record stores failure metadata for the lifetime of one request.
func Record(event *core.RequestEvent, failure Failure) {
	event.Set(key, failure)
}

// From returns the failure metadata recorded for this request.
func From(event *core.RequestEvent) (Failure, bool) {
	failure, ok := event.Get(key).(Failure)
	return failure, ok
}
