package itineraries

import (
	"net/url"
	"testing"

	itineraryworkflow "github.com/blackfyre/wga/internal/itineraries"
)

func TestProjectFromTokenReflectsDraft(t *testing.T) {
	app, mux := newItineraryMux(t)

	artwork, err := app.FindRecordById("artworks", testArtworkID)
	if err != nil {
		t.Fatalf("find artwork: %v", err)
	}
	artwork.Set("image", "work.jpg")
	if err := app.Save(artwork); err != nil {
		t.Fatalf("save artwork image: %v", err)
	}

	cookie, csrf := sessionForMux(t, mux)
	postForm(t, mux, "/itineraries/draft/add", cookie, csrf, url.Values{"artwork_id": {testArtworkID}})

	session := projectFromToken(app, cookie.Value)

	if want := itineraryworkflow.CSRFToken(cookie.Value); session.CSRF != want {
		t.Errorf("CSRF = %q, want %q", session.CSRF, want)
	}
	if session.Tray.Count != 1 {
		t.Errorf("tray count = %d, want 1", session.Tray.Count)
	}
	if !session.IsAdded(testArtworkID) {
		t.Error("projection must report the added artwork")
	}
	if len(session.Tray.Thumbs) != 1 {
		t.Errorf("tray thumbnails = %d, want 1", len(session.Tray.Thumbs))
	}
}
