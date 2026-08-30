package artworks

import (
	"net/url"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/assets/templ/pages"
)

func TestFiltersBuildFilterIncludesDateRange(t *testing.T) {
	filters := &filters{
		Title:    "Still life",
		YearFrom: "1902",
		YearTo:   "1911",
	}

	filterString, params := filters.BuildFilter()

	for _, condition := range []string{
		"title ~ {:title}",
		"year >= {:year_from}",
		"year <= {:year_to}",
	} {
		if !strings.Contains(filterString, condition) {
			t.Errorf("filter %q does not contain %q", filterString, condition)
		}
	}

	for key, want := range map[string]string{
		"title":     "Still life",
		"year_from": "1902",
		"year_to":   "1911",
	} {
		if got := params[key]; got != want {
			t.Errorf("parameter %q = %q, want %q", key, got, want)
		}
	}
}

func TestFiltersBuildFilterMatchesTitleOrArtistQuery(t *testing.T) {
	filters := &filters{Query: "milkmaid"}

	filterString, params := filters.BuildFilter()

	if !strings.Contains(filterString, "(title ~ {:query} || author.filing_name ~ {:query})") {
		t.Errorf("filter %q does not match title or artist", filterString)
	}

	if got := params["query"]; got != "milkmaid" {
		t.Errorf("parameter query = %q, want %q", got, "milkmaid")
	}
}

func TestFiltersBuildFilterUsesExactArtistID(t *testing.T) {
	filters := &filters{ArtistString: "Aachen, Hans von", ArtistID: "artistone000001"}

	filterString, params := filters.BuildFilter()

	if !strings.Contains(filterString, "author.id ?= {:artist_id}") {
		t.Errorf("filter %q does not contain exact artist condition", filterString)
	}
	if strings.Contains(filterString, "author.filing_name ~ {:artist}") {
		t.Errorf("filter %q must not contain the legacy artist condition", filterString)
	}
	if got := params["artist_id"]; got != "artistone000001" {
		t.Errorf("artist_id = %q, want artistone000001", got)
	}
	if _, ok := params["artist"]; ok {
		t.Errorf("legacy artist parameter = %q, want omitted", params["artist"])
	}
}

func TestBuildFiltersExactArtistIDTakesPrecedence(t *testing.T) {
	filters := buildFilters(url.Values{
		"artist":    {"Aachen, Hans von"},
		"artist_id": {"artistone000001"},
	})

	if filters.ArtistString != "" {
		t.Errorf("legacy artist = %q, want empty", filters.ArtistString)
	}
	if filters.ArtistID != "artistone000001" {
		t.Errorf("artist ID = %q, want artistone000001", filters.ArtistID)
	}
	if got, want := filters.BuildPath("/artworks"), "/artworks?artist_id=artistone000001"; got != want {
		t.Errorf("path = %q, want %q", got, want)
	}
}

func TestBuildArtworkSearchPathPreservesExactArtistIDInDualMode(t *testing.T) {
	path := buildArtworkSearchPath("/artworks", &filters{ArtistID: "artistone000001"}, &pages.ArtworkSearchDualMode{
		LeftPath:      "/artists/left",
		RightPath:     "/artists/right",
		LeftRenderTo:  "#left",
		RightRenderTo: "#right",
		Target:        "right",
	})

	values, err := url.Parse(path)
	if err != nil {
		t.Fatalf("parse path: %v", err)
	}
	if got := values.Query().Get("artist_id"); got != "artistone000001" {
		t.Errorf("artist ID = %q, want artistone000001", got)
	}
}

func TestFiltersBuildPathPreservesViewAndDateRange(t *testing.T) {
	filters := &filters{
		SchoolString: "bohemian",
		YearFrom:     "1902",
		YearTo:       "1911",
		View:         "list",
		Page:         "2",
	}

	path := filters.BuildPath("/artworks/results")

	for _, value := range []string{
		"art_school=bohemian",
		"year_from=1902",
		"year_to=1911",
		"view=list",
		"page=2",
	} {
		if !strings.Contains(path, value) {
			t.Errorf("path %q does not contain %q", path, value)
		}
	}
}

func TestArtworkSearchViewDefaultsToGrid(t *testing.T) {
	if got := artworkSearchView("unexpected"); got != "grid" {
		t.Errorf("unexpected view = %q, want grid", got)
	}
}

func TestFiltersBuildPathOmitsDefaultGridView(t *testing.T) {
	path := (&filters{View: "grid"}).BuildPath("/artworks")

	if path != "/artworks" {
		t.Errorf("path = %q, want /artworks", path)
	}
}

func TestFiltersBuildFilterIncludesTechnique(t *testing.T) {
	filterString, params := (&filters{TechniqueString: "fresco"}).BuildFilter()

	if !strings.Contains(filterString, "technique ~ {:technique}") {
		t.Errorf("filter %q does not contain technique clause", filterString)
	}
	if got := params["technique"]; got != "fresco" {
		t.Errorf("parameter technique = %q, want %q", got, "fresco")
	}
}

func TestFiltersBuildFilterIncludesPeriodAndLocation(t *testing.T) {
	filterString, params := (&filters{
		PeriodString:   "periodbaroque1",
		LocationString: "locflorence01",
	}).BuildFilter()

	for _, condition := range []string{
		"art_period_id = {:period}",
		"current_location_id = {:location}",
	} {
		if !strings.Contains(filterString, condition) {
			t.Errorf("filter %q does not contain %q", filterString, condition)
		}
	}

	for key, want := range map[string]string{
		"period":   "periodbaroque1",
		"location": "locflorence01",
	} {
		if got := params[key]; got != want {
			t.Errorf("parameter %q = %q, want %q", key, got, want)
		}
	}
}

func TestFiltersBuildFilterCombinesStoredFilters(t *testing.T) {
	filterString, params := (&filters{
		SchoolString:    "dutch",
		ArtFormString:   "painting",
		TechniqueString: "oil",
	}).BuildFilter()

	for _, condition := range []string{
		"school.slug = {:art_school}",
		"form.slug = {:art_form}",
		"technique ~ {:technique}",
	} {
		if !strings.Contains(filterString, condition) {
			t.Errorf("filter %q does not contain %q", filterString, condition)
		}
	}

	for key, want := range map[string]string{
		"art_school": "dutch",
		"art_form":   "painting",
		"technique":  "oil",
	} {
		if got := params[key]; got != want {
			t.Errorf("parameter %q = %q, want %q", key, got, want)
		}
	}
}

func TestBuildFiltersParsesCatalogueFilters(t *testing.T) {
	f := buildFilters(url.Values{
		"art_school": {"dutch"},
		"period":     {"baroque"},
		"art_form":   {"painting"},
		"technique":  {"fresco"},
		"location":   {"Florence"},
	})

	if f.SchoolString != "dutch" || f.PeriodString != "baroque" || f.ArtFormString != "painting" || f.TechniqueString != "fresco" || f.LocationString != "Florence" {
		t.Errorf("parsed filters = %#v", f)
	}
}

func TestBuildFiltersRoundTripsPeriodAndVenue(t *testing.T) {
	f := buildFilters(url.Values{
		"period": {"baroque"},
		"venue":  {"Florence"},
	})

	path := f.BuildPath("/artworks")
	for _, value := range []string{"period=baroque", "venue=Florence"} {
		if !strings.Contains(path, value) {
			t.Errorf("path %q does not round-trip %q", path, value)
		}
	}
	if strings.Contains(path, "tone") {
		t.Errorf("path %q must not retain a deferred tone parameter", path)
	}
}

func TestFiltersPathOmitsDefaultSortAndDirection(t *testing.T) {
	path := (&filters{Sort: "catalogue", SortDir: "asc"}).BuildPath("/artworks")

	if path != "/artworks" {
		t.Errorf("path = %q, want /artworks (default sort and direction omitted)", path)
	}
}

func TestFiltersPathEmitsSortAndDirection(t *testing.T) {
	path := (&filters{Sort: "date", SortDir: "desc"}).BuildPath("/artworks")

	for _, value := range []string{"sort=date", "dir=desc"} {
		if !strings.Contains(path, value) {
			t.Errorf("path %q does not contain %q", path, value)
		}
	}
}

func TestForSortResetsPageAndDirection(t *testing.T) {
	f := buildFilters(url.Values{"sort": {"date"}, "dir": {"desc"}, "page": {"3"}})
	next := f.forSort("title")

	if next.Sort != "title" || next.SortDir != "asc" || next.Page != "" {
		t.Errorf("forSort = %#v, want title/asc/empty-page", next)
	}
}

func TestForSortPreservesUnrelatedFilters(t *testing.T) {
	f := buildFilters(url.Values{"q": {"milkmaid"}, "art_school": {"dutch"}, "view": {"list"}})
	next := f.forSort("artist")

	if next.Query != "milkmaid" || next.SchoolString != "dutch" || next.View != "list" {
		t.Errorf("forSort lost unrelated filters: %#v", next)
	}
}
