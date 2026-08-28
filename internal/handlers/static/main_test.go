package static

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/handlers/landing"
	"github.com/blackfyre/wga/internal/testutils"
	apputils "github.com/blackfyre/wga/internal/utils"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
)

func newStaticTestApp(t testing.TB) *tests.TestApp {
	t.Helper()

	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("create test app: %v", err)
	}
	app.Settings().Logs.MaxDays = 1

	collection := core.NewBaseCollection("Static_pages")
	collection.Id = "static_pages"
	collection.MarkAsNew()
	collection.Fields.Add(
		&core.TextField{Id: "sp_title", Name: "title", Required: true},
		&core.TextField{Id: "sp_slug", Name: "slug"},
		&core.EditorField{Id: "sp_content", Name: "content"},
	)
	if err := app.Save(collection); err != nil {
		t.Fatalf("save static_pages collection: %v", err)
	}

	return app
}

func TestAssetCacheControl(t *testing.T) {
	cases := map[string]string{
		"js/app.js":              "no-cache",
		"js/bootstrap-abc123.js": "public, max-age=31536000, immutable",
		"css/style.css":           "",
	}
	for path, want := range cases {
		if got := assetCacheControl(path); got != want {
			t.Errorf("assetCacheControl(%q) = %q, want %q", path, got, want)
		}
	}
}

func createStaticPage(t testing.TB, app core.App, slug, title, content string) {
	t.Helper()

	collection, err := app.FindCollectionByNameOrId("static_pages")
	if err != nil {
		t.Fatalf("find static_pages collection: %v", err)
	}
	record := core.NewRecord(collection)
	record.Set("title", title)
	record.Set("slug", slug)
	record.Set("content", content)
	if err := app.Save(record); err != nil {
		t.Fatalf("save static page: %v", err)
	}
}

// registerFakeHome registers a bare "GET /" route, mirroring the landing home
// route's ServeMux pattern, so route tests can prove the public fallback
// against the same catch-all shadowing the real application has.
func registerFakeHome(app core.App) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/", func(c *core.RequestEvent) error {
			return c.HTML(http.StatusOK, "HOME")
		})
		return se.Next()
	})
}

// newFullAppTestApp creates a real PocketBase app with the collections the
// landing home route and static pages require, for full-application routing
// tests against the actual landing "GET /" registration.
func newFullAppTestApp(t testing.TB) *pocketbase.PocketBase {
	t.Helper()

	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap full app: %v", err)
	}
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Errorf("reset full app: %v", err)
		}
	})

	staticPages := core.NewBaseCollection("Static_pages")
	staticPages.Id = "static_pages"
	staticPages.MarkAsNew()
	staticPages.Fields.Add(
		&core.TextField{Id: "sp_title", Name: "title", Required: true},
		&core.TextField{Id: "sp_slug", Name: "slug"},
		&core.EditorField{Id: "sp_content", Name: "content"},
	)
	if err := app.Save(staticPages); err != nil {
		t.Fatalf("save static_pages collection: %v", err)
	}

	artists := core.NewBaseCollection("Artists")
	artists.Id = "test_artists"
	artists.MarkAsNew()
	artists.Fields.Add(
		&core.TextField{Id: "artist_name", Name: "name", Required: true},
		&core.BoolField{Id: "artist_published", Name: "published"},
	)
	if err := app.Save(artists); err != nil {
		t.Fatalf("save artists collection: %v", err)
	}

	artworks := core.NewBaseCollection("Artworks")
	artworks.Id = "test_artworks"
	artworks.MarkAsNew()
	artworks.Fields.Add(
		&core.TextField{Id: "artwork_title", Name: "title", Required: true},
		&core.TextField{Id: "artwork_year", Name: "year"},
		&core.TextField{Id: "artwork_image", Name: "image"},
		&core.NumberField{Id: "artwork_image_width", Name: "image_width"},
		&core.DateField{Id: "artwork_created", Name: "created"},
		&core.BoolField{Id: "artwork_published", Name: "published"},
		&core.RelationField{Id: "artwork_author", Name: "author", CollectionId: artists.Id, MinSelect: 1, MaxSelect: 10},
	)
	if err := app.Save(artworks); err != nil {
		t.Fatalf("save artworks collection: %v", err)
	}

	schools := core.NewBaseCollection("Schools")
	schools.Id = "test_schools"
	schools.MarkAsNew()
	if err := app.Save(schools); err != nil {
		t.Fatalf("save schools collection: %v", err)
	}

	return app
}

func configureStaticPublicURL(t testing.TB) {
	t.Helper()

	configuration := config.LoadFrom(func(key string) string {
		return map[string]string{
			"WGA_ENV":                "development",
			"WGA_PROTOCOL":           "https",
			"WGA_HOSTNAME":           "gallery.example",
			"WGA_SENDER_NAME":        "WGA",
			"WGA_SENDER_ADDRESS":     "sender@example.com",
			"WGA_POSTCARD_FREQUENCY": "*/1 * * * *",
		}[key]
	})
	server, err := configuration.Server()
	if err != nil {
		t.Fatalf("load server configuration: %v", err)
	}
	apputils.ConfigurePublicURL(server.PublicURL)
	t.Cleanup(func() { apputils.ConfigurePublicURL(config.PublicURL{}) })
}

func TestStaticPageRouteRendersManagedPageWithMetadataAndContents(t *testing.T) {
	scenario := tests.ApiScenario{
		Name:           "about page renders metadata, kicker, canonical and contents",
		Method:         http.MethodGet,
		URL:            "/pages/about",
		ExpectedStatus: http.StatusOK,
		ExpectedContent: []string{
			"About - WGA",
			"14 — ABOUT THE COLLECTION",
			`<link rel="canonical" href="https://gallery.example/pages/about"`,
			`<h2 id="the-collection">The collection</h2>`,
			"The archive.",
			`aria-label="Contents"`,
			`href="#the-collection"`,
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			configureStaticPublicURL(t)
			app := newStaticTestApp(t)
			RegisterHandlers(app, config.EnvironmentProduction)
			createStaticPage(t, app, "about", "About", `<h2>The collection</h2><p>The archive.</p>`)
			return app
		},
		AfterTestFunc: func(t testing.TB, _ *tests.TestApp, response *http.Response) {
			if got := response.Header.Get("HX-Push-Url"); got != "/pages/about" {
				t.Fatalf("HX-Push-Url = %q, want /pages/about", got)
			}
		},
	}

	scenario.Test(t)
}

func TestStaticPageRouteRendersPrivacyAndOtherRecords(t *testing.T) {
	cases := []struct {
		name      string
		slug      string
		title     string
		kicker    string
		content   string
		fragments []string
	}{
		{
			name:    "privacy policy",
			slug:    "privacy-policy",
			title:   "Privacy policy",
			kicker:  "06 — GENERAL CONTENT",
			content: `<h2>Data we collect</h2><p>Cookies.</p>`,
			fragments: []string{
				`<h2 id="data-we-collect">Data we collect</h2>`,
				"Cookies.",
			},
		},
		{
			name:    "generic record",
			slug:    "contact",
			title:   "Contact",
			kicker:  "PUBLIC INFORMATION",
			content: `<h2>Write to us</h2><p>Email.</p>`,
			fragments: []string{
				`<h2 id="write-to-us">Write to us</h2>`,
				"Email.",
			},
		},
	}

	for _, tc := range cases {
		expected := append([]string{
			tc.title + " - WGA",
			tc.kicker,
			`<link rel="canonical" href="https://gallery.example/pages/` + tc.slug + `"`,
		}, tc.fragments...)

		scenario := tests.ApiScenario{
			Name:            "static " + tc.slug + " renders its own kicker and canonical",
			Method:          http.MethodGet,
			URL:             "/pages/" + tc.slug,
			ExpectedStatus:  http.StatusOK,
			ExpectedContent: expected,
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				configureStaticPublicURL(t)
				app := newStaticTestApp(t)
				RegisterHandlers(app, config.EnvironmentProduction)
				createStaticPage(t, app, tc.slug, tc.title, tc.content)
				return app
			},
		}

		scenario.Test(t)
	}
}

func TestStaticPageRouteMissingRecordReturnsShared404WithoutErrorLog(t *testing.T) {
	var captured func() []*core.Log

	scenario := tests.ApiScenario{
		Name:           "missing static page returns shared 404 without error logging",
		Method:         http.MethodGet,
		URL:            "/pages/no-such-reference-page",
		ExpectedStatus: http.StatusNotFound,
		ExpectedContent: []string{
			"This record is not in the collection.",
			"RETURN TO THE GALLERY",
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			app := newStaticTestApp(t)
			captured = testutils.CaptureLogs(app)
			RegisterHandlers(app, config.EnvironmentProduction)
			return app
		},
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, _ *http.Response) {
			testutils.FlushLogs(t, app)
			if entries := testutils.LogsWithEvent(captured(), "static_page.request.failed"); len(entries) != 0 {
				t.Fatalf("expected no static page failure logs for an absent record, got %d", len(entries))
			}
		},
	}

	scenario.Test(t)
}

func TestStaticPageRouteDatabaseFailureReturnsShared500(t *testing.T) {
	const sensitiveDetail = "no such column: slug"
	var captured func() []*core.Log

	scenario := tests.ApiScenario{
		Name:            "static page database failure returns shared 500 with safe logging",
		Method:          http.MethodGet,
		URL:             "/pages/about",
		ExpectedStatus:  http.StatusInternalServerError,
		ExpectedContent: []string{"The archive could not complete that request."},
		NotExpectedContent: []string{
			sensitiveDetail,
			"Internal server error",
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			app, err := tests.NewTestApp()
			if err != nil {
				t.Fatalf("create test app: %v", err)
			}
			app.Settings().Logs.MaxDays = 1
			captured = testutils.CaptureLogs(app)

			// A static_pages collection without the expected slug field forces
			// the slug lookup to fail with a persistence error, not absence.
			collection := core.NewBaseCollection("Static_pages")
			collection.Id = "static_pages"
			collection.MarkAsNew()
			collection.Fields.Add(
				&core.TextField{Id: "sp_title", Name: "title", Required: true},
				&core.EditorField{Id: "sp_content", Name: "content"},
			)
			if err := app.Save(collection); err != nil {
				t.Fatalf("save broken static_pages collection: %v", err)
			}

			RegisterHandlers(app, config.EnvironmentProduction)
			return app
		},
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, _ *http.Response) {
			testutils.FlushLogs(t, app)
			entry := testutils.LogWithEvent(captured(), "static_page.request.failed")
			if entry == nil {
				t.Fatal("expected a static page failure log")
			}
			if entry.Data["outcome"] != "fetch_error" {
				t.Fatalf("outcome = %v, want fetch_error", entry.Data["outcome"])
			}
			if strings.Contains(fmt.Sprint(testutils.LogData(captured())), sensitiveDetail) {
				t.Fatalf("captured log contains %q", sensitiveDetail)
			}
		},
	}

	scenario.Test(t)
}

func TestStaticPageServerErrorIsClientSafe(t *testing.T) {
	const sensitiveDetail = "credential token=secret-value"
	var captured func() []*core.Log

	scenario := tests.ApiScenario{
		Name:            "static page server error omits internal detail",
		Method:          http.MethodGet,
		URL:             "/pages/server-error",
		ExpectedStatus:  http.StatusInternalServerError,
		ExpectedContent: []string{"The archive could not complete that request."},
		NotExpectedContent: []string{
			sensitiveDetail,
			"Internal server error",
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			app, err := tests.NewTestApp()
			if err != nil {
				t.Fatalf("create test app: %v", err)
			}
			app.Settings().Logs.MaxDays = 1
			captured = testutils.CaptureLogs(app)

			app.OnServe().BindFunc(func(se *core.ServeEvent) error {
				se.Router.GET("/pages/server-error", func(e *core.RequestEvent) error {
					return staticPageServerError(app, e, "render_error", fmt.Errorf("%s", sensitiveDetail))
				})
				return se.Next()
			})

			return app
		},
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, _ *http.Response) {
			testutils.FlushLogs(t, app)
			entry := testutils.LogWithEvent(captured(), "static_page.request.failed")
			if entry == nil {
				t.Fatal("expected a static page failure log")
			}
			if entry.Data["outcome"] != "render_error" {
				t.Fatalf("outcome = %v, want render_error", entry.Data["outcome"])
			}
			if entry.Data["error_type"] == "" {
				t.Fatal("expected a recorded error type")
			}
			if strings.Contains(fmt.Sprint(testutils.LogData(captured())), sensitiveDetail) {
				t.Fatalf("captured log contains %q", sensitiveDetail)
			}
		},
	}

	scenario.Test(t)
}

func TestCatchAllUnknownPublicGetReturnsHtml404(t *testing.T) {
	scenario := tests.ApiScenario{
		Name:           "unknown public GET returns shared HTML 404",
		Method:         http.MethodGet,
		URL:            "/no-such-public-page",
		ExpectedStatus: http.StatusNotFound,
		ExpectedContent: []string{
			"This record is not in the collection.",
			"RETURN TO THE GALLERY",
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			app := newStaticTestApp(t)
			registerFakeHome(app)
			RegisterHandlers(app, config.EnvironmentProduction)
			return app
		},
		AfterTestFunc: func(t testing.TB, _ *tests.TestApp, response *http.Response) {
			if got := response.Header.Get("Content-Type"); !strings.Contains(got, "text/html") {
				t.Fatalf("Content-Type = %q, want text/html", got)
			}
		},
	}

	scenario.Test(t)
}

func TestCatchAllKeepsTechnicalBoundariesTechnical(t *testing.T) {
	assertNotHTML := func(t testing.TB, response *http.Response) {
		t.Helper()
		if got := response.Header.Get("Content-Type"); strings.Contains(got, "text/html") {
			t.Fatalf("Content-Type = %q, must stay technical (non-HTML)", got)
		}
	}

	scenarios := []tests.ApiScenario{
		{
			Name:           "API miss stays a JSON 404",
			Method:         http.MethodGet,
			URL:            "/api/no-such-public-endpoint",
			ExpectedStatus: http.StatusNotFound,
			ExpectedContent: []string{
				`"status":404`,
				"The requested resource wasn't found.",
			},
			NotExpectedContent: []string{"This record is not in the collection."},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := newStaticTestApp(t)
				registerFakeHome(app)
				RegisterHandlers(app, config.EnvironmentProduction)
				return app
			},
			AfterTestFunc: func(t testing.TB, _ *tests.TestApp, response *http.Response) {
				assertNotHTML(t, response)
			},
		},
		{
			Name:           "admin UI miss stays technical",
			Method:         http.MethodGet,
			URL:            "/_/no-such-admin-page",
			ExpectedStatus: http.StatusNotFound,
			NotExpectedContent: []string{
				"This record is not in the collection.",
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := newStaticTestApp(t)
				registerFakeHome(app)
				RegisterHandlers(app, config.EnvironmentProduction)
				return app
			},
			AfterTestFunc: func(t testing.TB, _ *tests.TestApp, response *http.Response) {
				assertNotHTML(t, response)
			},
		},
		{
			Name:           "non-GET public miss stays a JSON 404",
			Method:         http.MethodPost,
			URL:            "/no-such-public-page",
			ExpectedStatus: http.StatusNotFound,
			ExpectedContent: []string{
				`"status":404`,
			},
			NotExpectedContent: []string{"This record is not in the collection."},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := newStaticTestApp(t)
				registerFakeHome(app)
				RegisterHandlers(app, config.EnvironmentProduction)
				return app
			},
			AfterTestFunc: func(t testing.TB, _ *tests.TestApp, response *http.Response) {
				assertNotHTML(t, response)
			},
		},
		{
			Name:           "unknown HEAD public miss stays technical",
			Method:         http.MethodHead,
			URL:            "/no-such-public-page",
			ExpectedStatus: http.StatusNotFound,
			NotExpectedContent: []string{
				"This record is not in the collection.",
			},
			TestAppFactory: func(t testing.TB) *tests.TestApp {
				app := newStaticTestApp(t)
				registerFakeHome(app)
				RegisterHandlers(app, config.EnvironmentProduction)
				return app
			},
			AfterTestFunc: func(t testing.TB, _ *tests.TestApp, response *http.Response) {
				assertNotHTML(t, response)
			},
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestReservedBoundary(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{path: "/api", want: true},
		{path: "/api/collections", want: true},
		{path: "/_/admins", want: true},
		{path: "/assets/css/style.css", want: true},
		{path: "/sitemap/sitemap.xml", want: true},
		{path: "/tmp/visual-overhaul", want: true},
		{path: "/apix", want: false},
		{path: "/pages/about", want: false},
		{path: "/artworks", want: false},
		{path: "/", want: false},
		{path: "/apiary", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			if got := isReservedBoundary(tc.path); got != tc.want {
				t.Errorf("isReservedBoundary(%q) = %t, want %t", tc.path, got, tc.want)
			}
		})
	}
}

func TestFullAppRoutingPreservesHomeAndKnownRoutes(t *testing.T) {
	app := newFullAppTestApp(t)
	configureStaticPublicURL(t)

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/known", func(c *core.RequestEvent) error {
			return c.HTML(http.StatusOK, "KNOWN")
		})
		return se.Next()
	})
	landing.RegisterHandlers(app)
	RegisterHandlers(app, config.EnvironmentProduction)
	createStaticPage(t, app, "about", "About", `<h2>The collection</h2><p>The archive.</p>`)

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

		cases := []struct {
			name        string
			method      string
			path        string
			status      int
			contains    string
			notContains string
			contentType string
		}{
			{name: "home GET", method: http.MethodGet, path: "/", status: http.StatusOK, contains: "Web Gallery of Art"},
			{name: "home HEAD", method: http.MethodHead, path: "/", status: http.StatusOK},
			{name: "known route", method: http.MethodGet, path: "/known", status: http.StatusOK, contains: "KNOWN"},
			{name: "known route HEAD", method: http.MethodHead, path: "/known", status: http.StatusOK},
			{name: "static page", method: http.MethodGet, path: "/pages/about", status: http.StatusOK, contains: "About"},
			{name: "unknown public GET", method: http.MethodGet, path: "/no-such-public-page", status: http.StatusNotFound, contains: "This record is not in the collection.", contentType: "text/html"},
			{name: "unknown public HEAD", method: http.MethodHead, path: "/no-such-public-page", status: http.StatusNotFound, notContains: "This record is not in the collection.", contentType: "application/json"},
			{name: "unknown API GET", method: http.MethodGet, path: "/api/no-such", status: http.StatusNotFound, contains: `"status":404`, notContains: "This record is not in the collection.", contentType: "application/json"},
			{name: "missing asset", method: http.MethodGet, path: "/assets/no-such.css", status: http.StatusNotFound, contains: "File not found", notContains: "This record is not in the collection.", contentType: "application/json"},
			{name: "POST unknown", method: http.MethodPost, path: "/no-such-public-page", status: http.StatusNotFound, contains: `"status":404`, notContains: "This record is not in the collection.", contentType: "application/json"},
		}

		for _, tc := range cases {
			recorder := httptest.NewRecorder()
			mux.ServeHTTP(recorder, httptest.NewRequest(tc.method, tc.path, nil))

			if recorder.Code != tc.status {
				t.Errorf("%s: status = %d, want %d", tc.name, recorder.Code, tc.status)
			}
			body := recorder.Body.String()
			if tc.contains != "" && !strings.Contains(body, tc.contains) {
				t.Errorf("%s: body does not contain %q", tc.name, tc.contains)
			}
			if tc.notContains != "" && strings.Contains(body, tc.notContains) {
				t.Errorf("%s: body must not contain %q", tc.name, tc.notContains)
			}
			if tc.contentType != "" && !strings.Contains(recorder.Header().Get("Content-Type"), tc.contentType) {
				t.Errorf("%s: Content-Type = %q, want %q", tc.name, recorder.Header().Get("Content-Type"), tc.contentType)
			}
		}

		return nil
	}); err != nil {
		t.Fatalf("trigger serve event: %v", err)
	}
}

func TestStaticPageHandlerRenderFailureStaysSafe(t *testing.T) {
	var captured func() []*core.Log

	app := newStaticTestApp(t)
	captured = testutils.CaptureLogs(app)
	createStaticPage(t, app, "about", "About", `<h2>The collection</h2><p>The archive.</p>`)

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	request := httptest.NewRequest(http.MethodGet, "/pages/about", nil).WithContext(cancelled)
	request.SetPathValue("slug", "about")

	recorder := httptest.NewRecorder()
	event := &core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  request,
			Response: recorder,
		},
	}

	err := staticPageHandler(app, event)
	if err == nil {
		t.Fatal("expected the render failure to surface as an error")
	}

	testutils.FlushLogs(t, app)
	entry := testutils.LogWithEvent(captured(), "static_page.request.failed")
	if entry == nil {
		t.Fatal("expected a static page failure log")
	}
	if entry.Data["outcome"] != "render_error" {
		t.Fatalf("outcome = %v, want render_error", entry.Data["outcome"])
	}

	body := recorder.Body.String()
	if strings.Contains(body, "Internal server error") {
		t.Errorf("render failure must not return a bare internal server error: %s", body)
	}
	if strings.Contains(body, "The archive.") {
		t.Errorf("render failure must not leak managed content: %s", body)
	}
}
