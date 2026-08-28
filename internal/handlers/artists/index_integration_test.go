package artists

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	neturl "net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/blackfyre/wga/internal/config"
	apputils "github.com/blackfyre/wga/internal/utils"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func newArtistsIndexApp(t *testing.T) *pocketbase.PocketBase {
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

	schools := core.NewBaseCollection("Schools")
	schools.Id = "test_schools"
	schools.MarkAsNew()
	schools.Fields.Add(
		&core.TextField{Id: "school_name", Name: "name", Required: true},
		&core.TextField{Id: "school_slug", Name: "slug"},
	)
	if err := app.Save(schools); err != nil {
		t.Fatalf("save schools: %v", err)
	}

	periods := core.NewBaseCollection("Art_periods")
	periods.Id = "test_art_periods"
	periods.MarkAsNew()
	periods.Fields.Add(
		&core.TextField{Id: "period_name", Name: "name", Required: true},
		&core.NumberField{Id: "period_start", Name: "start"},
		&core.NumberField{Id: "period_end", Name: "end"},
	)
	if err := app.Save(periods); err != nil {
		t.Fatalf("save art_periods: %v", err)
	}

	artists := core.NewBaseCollection("Artists")
	artists.Id = "test_artists"
	artists.MarkAsNew()
	artists.Fields.Add(
		&core.TextField{Id: "artist_name", Name: "name", Required: true},
		&core.TextField{Id: "artist_filing_name", Name: "filing_name"},
		&core.TextField{Id: "artist_short_name", Name: "short_name"},
		&core.TextField{Id: "artist_slug", Name: "slug"},
		&core.NumberField{Id: "artist_yob", Name: "year_of_birth"},
		&core.NumberField{Id: "artist_yod", Name: "year_of_death"},
		&core.TextField{Id: "artist_profession", Name: "profession"},
		&core.TextField{Id: "artist_portrait", Name: "portrait"},
		&core.NumberField{Id: "artist_portrait_width", Name: "biography_image_width"},
		&core.RelationField{Id: "artist_school", Name: "school", CollectionId: schools.Id, MinSelect: 0, MaxSelect: 10},
		&core.BoolField{Id: "artist_published", Name: "published"},
	)
	if err := app.Save(artists); err != nil {
		t.Fatalf("save artists: %v", err)
	}

	artworks := core.NewBaseCollection("Artworks")
	artworks.Id = "test_artworks"
	artworks.MarkAsNew()
	artworks.Fields.Add(
		&core.TextField{Id: "artwork_title", Name: "title", Required: true},
		&core.RelationField{Id: "artwork_author", Name: "author", CollectionId: artists.Id, MinSelect: 1, MaxSelect: 10},
		&core.BoolField{Id: "artwork_published", Name: "published"},
	)
	if err := app.Save(artworks); err != nil {
		t.Fatalf("save artworks: %v", err)
	}

	return app
}

type indexArtistSeed struct {
	id            string
	name          string
	filingName    string
	shortName     string
	slug          string
	birth         int
	death         int
	profession    string
	portrait      string
	portraitWidth int
	schools       []string
	published     bool
}

func saveIndexArtist(t *testing.T, app *pocketbase.PocketBase, seed indexArtistSeed) {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId("artists")
	if err != nil {
		t.Fatalf("find artists: %v", err)
	}
	record := core.NewRecord(collection)
	record.Id = seed.id
	record.Set("name", seed.name)
	filingName := seed.filingName
	if filingName == "" {
		filingName = seed.name
	}
	shortName := seed.shortName
	if shortName == "" {
		shortName = seed.name
	}
	record.Set("filing_name", filingName)
	record.Set("short_name", shortName)
	record.Set("slug", seed.slug)
	record.Set("year_of_birth", seed.birth)
	record.Set("year_of_death", seed.death)
	record.Set("profession", seed.profession)
	if seed.portrait != "" {
		record.Set("portrait", seed.portrait)
	}
	record.Set("biography_image_width", seed.portraitWidth)
	record.Set("school", seed.schools)
	record.Set("published", seed.published)
	if err := app.Save(record); err != nil {
		t.Fatalf("save artist %s: %v", seed.id, err)
	}
}

func saveIndexSchool(t *testing.T, app *pocketbase.PocketBase, id, slug, name string) {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId("schools")
	if err != nil {
		t.Fatalf("find schools: %v", err)
	}
	record := core.NewRecord(collection)
	record.Id = id
	record.Set("name", name)
	record.Set("slug", slug)
	if err := app.Save(record); err != nil {
		t.Fatalf("save school %s: %v", id, err)
	}
}

func saveIndexArtwork(t *testing.T, app *pocketbase.PocketBase, id string, authors []string, published bool) {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId("artworks")
	if err != nil {
		t.Fatalf("find artworks: %v", err)
	}
	record := core.NewRecord(collection)
	record.Id = id
	record.Set("title", "Work "+id)
	record.Set("author", authors)
	record.Set("published", published)
	if err := app.Save(record); err != nil {
		t.Fatalf("save artwork %s: %v", id, err)
	}
}

func TestBuildArtistIndexViewUsesCanonicalArtistURLAndPortrait(t *testing.T) {
	app := newArtistsIndexApp(t)
	// Portrait source width 800 > 500 target, so the 500 profile is used.
	saveIndexArtist(t, app, indexArtistSeed{
		id: "artistrembrandt", name: "Rembrandt van Rijn", filingName: "Rijn, Rembrandt van", shortName: "Rembrandt", slug: "rembrandt-van-rijn",
		birth: 1606, death: 1669, profession: "painter",
		portrait: "portrait.jpg", portraitWidth: 800, published: true,
	})
	// Portrait source width 300 <= 500 target, so the original is served.
	saveIndexArtist(t, app, indexArtistSeed{
		id: "artistvermeer00", name: "Johannes Vermeer", filingName: "Vermeer, Johannes", shortName: "Vermeer", slug: "johannes-vermeer",
		birth: 1632, death: 1675, profession: "painter",
		portrait: "small.jpg", portraitWidth: 300, published: true,
	})

	view, canonical, err := buildArtistIndexView(app, neturl.Values{})
	if err != nil {
		t.Fatalf("build view: %v", err)
	}
	if canonical != "/artists" {
		t.Errorf("canonical = %q, want /artists", canonical)
	}
	if len(view.Artists) != 2 {
		t.Fatalf("artists = %d, want 2", len(view.Artists))
	}

	byName := map[string]string{}
	for _, artist := range view.Artists {
		byName[artist.Name] = artist.URL
	}
	if byName["Rijn, Rembrandt van"] != "/artists/rembrandt-van-rijn-artistrembrandt" {
		t.Errorf("canonical URL = %q, want helper output", byName["Rijn, Rembrandt van"])
	}

	thumbs := map[string]string{}
	for _, artist := range view.Artists {
		thumbs[artist.Name] = artist.Thumb
	}
	if thumbs["Rijn, Rembrandt van"] != "/api/files/artists/artistrembrandt/portrait.jpg?thumb=500x0" {
		t.Errorf("500px portrait = %q", thumbs["Rijn, Rembrandt van"])
	}
	if thumbs["Vermeer, Johannes"] != "/api/files/artists/artistvermeer00/small.jpg" {
		t.Errorf("original portrait = %q, want original without upscale", thumbs["Vermeer, Johannes"])
	}
}

func TestBuildArtistIndexViewMarksAvailability(t *testing.T) {
	app := newArtistsIndexApp(t)
	saveIndexArtist(t, app, indexArtistSeed{id: "artistavailable", name: "Available Artist", published: true})
	saveIndexArtist(t, app, indexArtistSeed{id: "artistunavail01", name: "Unavailable Artist", published: true})
	saveIndexArtwork(t, app, "workavail000001", []string{"artistavailable"}, true)
	saveIndexArtwork(t, app, "workhidden00001", []string{"artistunavail01"}, false)

	view, _, err := buildArtistIndexView(app, neturl.Values{})
	if err != nil {
		t.Fatalf("build view: %v", err)
	}

	availability := map[string]bool{}
	for _, artist := range view.Artists {
		availability[artist.Name] = artist.Available
	}
	if !availability["Available Artist"] {
		t.Error("Available Artist should be available via a published artwork")
	}
	if availability["Unavailable Artist"] {
		t.Error("Unavailable Artist should be unavailable (only unpublished artworks)")
	}
}

func TestBuildArtistIndexViewBornRangeWhenBoundsExist(t *testing.T) {
	app := newArtistsIndexApp(t)
	saveIndexArtist(t, app, indexArtistSeed{id: "artistlow000001", name: "Low", birth: 1200, published: true})
	saveIndexArtist(t, app, indexArtistSeed{id: "artisthigh00001", name: "High", birth: 1900, published: true})

	view, _, err := buildArtistIndexView(app, neturl.Values{})
	if err != nil {
		t.Fatalf("build view: %v", err)
	}
	if !view.HasBirthBounds {
		t.Fatal("expected HasBirthBounds when published birth years exist")
	}
	if view.BornRange.Min != 1200 || view.BornRange.Max != 1900 {
		t.Errorf("born bounds = (%d, %d), want (1200, 1900)", view.BornRange.Min, view.BornRange.Max)
	}
	if view.BornRange.FromValue != 1200 || view.BornRange.ToValue != 1900 {
		t.Errorf("born values = (%d, %d), want normalized (1200, 1900)", view.BornRange.FromValue, view.BornRange.ToValue)
	}

	clamped, _, err := buildArtistIndexView(app, neturl.Values{"born_from": {"1000"}, "born_to": {"9999"}})
	if err != nil {
		t.Fatalf("build clamped view: %v", err)
	}
	if clamped.BornRange.FromValue != 1200 || clamped.BornRange.ToValue != 1900 {
		t.Errorf("clamped born values = (%d, %d), want (1200, 1900)", clamped.BornRange.FromValue, clamped.BornRange.ToValue)
	}
}

func TestBuildArtistIndexViewNoBornRangeWithoutBounds(t *testing.T) {
	app := newArtistsIndexApp(t)
	view, _, err := buildArtistIndexView(app, neturl.Values{})
	if err != nil {
		t.Fatalf("build view: %v", err)
	}
	if view.HasBirthBounds {
		t.Error("expected no birth bounds when no published artist has a birth year")
	}
}

func TestBuildArtistIndexViewClearsUnknownSchool(t *testing.T) {
	app := newArtistsIndexApp(t)
	saveIndexSchool(t, app, "schooldutch0001", "dutch", "Dutch")
	saveIndexArtist(t, app, indexArtistSeed{id: "artistone000001", name: "Artist", schools: []string{"schooldutch0001"}, published: true})

	view, canonical, err := buildArtistIndexView(app, neturl.Values{"school": {"unknown-school"}})
	if err != nil {
		t.Fatalf("build view: %v", err)
	}
	if canonical != "/artists" {
		t.Errorf("canonical = %q, want /artists (unknown school dropped)", canonical)
	}
	for _, option := range view.Schools {
		if option.Value != "" && option.Checked {
			t.Errorf("no known school option should be checked after unknown-school normalisation, got %#v", option)
		}
	}
}

func TestBuildArtistIndexViewClampsPageWhenEmpty(t *testing.T) {
	app := newArtistsIndexApp(t)
	saveIndexArtist(t, app, indexArtistSeed{id: "artistone000001", name: "Only Artist", published: true})

	_, canonical, err := buildArtistIndexView(app, neturl.Values{"q": {"no-such-artist"}, "page": {"999"}})
	if err != nil {
		t.Fatalf("build view: %v", err)
	}
	if canonical != "/artists?q=no-such-artist" {
		t.Errorf("canonical = %q, want /artists?q=no-such-artist (out-of-range page reset when no results)", canonical)
	}
}

func TestBuildArtistIndexViewAllUrlPreservesFilters(t *testing.T) {
	app := newArtistsIndexApp(t)
	saveIndexSchool(t, app, "schooldutch0001", "dutch", "Dutch")
	saveIndexArtist(t, app, indexArtistSeed{id: "artistone000001", name: "Artist", schools: []string{"schooldutch0001"}, published: true})

	view, _, err := buildArtistIndexView(app, neturl.Values{"letter": {"A"}, "school": {"dutch"}, "page": {"1"}})
	if err != nil {
		t.Fatalf("build view: %v", err)
	}
	if view.AllUrl != "/artists?school=dutch" {
		t.Errorf("AllUrl = %q, want /artists?school=dutch (clears letter, preserves school, resets page)", view.AllUrl)
	}
	if view.ResetUrl != "/artists" {
		t.Errorf("ResetUrl = %q, want /artists (full reset)", view.ResetUrl)
	}
}

func TestBuildArtistIndexViewDiscardsBornBoundsWithoutKnownYears(t *testing.T) {
	app := newArtistsIndexApp(t)
	saveIndexArtist(t, app, indexArtistSeed{id: "artistone000001", name: "Artist", published: true})

	view, canonical, err := buildArtistIndexView(app, neturl.Values{"born_from": {"1600"}, "born_to": {"1700"}})
	if err != nil {
		t.Fatalf("build view: %v", err)
	}
	if view.HasBirthBounds {
		t.Error("expected no birth bounds when no published artist has a known birth year")
	}
	if canonical != "/artists" {
		t.Errorf("canonical = %q, want /artists (supplied bounds discarded without known-year range)", canonical)
	}
}

func TestBuildArtistIndexViewIssuesBoundedQueries(t *testing.T) {
	small := newArtistsIndexApp(t)
	for i := 0; i < 5; i++ {
		saveIndexArtist(t, small, indexArtistSeed{
			id: fmt.Sprintf("artsmall%07d", i), name: fmt.Sprintf("Small Artist %d", i),
			birth: 1500 + i, published: true,
		})
	}
	large := newArtistsIndexApp(t)
	for i := 0; i < 60; i++ {
		saveIndexArtist(t, large, indexArtistSeed{
			id: fmt.Sprintf("artlarge%07d", i), name: fmt.Sprintf("Large Artist %d", i),
			birth: 1500 + i, published: true,
		})
	}

	smallCount, err := countIndexQueries(small, func() error {
		_, _, err := buildArtistIndexView(small, neturl.Values{})
		return err
	})
	if err != nil {
		t.Fatalf("small build: %v", err)
	}
	largeCount, err := countIndexQueries(large, func() error {
		_, _, err := buildArtistIndexView(large, neturl.Values{})
		return err
	})
	if err != nil {
		t.Fatalf("large build: %v", err)
	}

	if smallCount == 0 {
		t.Fatal("expected to observe queries, got 0")
	}
	if smallCount != largeCount {
		t.Errorf("query count grew with artist count: %d (5 artists) vs %d (60 artists)", smallCount, largeCount)
	}
}

func countIndexQueries(app *pocketbase.PocketBase, fn func() error) (int, error) {
	concurrent, ok := app.ConcurrentDB().(*dbx.DB)
	if !ok {
		return 0, fmt.Errorf("ConcurrentDB is %T, want *dbx.DB", app.ConcurrentDB())
	}
	nonconcurrent, _ := app.NonconcurrentDB().(*dbx.DB)

	var count int64
	queryLog := func(_ context.Context, _ time.Duration, _ string, _ *sql.Rows, _ error) {
		atomic.AddInt64(&count, 1)
	}
	execLog := func(_ context.Context, _ time.Duration, _ string, _ sql.Result, _ error) {
		atomic.AddInt64(&count, 1)
	}

	concurrent.QueryLogFunc = queryLog
	concurrent.ExecLogFunc = execLog
	if nonconcurrent != nil {
		nonconcurrent.QueryLogFunc = queryLog
		nonconcurrent.ExecLogFunc = execLog
	}
	defer func() {
		concurrent.QueryLogFunc = nil
		concurrent.ExecLogFunc = nil
		if nonconcurrent != nil {
			nonconcurrent.QueryLogFunc = nil
			nonconcurrent.ExecLogFunc = nil
		}
	}()

	if err := fn(); err != nil {
		return 0, err
	}

	return int(atomic.LoadInt64(&count)), nil
}

func TestArtistsRouteRendersMetadataAndSingleHeading(t *testing.T) {
	app := newArtistsIndexApp(t)
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
	t.Cleanup(func() {
		apputils.ConfigurePublicURL(config.PublicURL{})
	})

	RegisterHandlers(app, configuration.Environment())

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

		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/artists", nil))
		if response.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", response.Code, http.StatusOK)
		}
		if got := response.Header().Get("HX-Push-Url"); got != "/artists" {
			t.Errorf("HX-Push-Url = %q, want /artists", got)
		}

		body := response.Body.String()
		for _, expected := range []string{
			"<title>", "Artists - WGA",
			`<meta name="description"`, "Check out the artists in the gallery.",
			`<link rel="canonical"`, "https://gallery.example/artists",
		} {
			if !strings.Contains(body, expected) {
				t.Errorf("response missing %q", expected)
			}
		}
		if count := strings.Count(body, "<h1"); count != 1 {
			t.Errorf("expected exactly one h1, got %d", count)
		}

		return nil
	}); err != nil {
		t.Fatalf("trigger serve event: %v", err)
	}
}

func TestArtistsRoutePushesCanonicalQueryURL(t *testing.T) {
	app := newArtistsIndexApp(t)
	apputils.ConfigurePublicURL(config.PublicURL{})
	t.Cleanup(func() {
		apputils.ConfigurePublicURL(config.PublicURL{})
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

		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/artists?view=grid&sort=az&page=1&letter=s", nil))
		if response.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
		}
		if got := response.Header().Get("HX-Push-Url"); got != "/artists?letter=S" {
			t.Errorf("HX-Push-Url = %q, want normalized /artists?letter=S", got)
		}

		return nil
	}); err != nil {
		t.Fatalf("trigger serve event: %v", err)
	}
}
