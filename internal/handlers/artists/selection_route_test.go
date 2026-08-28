package artists

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/config"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func createSelectionRouteCollections(t *testing.T, app *pocketbase.PocketBase) {
	t.Helper()

	artists := core.NewBaseCollection("Artists")
	artists.Id = "artists"
	artists.MarkAsNew()
	artists.Fields.Add(
		&core.TextField{Name: "name", Required: true},
		&core.TextField{Name: "filing_name"},
		&core.TextField{Name: "short_name"},
		&core.TextField{Name: "slug"},
		&core.NumberField{Name: "year_of_birth"},
		&core.NumberField{Name: "year_of_death"},
		&core.TextField{Name: "place_of_birth"},
		&core.TextField{Name: "place_of_death"},
		&core.TextField{Name: "profession"},
		&core.EditorField{Name: "bio"},
		&core.BoolField{Name: "published"},
	)
	if err := app.Save(artists); err != nil {
		t.Fatalf("save artists: %v", err)
	}

	artworks := core.NewBaseCollection("Artworks")
	artworks.Id = "artworks"
	artworks.MarkAsNew()
	artworks.Fields.Add(
		&core.TextField{Name: "title", Required: true},
		&core.RelationField{Name: "author", CollectionId: artists.Id, MinSelect: 1, MaxSelect: 10},
		&core.TextField{Name: "technique"},
		&core.TextField{Name: "comment"},
		&core.TextField{Name: "image"},
		&core.NumberField{Name: "image_width"},
		&core.NumberField{Name: "image_height"},
		&core.BoolField{Name: "published"},
	)
	if err := app.Save(artworks); err != nil {
		t.Fatalf("save artworks: %v", err)
	}

	selections := core.NewBaseCollection("Art_selections")
	selections.Id = "art_selections"
	selections.MarkAsNew()
	selections.Fields.Add(
		&core.RelationField{Name: "artist", CollectionId: artists.Id, MinSelect: 1, MaxSelect: 1, Required: true},
		&core.TextField{Name: "title", Required: true},
		&core.TextField{Name: "context"},
		&core.TextField{Name: "display_title", Required: true},
		&core.EditorField{Name: "commentary"},
		&core.RelationField{Name: "artworks", CollectionId: artworks.Id, MinSelect: 1, MaxSelect: 1000, Required: true},
		&core.TextField{Name: "source_path", Required: true, Hidden: true},
		&core.TextField{Name: "source_hash", Required: true, Hidden: true},
		&core.BoolField{Name: "published", Required: true},
	)
	if err := app.Save(selections); err != nil {
		t.Fatalf("save selections: %v", err)
	}
}

func saveSelectionRouteRecord(t *testing.T, app *pocketbase.PocketBase, collection string, id string, fields map[string]any) {
	t.Helper()
	coll, err := app.FindCollectionByNameOrId(collection)
	if err != nil {
		t.Fatalf("find %s: %v", collection, err)
	}
	record := core.NewRecord(coll)
	record.Id = id
	if collection == "artists" {
		fields["filing_name"] = fields["name"]
		fields["short_name"] = fields["name"]
	}
	for key, value := range fields {
		record.Set(key, value)
	}
	if err := app.Save(record); err != nil {
		t.Fatalf("save %s %s: %v", collection, id, err)
	}
}

func newSelectionRouteApp(t *testing.T) (*pocketbase.PocketBase, func(string, ...bool) *httptest.ResponseRecorder) {
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

	createSelectionRouteCollections(t, app)
	saveSelectionRouteRecord(t, app, "artists", "artistone000001", map[string]any{
		"name": "Synthetic Artist", "slug": "synthetic-artist", "published": true,
	})
	saveSelectionRouteRecord(t, app, "artists", "artisttwo000001", map[string]any{
		"name": "Other Artist", "slug": "other-artist", "published": true,
	})
	saveSelectionRouteRecord(t, app, "artworks", "workone00000001", map[string]any{
		"title": "A Painting", "author": []string{"artistone000001"}, "published": true, "image": "painting.jpg", "image_width": 800, "technique": "Oil",
	})
	saveSelectionRouteRecord(t, app, "art_selections", "rselect00000001", map[string]any{
		"artist": []string{"artistone000001"}, "title": "Paintings", "display_title": "Dürer: Paintings",
		"commentary": "<p>An editorial lede.</p>", "artworks": []string{"workone00000001"},
		"source_path": "html/a/artist/paintings/index.html", "source_hash": "source-hash", "published": true,
	})
	saveSelectionRouteRecord(t, app, "art_selections", "rselect00000002", map[string]any{
		"artist": []string{"artistone000001"}, "title": "Studies", "display_title": "Dürer: Studies",
		"artworks":    []string{"workone00000001"},
		"source_path": "html/a/artist/studies/index.html", "source_hash": "source-hash", "published": true,
	})
	saveSelectionRouteRecord(t, app, "art_selections", "rselect00000004", map[string]any{
		"artist": []string{"artisttwo000001"}, "title": "Foreign", "display_title": "Other: Foreign",
		"artworks":    []string{"workone00000001"},
		"source_path": "html/a/artist/foreign/index.html", "source_hash": "source-hash", "published": true,
	})

	RegisterHandlers(app, config.EnvironmentTest)

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

func TestSelectionRouteRendersFullPage(t *testing.T) {
	_, request := newSelectionRouteApp(t)

	recorder := request("/artists/synthetic-artist-artistone000001/selections/rselect00000001")

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	body := recorder.Body.String()
	for _, expected := range []string{
		"<!doctype html>",
		"Dürer: Paintings",
		"An editorial lede.",
		"A Painting",
		"Dürer: Studies",
		"VIEW FULL HOLDING",
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("expected response to contain %q", expected)
		}
	}
	if got := recorder.Header().Get("HX-Push-Url"); got != "/artists/synthetic-artist-artistone000001/selections/rselect00000001" {
		t.Errorf("HX-Push-Url = %q, want canonical selection URL", got)
	}
	if strings.Contains(body, "Other: Foreign") {
		t.Error("foreign-artist selection must not appear among other selections")
	}
}

func TestSelectionRouteRendersShortIdentitySectionGridAndFullHTMXParity(t *testing.T) {
	app, request := newSelectionRouteApp(t)
	artist, err := app.FindRecordById("artists", "artistone000001")
	if err != nil {
		t.Fatalf("find artist: %v", err)
	}
	artist.Set("filing_name", "Surname, Given")
	artist.Set("short_name", "Given")
	if err := app.Save(artist); err != nil {
		t.Fatalf("save artist identity: %v", err)
	}

	full := request("/artists/synthetic-artist-artistone000001/selections/rselect00000001")
	fragment := request("/artists/synthetic-artist-artistone000001/selections/rselect00000001", true)
	if full.Code != http.StatusOK || fragment.Code != http.StatusOK {
		t.Fatalf("full/HTMX status = %d/%d, want 200/200", full.Code, fragment.Code)
	}
	body := full.Body.String()
	for _, expected := range []string{
		"Given",
		"21 — SELECTION",
		`grid grid-cols-2 md:grid-cols-4`,
		"An editorial lede.",
		"VIEW FULL HOLDING",
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("expected response to contain %q", expected)
		}
	}
	if strings.Contains(body, "Given, Surname") {
		t.Error("selection route must not reconstruct the artist identity")
	}
	if !strings.Contains(fragment.Body.String(), "Dürer: Paintings") || !strings.Contains(fragment.Body.String(), "21 — SELECTION") {
		t.Error("HTMX response must preserve the canonical selection record")
	}
	if got := fragment.Header().Get("HX-Push-Url"); got != "/artists/synthetic-artist-artistone000001/selections/rselect00000001" {
		t.Errorf("HTMX HX-Push-Url = %q, want canonical selection URL", got)
	}
}

func TestSelectionRouteSanitisesCommentary(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Errorf("reset: %v", err)
		}
	})
	createSelectionRouteCollections(t, app)
	saveSelectionRouteRecord(t, app, "artists", "artistone000001", map[string]any{
		"name": "Synthetic Artist", "slug": "synthetic-artist", "published": true,
	})
	saveSelectionRouteRecord(t, app, "artworks", "workone00000001", map[string]any{
		"title": "A Painting", "author": []string{"artistone000001"}, "published": true, "image": "painting.jpg", "image_width": 800,
	})
	saveSelectionRouteRecord(t, app, "art_selections", "rselect00000001", map[string]any{
		"artist": []string{"artistone000001"}, "title": "Paintings", "display_title": "Paintings",
		"commentary": "<p>Safe prose.</p><script>alert(1)</script>", "artworks": []string{"workone00000001"},
		"source_path": "html/a/artist/paintings/index.html", "source_hash": "source-hash", "published": true,
	})

	RegisterHandlers(app, config.EnvironmentTest)
	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatalf("create router: %v", err)
	}
	serveEvent := &core.ServeEvent{App: app, Router: router}
	if err := app.OnServe().Trigger(serveEvent, func(event *core.ServeEvent) error { return nil }); err != nil {
		t.Fatalf("trigger serve event: %v", err)
	}
	mux, err := serveEvent.Router.BuildMux()
	if err != nil {
		t.Fatalf("build mux: %v", err)
	}

	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/artists/synthetic-artist-artistone000001/selections/rselect00000001", nil))

	body := recorder.Body.String()
	if !strings.Contains(body, "Safe prose.") {
		t.Error("expected sanitised commentary to retain safe prose")
	}
	if strings.Contains(body, "<script>alert(1)</script>") {
		t.Error("unsanitised script must not be rendered")
	}
}

func TestSelectionRouteRejectsForeignAndMissing(t *testing.T) {
	_, request := newSelectionRouteApp(t)

	for _, path := range []string{
		"/artists/synthetic-artist-artistone000001/selections/rselect00000004", // foreign artist
		"/artists/synthetic-artist-artistone000001/selections/rmissing0000001", // missing
	} {
		recorder := request(path)
		if recorder.Code != http.StatusNotFound {
			t.Errorf("%s status = %d, want 404", path, recorder.Code)
		}
	}
}

func TestSelectionRouteNormalisesArtistSlug(t *testing.T) {
	_, request := newSelectionRouteApp(t)

	recorder := request("/artists/wrong-slug-artistone000001/selections/rselect00000001")

	if recorder.Code != http.StatusMovedPermanently {
		t.Fatalf("status = %d, want 301", recorder.Code)
	}
	if got, want := recorder.Header().Get("Location"), "/artists/synthetic-artist-artistone000001/selections/rselect00000001"; got != want {
		t.Errorf("Location = %q, want %q", got, want)
	}
}

func TestSelectionRouteRendersCitation(t *testing.T) {
	_, request := newSelectionRouteApp(t)

	recorder := request("/artists/synthetic-artist-artistone000001/selections/rselect00000001")

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	body := recorder.Body.String()
	for _, expected := range []string{
		"CITE THIS RECORD — BIBTEX",
		"wga-rselect00000001",
		"Dürer: Paintings (selection)",
		"/artists/synthetic-artist-artistone000001/selections/rselect00000001",
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("expected selection citation to contain %q", expected)
		}
	}
}

func TestSelectionRouteRendersHonestMissingCommentary(t *testing.T) {
	_, request := newSelectionRouteApp(t)

	recorder := request("/artists/synthetic-artist-artistone000001/selections/rselect00000002")

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if body := recorder.Body.String(); !strings.Contains(body, "Commentary is unavailable for this selection.") {
		t.Error("expected honest missing-commentary state on the selection page")
	}
}

func TestSelectionRouteRendersServerSideWithoutViewerHooks(t *testing.T) {
	_, request := newSelectionRouteApp(t)

	recorder := request("/artists/synthetic-artist-artistone000001/selections/rselect00000001")

	body := recorder.Body.String()
	if !strings.Contains(body, "<html") {
		t.Error("selection route must render the complete server-side document")
	}
	if strings.Contains(body, "data-viewer") {
		t.Error("selection works must not carry artwork viewer hooks")
	}
}

func TestSelectionRouteRendersHoldingWithZeroRenderableWorks(t *testing.T) {
	_, request := newSelectionRouteApp(t)

	// rselect00000004 belongs to Other Artist but references Synthetic Artist's
	// work, so its ordered membership resolves to zero renderable works.
	recorder := request("/artists/other-artist-artisttwo000001/selections/rselect00000004")

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "No selected works are available for this selection.") {
		t.Error("expected honest empty-works state for the zero-member selection")
	}
	if !strings.Contains(body, "VIEW FULL HOLDING") {
		t.Error("selection with zero renderable works must still expose the wider-holding link")
	}
	if !strings.Contains(body, `href="/artworks?artist=Other+Artist"`) {
		t.Error("holding link must resolve to the artist's wider catalogue route")
	}
}
