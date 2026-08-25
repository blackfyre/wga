package itineraries

import (
	"strings"
	"testing"

	itineraryworkflow "github.com/blackfyre/wga/internal/itineraries"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

// createArtist adds an artist with the supplied id and name to the test schema.
func createArtist(t *testing.T, app *pocketbase.PocketBase, id string, name string) {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId("artists")
	if err != nil {
		t.Fatalf("find artists: %v", err)
	}
	record := core.NewRecord(collection)
	record.Id = id
	record.Set("name", name)
	record.Set("slug", strings.ToLower(strings.ReplaceAll(name, " ", "-")))
	if err := app.Save(record); err != nil {
		t.Fatalf("create artist %s: %v", id, err)
	}
}

// createArtworkWithImage adds a published artwork carrying an image and a
// declared source width so delivery-profile selection is exercised.
func createArtworkWithImage(t *testing.T, app *pocketbase.PocketBase, id string, title string, author string, width int) {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId("artworks")
	if err != nil {
		t.Fatalf("find artworks: %v", err)
	}
	record := core.NewRecord(collection)
	record.Id = id
	record.Set("title", title)
	record.Set("published", true)
	record.Set("image", "img.jpg")
	record.Set("image_width", width)
	record.Set("author", []string{author})
	if err := app.Save(record); err != nil {
		t.Fatalf("create artwork %s: %v", id, err)
	}
}

// TestPickerSearchesArtistName proves the builder picker query matches artist
// names as well as titles, the gap that reopened task 9.3.
func TestPickerSearchesArtistName(t *testing.T) {
	app, _ := newItineraryMux(t)
	createArtist(t, app, "ar0000000000002", "Rembrandt van Rijn")
	createArtworkWithImage(t, app, "aw0000000000002", "The Night Watch", "ar0000000000002", 800)

	// A title-only query would not match this work; the artist surname must.
	results, err := pickerWorks(app, "", "rembrandt")
	if err != nil {
		t.Fatalf("pickerWorks: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("artist search results = %d, want 1", len(results))
	}
	if results[0].Title != "The Night Watch" {
		t.Errorf("artist search returned %q, want %q", results[0].Title, "The Night Watch")
	}
	if results[0].Artist != "Rembrandt van Rijn" {
		t.Errorf("artist search byline = %q, want %q", results[0].Artist, "Rembrandt van Rijn")
	}

	// A title query still matches the same record through the other branch.
	titleResults, err := pickerWorks(app, "", "night watch")
	if err != nil {
		t.Fatalf("pickerWorks: %v", err)
	}
	if len(titleResults) != 1 || titleResults[0].Title != "The Night Watch" {
		t.Errorf("title search results = %v, want the Night Watch", titleResults)
	}
}

// TestPickerAndEditorUseDistinctNoUpscaleProfiles proves the picker rows use
// the 200px profile and the builder editor/filmstrip uses the 500px profile.
func TestPickerAndEditorUseDistinctNoUpscaleProfiles(t *testing.T) {
	app, mux := newItineraryMux(t)
	createArtworkWithImage(t, app, "aw0000000000002", "Wide Work", "ar0000000000001", 800)

	picker, err := pickerWorks(app, "", "")
	if err != nil {
		t.Fatalf("pickerWorks: %v", err)
	}
	if len(picker) == 0 {
		t.Fatal("picker returned no works")
	}
	if !strings.Contains(picker[0].ImageURL, "thumb=200x0") {
		t.Errorf("picker image URL = %q, want the 200x0 no-upscale profile", picker[0].ImageURL)
	}

	// Add the work to the draft and project the builder to inspect the
	// editor/filmstrip plate profile.
	cookie, _ := sessionForMux(t, mux)
	owner := itineraryworkflow.OwnerDigest(cookie.Value)
	if _, err := itineraryworkflow.AddStop(app, owner, "aw0000000000002"); err != nil {
		t.Fatalf("add stop: %v", err)
	}
	view, err := loadBuilderView(app, owner, "csrf", builderState{})
	if err != nil {
		t.Fatalf("loadBuilderView: %v", err)
	}
	if len(view.Stops) != 1 {
		t.Fatalf("builder stops = %d, want 1", len(view.Stops))
	}
	if !strings.Contains(view.Stops[0].ImageURL, "thumb=500x0") {
		t.Errorf("editor/filmstrip image URL = %q, want the 500x0 no-upscale profile", view.Stops[0].ImageURL)
	}
}

// TestViewerPlateUsesRecordAndViewerProfiles proves the public slideshow plate
// uses the 1400px record/tour profile for display and the 2000px viewer profile
// for the deliberate zoom, the "1400/2000" contract that reopened task 9.5.
func TestViewerPlateUsesRecordAndViewerProfiles(t *testing.T) {
	app, _ := newItineraryMux(t)
	createArtist(t, app, "ar0000000000002", "Rembrandt van Rijn")
	createArtworkWithImage(t, app, "aw0000000000002", "Wide Work", "ar0000000000002", 3000)

	record, err := app.FindRecordById("artworks", "aw0000000000002")
	if err != nil {
		t.Fatalf("find artwork: %v", err)
	}

	plate := viewerPlate(app, record)
	if !strings.Contains(plate.DisplayURL, "thumb=1400x0") {
		t.Errorf("viewer display URL = %q, want the 1400x0 no-upscale profile", plate.DisplayURL)
	}
	if !strings.Contains(plate.ZoomURL, "thumb=2000x0") {
		t.Errorf("viewer zoom URL = %q, want the 2000x0 no-upscale profile", plate.ZoomURL)
	}
}
