package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
	"github.com/blackfyre/wga/internal/logging"
	"github.com/blackfyre/wga/internal/testutils"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
)

func TestEligibleForTrustedHeadMarkup(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		accept string
		htmx   bool
		want   bool
	}{
		{name: "GET HTML document", method: http.MethodGet, path: "/", accept: "text/html,application/xhtml+xml", want: true},
		{name: "GET nested page", method: http.MethodGet, path: "/artists/record", accept: "text/html", want: true},
		{name: "HEAD HTML document", method: http.MethodHead, path: "/artists", accept: "text/html", want: true},
		{name: "POST rejected", method: http.MethodPost, path: "/", accept: "text/html", want: false},
		{name: "PUT rejected", method: http.MethodPut, path: "/", accept: "text/html", want: false},
		{name: "HTMX GET rejected", method: http.MethodGet, path: "/artists", accept: "text/html", htmx: true, want: false},
		{name: "non-HTML accept rejected", method: http.MethodGet, path: "/", accept: "application/json", want: false},
		{name: "empty accept rejected", method: http.MethodGet, path: "/", want: false},
		{name: "zero quality accept rejected", method: http.MethodGet, path: "/", accept: "text/html;q=0", want: false},
		{name: "mixed case media type accepted", method: http.MethodGet, path: "/", accept: "Text/HTML", want: true},
		{name: "api boundary rejected", method: http.MethodGet, path: "/api/collections/strings", accept: "text/html", want: false},
		{name: "pocketbase internal boundary rejected", method: http.MethodGet, path: "/_/health", accept: "text/html", want: false},
		{name: "assets boundary rejected", method: http.MethodGet, path: "/assets/css/style.css", accept: "text/html", want: false},
		{name: "sitemap boundary rejected", method: http.MethodGet, path: "/sitemap/sitemap.xml", accept: "text/html", want: false},
		{name: "visual overhaul boundary rejected", method: http.MethodGet, path: "/tmp/visual-overhaul", accept: "text/html", want: false},
		{name: "similar prefix is not a boundary", method: http.MethodGet, path: "/apian", accept: "text/html", want: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(tc.method, tc.path, nil)
			request.Header.Set("Accept", tc.accept)
			if tc.htmx {
				request.Header.Set("HX-Request", "true")
			}

			if got := eligibleForTrustedHeadMarkup(request); got != tc.want {
				t.Fatalf("eligibleForTrustedHeadMarkup() = %t, want %t", got, tc.want)
			}
		})
	}
}

func TestEligibleForTrustedHeadMarkupNilRequest(t *testing.T) {
	if eligibleForTrustedHeadMarkup(nil) {
		t.Fatal("expected nil request to be ineligible")
	}
}

func TestAcceptsHTML(t *testing.T) {
	tests := []struct {
		name   string
		accept string
		want   bool
	}{
		{name: "exact media type", accept: "text/html", want: true},
		{name: "media type within list", accept: "application/xml,text/html,application/xhtml+xml", want: true},
		{name: "default quality", accept: "text/html;level=1", want: true},
		{name: "maximum quality", accept: "text/html;q=1", want: true},
		{name: "positive quality", accept: "text/html;q=0.5", want: true},
		{name: "zero quality rejected", accept: "text/html;q=0", want: false},
		{name: "mixed case media type", accept: "Text/HTML", want: true},
		{name: "upper case media type with quality", accept: "TEXT/HTML;q=0.9", want: true},
		{name: "deceptive parameter substring rejected", accept: "application/xhtml+xml;profile=text/html", want: false},
		{name: "unrelated parameter value rejected", accept: "application/json;x=text/html", want: false},
		{name: "malformed media token rejected", accept: "text/htmlfoo", want: false},
		{name: "malformed quality rejected", accept: "text/html;q=abc", want: false},
		{name: "empty quality rejected", accept: "text/html;q=", want: false},
		{name: "out of range quality rejected", accept: "text/html;q=2", want: false},
		{name: "negative quality rejected", accept: "text/html;q=-0.1", want: false},
		{name: "wildcard subtype rejected", accept: "text/*", want: false},
		{name: "empty accept rejected", accept: "", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := acceptsHTML(tc.accept); got != tc.want {
				t.Fatalf("acceptsHTML(%q) = %t, want %t", tc.accept, got, tc.want)
			}
		})
	}
}

func newHeaderMarkupTestApp(t testing.TB, includeNameField bool) *tests.TestApp {
	t.Helper()

	app := testutils.NewTestApp(t)

	collection := core.NewBaseCollection("Strings")
	collection.Id = "strings"
	collection.MarkAsNew()
	if includeNameField {
		collection.Fields.Add(
			&core.TextField{Id: "strings_name", Name: "name", Required: true},
			&core.EditorField{Id: "strings_content", Name: "content"},
		)
	} else {
		// A strings collection without the expected name field forces the
		// lookup to fail with a persistence error rather than absence.
		collection.Fields.Add(
			&core.EditorField{Id: "strings_content", Name: "content"},
		)
	}
	if err := app.Save(collection); err != nil {
		t.Fatalf("save strings collection: %v", err)
	}

	return app
}

func createStringRecord(t testing.TB, app core.App, name, content string) {
	t.Helper()

	collection, err := app.FindCollectionByNameOrId("strings")
	if err != nil {
		t.Fatalf("find strings collection: %v", err)
	}
	record := core.NewRecord(collection)
	record.Set("name", name)
	record.Set("content", content)
	if err := app.Save(record); err != nil {
		t.Fatalf("save strings record: %v", err)
	}
}

func headerMarkupRequestEvent(app core.App, path string) *core.RequestEvent {
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.Header.Set("Accept", "text/html")

	return &core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  request,
			Response: httptest.NewRecorder(),
		},
	}
}

func TestTrustedHeadMarkupMiddlewareSetsContent(t *testing.T) {
	const markup = `<script src="/assets/js/trusted.js"></script>`
	app := newHeaderMarkupTestApp(t, true)
	createStringRecord(t, app, "scripts:header", markup)

	event := headerMarkupRequestEvent(app, "/")
	if err := prepareTrustedHeadMarkup(app, event); err != nil {
		t.Fatalf("prepareTrustedHeadMarkup: %v", err)
	}

	if got := tmplUtils.GetTrustedHeadMarkup(event.Request.Context()); got != markup {
		t.Fatalf("trusted markup = %q, want %q", got, markup)
	}
}

func TestTrustedHeadMarkupMiddlewareEmptyContent(t *testing.T) {
	app := newHeaderMarkupTestApp(t, true)
	createStringRecord(t, app, "scripts:header", "")

	event := headerMarkupRequestEvent(app, "/")
	if err := prepareTrustedHeadMarkup(app, event); err != nil {
		t.Fatalf("prepareTrustedHeadMarkup: %v", err)
	}

	if got := tmplUtils.GetTrustedHeadMarkup(event.Request.Context()); got != "" {
		t.Fatalf("trusted markup = %q, want empty", got)
	}
}

func TestTrustedHeadMarkupMiddlewareMissingRecord(t *testing.T) {
	app := newHeaderMarkupTestApp(t, true)
	captured := testutils.CaptureLogs(app)

	event := headerMarkupRequestEvent(app, "/")
	if err := prepareTrustedHeadMarkup(app, event); err != nil {
		t.Fatalf("prepareTrustedHeadMarkup: %v", err)
	}

	if got := tmplUtils.GetTrustedHeadMarkup(event.Request.Context()); got != "" {
		t.Fatalf("trusted markup = %q, want empty", got)
	}

	testutils.FlushLogs(t, app)
	if entry := testutils.LogWithEvent(captured(), "header_markup.lookup.failed"); entry != nil {
		t.Fatalf("absence must not be logged as a failure: %v", entry.Data)
	}
}

func TestTrustedHeadMarkupMiddlewareChangesPerRequest(t *testing.T) {
	app := newHeaderMarkupTestApp(t, true)
	createStringRecord(t, app, "scripts:header", "first")

	first := headerMarkupRequestEvent(app, "/")
	if err := prepareTrustedHeadMarkup(app, first); err != nil {
		t.Fatalf("prepareTrustedHeadMarkup: %v", err)
	}
	if got := tmplUtils.GetTrustedHeadMarkup(first.Request.Context()); got != "first" {
		t.Fatalf("trusted markup = %q, want %q", got, "first")
	}

	record, err := app.FindFirstRecordByData("strings", "name", "scripts:header")
	if err != nil {
		t.Fatalf("find scripts:header record: %v", err)
	}
	record.Set("content", "second")
	if err := app.Save(record); err != nil {
		t.Fatalf("update scripts:header record: %v", err)
	}

	second := headerMarkupRequestEvent(app, "/")
	if err := prepareTrustedHeadMarkup(app, second); err != nil {
		t.Fatalf("prepareTrustedHeadMarkup: %v", err)
	}
	if got := tmplUtils.GetTrustedHeadMarkup(second.Request.Context()); got != "second" {
		t.Fatalf("trusted markup = %q, want %q", got, "second")
	}
}

func TestTrustedHeadMarkupMiddlewareFailedLookupLogsRedacted(t *testing.T) {
	app := newHeaderMarkupTestApp(t, false)
	captured := testutils.CaptureLogs(app)

	event := headerMarkupRequestEvent(app, "/")
	logging.SetRequestID(event, "request-123")

	if err := prepareTrustedHeadMarkup(app, event); err != nil {
		t.Fatalf("expected middleware to continue without error, got %v", err)
	}

	if got := tmplUtils.GetTrustedHeadMarkup(event.Request.Context()); got != "" {
		t.Fatalf("trusted markup = %q, want empty after failed lookup", got)
	}

	testutils.FlushLogs(t, app)
	entry := testutils.LogWithEvent(captured(), "header_markup.lookup.failed")
	if entry == nil {
		t.Fatal("expected a header markup lookup failure log")
	}
	if got := entry.Data["request_id"]; got != "request-123" {
		t.Fatalf("request_id = %v, want %q", got, "request-123")
	}
	if got, ok := entry.Data["error_type"].(string); !ok || got == "" {
		t.Fatalf("error_type = %v, want non-empty string", entry.Data["error_type"])
	}
	if got := entry.Data["error"]; got != "[REDACTED]" {
		t.Fatalf("error = %v, want [REDACTED]", got)
	}

	output := fmt.Sprint(testutils.LogData(captured()))
	for _, sensitive := range []string{"scripts:header", "invalid or missing field"} {
		if strings.Contains(output, sensitive) {
			t.Fatalf("captured log contains %q: %s", sensitive, output)
		}
	}
}

func TestTrustedHeadMarkupMiddlewareRegisteredBeforeFeatureRoutes(t *testing.T) {
	const markup = `<script src="/assets/js/trusted.js"></script>`
	app := newHeaderMarkupTestApp(t, true)
	createStringRecord(t, app, "scripts:header", markup)

	registerTrustedHeadMarkupMiddleware(app)
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/known", func(c *core.RequestEvent) error {
			return c.HTML(http.StatusOK, tmplUtils.GetTrustedHeadMarkup(c.Request.Context()))
		})
		return se.Next()
	})

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

		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/known", nil)
		request.Header.Set("Accept", "text/html")
		mux.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
		}
		if got := recorder.Body.String(); got != markup {
			t.Fatalf("feature route saw %q, want trusted markup %q", got, markup)
		}

		return nil
	}); err != nil {
		t.Fatalf("trigger serve event: %v", err)
	}
}
