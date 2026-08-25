package timeline

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func TestTimelineRouteRendersFullAndHTMX(t *testing.T) {
	app := newTimelineApp(t)

	saveTimelineRecord(t, app, "artists", "artistpub000001", map[string]any{"name": "An Artist", "published": true})
	saveTimelineRecord(t, app, "art_periods", "periodbaro00001", map[string]any{
		"name": "Baroque", "start": 1500, "end": 1750, "description": "a period",
	})
	saveTimelineRecord(t, app, "artworks", "artworkon000001", map[string]any{
		"title": "Visible Work", "author": []string{"artistpub000001"}, "published": true,
		"date_start": 1600, "date_end": 1610,
	})
	saveTimelineRecord(t, app, "artworks", "artworkear00001", map[string]any{
		"title": "Early Work", "author": []string{"artistpub000001"}, "published": true,
		"date_start": 1500,
	})
	saveTimelineRecord(t, app, "artworks", "artworklat00001", map[string]any{
		"title": "Late Work", "author": []string{"artistpub000001"}, "published": true,
		"date_start": 1800,
	})

	RegisterHandlers(app)

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
		mux.ServeHTTP(full, httptest.NewRequest(http.MethodGet, "/timeline", nil))
		if full.Code != http.StatusOK {
			t.Fatalf("full status = %d, want 200", full.Code)
		}
		body := full.Body.String()
		if !strings.Contains(body, "<html") {
			t.Error("full response did not render a document")
		}
		if !strings.Contains(body, "Timeline") || !strings.Contains(body, "Visible Work") {
			t.Error("full response did not render the timeline content")
		}
		if got := strings.Count(body, `id="mc-area"`); got != 1 {
			t.Errorf("full response rendered %d #mc-area elements, want exactly 1", got)
		}
		if got := full.Header().Get("HX-Push-Url"); got != "/timeline" {
			t.Errorf("HX-Push-Url = %q, want /timeline", got)
		}

		// Boosted shell navigation targets #mc-area and must receive the
		// selectable full document rather than the local block.
		shell := httptest.NewRecorder()
		shellRequest := httptest.NewRequest(http.MethodGet, "/timeline", nil)
		shellRequest.Header.Set("HX-Request", "true")
		shellRequest.Header.Set("HX-Target", "mc-area")
		mux.ServeHTTP(shell, shellRequest)
		if shell.Code != http.StatusOK {
			t.Fatalf("shell status = %d, want 200", shell.Code)
		}
		if !strings.Contains(shell.Body.String(), "<html") {
			t.Error("shell navigation must render the full document")
		}
		if got := strings.Count(shell.Body.String(), `id="mc-area"`); got != 1 {
			t.Errorf("shell response rendered %d #mc-area elements, want exactly 1", got)
		}

		partial := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/timeline?from=1600&to=1700", nil)
		request.Header.Set("HX-Request", "true")
		request.Header.Set("HX-Target", "timeline")
		mux.ServeHTTP(partial, request)
		if partial.Code != http.StatusOK {
			t.Fatalf("partial status = %d, want 200", partial.Code)
		}
		if strings.Contains(partial.Body.String(), "<html") {
			t.Error("HTMX response should not render the full document")
		}
		if !strings.Contains(partial.Body.String(), "Visible Work") {
			t.Error("HTMX response should render the works panel")
		}
		if !strings.Contains(partial.Body.String(), `id="timeline"`) {
			t.Error("HTMX response should render the timeline block")
		}
		if strings.Contains(partial.Body.String(), `id="mc-area"`) {
			t.Error("feature-local timeline response must not carry #mc-area")
		}
		if got := partial.Header().Get("HX-Push-Url"); got != "/timeline?from=1600&to=1700" {
			t.Errorf("HX-Push-Url = %q, want /timeline?from=1600&to=1700", got)
		}

		return nil
	}); err != nil {
		t.Fatalf("trigger serve event: %v", err)
	}
}

func TestTimelineRouteRendersNoJavascriptFormAndLinks(t *testing.T) {
	app := newTimelineApp(t)

	saveTimelineRecord(t, app, "artists", "artistpub000001", map[string]any{"name": "An Artist", "published": true})
	saveTimelineRecord(t, app, "artworks", "artworkon000001", map[string]any{
		"title": "Visible Work", "author": []string{"artistpub000001"}, "published": true,
		"date_start": 1600,
	})

	RegisterHandlers(app)

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
		mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/timeline", nil))
		body := recorder.Body.String()

		for _, expected := range []string{
			`action="/timeline"`,
			`method="GET"`,
			`name="from"`,
			`name="to"`,
			`<noscript>`,
			`type="submit"`,
			"APPLY WINDOW",
			`href="/artists/an-artist-artistpub000001/visible-work-artworkon000001"`,
			`hx-select="#timeline"`,
			`hx-swap="outerHTML"`,
			`class="sr-only"`,
			"DECADE",
			"WORKS",
		} {
			if !strings.Contains(body, expected) {
				t.Errorf("no-JS response does not contain %q", expected)
			}
		}
		if strings.Contains(body, "VIEW ALL WORKS") {
			t.Error("timeline must not render the unfiltered /artworks continuation")
		}

		return nil
	}); err != nil {
		t.Fatalf("trigger serve event: %v", err)
	}
}

func TestTimelineRoutePaginatesWorksPreservingWindow(t *testing.T) {
	app := newTimelineApp(t)

	saveTimelineRecord(t, app, "artists", "artistpub000001", map[string]any{"name": "An Artist", "published": true})
	// Widen the bounds beyond the selected window so the window URL is not the
	// canonical full range.
	saveTimelineRecord(t, app, "artworks", "artworkearly001", map[string]any{
		"title": "Early Out", "author": []string{"artistpub000001"}, "published": true,
		"date_start": 1400,
	})
	for i := 0; i < 10; i++ {
		saveTimelineRecord(t, app, "artworks", fmt.Sprintf("artworkpag%05d", i+1), map[string]any{
			"title": fmt.Sprintf("Work %02d", i+1), "author": []string{"artistpub000001"}, "published": true,
			"date_start": 1500 + i,
		})
	}
	saveTimelineRecord(t, app, "artworks", "artworklater001", map[string]any{
		"title": "Late Out", "author": []string{"artistpub000001"}, "published": true,
		"date_start": 2000,
	})

	RegisterHandlers(app)

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

		page2 := httptest.NewRecorder()
		mux.ServeHTTP(page2, httptest.NewRequest(http.MethodGet, "/timeline?from=1500&to=1600&page=2", nil))
		body := page2.Body.String()

		if got := page2.Header().Get("HX-Push-Url"); got != "/timeline?from=1500&to=1600&page=2" {
			t.Errorf("HX-Push-Url = %q, want /timeline?from=1500&to=1600&page=2", got)
		}
		if !strings.Contains(body, "PAGE 2 OF 2") {
			t.Error("page 2 response did not render the pagination readout")
		}
		if !strings.Contains(body, "WORKS 9–10 OF 10") {
			t.Error("page 2 response did not render the truthful range label")
		}
		if !strings.Contains(body, ">Work 09</span>") || !strings.Contains(body, ">Work 10</span>") {
			t.Error("page 2 response did not render the second page of works")
		}
		if strings.Contains(body, ">Work 01</span>") {
			t.Error("page 2 response rendered a first-page work")
		}
		if !strings.Contains(body, `hx-get="/timeline?from=1500&amp;to=1600"`) {
			t.Error("page 2 response did not render a prev link back to page 1")
		}
		if strings.Contains(body, "NEXT →") && strings.Contains(body, `hx-get="/timeline?from=1500&amp;to=1600&amp;page=3"`) {
			t.Error("page 2 response rendered a next link past the last page")
		}

		clamped := httptest.NewRecorder()
		mux.ServeHTTP(clamped, httptest.NewRequest(http.MethodGet, "/timeline?from=1500&to=1600&page=999", nil))
		if got := clamped.Header().Get("HX-Push-Url"); got != "/timeline?from=1500&to=1600&page=2" {
			t.Errorf("HX-Push-Url for clamped page = %q, want /timeline?from=1500&to=1600&page=2", got)
		}

		return nil
	}); err != nil {
		t.Fatalf("trigger serve event: %v", err)
	}
}

func TestTimelineRouteNormalisesCanonicalUrl(t *testing.T) {
	app := newTimelineApp(t)

	saveTimelineRecord(t, app, "artists", "artistpub000001", map[string]any{"name": "An Artist", "published": true})
	saveTimelineRecord(t, app, "artworks", "artworkon000001", map[string]any{
		"title": "Work", "author": []string{"artistpub000001"}, "published": true,
		"date_start": 1600,
	})

	RegisterHandlers(app)

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
		mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/timeline?from=abc&to=9999", nil))

		if got := recorder.Header().Get("HX-Push-Url"); got != "/timeline" {
			t.Errorf("HX-Push-Url = %q, want canonical /timeline", got)
		}

		return nil
	}); err != nil {
		t.Fatalf("trigger serve event: %v", err)
	}
}

func TestTimelineRouteCanonicalisesPageOnEmptyWindows(t *testing.T) {
	// Period-only: a valid window with zero matching artworks.
	periodOnly := newTimelineApp(t)
	saveTimelineRecord(t, periodOnly, "art_periods", "periodonly00001", map[string]any{"name": "Only", "start": 1000, "end": 1200, "description": ""})

	RegisterHandlers(periodOnly)

	router, err := apis.NewRouter(periodOnly)
	if err != nil {
		t.Fatalf("create router: %v", err)
	}
	serveEvent := &core.ServeEvent{App: periodOnly, Router: router}
	if err := periodOnly.OnServe().Trigger(serveEvent, func(event *core.ServeEvent) error {
		mux, err := event.Router.BuildMux()
		if err != nil {
			return err
		}

		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/timeline?page=999", nil))
		if got := recorder.Header().Get("HX-Push-Url"); got != "/timeline" {
			t.Errorf("period-only HX-Push-Url = %q, want /timeline (page=999 canonicalised away)", got)
		}
		if !strings.Contains(recorder.Body.String(), "No works in this window") {
			t.Error("period-only empty window did not render the empty state")
		}

		return nil
	}); err != nil {
		t.Fatalf("trigger serve event: %v", err)
	}

	// Fully empty chronology.
	empty := newTimelineApp(t)
	RegisterHandlers(empty)

	emptyRouter, err := apis.NewRouter(empty)
	if err != nil {
		t.Fatalf("create empty router: %v", err)
	}
	emptyServeEvent := &core.ServeEvent{App: empty, Router: emptyRouter}
	if err := empty.OnServe().Trigger(emptyServeEvent, func(event *core.ServeEvent) error {
		mux, err := event.Router.BuildMux()
		if err != nil {
			return err
		}

		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/timeline?page=999", nil))
		if got := recorder.Header().Get("HX-Push-Url"); got != "/timeline" {
			t.Errorf("empty chronology HX-Push-Url = %q, want /timeline (page=999 canonicalised away)", got)
		}
		if !strings.Contains(recorder.Body.String(), "No chronology yet") {
			t.Error("empty chronology did not render the empty state")
		}

		return nil
	}); err != nil {
		t.Fatalf("trigger empty serve event: %v", err)
	}
}

func TestTimelineRouteRendersHonestEmptyState(t *testing.T) {
	app := newTimelineApp(t)

	RegisterHandlers(app)

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
		mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/timeline", nil))
		body := recorder.Body.String()

		if !strings.Contains(body, "No chronology yet") {
			t.Error("empty timeline did not render the honest empty state")
		}
		if !strings.Contains(body, `href="/artworks"`) {
			t.Error("empty timeline did not render the artwork recovery link")
		}

		return nil
	}); err != nil {
		t.Fatalf("trigger serve event: %v", err)
	}
}
