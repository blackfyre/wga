package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/handlers/artists"
	"github.com/blackfyre/wga/internal/handlers/landing"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// trustedHeadRouterMarker is a benign, unique marker used to prove that the
// middleware-provided trusted head markup survives the full-document render
// path. It is an inert HTML comment and never executes.
const trustedHeadRouterMarker = `<!--wga-trusted-head-router-marker-->`

// newTrustedHeadRouterTestApp bootstraps a disposable PocketBase app with the
// collections the registered feature routes need, plus the strings collection
// the header-markup middleware reads. No feature records are required: the
// homepage and artist index render their empty states.
func newTrustedHeadRouterTestApp(t *testing.T) *pocketbase.PocketBase {
	t.Helper()

	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap test app: %v", err)
	}
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Errorf("reset test app: %v", err)
		}
	})

	strings := core.NewBaseCollection("Strings")
	strings.Id = "strings"
	strings.MarkAsNew()
	strings.Fields.Add(
		&core.TextField{Id: "strings_name", Name: "name", Required: true},
		&core.EditorField{Id: "strings_content", Name: "content"},
	)
	if err := app.Save(strings); err != nil {
		t.Fatalf("save strings collection: %v", err)
	}

	schools := core.NewBaseCollection("Schools")
	schools.Id = "schools"
	schools.MarkAsNew()
	schools.Fields.Add(
		&core.TextField{Id: "school_name", Name: "name", Required: true},
		&core.TextField{Id: "school_slug", Name: "slug"},
	)
	if err := app.Save(schools); err != nil {
		t.Fatalf("save schools collection: %v", err)
	}

	artPeriods := core.NewBaseCollection("Art_periods")
	artPeriods.Id = "art_periods"
	artPeriods.MarkAsNew()
	artPeriods.Fields.Add(
		&core.TextField{Id: "period_name", Name: "name", Required: true},
		&core.NumberField{Id: "period_start", Name: "start"},
		&core.NumberField{Id: "period_end", Name: "end"},
	)
	if err := app.Save(artPeriods); err != nil {
		t.Fatalf("save art_periods collection: %v", err)
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
		t.Fatalf("save artists collection: %v", err)
	}

	artworks := core.NewBaseCollection("Artworks")
	artworks.Id = "artworks"
	artworks.MarkAsNew()
	artworks.Fields.Add(
		&core.TextField{Id: "artwork_title", Name: "title", Required: true},
		&core.TextField{Id: "artwork_year", Name: "year"},
		&core.TextField{Id: "artwork_image", Name: "image"},
		&core.NumberField{Id: "artwork_image_width", Name: "image_width"},
		&core.DateField{Id: "artwork_created", Name: "created"},
		&core.RelationField{Id: "artwork_author", Name: "author", CollectionId: artists.Id, MinSelect: 1, MaxSelect: 10},
		&core.BoolField{Id: "artwork_published", Name: "published"},
	)
	if err := app.Save(artworks); err != nil {
		t.Fatalf("save artworks collection: %v", err)
	}

	return app
}

// assertMarkerOnceBeforeHead verifies the trusted marker rendered exactly once
// and only before the closing head tag.
func assertMarkerOnceBeforeHead(t *testing.T, body string) {
	t.Helper()

	headEnd := strings.Index(body, "</head>")
	if headEnd == -1 {
		t.Fatalf("full document is missing </head>")
	}

	if count := strings.Count(body, trustedHeadRouterMarker); count != 1 {
		t.Fatalf("trusted marker rendered %d times, want exactly 1", count)
	}

	if markerIndex := strings.Index(body, trustedHeadRouterMarker); markerIndex == -1 || markerIndex > headEnd {
		t.Fatalf("trusted marker not rendered before </head>")
	}
}

// TestFullPageRoutesRenderTrustedHeadMarkup proves that the real homepage and
// artist-index routes carry the middleware-provided trusted markup through
// their full-document layout, and that an HTMX fragment omits it.
func TestFullPageRoutesRenderTrustedHeadMarkup(t *testing.T) {
	app := newTrustedHeadRouterTestApp(t)
	createStringRecord(t, app, "scripts:header", trustedHeadRouterMarker)

	registerTrustedHeadMarkupMiddleware(app)
	landing.RegisterHandlers(app)
	artists.RegisterHandlers(app, config.EnvironmentTest)

	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatalf("create router: %v", err)
	}
	serveEvent := &core.ServeEvent{App: app, Router: router}

	if err := app.OnServe().Trigger(serveEvent, func(se *core.ServeEvent) error {
		mux, err := se.Router.BuildMux()
		if err != nil {
			return err
		}

		serve := func(path string, htmx bool) *httptest.ResponseRecorder {
			request := httptest.NewRequest(http.MethodGet, path, nil)
			request.Header.Set("Accept", "text/html")
			if htmx {
				request.Header.Set("HX-Request", "true")
			}

			recorder := httptest.NewRecorder()
			mux.ServeHTTP(recorder, request)

			return recorder
		}

		homepage := serve("/", false)
		if homepage.Code != http.StatusOK {
			t.Errorf("homepage status = %d, want %d", homepage.Code, http.StatusOK)
		} else {
			assertMarkerOnceBeforeHead(t, homepage.Body.String())
		}

		artistIndex := serve("/artists", false)
		if artistIndex.Code != http.StatusOK {
			t.Errorf("artist index status = %d, want %d", artistIndex.Code, http.StatusOK)
		} else {
			assertMarkerOnceBeforeHead(t, artistIndex.Body.String())
		}

		fragment := serve("/artists", true)
		if fragment.Code != http.StatusOK {
			t.Errorf("artist index fragment status = %d, want %d", fragment.Code, http.StatusOK)
		} else {
			body := fragment.Body.String()
			if strings.Contains(body, trustedHeadRouterMarker) {
				t.Errorf("HTMX fragment must not contain the trusted marker")
			}
			if strings.Contains(body, "<html") {
				t.Errorf("HTMX fragment must not render the full document")
			}
		}

		return nil
	}); err != nil {
		t.Fatalf("trigger serve event: %v", err)
	}
}
