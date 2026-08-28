package artworks

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/assets/templ/pages"
	"github.com/blackfyre/wga/internal/config"
	apputils "github.com/blackfyre/wga/internal/utils"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func TestTask71VenueFiltersAndPublishedHoldingCounts(t *testing.T) {
	app := newArtworkSearchApp(t)
	saveSearchArtist(t, app, "artisttask71001", "Task Artist")
	saveSearchLocation(t, app, "loctask71alpha0", "Alpha Museum")
	saveSearchLocation(t, app, "loctask71beta01", "Beta Museum")
	saveSearchLocation(t, app, "loctask71gamma0", "Gamma Museum")
	seed := func(id, title, location string, published bool) {
		saveSearchArtwork(t, app, searchArtworkSeed{id: id, title: title, authors: []string{"artisttask71001"}, location: location, published: published})
	}
	seed("worktask7100001", "Alpha One", "loctask71alpha0", true)
	seed("worktask7100002", "Alpha Two", "loctask71alpha0", true)
	seed("worktask7100003", "Alpha Draft", "loctask71alpha0", false)
	seed("worktask7100004", "Beta One", "loctask71beta01", true)
	seed("worktask7100005", "Gamma Draft", "loctask71gamma0", false)

	options, err := getVenueOptions(app, "", "")
	if err != nil {
		t.Fatalf("venue options: %v", err)
	}
	if options.totalOptions != 2 || options.omittedOptions != 0 {
		t.Fatalf("eligible venue totals = %#v, want two published holdings", options)
	}
	if len(options.entries) != 2 || options.entries[0].label != "Alpha Museum" || options.entries[0].count != 2 || options.entries[1].count != 1 {
		t.Fatalf("venue options = %#v, want count-desc Alpha 2 then Beta 1", options.entries)
	}

	alpha, _, err := buildArtworkSearchView(app, url.Values{"venue": {"loctask71alpha0"}}, 1, 16)
	if err != nil {
		t.Fatalf("venue filter view: %v", err)
	}
	if alpha.Results.ResultCount != 2 {
		t.Fatalf("venue result count = %d, want 2", alpha.Results.ResultCount)
	}
	venueQuery, _, err := buildArtworkSearchView(app, url.Values{"venue_q": {"beta"}}, 1, 16)
	if err != nil {
		t.Fatalf("venue query view: %v", err)
	}
	if venueQuery.Results.ResultCount != 3 {
		t.Fatalf("venue_q changed results: %d, want all three published artworks", venueQuery.Results.ResultCount)
	}
	if venueQuery.Facets.ActiveCount != 0 || len(venueQuery.Facets.Collection.Options) != 1 {
		t.Fatalf("venue_q facet state = %#v", venueQuery.Facets)
	}
}

func TestTask71VenueFacetIsBoundedAndQueryCountStable(t *testing.T) {
	app := newArtworkSearchApp(t)
	saveSearchArtist(t, app, "artisttask71002", "Task Artist")
	for i := 0; i < artworkVenueOptionsLimit+1; i++ {
		id := fmt.Sprintf("loct71%09d", i)
		saveSearchLocation(t, app, id, fmt.Sprintf("Museum %02d", i))
		saveSearchArtwork(t, app, searchArtworkSeed{id: fmt.Sprintf("workt71%08d", i), title: fmt.Sprintf("Work %d", i), authors: []string{"artisttask71002"}, location: id, published: true})
	}
	count, err := countSearchQueries(app, func() error {
		options, err := getVenueOptions(app, "", "")
		if err == nil && (len(options.entries) != artworkVenueOptionsLimit || options.omittedOptions != 1 || options.omittedHoldings != 1) {
			return fmt.Errorf("bounded venue options = %#v", options)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("venue facet used %d queries, want one bounded aggregate", count)
	}
}

func TestTask71VenueFacetTieOrderAndHonestOmittedHoldingsNote(t *testing.T) {
	app := newArtworkSearchApp(t)
	saveSearchArtist(t, app, "artisttask71003", "Task Artist")
	for i := 0; i < artworkVenueOptionsLimit+2; i++ {
		id := fmt.Sprintf("loct71x%08d", i)
		name := "Same Museum"
		if i >= artworkVenueOptionsLimit {
			name = fmt.Sprintf("ZZZ Museum %d", i)
		}
		saveSearchLocation(t, app, id, name)
		saveSearchArtwork(t, app, searchArtworkSeed{id: fmt.Sprintf("workt71x%07d", i), title: fmt.Sprintf("Work %d", i), authors: []string{"artisttask71003"}, location: id, published: true})
	}
	options, err := getVenueOptions(app, "", "")
	if err != nil {
		t.Fatalf("venue options: %v", err)
	}
	if options.entries[0].label != "Same Museum" || options.entries[0].value >= options.entries[1].value {
		t.Fatalf("equal count/name tie was not id ordered: %#v", options.entries[:2])
	}
	note := venueFacetNote(options)
	if note != "Showing 40 of 42 collections; omitted collections hold 2 works. Keep typing to narrow." {
		t.Fatalf("omitted holdings note = %q", note)
	}
}

func TestTask71RouteFullHtmxAndOrdinaryGETParity(t *testing.T) {
	app := newArtworkSearchApp(t)
	saveSearchArtist(t, app, "artisttask71004", "Task Artist")
	saveSearchArtwork(t, app, searchArtworkSeed{id: "worktask7100006", title: "Task Work", authors: []string{"artisttask71004"}, published: true})
	configuration := config.LoadFrom(func(key string) string {
		return map[string]string{"WGA_ENV": "development", "WGA_PROTOCOL": "https", "WGA_HOSTNAME": "gallery.example", "WGA_SENDER_NAME": "WGA", "WGA_SENDER_ADDRESS": "sender@example.com", "WGA_POSTCARD_FREQUENCY": "*/1 * * * *"}[key]
	})
	server, err := configuration.Server()
	if err != nil {
		t.Fatalf("server config: %v", err)
	}
	apputils.ConfigurePublicURL(server.PublicURL)
	t.Cleanup(func() { apputils.ConfigurePublicURL(config.PublicURL{}) })
	RegisterArtworksHandlers(app)
	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatalf("router: %v", err)
	}
	if err := app.OnServe().Trigger(&core.ServeEvent{App: app, Router: router}, func(event *core.ServeEvent) error {
		mux, err := event.Router.BuildMux()
		if err != nil {
			return err
		}
		for _, request := range []struct {
			path string
			hx   bool
		}{
			{"/artworks?q=Task", false}, {"/artworks/results?q=Task", false}, {"/artworks/results?q=Task", true},
		} {
			recorder := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, request.path, nil)
			if request.hx {
				req.Header.Set("HX-Request", "true")
			}
			mux.ServeHTTP(recorder, req)
			if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "Task Work") {
				t.Errorf("GET %s (hx=%t): status %d/body missing result", request.path, request.hx, recorder.Code)
			}
			if !strings.Contains(recorder.Header().Get("HX-Push-Url"), "q=Task") {
				t.Errorf("GET %s (hx=%t): push URL %q", request.path, request.hx, recorder.Header().Get("HX-Push-Url"))
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("serve event: %v", err)
	}
}

// task71Option returns the collection option whose value matches wanted, or nil.
func task71Option(options []pages.ArtworkSearchCollectionOption, wanted string) *pages.ArtworkSearchCollectionOption {
	for i := range options {
		if options[i].Value == wanted {
			return &options[i]
		}
	}

	return nil
}

func TestTask71SelectedVenueRetainedOutsideCap(t *testing.T) {
	app := newArtworkSearchApp(t)
	saveSearchArtist(t, app, "artisttask71011", "Task Artist")
	for i := 0; i < artworkVenueOptionsLimit; i++ {
		location := fmt.Sprintf("loct71cap%06d", i)
		saveSearchLocation(t, app, location, fmt.Sprintf("Museum %02d", i))
		saveSearchArtwork(t, app, searchArtworkSeed{id: fmt.Sprintf("workt71cap%05d", i), title: fmt.Sprintf("Work %02d", i), authors: []string{"artisttask71011"}, location: location, published: true})
	}
	saveSearchLocation(t, app, "loct71zebra0001", "ZZZ Museum")
	saveSearchArtwork(t, app, searchArtworkSeed{id: "workt71zebra001", title: "Zebra Work", authors: []string{"artisttask71011"}, location: "loct71zebra0001", published: true})

	view, canonical, err := buildArtworkSearchView(app, url.Values{"venue": {"loct71zebra0001"}}, 1, 16)
	if err != nil {
		t.Fatalf("build view: %v", err)
	}
	if !strings.Contains(canonical, "venue=loct71zebra0001") {
		t.Fatalf("canonical dropped selected venue: %q", canonical)
	}
	if len(view.Facets.Collection.Options) > artworkVenueOptionsLimit {
		t.Fatalf("options = %d, want at most the %d cap", len(view.Facets.Collection.Options), artworkVenueOptionsLimit)
	}
	if displaced := task71Option(view.Facets.Collection.Options, "loct71cap000039"); displaced != nil {
		t.Fatalf("retained venue did not displace the final unselected option: %#v", displaced)
	}
	if note := view.Facets.Collection.Note; note != "Showing 40 of 41 collections; omitted collections hold 1 works. Keep typing to narrow." {
		t.Fatalf("hidden-holdings note = %q, want the displaced option back in the omitted count", note)
	}
	retained := task71Option(view.Facets.Collection.Options, "loct71zebra0001")
	if retained == nil || !retained.Selected || retained.Label != "ZZZ Museum" || retained.Count != 1 {
		t.Fatalf("retained selected venue = %#v", retained)
	}
	if view.Results.ResultCount != 1 {
		t.Fatalf("retained venue result count = %d, want 1", view.Results.ResultCount)
	}
}

func TestTask71SelectedVenueRetainedWhenExcludedByQuery(t *testing.T) {
	app := newArtworkSearchApp(t)
	saveSearchArtist(t, app, "artisttask71012", "Task Artist")
	saveSearchLocation(t, app, "loct71alpha0001", "Alpha Museum")
	saveSearchLocation(t, app, "loct71beta00001", "Beta Museum")
	saveSearchArtwork(t, app, searchArtworkSeed{id: "workt71alpha001", title: "Alpha One", authors: []string{"artisttask71012"}, location: "loct71alpha0001", published: true})
	saveSearchArtwork(t, app, searchArtworkSeed{id: "workt71alpha002", title: "Alpha Two", authors: []string{"artisttask71012"}, location: "loct71alpha0001", published: true})
	saveSearchArtwork(t, app, searchArtworkSeed{id: "workt71beta0001", title: "Beta One", authors: []string{"artisttask71012"}, location: "loct71beta00001", published: true})

	view, _, err := buildArtworkSearchView(app, url.Values{"venue": {"loct71alpha0001"}, "venue_q": {"beta"}}, 1, 16)
	if err != nil {
		t.Fatalf("build view: %v", err)
	}
	retained := task71Option(view.Facets.Collection.Options, "loct71alpha0001")
	if retained == nil || !retained.Selected || retained.Label != "Alpha Museum" || retained.Count != 2 {
		t.Fatalf("venue_q-excluded selection was not retained: %#v", view.Facets.Collection.Options)
	}
	if view.Results.ResultCount != 2 {
		t.Fatalf("venue_q narrowed results: %d, want the two Alpha works", view.Results.ResultCount)
	}
}

func TestTask71RetainedVenueKeepsHiddenHoldingsNoteHonest(t *testing.T) {
	app := newArtworkSearchApp(t)
	saveSearchArtist(t, app, "artisttask71023", "Task Artist")
	for i := 0; i < artworkVenueOptionsLimit+2; i++ {
		name := "Same Museum"
		if i >= artworkVenueOptionsLimit {
			name = "ZZZ Museum"
		}
		location := fmt.Sprintf("loct71n%08d", i)
		saveSearchLocation(t, app, location, name)
		saveSearchArtwork(t, app, searchArtworkSeed{id: fmt.Sprintf("workt71n%07d", i), title: fmt.Sprintf("Work %d", i), authors: []string{"artisttask71023"}, location: location, published: true})
	}

	view, _, err := buildArtworkSearchView(app, url.Values{"venue": {"loct71n00000040"}}, 1, 16)
	if err != nil {
		t.Fatalf("build view: %v", err)
	}
	if len(view.Facets.Collection.Options) > artworkVenueOptionsLimit {
		t.Fatalf("options = %d, want at most the %d cap", len(view.Facets.Collection.Options), artworkVenueOptionsLimit)
	}
	if note := view.Facets.Collection.Note; note != "Showing 40 of 42 collections; omitted collections hold 2 works. Keep typing to narrow." {
		t.Fatalf("hidden-holdings note = %q, want the retained venue kept and the displaced option counted", note)
	}
	if displaced := task71Option(view.Facets.Collection.Options, "loct71n00000039"); displaced != nil {
		t.Fatalf("retained venue did not displace the final unselected option: %#v", displaced)
	}
	if got := task71Option(view.Facets.Collection.Options, "loct71n00000040"); got == nil || !got.Selected || got.Label != "ZZZ Museum" {
		t.Fatalf("retained venue = %#v", got)
	}
}

func TestTask71UnknownSelectedVenueIsHonestAndEmpty(t *testing.T) {
	app := newArtworkSearchApp(t)
	saveSearchArtist(t, app, "artisttask71013", "Task Artist")
	saveSearchLocation(t, app, "loct71real00001", "Real Museum")
	saveSearchArtwork(t, app, searchArtworkSeed{id: "workt71real0001", title: "Real Work", authors: []string{"artisttask71013"}, location: "loct71real00001", published: true})

	view, canonical, err := buildArtworkSearchView(app, url.Values{"venue": {"loct71missing01"}}, 1, 16)
	if err != nil {
		t.Fatalf("build view: %v", err)
	}
	if !strings.Contains(canonical, "venue=loct71missing01") {
		t.Fatalf("canonical dropped unknown venue: %q", canonical)
	}
	if view.Results.ResultCount != 0 || len(view.Results.Artworks) != 0 {
		t.Fatalf("unknown venue yielded %d results, want empty", view.Results.ResultCount)
	}
	unknown := task71Option(view.Facets.Collection.Options, "loct71missing01")
	if unknown == nil || !unknown.Selected || unknown.Count != 0 || unknown.Label != "Unknown collection" {
		t.Fatalf("unknown venue choice = %#v, want a checked zero-count unavailable label", unknown)
	}
	if !view.Facets.Collection.Facet.Active {
		t.Fatal("unknown venue must remain an active filter")
	}
}

func TestTask71VenueHoldingEligibilityMatchesArtworkPredicate(t *testing.T) {
	app := newArtworkSearchApp(t)
	artists, err := app.FindCollectionByNameOrId("artists")
	if err != nil {
		t.Fatalf("find artists: %v", err)
	}
	artists.Fields.Add(&core.BoolField{Id: "artist_published", Name: "published"})
	if err := app.Save(artists); err != nil {
		t.Fatalf("add artists published field: %v", err)
	}

	saveSearchArtist(t, app, "artisttask71021", "Published Artist")
	saveSearchArtist(t, app, "artisttask71022", "Unpublished Artist")
	published, err := app.FindRecordById("artists", "artisttask71021")
	if err != nil {
		t.Fatalf("find published artist: %v", err)
	}
	published.Set("published", true)
	if err := app.Save(published); err != nil {
		t.Fatalf("save published artist: %v", err)
	}

	saveSearchLocation(t, app, "loct71pub000001", "Published Museum")
	saveSearchLocation(t, app, "loct71unpub0001", "Unpublished Museum")
	saveSearchArtwork(t, app, searchArtworkSeed{id: "workt71pub00001", title: "By Published", authors: []string{"artisttask71021"}, location: "loct71pub000001", published: true})
	saveSearchArtwork(t, app, searchArtworkSeed{id: "workt71unpub001", title: "By Unpublished", authors: []string{"artisttask71022"}, location: "loct71unpub0001", published: true})

	options, err := getVenueOptions(app, "", "")
	if err != nil {
		t.Fatalf("venue options: %v", err)
	}
	if options.totalOptions != 2 {
		t.Fatalf("eligible venues = %d, want both published and artist-unpublished holdings", options.totalOptions)
	}
	totalHoldings := 0
	for _, entry := range options.entries {
		totalHoldings += entry.count
	}
	if totalHoldings != 2 {
		t.Fatalf("eligible holdings = %d, want both published artworks counted", totalHoldings)
	}
}

func TestTask71SortViewChangeRefreshesRailWithNondefaultState(t *testing.T) {
	app := newArtworkSearchApp(t)
	saveSearchArtist(t, app, "artisttask71015", "Task Artist")
	saveSearchArtwork(t, app, searchArtworkSeed{id: "worktask7100015", title: "Task Work", authors: []string{"artisttask71015"}, published: true})
	configuration := config.LoadFrom(func(key string) string {
		return map[string]string{"WGA_ENV": "development", "WGA_PROTOCOL": "https", "WGA_HOSTNAME": "gallery.example", "WGA_SENDER_NAME": "WGA", "WGA_SENDER_ADDRESS": "sender@example.com", "WGA_POSTCARD_FREQUENCY": "*/1 * * * *"}[key]
	})
	server, err := configuration.Server()
	if err != nil {
		t.Fatalf("server config: %v", err)
	}
	apputils.ConfigurePublicURL(server.PublicURL)
	t.Cleanup(func() { apputils.ConfigurePublicURL(config.PublicURL{}) })
	RegisterArtworksHandlers(app)
	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatalf("router: %v", err)
	}
	if err := app.OnServe().Trigger(&core.ServeEvent{App: app, Router: router}, func(event *core.ServeEvent) error {
		mux, err := event.Router.BuildMux()
		if err != nil {
			return err
		}
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/artworks?sort=title&dir=desc&view=list", nil)
		req.Header.Set("HX-Request", "true")
		req.Header.Set("HX-Target", "artwork-search")
		mux.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusOK {
			t.Fatalf("sort/view rail replacement status = %d", recorder.Code)
		}
		body := recorder.Body.String()
		for _, expected := range []string{`id="artwork-filters"`, `name="sort" value="title"`, `name="dir" value="desc"`, `name="view" value="list"`} {
			if !strings.Contains(body, expected) {
				t.Errorf("sort/view rail replacement missing %s", expected)
			}
		}
		if got := recorder.Header().Get("HX-Push-Url"); got != "/artworks?dir=desc&sort=title&view=list" {
			t.Errorf("sort/view push URL = %q, want canonical /artworks", got)
		}
		return nil
	}); err != nil {
		t.Fatalf("serve event: %v", err)
	}
}

func TestTask71FilterFormHtmxReplacesRailAndResults(t *testing.T) {
	app := newArtworkSearchApp(t)
	saveSearchArtist(t, app, "artisttask71014", "Task Artist")
	saveSearchArtwork(t, app, searchArtworkSeed{id: "worktask7100014", title: "Task Work", authors: []string{"artisttask71014"}, published: true})
	configuration := config.LoadFrom(func(key string) string {
		return map[string]string{"WGA_ENV": "development", "WGA_PROTOCOL": "https", "WGA_HOSTNAME": "gallery.example", "WGA_SENDER_NAME": "WGA", "WGA_SENDER_ADDRESS": "sender@example.com", "WGA_POSTCARD_FREQUENCY": "*/1 * * * *"}[key]
	})
	server, err := configuration.Server()
	if err != nil {
		t.Fatalf("server config: %v", err)
	}
	apputils.ConfigurePublicURL(server.PublicURL)
	t.Cleanup(func() { apputils.ConfigurePublicURL(config.PublicURL{}) })
	RegisterArtworksHandlers(app)
	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatalf("router: %v", err)
	}
	if err := app.OnServe().Trigger(&core.ServeEvent{App: app, Router: router}, func(event *core.ServeEvent) error {
		mux, err := event.Router.BuildMux()
		if err != nil {
			return err
		}
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/artworks?q=Task", nil)
		req.Header.Set("HX-Request", "true")
		req.Header.Set("HX-Target", "artwork-search")
		mux.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusOK {
			t.Fatalf("rail replacement status = %d", recorder.Code)
		}
		body := recorder.Body.String()
		for _, expected := range []string{`id="artwork-search"`, `id="artwork-filters"`, `id="artwork-search-results"`} {
			if !strings.Contains(body, expected) {
				t.Errorf("rail replacement missing %s", expected)
			}
		}
		if strings.Contains(body, `id="mc-area"`) {
			t.Error("rail replacement leaked the shared layout shell")
		}
		if recorder.Header().Get("HX-Push-Url") != "/artworks?q=Task" {
			t.Errorf("rail replacement push URL = %q, want canonical /artworks", recorder.Header().Get("HX-Push-Url"))
		}
		return nil
	}); err != nil {
		t.Fatalf("serve event: %v", err)
	}
}
