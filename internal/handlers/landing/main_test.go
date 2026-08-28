package landing

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/repositories"
	apputils "github.com/blackfyre/wga/internal/utils"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func TestBuildHomePageUsesOnlyEligiblePublishedArtworks(t *testing.T) {
	app := newLandingTestApp(t)
	createLandingArtist(t, app, "artistalice0001", "Alice", "Alice, Filing", true)
	createLandingArtist(t, app, "artistbob000001", "Bob", "Bob, Filing", false)
	createLandingSchool(t, app)
	createLandingArtwork(t, app, "artworkalpha001", "Alpha Work", "artistalice0001", true, "2026-01-01 00:00:00.000Z", "small.jpg", 500)
	createLandingArtwork(t, app, "artworkbravo001", "Bravo Work", "artistalice0001", true, "2026-01-03 00:00:00.000Z", "large.jpg", 1200)
	createLandingArtwork(t, app, "artworkhidden01", "Hidden Work", "artistalice0001", false, "2026-01-04 00:00:00.000Z", "hidden.jpg", 1200)
	createLandingArtwork(t, app, "artworkprivate1", "Private Artist Work", "artistbob000001", true, "2026-01-05 00:00:00.000Z", "private.jpg", 1200)

	repo := repositories.NewLandingRepository(app)
	firstDate := time.Date(2026, time.January, 1, 18, 0, 0, 0, time.FixedZone("west", -5*60*60))
	page, err := buildHomePage(repo, firstDate)
	if err != nil {
		t.Fatalf("build home page: %v", err)
	}
	secondPage, err := buildHomePage(repo, firstDate)
	if err != nil {
		t.Fatalf("build same-day home page: %v", err)
	}

	if page.ArtistCount != "1" || page.ArtworkCount != "3" || page.SchoolCount != "1" {
		t.Errorf("counts = %q, %q, %q; want 1, 3, 1", page.ArtistCount, page.ArtworkCount, page.SchoolCount)
	}
	if page.FeaturedArtwork.Title != "Alpha Work" || secondPage.FeaturedArtwork != page.FeaturedArtwork {
		t.Errorf("same-day featured artwork = %#v, %#v; want stable Alpha Work", page.FeaturedArtwork, secondPage.FeaturedArtwork)
	}
	if page.FeaturedArtwork.Year != "1500" || page.RecentArtworks[0].Year != "1500" {
		t.Errorf("legacy home artwork years = %q, %q; want 1500", page.FeaturedArtwork.Year, page.RecentArtworks[0].Year)
	}
	if page.FeaturedArtwork.URL != "/artists/alice-artistalice0001/alpha-work-artworkalpha001" {
		t.Errorf("featured URL = %q", page.FeaturedArtwork.URL)
	}
	if page.FeaturedArtwork.Image != "/api/files/artworks/artworkalpha001/small.jpg" {
		t.Errorf("featured image = %q; want original without upscale", page.FeaturedArtwork.Image)
	}
	if len(page.RecentArtworks) != 2 {
		t.Fatalf("recent artworks = %#v; want two eligible works", page.RecentArtworks)
	}
	if page.RecentArtworks[0].Title != "Bravo Work" || page.RecentArtworks[1].Title != "Alpha Work" {
		t.Errorf("recent artworks = %#v; want newest eligible first", page.RecentArtworks)
	}
	if page.RecentArtworks[0].Image != "/api/files/artworks/artworkbravo001/large.jpg?thumb=500x0" {
		t.Errorf("recent image = %q; want 500px delivery profile", page.RecentArtworks[0].Image)
	}
	if page.RecentArtworks[1].Image != "/api/files/artworks/artworkalpha001/small.jpg" {
		t.Errorf("recent image = %q; want original without upscale", page.RecentArtworks[1].Image)
	}

	nextDay, err := buildHomePage(repo, firstDate.AddDate(0, 0, 1))
	if err != nil {
		t.Fatalf("build next-day home page: %v", err)
	}
	if nextDay.FeaturedArtwork.Title != "Bravo Work" {
		t.Errorf("next-day featured artwork = %q; want Bravo Work", nextDay.FeaturedArtwork.Title)
	}
	if nextDay.FeaturedArtwork.Image != "/api/files/artworks/artworkbravo001/large.jpg?thumb=900x0" {
		t.Errorf("next-day featured image = %q; want 900px delivery profile", nextDay.FeaturedArtwork.Image)
	}
}

func TestBuildHomePageUsesFilingNameBylines(t *testing.T) {
	app := newLandingTestApp(t)
	createLandingArtist(t, app, "artistalice0001", "Alice", "ALICE, Filing", true)
	createLandingSchool(t, app)
	createLandingArtwork(t, app, "artworkalpha001", "Alpha Work", "artistalice0001", true, "2026-01-01 00:00:00.000Z", "small.jpg", 500)

	page, err := buildHomePage(repositories.NewLandingRepository(app), time.Date(2026, time.January, 1, 18, 0, 0, 0, time.FixedZone("west", -5*60*60)))
	if err != nil {
		t.Fatalf("build home page: %v", err)
	}

	if page.FeaturedArtwork.Artist != "ALICE, Filing" {
		t.Errorf("featured byline = %q, want filing form %q", page.FeaturedArtwork.Artist, "ALICE, Filing")
	}
	if page.FeaturedArtwork.URL != "/artists/alice-artistalice0001/alpha-work-artworkalpha001" {
		t.Errorf("featured URL = %q, want legacy-name route", page.FeaturedArtwork.URL)
	}
	if len(page.RecentArtworks) != 1 || page.RecentArtworks[0].Artist != "ALICE, Filing" {
		t.Errorf("recent byline = %#v, want filing form", page.RecentArtworks)
	}
}

func TestBuildHomePageUsesArtworkDateEnd(t *testing.T) {
	app := newLandingTestApp(t)
	createLandingArtist(t, app, "artistalice0001", "Alice", "Alice, Filing", true)
	createLandingSchool(t, app)
	createLandingArtwork(t, app, "artworkalpha001", "Alpha Work", "artistalice0001", true, "2026-01-01 00:00:00.000Z", "small.jpg", 500)

	artwork, err := app.FindRecordById("artworks", "artworkalpha001")
	if err != nil {
		t.Fatalf("find artwork: %v", err)
	}
	artwork.Set("date_end", 1502)
	if err := app.Save(artwork); err != nil {
		t.Fatalf("save artwork date end: %v", err)
	}

	page, err := buildHomePage(repositories.NewLandingRepository(app), time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("build home page: %v", err)
	}
	if page.FeaturedArtwork.Year != "1502" || page.RecentArtworks[0].Year != "1502" {
		t.Errorf("home artwork years = %q, %q; want date_end 1502", page.FeaturedArtwork.Year, page.RecentArtworks[0].Year)
	}
}

func TestBuildHomePageOmitsBlankIdentityArtworks(t *testing.T) {
	app := newLandingTestApp(t)
	createLandingArtist(t, app, "artistblank0001", "Blank Identity", "", true)
	createLandingSchool(t, app)
	createLandingArtwork(t, app, "artworkblank001", "Blank Work", "artistblank0001", true, "2026-01-01 00:00:00.000Z", "blank.jpg", 500)

	page, err := buildHomePage(repositories.NewLandingRepository(app), time.Date(2026, time.January, 1, 18, 0, 0, 0, time.FixedZone("west", -5*60*60)))
	if err != nil {
		t.Fatalf("build home page: %v", err)
	}

	if page.FeaturedArtwork.Title != "" {
		t.Errorf("featured artwork = %#v, want none (blank identity fail closed)", page.FeaturedArtwork)
	}
	if len(page.RecentArtworks) != 0 {
		t.Errorf("recent artworks = %#v, want none (blank identity fail closed)", page.RecentArtworks)
	}
}

func TestBuildHomePageHandlesEmptyCollection(t *testing.T) {
	page, err := buildHomePage(repositories.NewLandingRepository(newLandingTestApp(t)), time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("build empty home page: %v", err)
	}
	if page.ArtistCount != "0" || page.ArtworkCount != "0" || page.SchoolCount != "0" {
		t.Errorf("empty counts = %#v", page)
	}
	if page.FeaturedArtwork.Title != "" || len(page.RecentArtworks) != 0 {
		t.Errorf("empty page artworks = %#v", page)
	}
}

func TestBuildHomePageLimitsRecentArtworksToFour(t *testing.T) {
	app := newLandingTestApp(t)
	createLandingArtist(t, app, "artistalice0001", "Alice", "Alice, Filing", true)
	for index := 1; index <= 5; index++ {
		value := strconv.Itoa(index)
		id := "artworkrecent0" + value
		created := "2026-01-0" + value + " 00:00:00.000Z"
		createLandingArtwork(t, app, id, "Recent Work "+value, "artistalice0001", true, created, "recent.jpg", 1200)
	}

	page, err := buildHomePage(repositories.NewLandingRepository(app), time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("build home page: %v", err)
	}
	if len(page.RecentArtworks) != recentArtworkLimit {
		t.Fatalf("recent artwork count = %d; want %d", len(page.RecentArtworks), recentArtworkLimit)
	}
	if page.RecentArtworks[0].Title != "Recent Work 5" || page.RecentArtworks[3].Title != "Recent Work 2" {
		t.Errorf("recent artwork order = %#v; want newest four first", page.RecentArtworks)
	}
}

func TestDailyArtworkIndexUsesUTCCalendarDaysAndStaysInBounds(t *testing.T) {
	tests := []struct {
		name  string
		date  time.Time
		count int
		want  int
	}{
		{name: "zero count", date: time.Unix(0, 0), count: 0, want: 0},
		{name: "negative count", date: time.Unix(0, 0), count: -1, want: 0},
		{name: "before epoch", date: time.Date(1969, time.December, 31, 23, 0, 0, 0, time.UTC), count: 3, want: 2},
		{name: "UTC day ignores source zone", date: time.Date(2026, time.January, 1, 0, 30, 0, 0, time.FixedZone("east", 2*60*60)), count: 3, want: 2},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := dailyArtworkIndex(test.date, test.count)
			if got != test.want {
				t.Fatalf("dailyArtworkIndex() = %d, want %d", got, test.want)
			}
			if test.count > 0 && (got < 0 || got >= test.count) {
				t.Fatalf("dailyArtworkIndex() = %d, want index in [0, %d)", got, test.count)
			}
		})
	}
}

func TestHomeRouteRendersMetadataForAnEmptyEligibleDataset(t *testing.T) {
	app := newLandingTestApp(t)
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

		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
		if response.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", response.Code, http.StatusOK)
		}
		if got := response.Header().Get("HX-Push-Url"); got != "/" {
			t.Errorf("HX-Push-Url = %q, want /", got)
		}

		body := response.Body.String()
		for _, expected := range []struct {
			name      string
			fragments []string
		}{
			{name: "title", fragments: []string{"<title>", "Web Gallery of Art | Explore artists and artworks - WGA"}},
			{name: "description", fragments: []string{`<meta name="description"`, "Explore artists, artworks, and side-by-side comparisons in the Web Gallery of Art."}},
			{name: "Open Graph URL", fragments: []string{`<meta name="og:url"`, "https://gallery.example/"}},
			{name: "canonical", fragments: []string{`<link rel="canonical"`, "https://gallery.example/"}},
		} {
			for _, fragment := range expected.fragments {
				if !strings.Contains(body, fragment) {
					t.Errorf("response %s does not contain %q", expected.name, fragment)
				}
			}
		}

		return nil
	}); err != nil {
		t.Fatalf("trigger serve event: %v", err)
	}
}

func newLandingTestApp(t *testing.T) *pocketbase.PocketBase {
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

	artists := core.NewBaseCollection("Artists")
	artists.Id = "test_artists"
	artists.MarkAsNew()
	artists.Fields.Add(
		&core.TextField{Id: "artist_name", Name: "name", Required: true},
		&core.TextField{Id: "artist_filing_name", Name: "filing_name"},
		&core.TextField{Id: "artist_short_name", Name: "short_name"},
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
		&core.NumberField{Id: "artwork_date_end", Name: "date_end"},
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

func createLandingSchool(t *testing.T, app *pocketbase.PocketBase) {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId("schools")
	if err != nil {
		t.Fatalf("find schools collection: %v", err)
	}
	if err := app.Save(core.NewRecord(collection)); err != nil {
		t.Fatalf("save school: %v", err)
	}
}

func createLandingArtist(t *testing.T, app *pocketbase.PocketBase, id string, name string, filingName string, published bool) {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId("artists")
	if err != nil {
		t.Fatalf("find artists collection: %v", err)
	}
	record := core.NewRecord(collection)
	record.Id = id
	record.Set("name", name)
	record.Set("filing_name", filingName)
	record.Set("short_name", filingName)
	record.Set("published", published)
	if err := app.Save(record); err != nil {
		t.Fatalf("save artist: %v", err)
	}
}

func createLandingArtwork(t *testing.T, app *pocketbase.PocketBase, id string, title string, author string, published bool, created string, image string, width int) {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId("artworks")
	if err != nil {
		t.Fatalf("find artworks collection: %v", err)
	}
	record := core.NewRecord(collection)
	record.Id = id
	record.Set("title", title)
	record.Set("year", "1500")
	record.Set("author", []string{author})
	record.Set("published", published)
	record.Set("image", image)
	record.Set("image_width", width)
	record.Set("created", created)
	if err := app.Save(record); err != nil {
		t.Fatalf("save artwork: %v", err)
	}
}
