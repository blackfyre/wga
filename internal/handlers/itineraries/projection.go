package itineraries

import (
	"net/http"
	"strings"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
	"github.com/blackfyre/wga/internal/constants"
	itineraryworkflow "github.com/blackfyre/wga/internal/itineraries"
	"github.com/blackfyre/wga/internal/utils/url"
	"github.com/pocketbase/pocketbase/core"
)

// PublicSession is the read-only projection of a visitor's itinerary session
// prepared once by the central middleware for a public document request. It
// carries the synchroniser token for add controls and the persistent tray view
// for the shared layout. It never creates records.
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
// the added-state projection stored in the request context by the middleware.
func (s PublicSession) AddedIDs() map[string]bool {
	return s.added
}

// ProjectSession resolves or issues the anonymous session cookie for a public
// document request and returns the read-only projection for the same request.
// A missing, duplicated, or malformed cookie is rotated to a fresh token under
// the supplied validated cookie policy. It never creates durable state: an
// absent draft yields an empty tray and added set while still carrying the
// synchroniser token so a first-time visitor can add from any supported
// surface.
func ProjectSession(app core.App, c *core.RequestEvent, cookie CookiePolicy) (PublicSession, error) {
	token := itineraryworkflow.TokenFromRequest(c.Request, cookie.Name)
	if token == "" {
		var err error
		token, err = itineraryworkflow.NewToken()
		if err != nil {
			return PublicSession{}, err
		}
		sessionCookie := itineraryworkflow.SessionCookie(token, cookie.Name, cookie.Secure)
		c.SetCookie(sessionCookie)
		// Replace every same-name cookie in the in-memory request with the
		// single fresh token so downstream handlers resolve it and do not mint
		// a second, conflicting cookie. Appending would leave rejected
		// duplicates/malformed values beside the fresh token and cause a second
		// rotation.
		replaceRequestCookie(c.Request, sessionCookie)
	}

	return projectFromToken(app, token), nil
}

// replaceRequestCookie removes every request cookie sharing cookie.Name and
// appends cookie, preserving unrelated cookies. It is the in-memory equivalent
// of the Set-Cookie replacement so downstream handlers see exactly one token.
func replaceRequestCookie(r *http.Request, cookie *http.Cookie) {
	if r == nil || cookie == nil {
		return
	}

	kept := make([]*http.Cookie, 0, len(r.Cookies())+1)
	for _, existing := range r.Cookies() {
		if existing.Name != cookie.Name {
			kept = append(kept, existing)
		}
	}
	kept = append(kept, &http.Cookie{Name: cookie.Name, Value: cookie.Value})

	parts := make([]string, 0, len(kept))
	for _, item := range kept {
		parts = append(parts, item.Name+"="+item.Value)
	}
	if len(parts) == 0 {
		r.Header.Del("Cookie")
		return
	}
	r.Header.Set("Cookie", strings.Join(parts, "; "))
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
