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

// newArtworkRouteApp builds the route test app in local development.
func newArtworkRouteApp(t *testing.T) (*pocketbase.PocketBase, func(string, ...bool) *httptest.ResponseRecorder) {
	t.Helper()
	return newArtworkRouteAppWithEnvironment(t, config.EnvironmentDevelopment)
}

func newArtworkRouteAppWithEnvironment(t *testing.T, environment config.Environment) (*pocketbase.PocketBase, func(string, ...bool) *httptest.ResponseRecorder) {
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
		&core.TextField{Name: "filing_name"},
		&core.TextField{Name: "short_name"},
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
		&core.NumberField{Name: "image_size_bytes"},
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
		"form": []string{}, "image": "painting.jpg", "image_width": 800, "image_size_bytes": 123456,
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

	request := func(path string, htmx ...bool) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		if len(htmx) > 0 && htmx[0] {
			req.Header.Set("HX-Request", "true")
		}
		mux.ServeHTTP(recorder, req)
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

func TestArtworkRouteNeverRendersSourceClaimsAndKeepsFullFile(t *testing.T) {
	for _, environment := range []config.Environment{
		config.EnvironmentDevelopment,
		config.EnvironmentTest,
		config.EnvironmentStaging,
		config.EnvironmentProduction,
	} {
		t.Run(string(environment), func(t *testing.T) {
			app, request := newArtworkRouteAppWithEnvironment(t, environment)
			artwork, err := app.FindRecordById(constants.CollectionArtworks, "workone00000001")
			if err != nil {
				t.Fatalf("find artwork: %v", err)
			}

			for _, sourceURL := range []string{
				"html/a/artist/painting.html",
				"https://example.com/html/a/artist/painting.html",
			} {
				artwork.Set("source_url", sourceURL)
				if err := app.Save(artwork); err != nil {
					t.Fatalf("save artwork source URL: %v", err)
				}

				recorder := request("/artists/synthetic-artist-artistone000001/a-painting-workone00000001")
				if recorder.Code != http.StatusOK {
					t.Fatalf("status = %d, want 200", recorder.Code)
				}
				body := recorder.Body.String()
				recordSection := body
				if start := strings.Index(body, "<figure"); start >= 0 {
					recordSection = body[start:]
				}
				if end := strings.Index(recordSection, "</figure>"); end >= 0 {
					recordSection = recordSection[:end]
				}
				if strings.Contains(recordSection, "VIEW ORIGINAL AT WEB GALLERY OF ART") {
					t.Errorf("%s source URL must not render a public source claim", sourceURL)
				}
				if strings.Contains(strings.ToUpper(recordSection), "LICENCE") {
					t.Errorf("%s source URL must not render a public licence claim", sourceURL)
				}
				if strings.Contains(recordSection, sourceURL) {
					t.Errorf("%s source URL must not render in the artwork record", sourceURL)
				}
				if !strings.Contains(body, "DOWNLOAD THE FULL FILE") {
					t.Errorf("%s source URL must retain the full-file link", sourceURL)
				}
			}
		})
	}
}

func TestArtworkRouteRendersRecordIdentityDateAndEvidenceBackedFile(t *testing.T) {
	app, request := newArtworkRouteApp(t)
	artist, err := app.FindRecordById(constants.CollectionArtists, "artistone000001")
	if err != nil {
		t.Fatalf("find artist: %v", err)
	}
	artist.Set("filing_name", "Surname, Given")
	artist.Set("short_name", "Given")
	if err := app.Save(artist); err != nil {
		t.Fatalf("save artist identity: %v", err)
	}
	artwork, err := app.FindRecordById(constants.CollectionArtworks, "workone00000001")
	if err != nil {
		t.Fatalf("find artwork: %v", err)
	}
	artwork.Set("image_width", 1200)
	artwork.Set("image_height", 800)
	artwork.Set("image_size_bytes", 1_400_000)
	artwork.Set("date_start", 1900)
	artwork.Set("year", 1900)
	if err := app.Save(artwork); err != nil {
		t.Fatalf("save artwork evidence: %v", err)
	}

	recorder := request("/artists/synthetic-artist-artistone000001/a-painting-workone00000001")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	body := recorder.Body.String()
	for _, expected := range []string{
		"Given",
		"Surname, Given · 1900",
		"1200 × 800 px · JPEG · 1.4 MB",
		"DOWNLOAD THE FULL FILE",
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("expected response to contain %q", expected)
		}
	}
	if strings.Contains(body, "Given, Surname") {
		t.Error("artwork route must not reconstruct the artist filing name")
	}
}

func TestArtworkRouteRendersCountedHoldingAndFullHTMXParity(t *testing.T) {
	app, request := newArtworkRouteApp(t)
	titles := []string{"Related Work A", "Related Work B", "Related Work C", "Related Work D", "Related Work E"}
	for i, id := range []string{"related00000001", "related00000002", "related00000003", "related00000004", "related00000005"} {
		saveRecordRecord(t, app, constants.CollectionArtworks, id, map[string]any{
			"title": titles[i], "author": []string{"artistone000001"}, "published": true,
			"form": []string{}, "image": "related.jpg", "date_start": 1900 + i,
		})
	}

	full := request("/artists/synthetic-artist-artistone000001/a-painting-workone00000001")
	fragment := request("/artists/synthetic-artist-artistone000001/a-painting-workone00000001", true)
	if full.Code != http.StatusOK || fragment.Code != http.StatusOK {
		t.Fatalf("full/HTMX status = %d/%d, want 200/200", full.Code, fragment.Code)
	}
	if !strings.Contains(full.Body.String(), "FIND MORE 6 IN THE ARTWORK SEARCH") {
		t.Error("full response must expose the counted artist holding")
	}
	if !strings.Contains(full.Body.String(), `href="/artworks?artist=Synthetic+Artist"`) {
		t.Error("holding link must preserve the artist filter")
	}
	if !strings.Contains(fragment.Body.String(), "A Painting") || !strings.Contains(fragment.Body.String(), "FIND MORE 6 IN THE ARTWORK SEARCH") {
		t.Error("HTMX response must preserve the canonical artwork record and holding")
	}
	if got := fragment.Header().Get("HX-Push-Url"); got != "/artists/synthetic-artist-artistone000001/a-painting-workone00000001" {
		t.Errorf("HTMX HX-Push-Url = %q, want canonical artwork URL", got)
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

func TestArtworkRouteRedirectsPaletteBasis(t *testing.T) {
	_, request := newArtworkRouteApp(t)

	recorder := request("/artists/synthetic-artist-artistone000001/a-painting-workone00000001?basis=palette")

	if recorder.Code != http.StatusMovedPermanently {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusMovedPermanently)
	}
	if got := recorder.Header().Get("Location"); got != "/artists/synthetic-artist-artistone000001/a-painting-workone00000001" {
		t.Errorf("Location = %q, want canonical URL without palette basis", got)
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
