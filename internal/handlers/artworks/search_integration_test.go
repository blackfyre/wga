package artworks

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

	"github.com/blackfyre/wga/internal/assets/templ/pages"
	"github.com/blackfyre/wga/internal/config"
	apputils "github.com/blackfyre/wga/internal/utils"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// workID returns a deterministic 15-character artwork record id for a short tag.
func workID(tag string) string {
	id := "work" + tag
	if len(id) > 15 {
		id = id[:15]
	}
	return id + strings.Repeat("0", 15-len(id))
}

func newArtworkSearchApp(t *testing.T) *pocketbase.PocketBase {
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

	forms := core.NewBaseCollection("Art_forms")
	forms.Id = "test_art_forms"
	forms.MarkAsNew()
	forms.Fields.Add(
		&core.TextField{Id: "form_name", Name: "name", Required: true},
		&core.TextField{Id: "form_slug", Name: "slug"},
	)
	if err := app.Save(forms); err != nil {
		t.Fatalf("save art_forms: %v", err)
	}

	types := core.NewBaseCollection("Art_types")
	types.Id = "test_art_types"
	types.MarkAsNew()
	types.Fields.Add(
		&core.TextField{Id: "type_name", Name: "name", Required: true},
		&core.TextField{Id: "type_slug", Name: "slug"},
	)
	if err := app.Save(types); err != nil {
		t.Fatalf("save art_types: %v", err)
	}

	periods := core.NewBaseCollection("Art_periods")
	periods.Id = "test_art_periods"
	periods.MarkAsNew()
	periods.Fields.Add(
		&core.TextField{Id: "period_name", Name: "name", Required: true},
		&core.TextField{Id: "period_slug", Name: "slug"},
		&core.NumberField{Id: "period_start", Name: "start"},
		&core.NumberField{Id: "period_end", Name: "end"},
	)
	if err := app.Save(periods); err != nil {
		t.Fatalf("save art_periods: %v", err)
	}

	locations := core.NewBaseCollection("Locations")
	locations.Id = "test_locations"
	locations.MarkAsNew()
	locations.Fields.Add(
		&core.TextField{Id: "location_name", Name: "name", Required: true},
	)
	if err := app.Save(locations); err != nil {
		t.Fatalf("save locations: %v", err)
	}

	artists := core.NewBaseCollection("Artists")
	artists.Id = "test_artists"
	artists.MarkAsNew()
	artists.Fields.Add(
		&core.TextField{Id: "artist_name", Name: "name", Required: true},
		&core.TextField{Id: "artist_filing_name", Name: "filing_name"},
		&core.TextField{Id: "artist_short_name", Name: "short_name"},
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
		&core.RelationField{Id: "artwork_form", Name: "form", CollectionId: forms.Id, MinSelect: 1, MaxSelect: 20},
		&core.RelationField{Id: "artwork_type", Name: "type", CollectionId: types.Id, MinSelect: 1, MaxSelect: 20},
		&core.RelationField{Id: "artwork_school", Name: "school", CollectionId: schools.Id, MinSelect: 1, MaxSelect: 10},
		&core.RelationField{Id: "artwork_period", Name: "art_period_id", CollectionId: periods.Id, MinSelect: 0, MaxSelect: 1},
		&core.RelationField{Id: "artwork_location", Name: "current_location_id", CollectionId: locations.Id, MinSelect: 0, MaxSelect: 1},
		&core.TextField{Id: "artwork_technique", Name: "technique"},
		&core.NumberField{Id: "artwork_year", Name: "year"},
		&core.NumberField{Id: "artwork_source_row", Name: "source_row"},
		&core.NumberField{Id: "artwork_date_start", Name: "date_start"},
		&core.NumberField{Id: "artwork_date_end", Name: "date_end"},
		&core.BoolField{Id: "artwork_published", Name: "published"},
	)
	if err := app.Save(artworks); err != nil {
		t.Fatalf("save artworks: %v", err)
	}

	return app
}

type searchArtworkSeed struct {
	id        string
	title     string
	authors   []string
	form      string
	typeSlug  string
	school    string
	technique string
	year      int
	period    string
	location  string
	sourceRow int
	dateStart int
	dateEnd   int
	published bool
}

func saveSearchTaxonomy(t *testing.T, app *pocketbase.PocketBase, collection string, id string, slug string, name string) {
	t.Helper()
	model, err := app.FindCollectionByNameOrId(collection)
	if err != nil {
		t.Fatalf("find %s: %v", collection, err)
	}
	record := core.NewRecord(model)
	record.Id = id
	record.Set("name", name)
	record.Set("slug", slug)
	if err := app.Save(record); err != nil {
		t.Fatalf("save %s %s: %v", collection, id, err)
	}
}

func saveSearchArtPeriod(t *testing.T, app *pocketbase.PocketBase, id string, name string, start int, end int) {
	t.Helper()
	model, err := app.FindCollectionByNameOrId("art_periods")
	if err != nil {
		t.Fatalf("find art_periods: %v", err)
	}
	record := core.NewRecord(model)
	record.Id = id
	record.Set("name", name)
	record.Set("slug", strings.ToLower(name))
	record.Set("start", start)
	record.Set("end", end)
	if err := app.Save(record); err != nil {
		t.Fatalf("save art period %s: %v", id, err)
	}
}

func saveSearchLocation(t *testing.T, app *pocketbase.PocketBase, id string, name string) {
	t.Helper()
	model, err := app.FindCollectionByNameOrId("locations")
	if err != nil {
		t.Fatalf("find locations: %v", err)
	}
	record := core.NewRecord(model)
	record.Id = id
	record.Set("name", name)
	if err := app.Save(record); err != nil {
		t.Fatalf("save location %s: %v", id, err)
	}
}

func saveSearchArtist(t *testing.T, app *pocketbase.PocketBase, id string, name string) {
	t.Helper()
	model, err := app.FindCollectionByNameOrId("artists")
	if err != nil {
		t.Fatalf("find artists: %v", err)
	}
	record := core.NewRecord(model)
	record.Id = id
	record.Set("name", name)
	record.Set("filing_name", name)
	record.Set("short_name", name)
	if err := app.Save(record); err != nil {
		t.Fatalf("save artist %s: %v", id, err)
	}
}

func saveSearchArtwork(t *testing.T, app *pocketbase.PocketBase, seed searchArtworkSeed) {
	t.Helper()
	model, err := app.FindCollectionByNameOrId("artworks")
	if err != nil {
		t.Fatalf("find artworks: %v", err)
	}
	record := core.NewRecord(model)
	record.Id = seed.id
	record.Set("title", seed.title)
	record.Set("author", seed.authors)
	if seed.form != "" {
		record.Set("form", []string{seed.form})
	}
	if seed.typeSlug != "" {
		record.Set("type", []string{seed.typeSlug})
	}
	if seed.school != "" {
		record.Set("school", []string{seed.school})
	}
	if seed.period != "" {
		record.Set("art_period_id", []string{seed.period})
	}
	if seed.location != "" {
		record.Set("current_location_id", []string{seed.location})
	}
	record.Set("technique", seed.technique)
	record.Set("year", seed.year)
	record.Set("source_row", seed.sourceRow)
	record.Set("date_start", seed.dateStart)
	record.Set("date_end", seed.dateEnd)
	record.Set("published", seed.published)
	if err := app.Save(record); err != nil {
		t.Fatalf("save artwork %s: %v", seed.id, err)
	}
}

func TestBuildArtworkSearchViewFiltersBySchoolFormAndTechnique(t *testing.T) {
	app := newArtworkSearchApp(t)
	saveSearchTaxonomy(t, app, "schools", "schooldutch0001", "dutch", "Dutch")
	saveSearchTaxonomy(t, app, "schools", "schoolitali0001", "italian", "Italian")
	saveSearchTaxonomy(t, app, "art_forms", "formpaint000001", "painting", "Painting")
	saveSearchTaxonomy(t, app, "art_forms", "formsculp000001", "sculpture", "Sculpture")
	saveSearchArtist(t, app, "artistone000001", "Artist One")

	saveSearchArtwork(t, app, searchArtworkSeed{id: "workfresco00001", title: "Fresco Work", authors: []string{"artistone000001"}, form: "formpaint000001", school: "schooldutch0001", technique: "Fresco", year: 1600, published: true})
	saveSearchArtwork(t, app, searchArtworkSeed{id: "workoils0000001", title: "Oil Work", authors: []string{"artistone000001"}, form: "formsculp000001", school: "schoolitali0001", technique: "Oil on canvas", year: 1700, published: true})

	view, canonical, err := buildArtworkSearchView(app, neturl.Values{"art_school": {"dutch"}, "art_form": {"painting"}, "technique": {"Fresco"}}, 1, 16)
	if err != nil {
		t.Fatalf("build view: %v", err)
	}
	if canonical != "/artworks?art_form=painting&art_school=dutch&technique=Fresco" {
		t.Errorf("canonical = %q", canonical)
	}
	if view.Results.ResultCount != 1 || len(view.Results.Artworks) != 1 {
		t.Fatalf("results = %d (count %d), want 1", len(view.Results.Artworks), view.Results.ResultCount)
	}
	if view.Results.Artworks[0].Title != "Fresco Work" {
		t.Errorf("filtered artwork = %q, want Fresco Work", view.Results.Artworks[0].Title)
	}
}

func TestBuildArtworkSearchViewFiltersByExactArtistID(t *testing.T) {
	app := newArtworkSearchApp(t)
	saveSearchArtist(t, app, "artistone000001", "Aachen, Hans von")
	saveSearchArtist(t, app, "artisttwo000001", "Aachen, Hans von")

	saveSearchArtwork(t, app, searchArtworkSeed{id: "workfirst000001", title: "First Artist Work", authors: []string{"artistone000001"}, sourceRow: 1, published: true})
	saveSearchArtwork(t, app, searchArtworkSeed{id: "workshared00001", title: "Shared Work", authors: []string{"artisttwo000001", "artistone000001"}, sourceRow: 2, published: true})
	saveSearchArtwork(t, app, searchArtworkSeed{id: "worksecond00001", title: "Second Artist Work", authors: []string{"artisttwo000001"}, sourceRow: 3, published: true})
	saveSearchArtwork(t, app, searchArtworkSeed{id: "workhidden00001", title: "Hidden Work", authors: []string{"artistone000001"}, sourceRow: 4, published: false})

	view, canonical, err := buildArtworkSearchView(app, neturl.Values{
		"artist":    {"Aachen, Hans von"},
		"artist_id": {"artistone000001"},
	}, 1, 16)
	if err != nil {
		t.Fatalf("build exact artist view: %v", err)
	}
	if canonical != "/artworks?artist_id=artistone000001" {
		t.Errorf("canonical = %q", canonical)
	}
	if view.ArtistID != "artistone000001" {
		t.Errorf("view artist ID = %q, want artistone000001", view.ArtistID)
	}
	if view.Results.ListUrl != "/artworks?artist_id=artistone000001&view=list" {
		t.Errorf("list URL = %q", view.Results.ListUrl)
	}
	if view.Results.ResetUrl != "/artworks" {
		t.Errorf("reset URL = %q, want /artworks", view.Results.ResetUrl)
	}
	if !strings.Contains(view.Results.SortToggleUrl, "artist_id=artistone000001") {
		t.Errorf("sort toggle URL = %q, want retained artist ID", view.Results.SortToggleUrl)
	}
	if view.Results.ResultCount != 2 {
		t.Fatalf("result count = %d, want 2", view.Results.ResultCount)
	}
	assertTitles(t, view, []string{"First Artist Work", "Shared Work"})

	paged, _, err := buildArtworkSearchView(app, neturl.Values{"artist_id": {"artistone000001"}}, 1, 1)
	if err != nil {
		t.Fatalf("build paged artist view: %v", err)
	}
	if !strings.Contains(paged.Results.Pagination, "artist_id=artistone000001") {
		t.Errorf("pagination = %q, want retained artist ID", paged.Results.Pagination)
	}

	unknown, _, err := buildArtworkSearchView(app, neturl.Values{"artist_id": {"unknownartist001"}}, 1, 16)
	if err != nil {
		t.Fatalf("build unknown artist view: %v", err)
	}
	if unknown.Results.ResultCount != 0 || len(unknown.Results.Artworks) != 0 {
		t.Fatalf("unknown artist results = %d, want 0", unknown.Results.ResultCount)
	}
}

func TestBuildArtworkSearchViewSortsByTitleDesc(t *testing.T) {
	app := newArtworkSearchApp(t)
	saveSearchArtist(t, app, "artistone000001", "Artist One")

	saveSearchArtwork(t, app, searchArtworkSeed{id: "workbravo000001", title: "Bravo", authors: []string{"artistone000001"}, published: true})
	saveSearchArtwork(t, app, searchArtworkSeed{id: "workalpha000001", title: "Alpha", authors: []string{"artistone000001"}, published: true})
	saveSearchArtwork(t, app, searchArtworkSeed{id: "workcharl000001", title: "Charlie", authors: []string{"artistone000001"}, published: true})

	view, _, err := buildArtworkSearchView(app, neturl.Values{"sort": {"title"}, "dir": {"desc"}}, 1, 16)
	if err != nil {
		t.Fatalf("build view: %v", err)
	}

	got := []string{}
	for _, artwork := range view.Results.Artworks {
		got = append(got, artwork.Title)
	}
	want := []string{"Charlie", "Bravo", "Alpha"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("title desc order = %v, want %v", got, want)
		}
	}
}

func TestBuildArtworkSearchViewSortsByDateStartWithUnknownLast(t *testing.T) {
	app := newArtworkSearchApp(t)
	saveSearchArtist(t, app, "artistone000001", "Artist One")

	saveSearchArtwork(t, app, searchArtworkSeed{id: workID("unknown"), title: "Unknown", authors: []string{"artistone000001"}, dateStart: 0, published: true})
	saveSearchArtwork(t, app, searchArtworkSeed{id: workID("late"), title: "Late", authors: []string{"artistone000001"}, dateStart: 1800, published: true})
	saveSearchArtwork(t, app, searchArtworkSeed{id: workID("early"), title: "Early", authors: []string{"artistone000001"}, dateStart: 1200, published: true})

	asc, _, err := buildArtworkSearchView(app, neturl.Values{"sort": {"date"}}, 1, 16)
	if err != nil {
		t.Fatalf("build asc view: %v", err)
	}
	assertTitles(t, asc, []string{"Early", "Late", "Unknown"})

	desc, _, err := buildArtworkSearchView(app, neturl.Values{"sort": {"date"}, "dir": {"desc"}}, 1, 16)
	if err != nil {
		t.Fatalf("build desc view: %v", err)
	}
	assertTitles(t, desc, []string{"Late", "Early", "Unknown"})
}

func TestBuildArtworkSearchViewSortsByCatalogueSourceRow(t *testing.T) {
	app := newArtworkSearchApp(t)
	saveSearchArtist(t, app, "artistone000001", "Artist One")

	// IDs deliberately not in source_row order, so a correct archive order must
	// read source_row rather than fall back to id.
	saveSearchArtwork(t, app, searchArtworkSeed{id: workID("aaa"), title: "Third", authors: []string{"artistone000001"}, sourceRow: 3, published: true})
	saveSearchArtwork(t, app, searchArtworkSeed{id: workID("bbb"), title: "First", authors: []string{"artistone000001"}, sourceRow: 1, published: true})
	saveSearchArtwork(t, app, searchArtworkSeed{id: workID("ccc"), title: "Second", authors: []string{"artistone000001"}, sourceRow: 2, published: true})

	asc, _, err := buildArtworkSearchView(app, neturl.Values{}, 1, 16)
	if err != nil {
		t.Fatalf("build archive view: %v", err)
	}
	assertTitles(t, asc, []string{"First", "Second", "Third"})

	desc, _, err := buildArtworkSearchView(app, neturl.Values{"dir": {"desc"}}, 1, 16)
	if err != nil {
		t.Fatalf("build reversed archive view: %v", err)
	}
	assertTitles(t, desc, []string{"Third", "Second", "First"})
}

func TestBuildArtworkSearchViewMissingSourceRowSortsLastBothDirections(t *testing.T) {
	app := newArtworkSearchApp(t)
	saveSearchArtist(t, app, "artistone000001", "Artist One")

	saveSearchArtwork(t, app, searchArtworkSeed{id: workID("real"), title: "Real", authors: []string{"artistone000001"}, sourceRow: 1, published: true})
	saveSearchArtwork(t, app, searchArtworkSeed{id: workID("orph"), title: "Orphan", authors: []string{"artistone000001"}, sourceRow: 0, published: true})

	// source_row 0 (missing) sorts after the authoritative catalogue entries in
	// both directions.
	asc, _, err := buildArtworkSearchView(app, neturl.Values{}, 1, 16)
	if err != nil {
		t.Fatalf("build ascending view: %v", err)
	}
	assertTitles(t, asc, []string{"Real", "Orphan"})

	desc, _, err := buildArtworkSearchView(app, neturl.Values{"dir": {"desc"}}, 1, 16)
	if err != nil {
		t.Fatalf("build descending view: %v", err)
	}
	assertTitles(t, desc, []string{"Real", "Orphan"})
}

func TestBuildArtworkSearchViewPeriodAndLocationFilters(t *testing.T) {
	app := newArtworkSearchApp(t)
	saveSearchArtPeriod(t, app, "periodbaroque01", "Baroque", 1600, 1750)
	saveSearchArtPeriod(t, app, "periodrenai0001", "Renaissance", 1400, 1600)
	saveSearchLocation(t, app, "locflorence0001", "Florence")
	saveSearchLocation(t, app, "locparis0000001", "Paris")
	saveSearchArtist(t, app, "artistone000001", "Artist One")

	saveSearchArtwork(t, app, searchArtworkSeed{
		id: workID("barflor"), title: "Baroque Florence", authors: []string{"artistone000001"},
		period: "periodbaroque01", location: "locflorence0001", published: true,
	})
	saveSearchArtwork(t, app, searchArtworkSeed{
		id: workID("renparis"), title: "Renaissance Paris", authors: []string{"artistone000001"},
		period: "periodrenai0001", location: "locparis0000001", published: true,
	})

	// Individual filters.
	byPeriod, _, err := buildArtworkSearchView(app, neturl.Values{"period": {"periodbaroque01"}}, 1, 16)
	if err != nil {
		t.Fatalf("build period view: %v", err)
	}
	assertTitles(t, byPeriod, []string{"Baroque Florence"})

	byLocation, _, err := buildArtworkSearchView(app, neturl.Values{"location": {"locparis0000001"}}, 1, 16)
	if err != nil {
		t.Fatalf("build location view: %v", err)
	}
	assertTitles(t, byLocation, []string{"Renaissance Paris"})

	// Combined filters.
	combined, _, err := buildArtworkSearchView(app, neturl.Values{
		"period": {"periodbaroque01"}, "location": {"locflorence0001"},
	}, 1, 16)
	if err != nil {
		t.Fatalf("build combined view: %v", err)
	}
	assertTitles(t, combined, []string{"Baroque Florence"})

	// Negative: a period nobody has must filter everything out, not silently
	// return the unfiltered set.
	none, _, err := buildArtworkSearchView(app, neturl.Values{"period": {"periodrenai0001"}, "location": {"locflorence0001"}}, 1, 16)
	if err != nil {
		t.Fatalf("build negative view: %v", err)
	}
	if none.Results.ResultCount != 0 || len(none.Results.Artworks) != 0 {
		t.Fatalf("negative filter returned %d results, want 0", none.Results.ResultCount)
	}
}

func TestBuildArtworkSearchViewHonestEmptyGroupsWithoutData(t *testing.T) {
	app := newArtworkSearchApp(t)
	saveSearchArtist(t, app, "artistone000001", "Artist One")
	saveSearchArtwork(t, app, searchArtworkSeed{id: workID("one"), title: "Only", authors: []string{"artistone000001"}, published: true})

	view, _, err := buildArtworkSearchView(app, neturl.Values{}, 1, 16)
	if err != nil {
		t.Fatalf("build view: %v", err)
	}

	for name, group := range map[string]struct {
		options int
		note    string
	}{
		"period":   {options: len(view.PeriodGroup.Options), note: view.PeriodGroup.Note},
		"location": {options: len(view.LocationGroup.Options), note: view.LocationGroup.Note},
	} {
		if group.options != 0 {
			t.Errorf("%s group should render no options when upstream has none, got %d", name, group.options)
		}
		if group.note == "" {
			t.Errorf("%s group should carry an honest unavailable note", name)
		}
	}
}

func TestBuildArtworkSearchViewPeriodOptionsOrderedByStartYear(t *testing.T) {
	app := newArtworkSearchApp(t)
	saveSearchArtPeriod(t, app, "periodrenai0001", "Renaissance", 1400, 1600)
	saveSearchArtPeriod(t, app, "periodbaroque01", "Baroque", 1600, 1750)

	view, _, err := buildArtworkSearchView(app, neturl.Values{}, 1, 16)
	if err != nil {
		t.Fatalf("build view: %v", err)
	}

	// Options exclude the leading ALL chip; chronological order (start year).
	options := view.PeriodGroup.Options[1:]
	if len(options) != 2 {
		t.Fatalf("period options = %d, want 2", len(options))
	}
	if options[0].Label != "Renaissance" || options[1].Label != "Baroque" {
		t.Errorf("period options out of chronological order: %#v", options)
	}
}

func TestGetLocationOptionsOrderedByName(t *testing.T) {
	app := newArtworkSearchApp(t)
	saveSearchLocation(t, app, "loczebra0000001", "Zebra")
	saveSearchLocation(t, app, "localpha0000001", "Alpha")

	options, err := getLocationOptions(app)
	if err != nil {
		t.Fatalf("get location options: %v", err)
	}
	if len(options.entries) != 2 {
		t.Fatalf("location options = %d, want 2", len(options.entries))
	}
	if options.truncated {
		t.Error("location facet should not be truncated below its limit")
	}
	if options.entries[0].label != "Alpha" || options.entries[1].label != "Zebra" {
		t.Errorf("location options not name-ordered: %#v", options.entries)
	}
}

func TestGetLocationOptionsDuplicateLabelsDeterministicOrder(t *testing.T) {
	app := newArtworkSearchApp(t)
	// Two locations share the name "Florence"; the id tie-break must make the
	// result order deterministic rather than database-insertion dependent.
	saveSearchLocation(t, app, "loczzz000000001", "Florence")
	saveSearchLocation(t, app, "locaaa000000001", "Florence")

	options, err := getLocationOptions(app)
	if err != nil {
		t.Fatalf("get location options: %v", err)
	}
	if len(options.entries) != 2 {
		t.Fatalf("location options = %d, want 2", len(options.entries))
	}
	if options.entries[0].value != "locaaa000000001" || options.entries[1].value != "loczzz000000001" {
		t.Errorf("duplicate-label location order not id-deterministic: %#v", options.entries)
	}
}

func TestGetLocationOptionsExactLimitNotTruncated(t *testing.T) {
	app := newArtworkSearchApp(t)
	for i := 0; i < artworkLocationOptionsLimit; i++ {
		saveSearchLocation(t, app, fmt.Sprintf("loc%012d", i), fmt.Sprintf("Location %d", i))
	}

	options, err := getLocationOptions(app)
	if err != nil {
		t.Fatalf("get location options: %v", err)
	}
	if len(options.entries) != artworkLocationOptionsLimit {
		t.Fatalf("location options = %d, want exactly %d", len(options.entries), artworkLocationOptionsLimit)
	}
	if options.truncated {
		t.Error("location facet must not be truncated at exactly its limit")
	}
}

func TestGetLocationOptionsLimitPlusOneTruncated(t *testing.T) {
	app := newArtworkSearchApp(t)
	for i := 0; i < artworkLocationOptionsLimit+1; i++ {
		saveSearchLocation(t, app, fmt.Sprintf("loc%012d", i), fmt.Sprintf("Location %d", i))
	}

	options, err := getLocationOptions(app)
	if err != nil {
		t.Fatalf("get location options: %v", err)
	}
	if len(options.entries) != artworkLocationOptionsLimit {
		t.Fatalf("location options = %d, want capped %d", len(options.entries), artworkLocationOptionsLimit)
	}
	if !options.truncated {
		t.Error("expected the location facet to be truncated when an extra row exists")
	}
}

func TestGetPeriodOptionsDuplicateLabelsDeterministicOrder(t *testing.T) {
	app := newArtworkSearchApp(t)
	// Two periods share name and start; the id tie-break must make the result
	// order deterministic rather than database-insertion dependent.
	saveSearchArtPeriod(t, app, "periodzzz000001", "Baroque", 1600, 1750)
	saveSearchArtPeriod(t, app, "periodaaa000001", "Baroque", 1600, 1750)

	options, err := getArtPeriodOptions(app)
	if err != nil {
		t.Fatalf("get period options: %v", err)
	}
	if len(options.entries) != 2 {
		t.Fatalf("period options = %d, want 2", len(options.entries))
	}
	if options.entries[0].value != "periodaaa000001" || options.entries[1].value != "periodzzz000001" {
		t.Errorf("duplicate-label period order not id-deterministic: %#v", options.entries)
	}
}

func TestGetPeriodOptionsExactLimitNotTruncated(t *testing.T) {
	app := newArtworkSearchApp(t)
	for i := 0; i < artworkPeriodOptionsLimit; i++ {
		saveSearchArtPeriod(t, app, fmt.Sprintf("period%09d", i), fmt.Sprintf("Period %d", i), i, i)
	}

	options, err := getArtPeriodOptions(app)
	if err != nil {
		t.Fatalf("get period options: %v", err)
	}
	if len(options.entries) != artworkPeriodOptionsLimit {
		t.Fatalf("period options = %d, want exactly %d", len(options.entries), artworkPeriodOptionsLimit)
	}
	if options.truncated {
		t.Error("period facet must not be truncated at exactly its limit")
	}
}

func TestGetPeriodOptionsLimitPlusOneTruncated(t *testing.T) {
	app := newArtworkSearchApp(t)
	for i := 0; i < artworkPeriodOptionsLimit+1; i++ {
		saveSearchArtPeriod(t, app, fmt.Sprintf("period%09d", i), fmt.Sprintf("Period %d", i), i, i)
	}

	options, err := getArtPeriodOptions(app)
	if err != nil {
		t.Fatalf("get period options: %v", err)
	}
	if len(options.entries) != artworkPeriodOptionsLimit {
		t.Fatalf("period options = %d, want capped %d", len(options.entries), artworkPeriodOptionsLimit)
	}
	if !options.truncated {
		t.Error("expected the period facet to be truncated when an extra row exists")
	}
}

func TestBuildArtworkSearchViewClampsOutOfRangePage(t *testing.T) {
	app := newArtworkSearchApp(t)
	saveSearchArtist(t, app, "artistone000001", "Artist One")
	saveSearchArtwork(t, app, searchArtworkSeed{id: "workone00000001", title: "Only Work", authors: []string{"artistone000001"}, year: 1500, published: true})

	view, canonical, err := buildArtworkSearchView(app, neturl.Values{}, 999, 16)
	if err != nil {
		t.Fatalf("build view: %v", err)
	}
	if canonical != "/artworks" {
		t.Errorf("canonical = %q, want /artworks (out-of-range page reset)", canonical)
	}
	if view.Results.ResultCount != 1 {
		t.Errorf("result count = %d, want 1", view.Results.ResultCount)
	}
}

func TestBuildArtworkSearchViewIssuesBoundedQueries(t *testing.T) {
	small := newArtworkSearchApp(t)
	saveSearchArtist(t, small, "artistone000001", "Artist One")
	for i := 0; i < 5; i++ {
		saveSearchArtwork(t, small, searchArtworkSeed{
			id: fmt.Sprintf("worksmall%06d", i), title: fmt.Sprintf("Small %d", i),
			authors: []string{"artistone000001"}, year: 1500 + i, published: true,
		})
	}
	large := newArtworkSearchApp(t)
	saveSearchArtist(t, large, "artistone000001", "Artist One")
	for i := 0; i < 60; i++ {
		saveSearchArtwork(t, large, searchArtworkSeed{
			id: fmt.Sprintf("worklarge%06d", i), title: fmt.Sprintf("Large %d", i),
			authors: []string{"artistone000001"}, year: 1500 + i, published: true,
		})
	}

	smallCount, err := countSearchQueries(small, func() error {
		_, _, err := buildArtworkSearchView(small, neturl.Values{}, 1, 16)
		return err
	})
	if err != nil {
		t.Fatalf("small build: %v", err)
	}
	largeCount, err := countSearchQueries(large, func() error {
		_, _, err := buildArtworkSearchView(large, neturl.Values{}, 1, 16)
		return err
	})
	if err != nil {
		t.Fatalf("large build: %v", err)
	}

	if smallCount == 0 {
		t.Fatal("expected to observe queries, got 0")
	}
	if smallCount != largeCount {
		t.Errorf("query count grew with artwork count: %d (5 artworks) vs %d (60 artworks)", smallCount, largeCount)
	}
}

func countSearchQueries(app *pocketbase.PocketBase, fn func() error) (int, error) {
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

func assertTitles(t *testing.T, view pages.ArtworkSearchView, want []string) {
	t.Helper()
	got := make([]string, 0, len(view.Results.Artworks))
	for _, artwork := range view.Results.Artworks {
		got = append(got, artwork.Title)
	}
	if len(got) != len(want) {
		t.Fatalf("titles = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("titles = %v, want %v", got, want)
		}
	}
}

func TestArtworksRouteRendersFullPageAndFragment(t *testing.T) {
	app := newArtworkSearchApp(t)
	saveSearchArtist(t, app, "artistone000001", "Artist One")
	saveSearchArtwork(t, app, searchArtworkSeed{id: "workone00000001", title: "Only Work", authors: []string{"artistone000001"}, year: 1500, published: true})

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

	RegisterArtworksHandlers(app)

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
		mux.ServeHTTP(full, httptest.NewRequest(http.MethodGet, "/artworks?artist_id=artistone000001&sort=date", nil))
		if full.Code != http.StatusOK {
			t.Errorf("full page status = %d, want %d", full.Code, http.StatusOK)
		}
		if got := full.Header().Get("HX-Push-Url"); got != "/artworks?artist_id=artistone000001&sort=date" {
			t.Errorf("HX-Push-Url = %q, want retained artist ID", got)
		}
		body := full.Body.String()
		for _, expected := range []string{"<h1", "Artworks", "SCHOOL", "FORM", "TECHNIQUE", "PERIOD", "COLLECTION", "DATE"} {
			if !strings.Contains(body, expected) {
				t.Errorf("full page missing %q", expected)
			}
		}
		if strings.Contains(body, `name="tone"`) || strings.Contains(body, ">TONE<") {
			t.Error("full page must not expose deferred tone UI or state")
		}
		if !strings.Contains(body, `type="hidden" name="artist_id" value="artistone000001"`) {
			t.Error("full page must retain exact artist ID as non-visible GET state")
		}

		fragment := httptest.NewRecorder()
		fragmentRequest := httptest.NewRequest(http.MethodGet, "/artworks/results?artist_id=artistone000001&q=Only&sort=date", nil)
		fragmentRequest.Header.Set("HX-Request", "true")
		mux.ServeHTTP(fragment, fragmentRequest)
		if fragment.Code != http.StatusOK {
			t.Errorf("fragment status = %d, want %d", fragment.Code, http.StatusOK)
		}
		if got := fragment.Header().Get("HX-Push-Url"); got != "/artworks/results?artist_id=artistone000001&q=Only&sort=date" {
			t.Errorf("HX-Push-Url = %q, want retained artist ID", got)
		}
		fragmentBody := fragment.Body.String()
		if !strings.Contains(fragmentBody, "id=\"artwork-search-results\"") {
			t.Error("fragment missing results container")
		}
		if strings.Contains(fragmentBody, "id=\"artwork-filters\"") {
			t.Error("fragment must not include the filter form")
		}
		if !strings.Contains(fragmentBody, "Only Work") {
			t.Error("fragment must retain the artist holding while refining the catalogue query")
		}

		return nil
	}); err != nil {
		t.Fatalf("trigger serve event: %v", err)
	}
}
