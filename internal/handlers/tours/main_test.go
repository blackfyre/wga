package tours

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func TestRegisteredTourRoutesRenderEmptyIndexAndDenyMissingTour(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = app.ResetBootstrapState() })

	editors := core.NewBaseCollection("Guided_tour_editors")
	editors.Id = "guided_tour_editors"
	editors.MarkAsNew()
	editors.Fields.Add(&core.TextField{Name: "editor_key"}, &core.TextField{Name: "name"})
	if err := app.Save(editors); err != nil {
		t.Fatal(err)
	}
	tourCollection := core.NewBaseCollection("Guided_tours")
	tourCollection.Id = "guided_tours"
	tourCollection.MarkAsNew()
	tourCollection.Fields.Add(&core.TextField{Name: "slug"}, &core.TextField{Name: "title"}, &core.TextField{Name: "kind"}, &core.TextField{Name: "publication_status"}, &core.NumberField{Name: "series_position"}, &core.RelationField{Name: "editor", CollectionId: editors.Id, MaxSelect: 1})
	if err := app.Save(tourCollection); err != nil {
		t.Fatal(err)
	}
	revisions := core.NewBaseCollection("Guided_tour_revisions")
	revisions.Id = "guided_tour_revisions"
	revisions.MarkAsNew()
	revisions.Fields.Add(&core.RelationField{Name: "tour", CollectionId: tourCollection.Id, MaxSelect: 1})
	if err := app.Save(revisions); err != nil {
		t.Fatal(err)
	}
	tourCollection.Fields.Add(&core.RelationField{Name: "published_revision", CollectionId: revisions.Id, MaxSelect: 1})
	if err := app.Save(tourCollection); err != nil {
		t.Fatal(err)
	}

	RegisterHandlers(app)
	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	serve := &core.ServeEvent{App: app, Router: router}
	if err := app.OnServe().Trigger(serve, func(event *core.ServeEvent) error {
		mux, err := event.Router.BuildMux()
		if err != nil {
			return err
		}
		index := httptest.NewRecorder()
		mux.ServeHTTP(index, httptest.NewRequest(http.MethodGet, "/tours", nil))
		if index.Code != http.StatusOK {
			t.Fatalf("index status=%d", index.Code)
		}
		for _, expected := range []string{"Guided Tours", "No published tours yet", "Survey", "Artist", "Site", "Theme"} {
			if !strings.Contains(index.Body.String(), expected) {
				t.Errorf("index missing %q", expected)
			}
		}
		if !strings.Contains(index.Body.String(), `id="mc-area"`) {
			t.Errorf("full index response missing shared shell #mc-area")
		}

		htmxFragment := httptest.NewRecorder()
		fragmentReq := httptest.NewRequest(http.MethodGet, "/tours?kind=survey", nil)
		fragmentReq.Header.Set("HX-Request", "true")
		fragmentReq.Header.Set("HX-Target", "tours")
		mux.ServeHTTP(htmxFragment, fragmentReq)
		if htmxFragment.Code != http.StatusOK {
			t.Fatalf("htmx fragment status=%d", htmxFragment.Code)
		}
		if strings.Contains(htmxFragment.Body.String(), `id="mc-area"`) {
			t.Errorf("feature-local fragment must not carry #mc-area")
		}
		if !strings.Contains(htmxFragment.Body.String(), `id="tours"`) {
			t.Errorf("feature-local fragment missing #tours")
		}
		if got := htmxFragment.Header().Get("HX-Push-Url"); got != "/tours?kind=survey" {
			t.Errorf("HX-Push-Url = %q, want /tours?kind=survey", got)
		}

		htmxShell := httptest.NewRecorder()
		shellReq := httptest.NewRequest(http.MethodGet, "/tours", nil)
		shellReq.Header.Set("HX-Request", "true")
		shellReq.Header.Set("HX-Target", "mc-area")
		mux.ServeHTTP(htmxShell, shellReq)
		if htmxShell.Code != http.StatusOK {
			t.Fatalf("htmx shell status=%d", htmxShell.Code)
		}
		if !strings.Contains(htmxShell.Body.String(), `id="mc-area"`) {
			t.Errorf("shell navigation must return the full #mc-area page")
		}

		missing := httptest.NewRecorder()
		mux.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/tours/missing", nil))
		if missing.Code != http.StatusNotFound {
			t.Errorf("missing status=%d, want 404", missing.Code)
		}
		invalid := httptest.NewRecorder()
		mux.ServeHTTP(invalid, httptest.NewRequest(http.MethodGet, "/tours/missing/not-a-page", nil))
		if invalid.Code != http.StatusNotFound {
			t.Errorf("invalid address status=%d, want 404", invalid.Code)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestLegacyTourRoutesRedirectToCanonicalAddress(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = app.ResetBootstrapState() })

	editors := core.NewBaseCollection("Guided_tour_editors")
	editors.Id = "guided_tour_editors"
	editors.MarkAsNew()
	editors.Fields.Add(&core.TextField{Name: "editor_key"}, &core.TextField{Name: "name"})
	if err := app.Save(editors); err != nil {
		t.Fatal(err)
	}
	tours := core.NewBaseCollection("Guided_tours")
	tours.Id = "guided_tours"
	tours.MarkAsNew()
	tours.Fields.Add(&core.TextField{Name: "slug"}, &core.TextField{Name: "title"}, &core.TextField{Name: "kind"},
		&core.TextField{Name: "publication_status"}, &core.NumberField{Name: "series_position"}, &core.TextField{Name: "tour_number"},
		&core.RelationField{Name: "editor", CollectionId: editors.Id, MaxSelect: 1})
	if err := app.Save(tours); err != nil {
		t.Fatal(err)
	}
	revisions := core.NewBaseCollection("Guided_tour_revisions")
	revisions.Id = "guided_tour_revisions"
	revisions.MarkAsNew()
	revisions.Fields.Add(&core.RelationField{Name: "tour", CollectionId: tours.Id, MaxSelect: 1})
	if err := app.Save(revisions); err != nil {
		t.Fatal(err)
	}
	tours.Fields.Add(&core.RelationField{Name: "published_revision", CollectionId: revisions.Id, MaxSelect: 1})
	if err := app.Save(tours); err != nil {
		t.Fatal(err)
	}
	pages := core.NewBaseCollection("Guided_tour_pages")
	pages.Id = "guided_tour_pages"
	pages.MarkAsNew()
	pages.Fields.Add(&core.RelationField{Name: "tour", CollectionId: tours.Id, MaxSelect: 1}, &core.NumberField{Name: "page_position"})
	if err := app.Save(pages); err != nil {
		t.Fatal(err)
	}
	legacy := core.NewBaseCollection("Guided_tour_legacy_routes")
	legacy.Id = "guided_tour_legacy_routes"
	legacy.MarkAsNew()
	legacy.Fields.Add(&core.TextField{Name: "legacy_path"}, &core.RelationField{Name: "tour_page", CollectionId: pages.Id, MaxSelect: 1})
	if err := app.Save(legacy); err != nil {
		t.Fatal(err)
	}

	editor := core.NewRecord(editors)
	editor.Set("editor_key", "e")
	editor.Set("name", "Editor")
	if err := app.Save(editor); err != nil {
		t.Fatal(err)
	}
	tour := core.NewRecord(tours)
	tour.Set("slug", "legacy-tour")
	tour.Set("title", "Legacy Tour")
	tour.Set("kind", "survey")
	tour.Set("publication_status", "published")
	tour.Set("series_position", 1)
	tour.Set("tour_number", "6a")
	tour.Set("editor", editor.Id)
	if err := app.Save(tour); err != nil {
		t.Fatal(err)
	}
	revision := core.NewRecord(revisions)
	revision.Set("tour", tour.Id)
	if err := app.Save(revision); err != nil {
		t.Fatal(err)
	}
	tour.Set("published_revision", revision.Id)
	if err := app.Save(tour); err != nil {
		t.Fatal(err)
	}
	page := core.NewRecord(pages)
	page.Set("tour", tour.Id)
	page.Set("page_position", 1)
	if err := app.Save(page); err != nil {
		t.Fatal(err)
	}
	legacyRecord := core.NewRecord(legacy)
	legacyRecord.Set("legacy_path", "/tours/source/text.html")
	legacyRecord.Set("tour_page", page.Id)
	if err := app.Save(legacyRecord); err != nil {
		t.Fatal(err)
	}
	tourLegacyRecord := core.NewRecord(legacy)
	tourLegacyRecord.Set("legacy_path", "/tour/example/title.html")
	tourLegacyRecord.Set("tour_page", page.Id)
	if err := app.Save(tourLegacyRecord); err != nil {
		t.Fatal(err)
	}

	RegisterHandlers(app)
	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	serve := &core.ServeEvent{App: app, Router: router}
	if err := app.OnServe().Trigger(serve, func(event *core.ServeEvent) error {
		mux, err := event.Router.BuildMux()
		if err != nil {
			return err
		}
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/tours/source/text.html", nil))
		if rec.Code != http.StatusMovedPermanently {
			t.Fatalf("legacy status=%d, want 301", rec.Code)
		}
		if got := rec.Header().Get("Location"); got != "/tours/legacy-tour/2" {
			t.Fatalf("legacy Location=%q, want /tours/legacy-tour/2", got)
		}

		tourRec := httptest.NewRecorder()
		mux.ServeHTTP(tourRec, httptest.NewRequest(http.MethodGet, "/tour/example/title.html", nil))
		if tourRec.Code != http.StatusMovedPermanently {
			t.Fatalf("/tour legacy status=%d, want 301", tourRec.Code)
		}
		if got := tourRec.Header().Get("Location"); got != "/tours/legacy-tour/2" {
			t.Fatalf("/tour legacy Location=%q, want /tours/legacy-tour/2", got)
		}

		missing := httptest.NewRecorder()
		mux.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/tours/source/missing.html", nil))
		if missing.Code != http.StatusNotFound {
			t.Errorf("missing legacy status=%d, want 404", missing.Code)
		}

		crossOrigin := httptest.NewRecorder()
		mux.ServeHTTP(crossOrigin, httptest.NewRequest(http.MethodGet, "/\\evil.example", nil))
		if crossOrigin.Code != http.StatusNotFound {
			t.Errorf("cross-origin legacy status=%d, want 404 (no open redirect)", crossOrigin.Code)
		}
		if loc := crossOrigin.Header().Get("Location"); loc != "" {
			t.Errorf("cross-origin legacy Location=%q, want none", loc)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
