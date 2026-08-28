package tours

import (
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/blackfyre/wga/internal/assets"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// Task 8.6 test-only Guided Tours fixture and Playwright harness.
//
// newTask86FixtureApp + seedTask86Tours create exactly the same two synthetic
// tours described in internal/tours/synthetic_fixture_test.go. Every label names
// the material as synthetic, and the records live only in this test's disposable
// data directory, so they can never enter default or release application data.
//
// Disposable Playwright invocation (from the repository root):
//
//	WGA_TASK86_HARNESS=1 go test ./internal/handlers/tours -run '^TestTask86PlaywrightHarness$' -count=1 -timeout 0
//
// The harness prints `WGA_TASK86_HARNESS_READY http://127.0.0.1:<port>` to
// stderr, serves until the invoking process kills it, and honours
// WGA_TASK86_PORT (default 8090). Point Playwright at it with
// WGA_PROTOCOL=http and WGA_HOSTNAME=127.0.0.1:<port>.
const (
	task86RebuiltSlug = "synthetic-rebuilt-tour-task86"
	task86LegacySlug  = "synthetic-legacy-tour-task86"
)

func newTask86FixtureApp(t *testing.T) *pocketbase.PocketBase {
	t.Helper()
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	t.Cleanup(func() { _ = app.ResetBootstrapState() })

	artworks := task86Collection("Artworks", "artworks", &core.TextField{Name: "title"}, &core.TextField{Name: "image"}, &core.NumberField{Name: "image_width"}, &core.BoolField{Name: "published"})
	task86SaveCollection(t, app, artworks)
	editors := task86Collection("Guided_tour_editors", "guided_tour_editors", &core.TextField{Name: "editor_key"}, &core.TextField{Name: "name"})
	task86SaveCollection(t, app, editors)
	tours := task86Collection("Guided_tours", "guided_tours",
		&core.TextField{Name: "slug"}, &core.TextField{Name: "title"}, &core.TextField{Name: "blurb"}, &core.TextField{Name: "kind"},
		&core.TextField{Name: "tour_number"}, &core.RelationField{Name: "editor", CollectionId: editors.Id, MaxSelect: 1}, &core.NumberField{Name: "series_position"},
		&core.NumberField{Name: "published_year"}, &core.NumberField{Name: "revised_year"}, &core.TextField{Name: "legacy_url"},
		&core.TextField{Name: "presentation_status"}, &core.TextField{Name: "publication_status"})
	task86SaveCollection(t, app, tours)
	revisions := task86Collection("Guided_tour_revisions", "guided_tour_revisions", &core.RelationField{Name: "tour", CollectionId: tours.Id, MaxSelect: 1},
		&core.TextField{Name: "revision_key"}, &core.NumberField{Name: "revision_number"}, &core.TextField{Name: "label"}, &core.TextField{Name: "source_hash"})
	task86SaveCollection(t, app, revisions)
	tours.Fields.Add(&core.RelationField{Name: "published_revision", CollectionId: revisions.Id, MaxSelect: 1})
	if err := app.Save(tours); err != nil {
		t.Fatalf("add revision relation: %v", err)
	}
	sections := task86Collection("Guided_tour_sections", "guided_tour_sections", &core.RelationField{Name: "tour", CollectionId: tours.Id, MaxSelect: 1},
		&core.RelationField{Name: "revision", CollectionId: revisions.Id, MaxSelect: 1}, &core.NumberField{Name: "section_order"}, &core.TextField{Name: "title"})
	task86SaveCollection(t, app, sections)
	pages := task86Collection("Guided_tour_pages", "guided_tour_pages", &core.RelationField{Name: "tour", CollectionId: tours.Id, MaxSelect: 1},
		&core.RelationField{Name: "revision", CollectionId: revisions.Id, MaxSelect: 1}, &core.RelationField{Name: "section", CollectionId: sections.Id, MaxSelect: 1},
		&core.NumberField{Name: "page_position"}, &core.TextField{Name: "page_type"}, &core.TextField{Name: "title"}, &core.TextField{Name: "dateline"},
		&core.TextField{Name: "source_page_id"}, &core.TextField{Name: "source_path"}, &core.TextField{Name: "source_hash"},
		&core.RelationField{Name: "artwork", CollectionId: artworks.Id, MaxSelect: 1}, &core.TextField{Name: "credit"}, &core.TextField{Name: "work_target_path"})
	task86SaveCollection(t, app, pages)
	task86SaveCollection(t, app, task86Collection("Guided_tour_blocks", "guided_tour_blocks", &core.RelationField{Name: "page", CollectionId: pages.Id, MaxSelect: 1},
		&core.NumberField{Name: "block_order"}, &core.TextField{Name: "block_kind"}, &core.EditorField{Name: "content_html"}))
	task86SaveCollection(t, app, task86Collection("Guided_tour_index_rows", "guided_tour_index_rows", &core.RelationField{Name: "page", CollectionId: pages.Id, MaxSelect: 1},
		&core.NumberField{Name: "row_order"}, &core.TextField{Name: "name"}, &core.TextField{Name: "dates"}, &core.TextField{Name: "note"}, &core.TextField{Name: "target_path"}))
	task86SaveCollection(t, app, task86Collection("Guided_tour_bibliography", "guided_tour_bibliography", &core.RelationField{Name: "tour", CollectionId: tours.Id, MaxSelect: 1},
		&core.RelationField{Name: "revision", CollectionId: revisions.Id, MaxSelect: 1}, &core.NumberField{Name: "item_order"}, &core.TextField{Name: "citation"}))
	task86SaveCollection(t, app, task86Collection("Guided_tour_legacy_routes", "guided_tour_legacy_routes",
		&core.TextField{Name: "legacy_path"}, &core.RelationField{Name: "tour_page", CollectionId: pages.Id, MaxSelect: 1}))
	return app
}

func task86Collection(name, id string, fields ...core.Field) *core.Collection {
	collection := core.NewBaseCollection(name)
	collection.Id = id
	collection.MarkAsNew()
	collection.Fields.Add(fields...)
	return collection
}

func task86SaveCollection(t *testing.T, app core.App, collection *core.Collection) {
	t.Helper()
	if err := app.Save(collection); err != nil {
		t.Fatalf("save %s: %v", collection.Name, err)
	}
}

func task86SaveRecord(t *testing.T, app core.App, collection string, values map[string]any) *core.Record {
	t.Helper()
	coll, err := app.FindCollectionByNameOrId(collection)
	if err != nil {
		t.Fatal(err)
	}
	record := core.NewRecord(coll)
	for key, value := range values {
		record.Set(key, value)
	}
	if err := app.Save(record); err != nil {
		t.Fatalf("save %s: %v", collection, err)
	}
	return record
}

func seedTask86Tours(t *testing.T, app core.App) {
	t.Helper()
	seedTask86RebuiltTour(t, app)
	seedTask86LegacyTour(t, app)
}

func seedTask86RebuiltTour(t *testing.T, app core.App) {
	t.Helper()
	editor := task86SaveRecord(t, app, "guided_tour_editors", map[string]any{"editor_key": "syn-editor-" + task86RebuiltSlug, "name": "Synthetic Fixture Editor"})
	tour := task86SaveRecord(t, app, "guided_tours", map[string]any{"slug": task86RebuiltSlug,
		"title": "Synthetic Rebuilt Tour (Task 8.6 Fixture)", "blurb": "Deterministic synthetic tour. Not editorial content.", "kind": "survey",
		"tour_number": "T86", "editor": editor.Id, "series_position": 1, "published_year": 2001, "revised_year": 2002,
		"presentation_status": "rebuilt", "publication_status": "published"})
	revision := task86SaveRecord(t, app, "guided_tour_revisions", map[string]any{"tour": tour.Id, "revision_key": "syn-rev-1",
		"revision_number": 1, "label": "Synthetic revision 1", "source_hash": "syn-revision-source-hash"})
	tour.Set("published_revision", revision.Id)
	if err := app.Save(tour); err != nil {
		t.Fatalf("publish synthetic revision: %v", err)
	}
	section := task86SaveRecord(t, app, "guided_tour_sections", map[string]any{"tour": tour.Id, "revision": revision.Id,
		"section_order": 1, "title": "Synthetic Opening"})
	textPage := task86SaveRecord(t, app, "guided_tour_pages", map[string]any{"tour": tour.Id, "revision": revision.Id,
		"section": section.Id, "page_position": 1, "page_type": "text", "title": "Synthetic Text Page",
		"source_page_id": "syn-text", "source_path": "/tours/source/synthetic-text.html", "source_hash": "syn-text-hash"})
	task86SaveRecord(t, app, "guided_tour_blocks", map[string]any{"page": textPage.Id, "block_order": 1, "block_kind": "prose",
		"content_html": "<p>Synthetic tour prose. Not editorial content.</p>"})
	artwork := task86SaveRecord(t, app, "artworks", map[string]any{"title": "Synthetic Work", "image": "synthetic-work.jpg",
		"image_width": 2048, "published": true})
	task86SaveRecord(t, app, "guided_tour_pages", map[string]any{"tour": tour.Id, "revision": revision.Id, "section": section.Id,
		"page_position": 2, "page_type": "picture", "title": "Synthetic Picture Page",
		"source_page_id": "syn-picture", "source_path": "/tours/source/synthetic-picture.html", "source_hash": "syn-picture-hash",
		"artwork": artwork.Id, "credit": "Synthetic credit", "work_target_path": "/artworks/synthetic-work"})
	listPage := task86SaveRecord(t, app, "guided_tour_pages", map[string]any{"tour": tour.Id, "revision": revision.Id,
		"section": section.Id, "page_position": 3, "page_type": "list", "title": "Synthetic Index Page",
		"source_page_id": "syn-list", "source_path": "/tours/source/synthetic-list.html", "source_hash": "syn-list-hash"})
	task86SaveRecord(t, app, "guided_tour_index_rows", map[string]any{"page": listPage.Id, "row_order": 1, "name": "Synthetic safe row",
		"dates": "1500–1550", "note": "Synthetic note", "target_path": "/artists/synthetic"})
	task86SaveRecord(t, app, "guided_tour_bibliography", map[string]any{"tour": tour.Id, "revision": revision.Id,
		"item_order": 1, "citation": "Synthetic source citation. Not a real source."})
	task86SaveRecord(t, app, "guided_tour_legacy_routes", map[string]any{"legacy_path": "/tours/source/synthetic-text.html", "tour_page": textPage.Id})
	task86SaveRecord(t, app, "guided_tour_legacy_routes", map[string]any{"legacy_path": "/tours/source/synthetic-list.html", "tour_page": listPage.Id})
}

func seedTask86LegacyTour(t *testing.T, app core.App) {
	t.Helper()
	editor := task86SaveRecord(t, app, "guided_tour_editors", map[string]any{"editor_key": "syn-editor-" + task86LegacySlug, "name": "Synthetic Fixture Editor"})
	tour := task86SaveRecord(t, app, "guided_tours", map[string]any{"slug": task86LegacySlug,
		"title": "Synthetic Legacy Tour (Task 8.6 Fixture)", "blurb": "Deterministic synthetic tour. Not editorial content.", "kind": "site",
		"tour_number": "T87", "editor": editor.Id, "series_position": 2, "published_year": 2001, "revised_year": 2002,
		"legacy_url": "https://example.org/synthetic-legacy-original", "presentation_status": "original", "publication_status": "published"})
	revision := task86SaveRecord(t, app, "guided_tour_revisions", map[string]any{"tour": tour.Id, "revision_key": "syn-rev-1",
		"revision_number": 1, "label": "Synthetic revision 1", "source_hash": "syn-revision-source-hash"})
	tour.Set("published_revision", revision.Id)
	if err := app.Save(tour); err != nil {
		t.Fatalf("publish synthetic revision: %v", err)
	}
}

// registerTask86Mux registers the tours handler and the embedded static assets
// on a fresh router and returns it. Callers must run BuildMux inside an
// OnServe().Trigger callback because the tours handler binds its routes there.
func registerTask86Mux(t *testing.T, app *pocketbase.PocketBase) *core.ServeEvent {
	t.Helper()
	RegisterHandlers(app)
	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	fsys, err := fs.Sub(assets.PublicFiles, "public")
	if err != nil {
		t.Fatal(err)
	}
	router.GET("/assets/{path...}", apis.Static(fsys, false))
	return &core.ServeEvent{App: app, Router: router}
}

func TestTask86HarnessServesPopulatedRoutesAndAssets(t *testing.T) {
	app := newTask86FixtureApp(t)
	seedTask86Tours(t, app)
	serve := registerTask86Mux(t, app)

	if err := app.OnServe().Trigger(serve, func(event *core.ServeEvent) error {
		mux, err := event.Router.BuildMux()
		if err != nil {
			return err
		}
		get := func(path string) *httptest.ResponseRecorder {
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
			return rec
		}

		index := get("/tours")
		if index.Code != http.StatusOK {
			t.Fatalf("index status=%d", index.Code)
		}
		for _, expected := range []string{
			"Synthetic Rebuilt Tour (Task 8.6 Fixture)",
			"Synthetic Legacy Tour (Task 8.6 Fixture)",
			"REBUILT PAGE BY PAGE",
			"THE REST OF THE SERIES",
			"5 PAGES",
		} {
			if !strings.Contains(index.Body.String(), expected) {
				t.Errorf("index missing %q", expected)
			}
		}

		for address, marker := range map[int]string{
			1: "Synthetic Rebuilt Tour (Task 8.6 Fixture)",
			2: "Synthetic Text Page",
			3: "Synthetic Picture Page",
			4: "Synthetic Index Page",
			5: "Sources",
		} {
			rec := get(fmt.Sprintf("/tours/%s/%d", task86RebuiltSlug, address))
			if rec.Code != http.StatusOK {
				t.Errorf("address %d status=%d, want 200", address, rec.Code)
			}
			if !strings.Contains(rec.Body.String(), marker) {
				t.Errorf("address %d missing %q", address, marker)
			}
		}
		if rec := get("/tours/" + task86RebuiltSlug + "/6"); rec.Code != http.StatusNotFound {
			t.Errorf("beyond-sources status=%d, want 404", rec.Code)
		}

		legacy := get("/tours/" + task86LegacySlug)
		if legacy.Code != http.StatusOK || !strings.Contains(legacy.Body.String(), "Open the original tour") {
			t.Errorf("legacy tour status=%d body=%q", legacy.Code, legacy.Body.String())
		}
		if legacy.Body.String() == "" || !strings.Contains(legacy.Body.String(), "https://example.org/synthetic-legacy-original") {
			t.Errorf("legacy tour missing safe destination")
		}

		legacyRedirect := get("/tours/source/synthetic-text.html")
		if legacyRedirect.Code != http.StatusMovedPermanently {
			t.Errorf("legacy redirect status=%d, want 301", legacyRedirect.Code)
		}
		if got := legacyRedirect.Header().Get("Location"); got != "/tours/"+task86RebuiltSlug+"/2" {
			t.Errorf("legacy Location=%q", got)
		}

		asset := get("/assets/css/style.css")
		if asset.Code != http.StatusOK {
			t.Errorf("static asset status=%d, want 200", asset.Code)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

// TestTask86PlaywrightHarness is the disposable HTTP fixture for the external
// Playwright suite. It skips unless WGA_TASK86_HARNESS=1 so the ordinary
// go test ./... run never blocks.
func TestTask86PlaywrightHarness(t *testing.T) {
	if os.Getenv("WGA_TASK86_HARNESS") == "" {
		t.Skip("disposable Playwright harness; run with WGA_TASK86_HARNESS=1")
	}
	app := newTask86FixtureApp(t)
	seedTask86Tours(t, app)
	serve := registerTask86Mux(t, app)

	if err := app.OnServe().Trigger(serve, func(event *core.ServeEvent) error {
		mux, err := event.Router.BuildMux()
		if err != nil {
			return err
		}
		port := os.Getenv("WGA_TASK86_PORT")
		if port == "" {
			port = "8090"
		}
		addr := net.JoinHostPort("127.0.0.1", port)
		listener, err := net.Listen("tcp", addr)
		if err != nil {
			return fmt.Errorf("task 8.6 harness bind %s: %w", addr, err)
		}
		server := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
		fmt.Fprintf(os.Stderr, "WGA_TASK86_HARNESS_READY http://%s\n", addr)
		t.Logf("task 8.6 tour harness listening on http://%s", addr)
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
