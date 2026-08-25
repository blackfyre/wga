package inspire

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/config"
	apputils "github.com/blackfyre/wga/internal/utils"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func TestInspirationWorksReturnsOnlyEligiblePublishedFirstAuthors(t *testing.T) {
	app := newInspirationTestApp(t)
	createInspirationArtist(t, app, "artistpublic001", "Public Artist", true)
	createInspirationArtist(t, app, "artistprivate01", "Private Artist", false)

	for index := range 11 {
		id := "workpublic" + strconv.Itoa(index + 100000)[1:]
		createInspirationArtwork(t, app, id, "Public Work "+strconv.Itoa(index), []string{"artistpublic001"}, true, "large.jpg", 900)
	}
	createInspirationArtwork(t, app, "artworkhidden01", "Hidden Work", []string{"artistpublic001"}, false, "hidden.jpg", 900)
	createInspirationArtwork(t, app, "artworkprivate1", "Private Work", []string{"artistprivate01"}, true, "private.jpg", 900)
	createInspirationArtwork(t, app, "artworkfirstpr1", "Wrong First Author", []string{"artistprivate01", "artistpublic001"}, true, "first.jpg", 900)
	createInspirationArtwork(t, app, "artworknoauth01", "Missing Author Work", []string{}, true, "missing.jpg", 900)

	works, err := inspirationWorks(app)
	if err != nil {
		t.Fatalf("inspirationWorks: %v", err)
	}
	if len(works) != 10 {
		t.Fatalf("works = %d, want 10", len(works))
	}
	for _, work := range works {
		if work.Title == "Missing Author Work" {
			t.Errorf("missing-author artwork must not appear in inspiration results")
		}
		if work.Artist.Name != "Public Artist" {
			t.Errorf("work %#v has an ineligible author", work)
		}
		if !strings.HasPrefix(work.Url, "/artists/public-artist-artistpublic001/") {
			t.Errorf("work URL = %q, want canonical public artist URL", work.Url)
		}
		if work.Image != "/api/files/artworks/"+work.Id+"/large.jpg?thumb=500x0" {
			t.Errorf("work image = %q, want 500px delivery URL", work.Image)
		}
	}
}

func TestInspirationWorksUsesOriginalAndNoImageFallback(t *testing.T) {
	app := newInspirationTestApp(t)
	createInspirationArtist(t, app, "artistpublic001", "Public Artist", true)
	createInspirationArtwork(t, app, "artworksmall001", "Small Work", []string{"artistpublic001"}, true, "small.jpg", 500)
	createInspirationArtwork(t, app, "artworkunknown1", "Unknown Work", []string{"artistpublic001"}, true, "unknown.jpg", 0)
	createInspirationArtwork(t, app, "artworknoimage1", "No Image Work", []string{"artistpublic001"}, true, "", 0)

	works, err := inspirationWorks(app)
	if err != nil {
		t.Fatalf("inspirationWorks: %v", err)
	}
	images := map[string]string{}
	for _, work := range works {
		images[work.Title] = work.Image
	}
	if got := images["Small Work"]; got != "/api/files/artworks/artworksmall001/small.jpg" {
		t.Errorf("small image = %q, want original", got)
	}
	if got := images["Unknown Work"]; got != "/api/files/artworks/artworkunknown1/unknown.jpg" {
		t.Errorf("unknown-width image = %q, want original", got)
	}
	if got := images["No Image Work"]; got != apputils.AssetUrl("/assets/images/no-image.png") {
		t.Errorf("no-image fallback = %q", got)
	}
}

func TestInspirationWorksHandlesEmptyEligibleData(t *testing.T) {
	works, err := inspirationWorks(newInspirationTestApp(t))
	if err != nil {
		t.Fatalf("inspirationWorks: %v", err)
	}
	if len(works) != 0 {
		t.Errorf("works = %#v, want empty", works)
	}
}

func TestInspirationRouteRendersCanonicalMetadataAndEmptyState(t *testing.T) {
	app := newInspirationTestApp(t)
	configureInspirationPublicURL(t)
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
		mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/inspire", nil))
		if response.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", response.Code, http.StatusOK)
		}
		if got := response.Header().Get("HX-Push-Url"); got != "/inspire" {
			t.Errorf("HX-Push-Url = %q, want /inspire", got)
		}
		for _, fragment := range []string{
			"Inspiration - WGA",
			"Explore a shuffled selection of works from the Web Gallery of Art collection.",
			`<link rel="canonical" href="https://gallery.example/inspire"`,
			"There are no published works to explore here yet.",
		} {
			if !strings.Contains(response.Body.String(), fragment) {
				t.Errorf("response does not contain %q", fragment)
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("trigger serve event: %v", err)
	}
}

func TestInspirationRouteRendersPublishedArtworkLinks(t *testing.T) {
	app := newInspirationTestApp(t)
	createInspirationArtist(t, app, "artistpublic001", "Public Artist", true)
	createInspirationArtwork(t, app, "artworkpublic01", "Public Work", []string{"artistpublic001"}, true, "work.jpg", 900)
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
		mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/inspire", nil))
		if response.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", response.Code, http.StatusOK)
		}
		if !strings.Contains(response.Body.String(), `href="/artists/public-artist-artistpublic001/public-work-artworkpublic01"`) {
			t.Error("response does not contain the published artwork's canonical link")
		}
		return nil
	}); err != nil {
		t.Fatalf("trigger serve event: %v", err)
	}
}

func TestInspirationRouteSelectsTargetAwareResponse(t *testing.T) {
	app := newInspirationTestApp(t)
	createInspirationArtist(t, app, "artistpublic001", "Public Artist", true)
	createInspirationArtwork(t, app, "artworkpublic01", "Public Work", []string{"artistpublic001"}, true, "work.jpg", 900)
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
		mux.ServeHTTP(full, httptest.NewRequest(http.MethodGet, "/inspire", nil))
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
			request := httptest.NewRequest(http.MethodGet, "/inspire", nil)
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
		request := httptest.NewRequest(http.MethodGet, "/inspire", nil)
		request.Header.Set("HX-Request", "true")
		request.Header.Set("HX-Target", "inspiration")
		mux.ServeHTTP(local, request)
		if local.Code != http.StatusOK {
			t.Errorf("local status = %d, want %d", local.Code, http.StatusOK)
		}
		if strings.Contains(local.Body.String(), "<html") {
			t.Error("feature-local response should not render the full document")
		}
		if got := strings.Count(local.Body.String(), `id="inspiration"`); got != 1 {
			t.Errorf("feature-local response rendered %d #inspiration elements, want exactly 1", got)
		}
		if strings.Contains(local.Body.String(), `id="mc-area"`) {
			t.Error("feature-local response must not carry #mc-area")
		}

		return nil
	}); err != nil {
		t.Fatalf("trigger serve event: %v", err)
	}
}

func newInspirationTestApp(t *testing.T) *pocketbase.PocketBase {
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
		&core.TextField{Id: "artist_profession", Name: "profession"},
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
		&core.TextField{Id: "artwork_image", Name: "image"},
		&core.NumberField{Id: "artwork_image_width", Name: "image_width"},
		&core.TextField{Id: "artwork_comment", Name: "comment"},
		&core.TextField{Id: "artwork_technique", Name: "technique"},
		&core.BoolField{Id: "artwork_published", Name: "published"},
		&core.RelationField{Id: "artwork_author", Name: "author", CollectionId: artists.Id, MinSelect: 1, MaxSelect: 10},
	)
	if err := app.Save(artworks); err != nil {
		t.Fatalf("save artworks collection: %v", err)
	}
	return app
}

func createInspirationArtist(t *testing.T, app *pocketbase.PocketBase, id, name string, published bool) {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId("artists")
	if err != nil {
		t.Fatalf("find artists collection: %v", err)
	}
	record := core.NewRecord(collection)
	record.Id = id
	record.Set("name", name)
	record.Set("published", published)
	if err := app.Save(record); err != nil {
		t.Fatalf("save artist: %v", err)
	}
}

func createInspirationArtwork(t *testing.T, app *pocketbase.PocketBase, id, title string, authors []string, published bool, image string, imageWidth int) {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId("artworks")
	if err != nil {
		t.Fatalf("find artworks collection: %v", err)
	}
	record := core.NewRecord(collection)
	record.Id = id
	record.Set("title", title)
	record.Set("author", authors)
	record.Set("published", published)
	record.Set("image", image)
	record.Set("image_width", imageWidth)
	if err := app.Save(record); err != nil {
		t.Fatalf("save artwork: %v", err)
	}
}

func configureInspirationPublicURL(t *testing.T) {
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
