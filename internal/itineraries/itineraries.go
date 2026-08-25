// Package itineraries owns the anonymous visitor-itinerary capability.
//
// It persists one session-owned draft per anonymous visitor, exposes bounded
// ordering/narration mutations, publishes an immutable, expiring public token,
// and removes expired or abandoned records through an owned lifecycle job.
// Request handlers live in internal/handlers/itineraries and act only as
// transport adapters over this package's workflows.
package itineraries

import (
	"errors"
	"time"
)

// PocketBase collection identifiers owned by this capability.
const (
	CollectionItineraries    = "itineraries"
	CollectionItineraryStops = "itinerary_stops"

	// MaxStops bounds a single draft to fifteen unique published artworks.
	MaxStops = 15

	// Field length bounds match the accepted visual-overhaul reference.
	MaxTitleLength     = 80
	MaxIntroLength     = 400
	MaxCreatorLength   = 40
	MaxNarrationLength = 600

	// PublicationLifetime is the fixed one-year sharing window for a published
	// itinerary. It is also the maximum retention for an abandoned draft.
	PublicationLifetime = 365 * 24 * time.Hour

	// PublicationWindow is the rolling window used by the durable per-owner
	// publication limiter.
	PublicationWindow = time.Hour

	// PublicationBudget is the maximum number of itineraries one session may
	// publish within PublicationWindow.
	PublicationBudget = 3
)

// Status is the explicit itinerary lifecycle state (ADR 0014).
type Status string

const (
	// StatusDraft is the initial, session-owned mutable state.
	StatusDraft Status = "draft"
	// StatusPending is a legacy-only state produced by the pre-immediate
	// publication workflow. It is never written by new code; the backfill
	// migration reconciles existing pending records to approved. It remains in
	// the machine so operators can still approve or reject any stragglers.
	StatusPending Status = "pending"
	// StatusApproved is set by publication and permits the public token route
	// while unexpired. Publication is immediate: a validated draft becomes
	// readable straight away; the listed flag controls index discovery.
	StatusApproved Status = "approved"
	// StatusRejected hides the itinerary without deleting the maker's data. It
	// is an operational moderation outcome reached from approved or pending.
	StatusRejected Status = "rejected"
)

// statusValues is the authoritative set of persisted statuses.
var statusValues = []Status{StatusDraft, StatusPending, StatusApproved, StatusRejected}

// transitions is the authoritative state machine (ADR 0014). Publication is
// the only application-driven transition and moves a draft straight to
// approved so the token is immediately readable; moderation is an operational
// action performed through the PocketBase admin and may reject a live or
// legacy-pending itinerary.
var transitions = map[Status][]Status{
	StatusDraft:    {StatusApproved},
	StatusPending:  {StatusApproved, StatusRejected},
	StatusApproved: {StatusRejected},
	StatusRejected: {},
}

// ValidStatus reports whether value is an authoritative status.
func ValidStatus(value string) bool {
	for _, status := range statusValues {
		if Status(value) == status {
			return true
		}
	}

	return false
}

// CanTransition reports whether the lifecycle permits moving from status to next.
func CanTransition(status Status, next Status) bool {
	for _, allowed := range transitions[status] {
		if allowed == next {
			return true
		}
	}

	return false
}

// IsPublicStatus reports whether a published record may be served publicly.
func IsPublicStatus(value string) bool {
	return Status(value) == StatusApproved
}

// Capability-owned error values.
var (
	// ErrStopLimit is returned when a draft already holds MaxStops artworks.
	ErrStopLimit = errors.New("itinerary draft is limited to 15 stops")
	// ErrStopNotFound is returned when a stop is not in the caller's draft.
	ErrStopNotFound = errors.New("itinerary stop not found")
	// ErrArtworkUnavailable is returned when an artwork is missing or unpublished.
	ErrArtworkUnavailable = errors.New("artwork is unavailable")
	// ErrNoStops is returned when publishing an empty draft.
	ErrNoStops = errors.New("itinerary requires at least one stop")
	// ErrTitleRequired is returned when publishing without a title.
	ErrTitleRequired = errors.New("itinerary title is required")
	// ErrInvalidMove is returned for an unknown reorder direction.
	ErrInvalidMove = errors.New("invalid stop move direction")
	// ErrNotDraft is returned when a mutation targets a non-draft itinerary.
	ErrNotDraft = errors.New("itinerary is not a draft")
	// ErrPublishRateLimit is returned when an owner exceeds the bounded
	// publication budget within the rolling window.
	ErrPublishRateLimit = errors.New("itinerary publication rate limit exceeded")
	// ErrInvalidTransition is returned by the lifecycle hook when a status
	// transition is not permitted by the state machine.
	ErrInvalidTransition = errors.New("itinerary status transition is not allowed")
	// ErrImmutablePublication is returned by the lifecycle hook when immutable
	// publication content is changed on a non-draft itinerary.
	ErrImmutablePublication = errors.New("published itinerary content is immutable")
)
