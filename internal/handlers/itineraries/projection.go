package itineraries

import (
	"github.com/blackfyre/wga/internal/assets/templ/dto"
	"github.com/blackfyre/wga/internal/constants"
	itineraryworkflow "github.com/blackfyre/wga/internal/itineraries"
	"github.com/blackfyre/wga/internal/utils/url"
	"github.com/pocketbase/pocketbase/core"
)

// PublicSession is the read-only projection of a visitor's itinerary session
// served by the session fragment and used for mutation tray responses. It
// carries the synchroniser token for add controls and the persistent tray view.
// It never creates records.
type PublicSession struct {
	CSRF  string
	Tray  dto.ItineraryTrayView
	added map[string]bool
}

// IsAdded reports whether the artwork is already in the session draft.
func (s PublicSession) IsAdded(artworkID string) bool {
	if s.added == nil {
		return false
	}

	return s.added[artworkID]
}

// AddedIDs returns the set of artwork IDs already in the session draft. It is
// the added-state projection stored in the rendering context.
func (s PublicSession) AddedIDs() map[string]bool {
	return s.added
}

// projectFromToken derives the read-only projection for an already-validated
// bearer token. Any draft read failure yields an empty tray and added set
// rather than an error, so the projection stays safe and non-blocking.
func projectFromToken(app core.App, token string) PublicSession {
	session := PublicSession{
		CSRF: itineraryworkflow.CSRFToken(token),
		Tray: dto.ItineraryTrayView{BuilderURL: "/itineraries/new"},
	}

	owner := itineraryworkflow.OwnerDigest(token)

	draft, err := itineraryworkflow.FindDraft(app, owner)
	if err != nil {
		// No draft yet: carry the CSRF but no tray or added state.
		return session
	}

	stops, err := itineraryworkflow.LoadStops(app, draft.Id)
	if err != nil {
		return session
	}

	session.added = make(map[string]bool, len(stops))
	session.Tray.Count = len(stops)

	for index, stop := range stops {
		artworkID := stop.GetString("artwork")
		session.added[artworkID] = true

		if index >= 3 {
			continue
		}

		artwork, err := app.FindRecordById(constants.CollectionArtworks, artworkID)
		if err != nil || artwork.GetString("image") == "" {
			continue
		}
		session.Tray.Thumbs = append(session.Tray.Thumbs, url.GenerateArtworkImageURL(artwork, url.DeliveryProfileItineraryTray, ""))
	}

	return session
}
