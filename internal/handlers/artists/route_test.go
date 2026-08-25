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

func createArtistIndexCollections(t *testing.T, app *pocketbase.PocketBase) {
	t.Helper()

	schools := core.NewBaseCollection("Schools")
	schools.Id = "schools"
	schools.MarkAsNew()
	schools.Fields.Add(
		&core.TextField{Id: "school_name", Name: "name", Required: true},
		&core.TextField{Id: "school_slug", Name: "slug"},
	)
	if err := app.Save(schools); err != nil {
		t.Fatalf("save schools: %v", err)
	}

	periods := core.NewBaseCollection("Art_periods")
	periods.Id = "art_periods"
	periods.MarkAsNew()
	periods.Fields.Add(
		&core.TextField{Id: "period_name", Name: "name", Required: true},
		&core.NumberField{Id: "period_start", Name: "start"},
		&core.NumberField{Id: "period_end", Name: "end"},
	)
	if err := app.Save(periods); err != nil {
		t.Fatalf("save art periods: %v", err)
	}

	artists := core.NewBaseCollection("Artists")
	artists.Id = "artists"
	artists.MarkAsNew()
	artists.Fields.Add(
		&core.TextField{Id: "artist_name", Name: "name", Required: true},
		&core.TextField{Id: "artist_slug", Name: "slug"},
		&core.NumberField{Id: "artist_yob", Name: "year_of_birth"},
		&core.NumberField{Id: "artist_yod", Name: "year_of_death"},
		&core.TextField{Id: "artist_profession", Name: "profession"},
		&core.TextField{Id: "artist_portrait", Name: "portrait"},
		&core.NumberField{Id: "artist_portrait_width", Name: "biography_image_width"},
		&core.TextField{Id: "artist_pob", Name: "place_of_birth"},
		&core.TextField{Id: "artist_pod", Name: "place_of_death"},
		&core.EditorField{Id: "artist_bio", Name: "bio"},
		&core.RelationField{Id: "artist_school", Name: "school", CollectionId: schools.Id, MinSelect: 0, MaxSelect: 10},
		&core.BoolField{Id: "artist_published", Name: "published"},
	)
	if err := app.Save(artists); err != nil {
		t.Fatalf("save artists: %v", err)
	}

	artworks := core.NewBaseCollection("Artworks")
	artworks.Id = "artworks"
	artworks.MarkAsNew()
	artworks.Fields.Add(
		&core.TextField{Id: "artwork_title", Name: "title", Required: true},
		&core.RelationField{Id: "artwork_author", Name: "author", CollectionId: artists.Id, MinSelect: 1, MaxSelect: 10},
		&core.BoolField{Id: "artwork_published", Name: "published"},
	)
	if err := app.Save(artworks); err != nil {
		t.Fatalf("save artworks: %v", err)
	}
}

func saveIndexRecord(t *testing.T, app *pocketbase.PocketBase, collection string, id string, fields map[string]any) {
	t.Helper()

	coll, err := app.FindCollectionByNameOrId(collection)
	if err != nil {
		t.Fatalf("find %s: %v", collection, err)
	}
	record := core.NewRecord(coll)
	record.Id = id
	for key, value := range fields {
		record.Set(key, value)
	}
	if err := app.Save(record); err != nil {
		t.Fatalf("save %s %s: %v", collection, id, err)
	}
}

func TestArtistIndexRouteRendersFullAndHTMX(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Errorf("reset: %v", err)
		}
	})

	createArtistIndexCollections(t, app)
	saveIndexRecord(t, app, "schools", "schooldutch0001", map[string]any{"name": "Dutch", "slug": "dutch"})
	saveIndexRecord(t, app, "art_periods", "periodbaroque01", map[string]any{"name": "Baroque", "start": 1600, "end": 1750})
	saveIndexRecord(t, app, "artists", "artistone000001", map[string]any{
		"name": "Synthetic Artist", "slug": "synthetic-artist", "year_of_birth": 1606,
		"year_of_death": 1669, "profession": "painter", "school": []string{"schooldutch0001"}, "published": true,
	})
	saveIndexRecord(t, app, "artists", "artisttwo000001", map[string]any{
		"name": "Hidden Artist", "slug": "hidden-artist", "year_of_birth": 1606, "published": false,
	})
	saveIndexRecord(t, app, "artworks", "artworkone00001", map[string]any{
		"title": "Work", "author": []string{"artistone000001"}, "published": true,
	})

	RegisterHandlers(app, config.EnvironmentTest)

	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatalf("create router: %v", err)
	}
	serveEvent := &core.ServeEvent{App: app, Router: router}
	if err := app.OnServe().Trigger(serveEvent, func(event *core.ServeEvent) error {
		mux, err := event.Router.BuildMux()
		if err != nil {
			return err
		}

		full := httptest.NewRecorder()
		mux.ServeHTTP(full, httptest.NewRequest(http.MethodGet, "/artists?letter=S", nil))
		if full.Code != http.StatusOK {
			t.Errorf("full status = %d, want 200", full.Code)
		}
		if !strings.Contains(full.Body.String(), "<html") || !strings.Contains(full.Body.String(), "Synthetic Artist") || strings.Contains(full.Body.String(), "Hidden Artist") {
			t.Error("full response did not render published artists only")
		}
		if got := full.Header().Get("HX-Push-Url"); got != "/artists?letter=S" {
			t.Errorf("HX-Push-Url = %q, want /artists?letter=S", got)
		}

		partial := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/artists?letter=S&view=list", nil)
		request.Header.Set("HX-Request", "true")
		mux.ServeHTTP(partial, request)
		if partial.Code != http.StatusOK {
			t.Errorf("partial status = %d, want 200", partial.Code)
		}
		if strings.Contains(partial.Body.String(), "<html") {
			t.Error("HTMX response should not render the full document")
		}
		if !strings.Contains(partial.Body.String(), "NAME") || !strings.Contains(partial.Body.String(), "FORM") {
			t.Error("HTMX response should render the list table")
		}
		if got := partial.Header().Get("HX-Push-Url"); got != "/artists?letter=S&view=list" {
			t.Errorf("HX-Push-Url = %q, want /artists?letter=S&view=list", got)
		}

		return nil
	}); err != nil {
		t.Fatalf("trigger serve event: %v", err)
	}
}

func TestArtistIndexRouteSelectsTargetAwareResponse(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Errorf("reset: %v", err)
		}
	})

	createArtistIndexCollections(t, app)
	saveIndexRecord(t, app, "schools", "schooldutch0001", map[string]any{"name": "Dutch", "slug": "dutch"})
	saveIndexRecord(t, app, "art_periods", "periodbaroque01", map[string]any{"name": "Baroque", "start": 1600, "end": 1750})
	saveIndexRecord(t, app, "artists", "artistone000001", map[string]any{
		"name": "Synthetic Artist", "slug": "synthetic-artist", "year_of_birth": 1606,
		"year_of_death": 1669, "profession": "painter", "school": []string{"schooldutch0001"}, "published": true,
	})

	RegisterHandlers(app, config.EnvironmentTest)

	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatalf("create router: %v", err)
	}
	serveEvent := &core.ServeEvent{App: app, Router: router}
	if err := app.OnServe().Trigger(serveEvent, func(event *core.ServeEvent) error {
		mux, err := event.Router.BuildMux()
		if err != nil {
			return err
		}

		full := httptest.NewRecorder()
		mux.ServeHTTP(full, httptest.NewRequest(http.MethodGet, "/artists?letter=S", nil))
		if full.Code != http.StatusOK {
			t.Errorf("full status = %d, want %d", full.Code, http.StatusOK)
		}
		if !strings.Contains(full.Body.String(), "<html") {
			t.Error("full response should render the full document")
		}
		if got := strings.Count(full.Body.String(), `id="mc-area"`); got != 1 {
			t.Errorf("full response rendered %d #mc-area elements, want exactly 1", got)
		}

		for _, target := range []string{"mc-area", "#mc-area"} {
			shell := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/artists?letter=S", nil)
			request.Header.Set("HX-Request", "true")
			request.Header.Set("HX-Target", target)
			mux.ServeHTTP(shell, request)
			if shell.Code != http.StatusOK {
				t.Errorf("shell(%s) status = %d, want %d", target, shell.Code, http.StatusOK)
			}
			if !strings.Contains(shell.Body.String(), "<html") {
				t.Errorf("shell(%s) must render the full document", target)
			}
			if got := strings.Count(shell.Body.String(), `id="mc-area"`); got != 1 {
				t.Errorf("shell(%s) rendered %d #mc-area elements, want exactly 1", target, got)
			}
		}

		local := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/artists?letter=S", nil)
		request.Header.Set("HX-Request", "true")
		request.Header.Set("HX-Target", "artists")
		mux.ServeHTTP(local, request)
		if local.Code != http.StatusOK {
			t.Errorf("local status = %d, want %d", local.Code, http.StatusOK)
		}
		if strings.Contains(local.Body.String(), "<html") {
			t.Error("feature-local response should not render the full document")
		}
		if got := strings.Count(local.Body.String(), `id="artists"`); got != 1 {
			t.Errorf("feature-local response rendered %d #artists elements, want exactly 1", got)
		}
		if strings.Contains(local.Body.String(), `id="mc-area"`) {
			t.Error("feature-local response must not carry #mc-area")
		}

		return nil
	}); err != nil {
		t.Fatalf("trigger serve event: %v", err)
	}
}

func TestArtistIndexRouteNormalisesCanonicalUrl(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Errorf("reset: %v", err)
		}
	})

	createArtistIndexCollections(t, app)
	saveIndexRecord(t, app, "artists", "artistone000001", map[string]any{
		"name": "Synthetic Artist", "slug": "synthetic-artist", "published": true,
	})

	RegisterHandlers(app, config.EnvironmentTest)

	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatalf("create router: %v", err)
	}
	serveEvent := &core.ServeEvent{App: app, Router: router}
	if err := app.OnServe().Trigger(serveEvent, func(event *core.ServeEvent) error {
		mux, err := event.Router.BuildMux()
		if err != nil {
			return err
		}

		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/artists?letter=zz&page=999&born_from=abc&view=carousel&sort=desc", nil))

		if got := recorder.Header().Get("HX-Push-Url"); got != "/artists" {
			t.Errorf("HX-Push-Url = %q, want canonical /artists", got)
		}

		return nil
	}); err != nil {
		t.Fatalf("trigger serve event: %v", err)
	}
}
