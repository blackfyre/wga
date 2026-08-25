package itineraries

import "github.com/pocketbase/pocketbase/core"

// immutablePublicationFields are the itinerary fields that must not change once
// the record has left the draft state. Publication freezes owner, token, and
// all content/visibility fields; only the pending→approved/rejected moderation
// transition remains permitted.
var immutablePublicationFields = []string{
	"owner",
	"token",
	"title",
	"intro",
	"creator",
	"listed",
	"published",
	"expires_at",
}

// RegisterHooks registers the feature-owned production lifecycle enforcement.
// It binds a single OnRecordUpdate handler for the itineraries collection and
// holds no package-level mutable state; it is invoked once from the handler
// package's RegisterHandlers.
func RegisterHooks(app core.App) {
	app.OnRecordUpdate(CollectionItineraries).BindFunc(enforceItineraryLifecycle)
}

// enforceItineraryLifecycle centralises ADR 0014 lifecycle rules (ADR 0014):
// status may change only through the state machine (draft→pending→approved or
// rejected), and any non-draft record's publication content is immutable.
func enforceItineraryLifecycle(e *core.RecordEvent) error {
	original := e.Record.Original()
	originalStatus := Status(original.GetString("status"))
	newStatus := Status(e.Record.GetString("status"))

	if originalStatus != newStatus && !CanTransition(originalStatus, newStatus) {
		return ErrInvalidTransition
	}

	if originalStatus != StatusDraft {
		for _, field := range immutablePublicationFields {
			if original.GetString(field) != e.Record.GetString(field) {
				return ErrImmutablePublication
			}
		}
	}

	return e.Next()
}
