package timeline

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/migrations"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// realSeedPath resolves the authoritative producer SQLite source used for the
// updated-seed integration coverage. It is intentionally absent from CI: the
// test skips unless the operator points WGA_SEED_SQLITE_PATH at a production
// export or a sibling wga-src checkout is present.
//
// The resolved path is made absolute because seed.Import derives a file:// URL
// from it, and modernc/sqlite rejects relative ".." segments as an invalid URI
// authority.
func realSeedPath(t *testing.T) string {
	t.Helper()

	if path := os.Getenv("WGA_SEED_SQLITE_PATH"); path != "" {
		abs, err := filepath.Abs(path)
		if err != nil {
			t.Fatalf("resolve WGA_SEED_SQLITE_PATH %q: %v", path, err)
		}
		if _, err := os.Stat(abs); err != nil {
			t.Fatalf("WGA_SEED_SQLITE_PATH %q is not readable: %v", path, err)
		}
		return abs
	}

	// The timeline package's test working directory is
	// <repo>/internal/handlers/timeline, so the adjacent wga-src checkout sits
	// four levels up under a Projects/workspace root.
	candidate := filepath.Join("..", "..", "..", "..", "wga-src", "out", "production", "wga-src.sqlite")
	if info, err := os.Stat(candidate); err == nil && info.Mode().IsRegular() {
		abs, err := filepath.Abs(candidate)
		if err != nil {
			t.Fatalf("resolve sibling seed path: %v", err)
		}
		return abs
	}

	t.Skip("real producer seed SQLite not available; set WGA_SEED_SQLITE_PATH to run the updated-seed integration coverage")
	return ""
}

var configureImportedMigrations sync.Once

// newImportedTimelineApp bootstraps a fresh app and imports the authoritative
// producer seed through the real migration path (create schema, run pending
// migrations, seed.Import), matching how `serve` provisions a fresh data
// directory.
func newImportedTimelineApp(t *testing.T) *pocketbase.PocketBase {
	t.Helper()

	sqlitePath := realSeedPath(t)

	configureImportedMigrations.Do(func() {
		configuration := config.LoadFrom(func(key string) string {
			values := map[string]string{
				"WGA_PROTOCOL":       "http",
				"WGA_HOSTNAME":       "gallery.example",
				"WGA_SMTP_HOST":      "smtp.example",
				"WGA_SMTP_PORT":      "2525",
				"WGA_SENDER_NAME":    "WGA Test",
				"WGA_SENDER_ADDRESS": "sender@example.com",
				"WGA_SEED_SQLITE_PATH": sqlitePath,
			}
			return values[key]
		})
		if err := migrations.Configure(configuration.Migrations()); err != nil {
			t.Fatalf("configure migrations: %v", err)
		}
	})

	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Errorf("reset bootstrap state: %v", err)
		}
	})

	if err := app.RunAllMigrations(); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	return app
}

func assertImportedPeriodCount(t *testing.T, app *pocketbase.PocketBase) []artPeriod {
	t.Helper()

	periods, err := newRepository(app).listPeriods()
	if err != nil {
		t.Fatalf("listPeriods: %v", err)
	}
	if len(periods) != 32 {
		t.Fatalf("imported art periods = %d, want exactly 32", len(periods))
	}

	return periods
}

func TestImportedSeedProvidesExactly32ApprovedPeriods(t *testing.T) {
	app := newImportedTimelineApp(t)
	periods := assertImportedPeriodCount(t, app)

	// Source start_year/end_year must map verbatim to target start/end for a
	// spread of approved spans, proving the real import path carries the
	// producer chronology rather than synthesising it.
	expected := map[string][2]int{
		"Early Christian":               {100, 500},
		"Byzantine":                     {313, 1453},
		"Mannerism / Late Renaissance":  {1520, 1600},
		"Baroque":                       {1600, 1750},
		"Neoclassicism / Neoclassical":  {1750, 1800},
		"Impressionism":                 {1872, 1892},
		"Abstract Expressionism":        {1943, 1965},
		"Op Art":                        {1960, 1970},
	}
	found := map[string]artPeriod{}
	for _, period := range periods {
		found[period.name] = period
	}
	for name, span := range expected {
		period, ok := found[name]
		if !ok {
			t.Errorf("imported period %q missing", name)
			continue
		}
		if period.start != span[0] || period.end != span[1] {
			t.Errorf("imported period %q = (%d, %d), want source start_year/end_year (%d, %d)",
				name, period.start, period.end, span[0], span[1])
		}
	}

	// listPeriods must return a deterministic order (start, then name, then id).
	for i := 1; i < len(periods); i++ {
		prev, cur := periods[i-1], periods[i]
		if prev.start > cur.start {
			t.Errorf("periods not ordered by start: %q (%d) before %q (%d)", prev.name, prev.start, cur.name, cur.start)
		}
		if prev.start == cur.start && prev.name > cur.name {
			t.Errorf("periods not ordered by name on start tie: %q before %q", prev.name, cur.name)
		}
	}
}

func TestImportedSeedChronologySemantics(t *testing.T) {
	app := newImportedTimelineApp(t)
	repo := newRepository(app)

	artMin, artMax, err := repo.artworkBounds()
	if err != nil {
		t.Fatalf("artworkBounds: %v", err)
	}
	// Producer artwork dates span 101–1994; the period union widens the min to 100.
	if artMin != 101 || artMax != 1994 {
		t.Errorf("artworkBounds = (%d, %d), want (101, 1994)", artMin, artMax)
	}

	periods, err := repo.listPeriods()
	if err != nil {
		t.Fatalf("listPeriods: %v", err)
	}
	periodMin, periodMax := periodBounds(periods)
	min, max := unionBounds(artMin, artMax, periodMin, periodMax)
	if min != 100 || max != 1994 {
		t.Errorf("union bounds = (%d, %d), want (100, 1994)", min, max)
	}

	full, err := repo.countWorks(100, 1994)
	if err != nil {
		t.Fatalf("countWorks full: %v", err)
	}
	if full != 52866 {
		t.Errorf("full-range published works = %d, want 52866", full)
	}

	window, err := repo.countWorks(1500, 1600)
	if err != nil {
		t.Fatalf("countWorks window: %v", err)
	}
	if window != 12512 {
		t.Errorf("1500–1600 published works = %d, want 12512", window)
	}
}

func TestImportedSeedRendersFullShellAndHTMX(t *testing.T) {
	app := newImportedTimelineApp(t)
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
		fullBody := full.Body.String()
		if !strings.Contains(fullBody, "<html") {
			t.Error("full response did not render a document")
		}
		if got := full.Header().Get("HX-Push-Url"); got != "/timeline" {
			t.Errorf("HX-Push-Url = %q, want /timeline", got)
		}

		// The full range renders every approved preset: FULL RANGE plus 32 periods.
		if got := strings.Count(fullBody, `aria-current="page"`); got < 1 {
			t.Error("full response should mark the active preset")
		}
		for _, name := range []string{"FULL RANGE", "Early Christian", "Op Art", "Neoclassicism / Neoclassical"} {
			if !strings.Contains(fullBody, name) {
				t.Errorf("full response missing preset %q", name)
			}
		}

		// Bounded disclosure: at most 48 marks and 8 cards regardless of 52,866 works.
		assertImportedCaps(t, fullBody, 52866)

		// The timeline renders approved chronology without invented history.
		assertNoHistoricalEvents(t, fullBody)

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

		partial := httptest.NewRecorder()
		partialRequest := httptest.NewRequest(http.MethodGet, "/timeline?from=1500&to=1600", nil)
		partialRequest.Header.Set("HX-Request", "true")
		partialRequest.Header.Set("HX-Target", "timeline")
		mux.ServeHTTP(partial, partialRequest)
		if partial.Code != http.StatusOK {
			t.Fatalf("partial status = %d, want 200", partial.Code)
		}
		partialBody := partial.Body.String()
		if strings.Contains(partialBody, "<html") {
			t.Error("HTMX response should not render the full document")
		}
		if !strings.Contains(partialBody, `id="timeline"`) {
			t.Error("HTMX response should render the timeline block")
		}
		if got := partial.Header().Get("HX-Push-Url"); got != "/timeline?from=1500&to=1600" {
			t.Errorf("HX-Push-Url = %q, want /timeline?from=1500&to=1600", got)
		}
		if !strings.Contains(partialBody, "WORKS ACROSS THE WINDOW") {
			t.Error("HTMX window response should render the mark lane")
		}
		assertImportedCaps(t, partialBody, 12512)

		return nil
	}); err != nil {
		t.Fatalf("trigger serve event: %v", err)
	}
}

// assertImportedCaps verifies the rendered mark lane and works grid obey the
// 48-mark and 8-card disclosure caps against the given window total.
func assertImportedCaps(t *testing.T, body string, total int) {
	t.Helper()

	if got := strings.Count(body, "rotate-45"); got > markCap {
		t.Errorf("rendered marks = %d, want at most %d", got, markCap)
	}
	if got := strings.Count(body, `class="block hover:opacity-70`); got > worksPageSize {
		t.Errorf("rendered work cards = %d, want at most %d", got, worksPageSize)
	}
	want := fmt.Sprintf("MARKS SHOW THE FIRST %d OF %d WORKS", markCap, total)
	if !strings.Contains(body, want) {
		t.Errorf("response missing the truthful mark-cap note %q", want)
	}
}

// assertNoHistoricalEvents verifies the timeline presents only the approved
// art-period and published-artwork chronology, without fabricated event entries
// or prose (spec timeline-exploration / historical-event-data-unavailable).
func assertNoHistoricalEvents(t *testing.T, body string) {
	t.Helper()

	for _, marker := range []string{
		">Events<",
		"war broke out",
		"revolution",
		"battle of",
		"the discovery of",
		"Columbus",
		"Napoleon",
	} {
		if strings.Contains(body, marker) {
			t.Errorf("timeline rendered fabricated historical-event content %q", marker)
		}
	}
}

func TestImportedSeedImportsThroughRealPath(t *testing.T) {
	app := newImportedTimelineApp(t)

	// The import went through the real producer path, so the target now carries
	// the producer's published artwork count, not the synthetic fixture count.
	total, err := app.CountRecords("artworks")
	if err != nil {
		t.Fatalf("count artworks: %v", err)
	}
	if total != 52866 {
		t.Errorf("imported artworks = %d, want 52866", total)
	}

	periods, err := app.CountRecords("art_periods")
	if err != nil {
		t.Fatalf("count art_periods: %v", err)
	}
	if periods != 32 {
		t.Errorf("imported art_periods = %d, want 32", periods)
	}
}
