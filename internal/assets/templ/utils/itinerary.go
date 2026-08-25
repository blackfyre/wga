package utils

import (
	"context"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
)

// itineraryProjectionContextKey is a private typed key for the read-only
// itinerary session projection prepared by the central middleware. It is
// intentionally distinct from the generic ContextKey so the session state
// cannot collide with ordinary string context values.
type itineraryProjectionContextKey struct{}

// itineraryProjection carries the session state consumed by the shared layout
// tray and the supported add controls.
type itineraryProjection struct {
	csrf  string
	tray  dto.ItineraryTrayView
	added map[string]bool
}

// WithItineraryProjection returns a context carrying the read-only itinerary
// session projection. csrf is the synchroniser token for add forms; tray is the
// persistent tray view; added is the set of artwork IDs already in the draft.
func WithItineraryProjection(c context.Context, csrf string, tray dto.ItineraryTrayView, added map[string]bool) context.Context {
	return context.WithValue(c, itineraryProjectionContextKey{}, itineraryProjection{
		csrf:  csrf,
		tray:  tray,
		added: added,
	})
}

// itineraryProjectionFrom returns the stored projection, or a zero value.
func itineraryProjectionFrom(c context.Context) itineraryProjection {
	projection, _ := c.Value(itineraryProjectionContextKey{}).(itineraryProjection)
	return projection
}

// GetItineraryCSRF returns the synchroniser token for itinerary add forms, or
// an empty string when no session was projected. Add controls must not render
// without a token.
func GetItineraryCSRF(c context.Context) string {
	return itineraryProjectionFrom(c).csrf
}

// GetItineraryTray returns the persistent tray view, or a zero value when no
// session was projected.
func GetItineraryTray(c context.Context) dto.ItineraryTrayView {
	return itineraryProjectionFrom(c).tray
}

// GetItineraryAdded reports whether artworkID is already in the session draft.
func GetItineraryAdded(c context.Context, artworkID string) bool {
	projection := itineraryProjectionFrom(c)
	if projection.added == nil {
		return false
	}

	return projection.added[artworkID]
}
