package artists

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/repositories"
	apputils "github.com/blackfyre/wga/internal/utils"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func newArtistRecordApp(t *testing.T) *pocketbase.PocketBase {
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

	saveRecordCollection(t, app, core.NewBaseCollection("Schools"), "schools",
		&core.TextField{Name: "name", Required: true},
		&core.TextField{Name: "slug"},
	)
	saveRecordCollection(t, app, core.NewBaseCollection("Art_periods"), "art_periods",
		&core.TextField{Name: "name", Required: true},
		&core.TextField{Name: "slug"},
		&core.NumberField{Name: "start"},
		&core.NumberField{Name: "end"},
	)

	artists := core.NewBaseCollection("Artists")
	artists.Id = "artists"
	artists.MarkAsNew()
	artists.Fields.Add(
		&core.TextField{Name: "name", Required: true},
		&core.TextField{Name: "slug"},
		&core.EditorField{Name: "bio"},
		&core.NumberField{Name: "year_of_birth"},
		&core.NumberField{Name: "year_of_death"},
		&core.TextField{Name: "place_of_birth"},
		&core.TextField{Name: "place_of_death"},
		&core.BoolField{Name: "exact_year_of_birth"},
		&core.BoolField{Name: "exact_year_of_death"},
		&core.TextField{Name: "profession"},
		&core.SelectField{Name: "known_place_of_birth", Values: []string{"yes", "no", "n/a"}, MaxSelect: 1},
		&core.SelectField{Name: "known_place_of_death", Values: []string{"yes", "no", "n/a"}, MaxSelect: 1},
		&core.RelationField{Name: "school", CollectionId: "schools", MinSelect: 0, MaxSelect: 10},
		&core.TextField{Name: "portrait"},
		&core.NumberField{Name: "biography_image_width"},
		&core.NumberField{Name: "biography_image_height"},
		&core.BoolField{Name: "published"},
	)
	if err := app.Save(artists); err != nil {
		t.Fatalf("save artists collection: %v", err)
	}
	artists.Fields.Add(&core.RelationField{Name: "also_known_as", CollectionId: "artists", MinSelect: 0, MaxSelect: 0})
	if err := app.Save(artists); err != nil {
		t.Fatalf("save artists also_known_as field: %v", err)
	}

	saveRecordCollection(t, app, core.NewBaseCollection("Artworks"), "artworks",
		&core.TextField{Name: "title", Required: true},
		&core.RelationField{Name: "author", CollectionId: "artists", MinSelect: 1, MaxSelect: 10},
		&core.TextField{Name: "technique"},
		&core.EditorField{Name: "comment"},
		&core.BoolField{Name: "published"},
		&core.TextField{Name: "image"},
		&core.NumberField{Name: "image_width"},
	)

	saveRecordCollection(t, app, core.NewBaseCollection("Art_selections"), "art_selections",
		&core.RelationField{Name: "artist", CollectionId: "artists", MinSelect: 1, MaxSelect: 1, Required: true},
		&core.TextField{Name: "title", Required: true},
		&core.TextField{Name: "context"},
		&core.TextField{Name: "display_title", Required: true},
		&core.EditorField{Name: "commentary"},
		&core.RelationField{Name: "artworks", CollectionId: "artworks", MinSelect: 1, MaxSelect: 1000, Required: true},
		&core.TextField{Name: "source_path", Required: true, Hidden: true},
		&core.TextField{Name: "source_hash", Required: true, Hidden: true},
		&core.TextField{Name: "content_hash", Required: true, Hidden: true},
		&core.BoolField{Name: "published", Required: true},
	)

	saveRecordCollection(t, app, core.NewBaseCollection("Glossary"), "glossary",
		&core.TextField{Name: "expression", Required: true},
		&core.TextField{Name: "definition", Required: true},
	)

	saveRecordCollection(t, app, core.NewBaseCollection("Music_composer"), "music_composer",
		&core.TextField{Name: "name", Required: true},
		&core.SelectField{Name: "century", Values: []string{"12", "13", "14", "15", "16", "17", "18", "19", "20", "21"}, MaxSelect: 1},
		&core.BoolField{Name: "published"},
	)

	saveRecordCollection(t, app, core.NewBaseCollection("Music_song"), "music_song",
		&core.TextField{Name: "title", Required: true},
		&core.RelationField{Name: "composer", CollectionId: "music_composer", MinSelect: 1, MaxSelect: 20},
		&core.TextField{Name: "source"},
		&core.BoolField{Name: "published"},
	)

	return app
}

func saveRecordCollection(t *testing.T, app *pocketbase.PocketBase, collection *core.Collection, id string, fields ...core.Field) {
	t.Helper()
	collection.Id = id
	collection.MarkAsNew()
	collection.Fields.Add(fields...)
	if err := app.Save(collection); err != nil {
		t.Fatalf("save %s collection: %v", id, err)
	}
}

func saveRecordRecord(t *testing.T, app *pocketbase.PocketBase, collection string, id string, fields map[string]any) {
	t.Helper()
	coll, err := app.FindCollectionByNameOrId(collection)
	if err != nil {
		t.Fatalf("find %s: %v", collection, err)
	}
	record := core.NewRecord(coll)
	record.Id = id
	for key, value := range fields {
		record.Set(key, value)
	}
	if err := app.Save(record); err != nil {
		t.Fatalf("save %s %s: %v", collection, id, err)
	}
}

func serveArtistRecordRequests(t *testing.T, app *pocketbase.PocketBase, requests []recordRequest) []*httptest.ResponseRecorder {
	t.Helper()
	RegisterHandlers(app, config.EnvironmentTest)

	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatalf("create router: %v", err)
	}
	serveEvent := &core.ServeEvent{App: app, Router: router}

	recorders := make([]*httptest.ResponseRecorder, len(requests))
	if err := app.OnServe().Trigger(serveEvent, func(event *core.ServeEvent) error {
		mux, err := event.Router.BuildMux()
		if err != nil {
			return err
		}
		for i, req := range requests {
			request := httptest.NewRequest(http.MethodGet, req.path, nil)
			if req.htmx {
				request.Header.Set("HX-Request", "true")
			}
			recorders[i] = httptest.NewRecorder()
			mux.ServeHTTP(recorders[i], request)
		}
		return nil
	}); err != nil {
		t.Fatalf("trigger serve event: %v", err)
	}

	return recorders
}

type recordRequest struct {
	path string
	htmx bool
}

func serveArtistRecordRequest(t *testing.T, app *pocketbase.PocketBase, path string, htmx bool) *httptest.ResponseRecorder {
	t.Helper()
	return serveArtistRecordRequests(t, app, []recordRequest{{path: path, htmx: htmx}})[0]
}

func seedPublishedArtist(t *testing.T, app *pocketbase.PocketBase) {
	t.Helper()
	saveRecordRecord(t, app, "schools", "schooldutch0001", map[string]any{"name": "Dutch", "slug": "dutch"})
	saveRecordRecord(t, app, "art_periods", "periodbaroque01", map[string]any{"name": "Baroque", "slug": "baroque", "start": 1600, "end": 1750})
	saveRecordRecord(t, app, "artists", "artistone000001", map[string]any{
		"name":                 "Synthetic Artist",
		"slug":                 "synthetic-artist",
		"bio":                  "<p>He trained in the <strong>workshop</strong> and used chiaroscuro.</p><script>alert(1)</script>",
		"year_of_birth":        1606,
		"year_of_death":        1669,
		"place_of_birth":       "Leiden",
		"place_of_death":       "Amsterdam",
		"profession":           "painter",
		"school":               []string{"schooldutch0001"},
		"portrait":             "portrait.jpg",
		"biography_image_width": 800,
		"published":            true,
	})
	saveRecordRecord(t, app, "artworks", "artworkone00001", map[string]any{
		"title": "Alpha Work", "author": []string{"artistone000001"}, "published": true, "image": "alpha.jpg", "image_width": 900,
	})
	saveRecordRecord(t, app, "artworks", "artworktwo00001", map[string]any{
		"title": "Beta Work", "author": []string{"artistone000001"}, "published": true,
	})
	saveRecordRecord(t, app, "artworks", "artworkhid00001", map[string]any{
		"title": "Hidden Work", "author": []string{"artistone000001"}, "published": false,
	})
	saveRecordRecord(t, app, "glossary", "glosschiaro0000", map[string]any{"expression": "chiaroscuro", "definition": "A treatment of light and shade."})
	saveRecordRecord(t, app, "music_composer", "composer1700000", map[string]any{"name": "Sweelinck", "century": "17", "published": true})
	saveRecordRecord(t, app, "music_song", "songone00000000", map[string]any{"title": "Fantasia chromatica", "composer": []string{"composer1700000"}, "source": "fantasia.mp3", "published": true})
}

func TestArtistRecordRouteRedirectsToCanonicalSlug(t *testing.T) {
	app := newArtistRecordApp(t)
	seedPublishedArtist(t, app)

	recorder := serveArtistRecordRequest(t, app, "/artists/wrong-slug-artistone000001", false)
	if recorder.Code != http.StatusMovedPermanently {
		t.Fatalf("status = %d, want 301", recorder.Code)
	}
	if got := recorder.Header().Get("Location"); got != "/artists/synthetic-artist-artistone000001" {
		t.Errorf("Location = %q, want canonical /artists/synthetic-artist-artistone000001", got)
	}
}

func TestArtistRecordRouteRejectsUnpublishedArtist(t *testing.T) {
	app := newArtistRecordApp(t)
	saveRecordRecord(t, app, "artists", "artisthid100000", map[string]any{"name": "Hidden Artist", "slug": "hidden-artist", "published": false})

	recorder := serveArtistRecordRequest(t, app, "/artists/hidden-artist-artisthid100000", false)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
}

func TestArtistRecordRouteRendersFullAndHTMX(t *testing.T) {
	app := newArtistRecordApp(t)
	seedPublishedArtist(t, app)

	recorders := serveArtistRecordRequests(t, app, []recordRequest{
		{path: "/artists/synthetic-artist-artistone000001", htmx: false},
		{path: "/artists/synthetic-artist-artistone000001", htmx: true},
	})
	full := recorders[0]
	partial := recorders[1]

	if full.Code != http.StatusOK {
		t.Fatalf("full status = %d, want 200", full.Code)
	}
	if !strings.Contains(full.Body.String(), "<html") || !strings.Contains(full.Body.String(), "Synthetic Artist") {
		t.Error("full response should render the complete document")
	}
	if got := full.Header().Get("HX-Push-Url"); got != "/artists/synthetic-artist-artistone000001" {
		t.Errorf("HX-Push-Url = %q, want canonical", got)
	}

	if partial.Code != http.StatusOK {
		t.Fatalf("partial status = %d, want 200", partial.Code)
	}
	if strings.Contains(partial.Body.String(), "<html") {
		t.Error("HTMX response should not render the full document")
	}
	if !strings.Contains(partial.Body.String(), `<main id="mc-area"`) {
		t.Error("HTMX response should render the main content area block")
	}
	if !strings.Contains(partial.Body.String(), "Synthetic Artist") {
		t.Error("HTMX response should render the artist record")
	}
}

func TestArtistRecordViewResolvesMetadataPortraitAndParity(t *testing.T) {
	app := newArtistRecordApp(t)
	seedPublishedArtist(t, app)

	artist, err := repositoriesFindPublishedArtist(app, "artistone000001")
	if err != nil {
		t.Fatalf("find artist: %v", err)
	}
	view, err := buildArtistRecordView(app, artist)
	if err != nil {
		t.Fatalf("build view: %v", err)
	}

	if view.Name != "Synthetic Artist" {
		t.Errorf("name = %q", view.Name)
	}
	if view.LifeSummary != "b. 1606 Leiden, d. 1669 Amsterdam" {
		t.Errorf("life summary = %q", view.LifeSummary)
	}
	if view.Schools != "Dutch" {
		t.Errorf("schools = %q, want Dutch", view.Schools)
	}
	if view.Period != "Baroque" {
		t.Errorf("period = %q, want Baroque", view.Period)
	}
	if view.Profession != "painter" {
		t.Errorf("profession = %q", view.Profession)
	}
	if want := "/api/files/artists/artistone000001/portrait.jpg?thumb=600x0"; view.Portrait != want {
		t.Errorf("portrait = %q, want 600-profile %q", view.Portrait, want)
	}
	if view.WorkCount != 2 {
		t.Errorf("work count = %d, want 2 (unpublished excluded)", view.WorkCount)
	}
	if view.WorksURL != "/artworks?artist=Synthetic+Artist" {
		t.Errorf("works URL = %q", view.WorksURL)
	}
	if view.Citation.Key != "wga-synthetic-artist" {
		t.Errorf("citation key = %q", view.Citation.Key)
	}

	// Visible portrait, Open Graph image, and Person.image JSON-LD match.
	og := artistOpenGraphImage(view)
	if og != apputils.AssetUrl(view.Portrait) {
		t.Errorf("Open Graph image = %q, want %q", og, apputils.AssetUrl(view.Portrait))
	}
	if !strings.Contains(view.Jsonld, apputils.AssetUrl(view.Portrait)) {
		t.Errorf("JSON-LD missing portrait image %q", apputils.AssetUrl(view.Portrait))
	}
}

func TestArtistRecordViewSanitisesBiographyAndAnnotatesGlossary(t *testing.T) {
	app := newArtistRecordApp(t)
	seedPublishedArtist(t, app)

	artist, err := repositoriesFindPublishedArtist(app, "artistone000001")
	if err != nil {
		t.Fatalf("find artist: %v", err)
	}
	view, err := buildArtistRecordView(app, artist)
	if err != nil {
		t.Fatalf("build view: %v", err)
	}

	if strings.Contains(view.Bio, "<script") {
		t.Error("biography must not retain script markup")
	}
	if !strings.Contains(view.Bio, "<strong>workshop</strong>") {
		t.Error("biography should retain legitimate structure")
	}
	for _, expected := range []string{
		`class="wga-term"`,
		`role="note"`,
		`tabindex="0"`,
		`aria-label="chiaroscuro: A treatment of light and shade."`,
		`class="wga-tooltip__body">A treatment of light and shade.</span>`,
	} {
		if !strings.Contains(view.Bio, expected) {
			t.Errorf("biography should annotate glossary terms with the task-4.4 contract, missing %q", expected)
		}
	}
	if strings.Contains(view.Bio, `class="glossary-term"`) {
		t.Error("biography should not use the legacy glossary-term annotation")
	}
}

func TestArtistRecordViewDegradesWithoutGlossary(t *testing.T) {
	app := newArtistRecordApp(t)
	seedPublishedArtist(t, app)
	// Remove the glossary collection so loading entries fails. The biography
	// must still render sanitized and readable.
	glossaryCollection, err := app.FindCollectionByNameOrId("glossary")
	if err != nil {
		t.Fatalf("find glossary: %v", err)
	}
	if err := app.Delete(glossaryCollection); err != nil {
		t.Fatalf("delete glossary: %v", err)
	}

	artist, err := repositories.NewArtistRecordRepository(app).FindPublishedArtist("artistone000001")
	if err != nil {
		t.Fatalf("find artist: %v", err)
	}
	view, err := buildArtistRecordView(app, artist)
	if err != nil {
		t.Fatalf("build view: %v", err)
	}

	if strings.Contains(view.Bio, "<script") {
		t.Error("biography must not retain script markup")
	}
	if !strings.Contains(view.Bio, "<strong>workshop</strong>") {
		t.Error("biography should remain readable without glossary")
	}
}

func TestArtistRecordViewUsesOriginalPortraitWhenNarrower(t *testing.T) {
	app := newArtistRecordApp(t)
	seedPublishedArtist(t, app)
	saveRecordRecord(t, app, "artists", "artistnarrow000", map[string]any{
		"name": "Narrow Artist", "slug": "narrow-artist", "portrait": "small.jpg", "biography_image_width": 400, "published": true,
	})

	artist, err := repositoriesFindPublishedArtist(app, "artistnarrow000")
	if err != nil {
		t.Fatalf("find artist: %v", err)
	}
	view, err := buildArtistRecordView(app, artist)
	if err != nil {
		t.Fatalf("build view: %v", err)
	}

	if want := "/api/files/artists/artistnarrow000/small.jpg"; view.Portrait != want {
		t.Errorf("portrait = %q, want original %q", view.Portrait, want)
	}
}

func TestArtistRecordViewFallsBackToWorkImageWithoutPortrait(t *testing.T) {
	app := newArtistRecordApp(t)
	seedPublishedArtist(t, app)
	saveRecordRecord(t, app, "artists", "artistplain0000", map[string]any{
		"name": "Plain Artist", "slug": "plain-artist", "published": true,
	})
	saveRecordRecord(t, app, "artworks", "artworkplain001", map[string]any{
		"title": "Only Work", "author": []string{"artistplain0000"}, "published": true, "image": "only.jpg", "image_width": 900,
	})

	artist, err := repositoriesFindPublishedArtist(app, "artistplain0000")
	if err != nil {
		t.Fatalf("find artist: %v", err)
	}
	view, err := buildArtistRecordView(app, artist)
	if err != nil {
		t.Fatalf("build view: %v", err)
	}

	if view.Portrait != "" {
		t.Errorf("portrait = %q, want empty", view.Portrait)
	}
	if strings.Contains(view.Jsonld, `"image"`) {
		t.Error("JSON-LD should omit image without a portrait")
	}
	if got, want := artistOpenGraphImage(view), apputils.AssetUrl(view.Works[0].Image); got != want {
		t.Errorf("Open Graph image = %q, want first work %q", got, want)
	}
}

func TestArtistRecordViewUsesCanonicalPublishedWorkLinks(t *testing.T) {
	app := newArtistRecordApp(t)
	seedPublishedArtist(t, app)

	artist, err := repositoriesFindPublishedArtist(app, "artistone000001")
	if err != nil {
		t.Fatalf("find artist: %v", err)
	}
	view, err := buildArtistRecordView(app, artist)
	if err != nil {
		t.Fatalf("build view: %v", err)
	}

	if len(view.Works) != 2 {
		t.Fatalf("works = %d, want 2 published works", len(view.Works))
	}
	if want := "/artists/synthetic-artist-artistone000001/alpha-work-artworkone00001"; view.Works[0].Url != want {
		t.Errorf("first work URL = %q, want %q", view.Works[0].Url, want)
	}
	if want := "/api/files/artworks/artworkone00001/alpha.jpg?thumb=500x0"; view.Works[0].Image != want {
		t.Errorf("first work image = %q, want %q", view.Works[0].Image, want)
	}
}

func TestArtistRecordViewMatchesPeriodMusic(t *testing.T) {
	app := newArtistRecordApp(t)
	seedPublishedArtist(t, app)

	artist, err := repositoriesFindPublishedArtist(app, "artistone000001")
	if err != nil {
		t.Fatalf("find artist: %v", err)
	}
	view, err := buildArtistRecordView(app, artist)
	if err != nil {
		t.Fatalf("build view: %v", err)
	}

	if view.Music.Piece != "Fantasia chromatica" {
		t.Errorf("music piece = %q, want Fantasia chromatica", view.Music.Piece)
	}
	if view.Music.SongID != "songone00000000" {
		t.Errorf("music song id = %q, want songone00000000", view.Music.SongID)
	}
	if view.Music.PlayerURL != "/player?song=songone00000000" {
		t.Errorf("music player URL = %q", view.Music.PlayerURL)
	}
}

func repositoriesFindPublishedArtist(app *pocketbase.PocketBase, id string) (*core.Record, error) {
	return repositories.NewArtistRecordRepository(app).FindPublishedArtist(id)
}

const selectionTestContentHash = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func saveSelectionRecord(t *testing.T, app *pocketbase.PocketBase, id string, displayTitle string, commentary string, artworks []string) {
	t.Helper()
	saveRecordRecord(t, app, "art_selections", id, map[string]any{
		"artist":        []string{"artistone000001"},
		"title":         displayTitle,
		"display_title": displayTitle,
		"commentary":    commentary,
		"artworks":      artworks,
		"source_path":   "html/a/artist/" + id + "/index.html",
		"source_hash":   "source-hash",
		"content_hash":  selectionTestContentHash,
		"published":     true,
	})
}

func TestArtistRecordViewBuildsSelectionPreviewsAboveThreshold(t *testing.T) {
	app := newArtistRecordApp(t)
	seedPublishedArtist(t, app)
	saveSelectionRecord(t, app, "rselect00000001", "Synthetic: Paintings", "<p>A supplied lede.</p>", []string{"artworkone00001", "artworktwo00001"})
	saveSelectionRecord(t, app, "rselect00000002", "Synthetic: Studies", "", []string{"artworkone00001"})

	artist, err := repositoriesFindPublishedArtist(app, "artistone000001")
	if err != nil {
		t.Fatalf("find artist: %v", err)
	}
	view, err := buildArtistRecordView(app, artist)
	if err != nil {
		t.Fatalf("build view: %v", err)
	}

	if len(view.Selections) != 2 {
		t.Fatalf("selections = %d, want 2 previews", len(view.Selections))
	}

	first := view.Selections[0]
	if first.DisplayTitle != "Synthetic: Paintings" {
		t.Errorf("first display title = %q, want Synthetic: Paintings", first.DisplayTitle)
	}
	if first.SelectedCount != 2 {
		t.Errorf("first selected count = %d, want 2", first.SelectedCount)
	}
	if first.CataloguedCount != 2 {
		t.Errorf("first catalogued count = %d, want 2", first.CataloguedCount)
	}
	if !first.HasCommentary {
		t.Error("first preview should carry its supplied commentary")
	}
	if !strings.Contains(first.Commentary, "A supplied lede.") {
		t.Errorf("first commentary = %q, want supplied lede", first.Commentary)
	}
	if want := "/artists/synthetic-artist-artistone000001/selections/rselect00000001"; first.URL != want {
		t.Errorf("first preview URL = %q, want %q", first.URL, want)
	}
	if len(first.Works) != 2 {
		t.Errorf("first representative works = %d, want 2", len(first.Works))
	}

	second := view.Selections[1]
	if second.DisplayTitle != "Synthetic: Studies" {
		t.Errorf("second display title = %q, want Synthetic: Studies", second.DisplayTitle)
	}
	if second.HasCommentary {
		t.Error("second preview must not claim commentary when none is supplied")
	}
	if second.SelectedCount != 1 {
		t.Errorf("second selected count = %d, want 1", second.SelectedCount)
	}
}

func TestArtistRecordViewOmitsPreviewsBelowThreshold(t *testing.T) {
	for _, selections := range []int{0, 1} {
		t.Run(string(rune('0'+selections)), func(t *testing.T) {
			app := newArtistRecordApp(t)
			seedPublishedArtist(t, app)
			if selections == 1 {
				saveSelectionRecord(t, app, "rselect00000001", "Synthetic: Paintings", "<p>A lede.</p>", []string{"artworkone00001"})
			}

			artist, err := repositoriesFindPublishedArtist(app, "artistone000001")
			if err != nil {
				t.Fatalf("find artist: %v", err)
			}
			view, err := buildArtistRecordView(app, artist)
			if err != nil {
				t.Fatalf("build view: %v", err)
			}

			if len(view.Selections) != 0 {
				t.Fatalf("selections = %d, want 0 (single selection must retain ordinary works presentation)", len(view.Selections))
			}
		})
	}
}

func TestArtistRecordViewExcludesForeignSelections(t *testing.T) {
	app := newArtistRecordApp(t)
	seedPublishedArtist(t, app)
	saveSelectionRecord(t, app, "rselect00000001", "Synthetic: Paintings", "<p>A lede.</p>", []string{"artworkone00001"})
	saveSelectionRecord(t, app, "rselect00000002", "Synthetic: Studies", "", []string{"artworkone00001"})

	// A second artist owns one selection; it must never appear as a preview for
	// the primary artist.
	saveRecordRecord(t, app, "artists", "artisttwo000001", map[string]any{"name": "Other Artist", "slug": "other-artist", "published": true})
	saveRecordRecord(t, app, "art_selections", "rselect00000003", map[string]any{
		"artist":        []string{"artisttwo000001"},
		"title":         "Foreign",
		"display_title": "Foreign",
		"artworks":      []string{"artworkone00001"},
		"source_path":   "html/a/artist/foreign/index.html",
		"source_hash":   "source-hash",
		"content_hash":  selectionTestContentHash,
		"published":     true,
	})

	artist, err := repositoriesFindPublishedArtist(app, "artistone000001")
	if err != nil {
		t.Fatalf("find artist: %v", err)
	}
	view, err := buildArtistRecordView(app, artist)
	if err != nil {
		t.Fatalf("build view: %v", err)
	}

	if len(view.Selections) != 2 {
		t.Fatalf("selections = %d, want 2 (foreign selection excluded)", len(view.Selections))
	}
	for _, preview := range view.Selections {
		if preview.DisplayTitle == "Foreign" {
			t.Error("foreign artist's selection must not appear as a preview")
		}
	}
}

func TestRenderArtistContentSanitisesLegacyBiography(t *testing.T) {
	app := newArtistRecordApp(t)
	seedPublishedArtist(t, app)

	artist, err := repositoriesFindPublishedArtist(app, "artistone000001")
	if err != nil {
		t.Fatalf("find artist: %v", err)
	}
	artist.Set("bio", `<p>He used chiaroscuro in the <strong>workshop</strong>.</p><img src=x onerror="alert(1)"><script>alert(2)</script>`)

	content, err := RenderArtistContent(app, nil, artist, "#dual-area", false)
	if err != nil {
		t.Fatalf("render artist content: %v", err)
	}

	for _, unsafe := range []string{"<script", "onerror", "<img"} {
		if strings.Contains(content.Bio, unsafe) {
			t.Errorf("legacy bio must not contain %q, got %q", unsafe, content.Bio)
		}
	}
	if !strings.Contains(content.Bio, "<strong>workshop</strong>") {
		t.Errorf("legacy bio should retain legitimate structure, got %q", content.Bio)
	}
	if !strings.Contains(content.Bio, `class="wga-term"`) {
		t.Errorf("legacy bio should annotate glossary terms, got %q", content.Bio)
	}
}

func TestRenderArtistContentSanitisesWithoutGlossary(t *testing.T) {
	app := newArtistRecordApp(t)
	seedPublishedArtist(t, app)

	// Remove the glossary collection so entry loading fails.
	glossaryCollection, err := app.FindCollectionByNameOrId("glossary")
	if err != nil {
		t.Fatalf("find glossary: %v", err)
	}
	if err := app.Delete(glossaryCollection); err != nil {
		t.Fatalf("delete glossary: %v", err)
	}

	artist, err := repositoriesFindPublishedArtist(app, "artistone000001")
	if err != nil {
		t.Fatalf("find artist: %v", err)
	}
	artist.Set("bio", `<p>He used chiaroscuro.</p><script>alert(1)</script>`)

	content, err := RenderArtistContent(app, nil, artist, "#dual-area", false)
	if err != nil {
		t.Fatalf("render artist content: %v", err)
	}

	if strings.Contains(content.Bio, "<script") {
		t.Error("legacy bio must sanitise even when glossary loading fails")
	}
	if !strings.Contains(content.Bio, "<p>He used chiaroscuro.</p>") {
		t.Errorf("legacy bio should remain readable without glossary, got %q", content.Bio)
	}
	if strings.Contains(content.Bio, `class="wga-term"`) {
		t.Error("legacy bio should not annotate without glossary entries")
	}
}
