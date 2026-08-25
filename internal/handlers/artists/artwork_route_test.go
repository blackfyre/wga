package artists

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/constants"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// newArtworkRouteApp builds the route test app in local development, where the
// WGA reproduction source link is populated.
func newArtworkRouteApp(t *testing.T) (*pocketbase.PocketBase, func(string) *httptest.ResponseRecorder) {
	t.Helper()
	return newArtworkRouteAppWithEnvironment(t, config.EnvironmentDevelopment)
}

func newArtworkRouteAppWithEnvironment(t *testing.T, environment config.Environment) (*pocketbase.PocketBase, func(string) *httptest.ResponseRecorder) {
	t.Helper()

	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Errorf("reset: %v", err)
		}
	})

	saveRecordCollection(t, app, core.NewBaseCollection("Schools"), constants.CollectionSchools,
		&core.TextField{Name: "name"},
		&core.TextField{Name: "slug"},
	)
	saveRecordCollection(t, app, core.NewBaseCollection("Art_forms"), constants.CollectionArtForms,
		&core.TextField{Name: "name"},
		&core.TextField{Name: "slug"},
	)
	saveRecordCollection(t, app, core.NewBaseCollection("Art_types"), constants.CollectionArtTypes,
		&core.TextField{Name: "name"},
		&core.TextField{Name: "slug"},
	)
	saveRecordCollection(t, app, core.NewBaseCollection("Locations"), constants.CollectionLocations,
		&core.TextField{Name: "name", Required: true},
		&core.BoolField{Name: "museum"},
		&core.BoolField{Name: "is_public"},
	)
	saveRecordCollection(t, app, core.NewBaseCollection("Glossary"), constants.CollectionGlossary,
		&core.TextField{Name: "expression", Required: true},
		&core.TextField{Name: "definition", Required: true},
	)
	saveRecordCollection(t, app, core.NewBaseCollection("Music_composer"), "music_composer",
		&core.TextField{Name: "name", Required: true},
		&core.SelectField{Name: "century", Values: []string{"17"}, MaxSelect: 1},
		&core.BoolField{Name: "published"},
	)
	saveRecordCollection(t, app, core.NewBaseCollection("Art_periods"), "art_periods",
		&core.TextField{Name: "name"},
		&core.TextField{Name: "slug"},
		&core.NumberField{Name: "start"},
		&core.NumberField{Name: "end"},
	)
	saveRecordCollection(t, app, core.NewBaseCollection("Artists"), constants.CollectionArtists,
		&core.TextField{Name: "name"},
		&core.TextField{Name: "slug"},
		&core.TextField{Name: "bio"},
		&core.TextField{Name: "profession"},
		&core.RelationField{Name: "school", CollectionId: constants.CollectionSchools, MinSelect: 0, MaxSelect: 10},
		&core.BoolField{Name: "published"},
	)
	saveRecordCollection(t, app, core.NewBaseCollection("Music_song"), "music_song",
		&core.TextField{Name: "title", Required: true},
		&core.RelationField{Name: "composer", CollectionId: "music_composer", MinSelect: 1, MaxSelect: 20},
		&core.TextField{Name: "source"},
		&core.BoolField{Name: "published"},
	)
	saveRecordCollection(t, app, core.NewBaseCollection("Artworks"), constants.CollectionArtworks,
		&core.TextField{Name: "title", Required: true},
		&core.RelationField{Name: "author", CollectionId: constants.CollectionArtists, MinSelect: 1, MaxSelect: 10},
		&core.RelationField{Name: "form", CollectionId: constants.CollectionArtForms, MinSelect: 1, MaxSelect: 20},
		&core.RelationField{Name: "type", CollectionId: constants.CollectionArtTypes, MinSelect: 0, MaxSelect: 20},
		&core.RelationField{Name: "school", CollectionId: constants.CollectionSchools, MinSelect: 0, MaxSelect: 10},
		&core.TextField{Name: "technique"},
		&core.EditorField{Name: "comment"},
		&core.BoolField{Name: "published"},
		&core.TextField{Name: "image"},
		&core.NumberField{Name: "image_width"},
		&core.NumberField{Name: "image_height"},
		&core.NumberField{Name: "source_row"},
		&core.NumberField{Name: "date_start"},
		&core.NumberField{Name: "date_end"},
		&core.BoolField{Name: "is_circa"},
		&core.TextField{Name: "date_qualifier"},
		&core.TextField{Name: "timeframe_text"},
		&core.RelationField{Name: "current_location_id", CollectionId: constants.CollectionLocations, MinSelect: 0, MaxSelect: 1},
		&core.RelationField{Name: "art_period_id", CollectionId: "art_periods", MinSelect: 0, MaxSelect: 1},
		&core.TextField{Name: "source_url"},
		&core.TextField{Name: "source_path"},
		&core.TextField{Name: "source_comment"},
		&core.JSONField{Name: "colour_palette"},
		&core.JSONField{Name: "colour_signature"},
		&core.TextField{Name: "colour_profile_version"},
		&core.TextField{Name: "colour_image_hash"},
		&core.NumberField{Name: "year"},
	)

	saveRecordRecord(t, app, constants.CollectionArtists, "artistone000001", map[string]any{
		"name": "Synthetic Artist", "slug": "synthetic-artist", "published": true,
	})
	saveRecordRecord(t, app, constants.CollectionArtworks, "workone00000001", map[string]any{
		"title": "A Painting", "author": []string{"artistone000001"}, "published": true,
		"form": []string{}, "image": "painting.jpg", "image_width": 800,
	})

	RegisterHandlers(app, environment)

	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatalf("create router: %v", err)
	}
	serveEvent := &core.ServeEvent{App: app, Router: router}
	if err := app.OnServe().Trigger(serveEvent, func(event *core.ServeEvent) error {
		return nil
	}); err != nil {
		t.Fatalf("trigger serve event: %v", err)
	}
	mux, err := serveEvent.Router.BuildMux()
	if err != nil {
		t.Fatalf("build mux: %v", err)
	}

	request := func(path string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		return recorder
	}

	return app, request
}

func TestArtworkRouteRendersFullPageWithDefaultBasis(t *testing.T) {
	_, request := newArtworkRouteApp(t)

	recorder := request("/artists/synthetic-artist-artistone000001/a-painting-workone00000001")

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "A Painting") {
		t.Error("expected artwork title in response")
	}
	if got := recorder.Header().Get("HX-Push-Url"); got != "/artists/synthetic-artist-artistone000001/a-painting-workone00000001" {
		t.Errorf("HX-Push-Url = %q, want canonical URL without basis", got)
	}
}

func TestArtworkRouteRendersCanonicalWGAProvenanceOnlyForSafeSource(t *testing.T) {
	app, request := newArtworkRouteApp(t)
	artwork, err := app.FindRecordById(constants.CollectionArtworks, "workone00000001")
	if err != nil {
		t.Fatalf("find artwork: %v", err)
	}
	artwork.Set("source_url", "html/a/artist/painting.html")
	if err := app.Save(artwork); err != nil {
		t.Fatalf("save artwork source URL: %v", err)
	}

	path := "/artists/synthetic-artist-artistone000001/a-painting-workone00000001"
	recorder := request(path)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	body := recorder.Body.String()
	for _, expected := range []string{
		"VIEW ORIGINAL AT WEB GALLERY OF ART",
		`href="https://www.wga.hu/html/a/artist/painting.html"`,
		`target="_blank"`,
		`rel="noopener noreferrer"`,
		"DOWNLOAD THE FULL FILE",
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("response missing %q", expected)
		}
	}

	artwork.Set("source_url", "https://example.com/html/a/artist/painting.html")
	if err := app.Save(artwork); err != nil {
		t.Fatalf("save unsafe artwork source URL: %v", err)
	}
	recorder = request(path)
	if strings.Contains(recorder.Body.String(), "VIEW ORIGINAL AT WEB GALLERY OF ART") {
		t.Error("unsafe producer source URL must fail closed")
	}
}

// TestArtworkRouteHidesWGAProvenanceOutsideDevelopment proves the WGA
// reproduction source link is a local-development convenience: it must not
// render in any deployed environment, even for a safe, allow-listed source
// URL, while the deliberate source-file download stays available everywhere.
func TestArtworkRouteHidesWGAProvenanceOutsideDevelopment(t *testing.T) {
	app, request := newArtworkRouteAppWithEnvironment(t, config.EnvironmentProduction)
	artwork, err := app.FindRecordById(constants.CollectionArtworks, "workone00000001")
	if err != nil {
		t.Fatalf("find artwork: %v", err)
	}
	artwork.Set("source_url", "html/a/artist/painting.html")
	if err := app.Save(artwork); err != nil {
		t.Fatalf("save artwork source URL: %v", err)
	}

	recorder := request("/artists/synthetic-artist-artistone000001/a-painting-workone00000001")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	body := recorder.Body.String()
	if strings.Contains(body, "VIEW ORIGINAL AT WEB GALLERY OF ART") {
		t.Error("WGA provenance link must not render outside local development")
	}
	if !strings.Contains(body, "DOWNLOAD THE FULL FILE") {
		t.Error("the deliberate source-file download must still render outside development")
	}
}

func TestArtworkRoutePreservesNonDefaultBasis(t *testing.T) {
	_, request := newArtworkRouteApp(t)

	recorder := request("/artists/synthetic-artist-artistone000001/a-painting-workone00000001?basis=collection")

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if got := recorder.Header().Get("HX-Push-Url"); got != "/artists/synthetic-artist-artistone000001/a-painting-workone00000001?basis=collection" {
		t.Errorf("HX-Push-Url = %q, want canonical URL with basis=collection", got)
	}
}

func TestArtworkRouteCanonicalisesInvalidBasis(t *testing.T) {
	_, request := newArtworkRouteApp(t)

	recorder := request("/artists/synthetic-artist-artistone000001/a-painting-workone00000001?basis=garbage")

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if got := recorder.Header().Get("HX-Push-Url"); got != "/artists/synthetic-artist-artistone000001/a-painting-workone00000001" {
		t.Errorf("HX-Push-Url = %q, want canonical URL without basis (invalid input)", got)
	}
}

func TestArtworkRouteRejectsUnpublishedArtist(t *testing.T) {
	app, request := newArtworkRouteApp(t)
	saveRecordRecord(t, app, constants.CollectionArtists, "hiddenartist001", map[string]any{
		"name": "Hidden Artist", "slug": "hidden-artist", "published": false,
	})

	recorder := request("/artists/hidden-artist-hiddenartist001/a-painting-workone00000001")
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (unpublished artist)", recorder.Code)
	}
}

func TestArtworkRouteRejectsUnpublishedArtwork(t *testing.T) {
	app, request := newArtworkRouteApp(t)
	saveRecordRecord(t, app, constants.CollectionArtworks, "hiddenwork00001", map[string]any{
		"title": "Hidden Work", "author": []string{"artistone000001"}, "published": false,
	})

	recorder := request("/artists/synthetic-artist-artistone000001/hidden-work-hiddenwork00001")
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (unpublished artwork)", recorder.Code)
	}
}

func TestArtworkRouteRejectsMismatchedArtist(t *testing.T) {
	app, request := newArtworkRouteApp(t)
	saveRecordRecord(t, app, constants.CollectionArtists, "artisttwo000001", map[string]any{
		"name": "Other Artist", "slug": "other-artist", "published": true,
	})
	saveRecordRecord(t, app, constants.CollectionArtworks, "otherwork000001", map[string]any{
		"title": "Other Work", "author": []string{"artisttwo000001"}, "published": true,
	})

	recorder := request("/artists/synthetic-artist-artistone000001/other-work-otherwork000001")
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (mismatched artist)", recorder.Code)
	}
}
