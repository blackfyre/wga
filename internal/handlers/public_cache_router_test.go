package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/handlers/artists"
	"github.com/blackfyre/wga/internal/handlers/landing"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// TestSharedCatalogueDocumentsCarryNoVisitorState proves that the real
// homepage and artist index, served through the shared-cache middleware, are
// publicly cacheable only while they carry no cookie, synchroniser token, or
// itinerary tray content, for both GET and HEAD.
func TestSharedCatalogueDocumentsCarryNoVisitorState(t *testing.T) {
	app := newTrustedHeadRouterTestApp(t)

	registerPublicCatalogueCache(app)
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

		for _, method := range []string{http.MethodGet, http.MethodHead} {
			for _, path := range []string{"/", "/artists"} {
				request := httptest.NewRequest(method, path, nil)
				request.Header.Set("Accept", "text/html")
				response := httptest.NewRecorder()
				mux.ServeHTTP(response, request)

				if response.Code != http.StatusOK {
					t.Errorf("%s %s status = %d, want 200", method, path, response.Code)
					continue
				}
				if got := response.Header().Get("Cache-Control"); got != publicCatalogueCacheControl {
					t.Errorf("%s %s Cache-Control = %q, want %q", method, path, got, publicCatalogueCacheControl)
				}
				if cookies := response.Result().Cookies(); len(cookies) != 0 {
					t.Errorf("%s %s set cookies on a shared document: %v", method, path, cookies)
				}
				if method == http.MethodHead {
					continue
				}
				body := response.Body.String()
				if strings.Contains(body, `data-itinerary-csrf="`) || strings.Contains(body, `role="region" aria-label="Itinerary draft"`) {
					t.Errorf("%s %s rendered visitor itinerary state", method, path)
				}
				// The background tray load must not inherit the body's main-area
				// target or progress indicator, nor run a document view transition.
				for _, attribute := range []string{
					`hx-get="/itineraries/session"`,
					`hx-target="this"`,
					`hx-indicator="#itinerary-tray"`,
					`hx-swap="outerHTML transition:false"`,
				} {
					if !strings.Contains(body, attribute) {
						t.Errorf("%s %s session tray placeholder lacks %s", method, path, attribute)
					}
				}
			}
		}

		return nil
	}); err != nil {
		t.Fatalf("trigger serve event: %v", err)
	}
}
