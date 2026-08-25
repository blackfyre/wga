package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
	"github.com/blackfyre/wga/internal/handlers/itineraries"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func TestEligibleForItineraryProjection(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		htmx   bool
		accept string
		want   bool
	}{
		{name: "full html get", method: http.MethodGet, path: "/artworks", accept: "text/html", want: true},
		{name: "head is eligible", method: http.MethodHead, path: "/", accept: "text/html", want: true},
		{name: "htmx get fragment included", method: http.MethodGet, path: "/artworks", htmx: true, accept: "text/html", want: true},
		{name: "htmx get fragment without html accept still included", method: http.MethodGet, path: "/timeline", htmx: true, accept: "*/*", want: true},
		{name: "post skipped", method: http.MethodPost, path: "/", accept: "text/html", want: false},
		{name: "put skipped", method: http.MethodPut, path: "/", accept: "text/html", want: false},
		{name: "api boundary skipped", method: http.MethodGet, path: "/api/collections/artworks", accept: "text/html", want: false},
		{name: "api htmx skipped", method: http.MethodGet, path: "/api/collections/artworks", htmx: true, want: false},
		{name: "admin boundary skipped", method: http.MethodGet, path: "/_/", accept: "text/html", want: false},
		{name: "assets boundary skipped", method: http.MethodGet, path: "/assets/js/app.js", accept: "text/html", want: false},
		{name: "assets htmx skipped", method: http.MethodGet, path: "/assets/js/app.js", htmx: true, want: false},
		{name: "sitemap boundary skipped", method: http.MethodGet, path: "/sitemap", accept: "text/html", want: false},
		{name: "non-html accept skipped", method: http.MethodGet, path: "/artworks", accept: "application/json", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.path, nil)
			request.Header.Set("Accept", test.accept)
			if test.htmx {
				request.Header.Set("HX-Request", "true")
			}

			if got := eligibleForItineraryProjection(request); got != test.want {
				t.Fatalf("eligible = %t, want %t", got, test.want)
			}
		})
	}
}

func TestEligibleForItineraryProjectionNilRequest(t *testing.T) {
	if eligibleForItineraryProjection(nil) {
		t.Fatal("nil request must not be eligible")
	}
}

func TestProjectionMiddlewareIntegration(t *testing.T) {
	app := newProjectionMiddlewareTestApp(t)

	registerItinerarySessionMiddleware(app, itineraries.CookiePolicy{Name: "wga_itinerary_dev", Secure: false})

	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatalf("create router: %v", err)
	}

	// Register a probe route that echoes the projected CSRF token so the test
	// can prove the middleware prepared the request context.
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/probe", func(c *core.RequestEvent) error {
			return c.String(http.StatusOK, "csrf="+tmplUtils.GetItineraryCSRF(c.Request.Context()))
		})
		return se.Next()
	})

	serveEvent := &core.ServeEvent{App: app, Router: router}
	if err := app.OnServe().Trigger(serveEvent, func(se *core.ServeEvent) error {
		mux, err := se.Router.BuildMux()
		if err != nil {
			return err
		}

		serve := func(method string, path string, htmx bool) *httptest.ResponseRecorder {
			request := httptest.NewRequest(method, path, nil)
			request.Header.Set("Accept", "text/html")
			if htmx {
				request.Header.Set("HX-Request", "true")
			}
			recorder := httptest.NewRecorder()
			mux.ServeHTTP(recorder, request)
			return recorder
		}

		// First full-page request: cookie issued, CSRF projected, no-store.
		first := serve(http.MethodGet, "/probe", false)
		if first.Code != http.StatusOK {
			t.Fatalf("full status = %d, want 200", first.Code)
		}
		if !strings.HasPrefix(first.Body.String(), "csrf=") || strings.TrimSpace(first.Body.String()) == "csrf=" {
			t.Errorf("full response did not carry a projected CSRF token: %q", first.Body.String())
		}
		if got := first.Header().Get("Cache-Control"); got != "private, no-store" {
			t.Errorf("full Cache-Control = %q, want private, no-store", got)
		}
		cookies := first.Result().Cookies()
		if len(cookies) != 1 || cookies[0].Name != "wga_itinerary_dev" {
			t.Fatalf("full response cookies = %+v, want one wga_itinerary_dev cookie", cookies)
		}

		// HTMX fragment request also carries the projection and no-store.
		fragment := serve(http.MethodGet, "/probe", true)
		if fragment.Code != http.StatusOK {
			t.Fatalf("htmx status = %d, want 200", fragment.Code)
		}
		if !strings.HasPrefix(fragment.Body.String(), "csrf=") || strings.TrimSpace(fragment.Body.String()) == "csrf=" {
			t.Errorf("htmx response did not carry a projected CSRF token: %q", fragment.Body.String())
		}
		if got := fragment.Header().Get("Cache-Control"); got != "private, no-store" {
			t.Errorf("htmx Cache-Control = %q, want private, no-store", got)
		}

		// POST is never projected and never sets a cookie.
		post := serve(http.MethodPost, "/probe", false)
		if strings.HasPrefix(post.Body.String(), "csrf=") {
			t.Errorf("POST response unexpectedly projected a CSRF token: %q", post.Body.String())
		}
		if len(post.Result().Cookies()) != 0 {
			t.Errorf("POST response set cookies: %+v", post.Result().Cookies())
		}

		return nil
	}); err != nil {
		t.Fatalf("trigger serve event: %v", err)
	}

	// No durable records are created by the cookie-less GET projections.
	records, err := app.FindRecordsByFilter("itineraries", "", "", 0, 0)
	if err != nil {
		t.Fatalf("find itineraries: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("cookie-less GETs allocated %d itinerary records, want 0", len(records))
	}
}

func newProjectionMiddlewareTestApp(t *testing.T) *pocketbase.PocketBase {
	t.Helper()

	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap app: %v", err)
	}
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Errorf("reset app: %v", err)
		}
	})

	itineraries := core.NewBaseCollection("Itineraries")
	itineraries.Id = "itineraries"
	itineraries.MarkAsNew()
	itineraries.Fields.Add(
		&core.TextField{Name: "owner"},
		&core.SelectField{Name: "status", Values: []string{"draft", "pending", "approved", "rejected"}, MaxSelect: 1},
	)
	if err := app.Save(itineraries); err != nil {
		t.Fatalf("create itineraries collection: %v", err)
	}

	return app
}
