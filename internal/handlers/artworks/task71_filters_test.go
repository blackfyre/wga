package artworks

import (
	"net/url"
	"strings"
	"testing"
)

func TestTask71BuildFiltersCanonicalVenueState(t *testing.T) {
	f := buildFilters(url.Values{
		"venue": {"loc-paris"}, "venue_q": {"par"}, "year_from": {"1600"},
		"year_to": {"1700"}, "sort": {"title"}, "dir": {"desc"},
		"view": {"list"}, "page": {"3"},
	})

	if f.selectedVenue() != "loc-paris" || f.LocationString != "loc-paris" {
		t.Fatalf("venue state = %#v", f)
	}
	if f.ActiveFilterCount() != 2 {
		t.Fatalf("active filter count = %d, want venue plus year range", f.ActiveFilterCount())
	}
	path := f.BuildPath("/artworks")
	for _, part := range []string{"venue=loc-paris", "venue_q=par", "year_from=1600", "year_to=1700", "sort=title", "dir=desc", "view=list", "page=3"} {
		if !strings.Contains(path, part) {
			t.Errorf("canonical path %q missing %q", path, part)
		}
	}
	if strings.Contains(path, "location=") || strings.Contains(path, "tone=") {
		t.Errorf("canonical path exposes obsolete state: %q", path)
	}
}

func TestTask71LegacyLocationTranslationAndVenueConflict(t *testing.T) {
	legacy := buildFilters(url.Values{"location": {"loc-paris"}})
	if legacy.selectedVenue() != "loc-paris" || legacy.BuildPath("/artworks") != "/artworks?venue=loc-paris" {
		t.Fatalf("legacy location was not translated canonically: %#v, %q", legacy, legacy.BuildPath("/artworks"))
	}

	equal := buildFilters(url.Values{"venue": {"loc-paris"}, "location": {"loc-paris"}})
	if equal.VenueConflict || equal.BuildPath("/artworks") != "/artworks?venue=loc-paris" {
		t.Fatalf("equal canonical and legacy values should be accepted: %#v", equal)
	}

	conflict := buildFilters(url.Values{"venue": {"loc-paris"}, "location": {"loc-amsterdam"}})
	if !conflict.VenueConflict {
		t.Fatal("conflicting canonical and legacy venue values were accepted")
	}
}

func TestTask71VenueQueryIsNotAResultFilterOrActiveCount(t *testing.T) {
	without := buildFilters(url.Values{})
	withQuery := buildFilters(url.Values{"venue_q": {"museum"}})
	if withQuery.VenueQuery != "museum" || withQuery.ActiveFilterCount() != without.ActiveFilterCount() || withQuery.AnyFilterActive() {
		t.Fatalf("venue_q changed result filter state: %#v", withQuery)
	}
	filter, params := withQuery.BuildFilter()
	if strings.Contains(filter, "venue") || len(params) != 0 {
		t.Fatalf("venue_q entered artwork predicate: %q %#v", filter, params)
	}
}

func TestTask71SortAndViewLinksUseCanonicalPaths(t *testing.T) {
	f := buildFilters(url.Values{"q": {"milkmaid"}, "sort": {"title"}, "dir": {"desc"}, "view": {"list"}})

	for _, option := range buildSortOptions(f, nil) {
		if strings.HasPrefix(option.Href, "/artworks/results") {
			t.Errorf("sort option %q must target canonical /artworks, got %q", option.Key, option.Href)
		}
		if !strings.HasPrefix(option.Href, "/artworks") {
			t.Errorf("sort option %q must target /artworks, got %q", option.Key, option.Href)
		}
	}

	var title string
	for _, option := range buildSortOptions(f, nil) {
		if option.Key == "title" {
			title = option.Href
		}
	}
	for _, part := range []string{"q=milkmaid", "sort=title", "view=list"} {
		if !strings.Contains(title, part) {
			t.Errorf("title sort link %q missing %q", title, part)
		}
	}
	if strings.Contains(title, "dir=") {
		t.Errorf("title sort link %q must reset direction to the criterion default", title)
	}
}

func TestTask71SortViewPageAndDualStateRoundTrip(t *testing.T) {
	f := buildFilters(url.Values{"q": {"milkmaid"}, "sort": {"date"}, "dir": {"desc"}, "view": {"list"}, "page": {"4"}})
	dual := getDualModeSearchContext(url.Values{
		"dual_target": {"right"}, "dual_left": {"/artworks?a=1"}, "dual_right": {"default"},
		"dual_left_render_to": {"right"}, "dual_right_render_to": {"left"},
	})
	path := buildArtworkSearchPath("/artworks/results", f, dual)
	for _, part := range []string{"q=milkmaid", "sort=date", "dir=desc", "view=list", "page=4", "dual_target=right", "dual_left_render_to=right", "dual_right_render_to=left"} {
		if !strings.Contains(path, part) {
			t.Errorf("state path %q missing %q", path, part)
		}
	}
}

func TestTask71FacetDisclosureStateAndToneAbsence(t *testing.T) {
	app := newArtworkSearchApp(t)
	view, _, err := buildArtworkSearchView(app, url.Values{"q": {"milkmaid"}, "venue_q": {"museum"}, "year_from": {"1600"}}, 1, 16)
	if err != nil {
		t.Fatalf("build view: %v", err)
	}
	f := view.Facets
	if f.ActiveCount != 2 || !f.Query.Active || !f.Query.Open || !f.Year.Active || !f.Year.Open {
		t.Fatalf("active facet state = %#v", f)
	}
	if !f.Collection.Facet.Open || f.Collection.Facet.Active {
		t.Fatalf("venue_q disclosure state = %#v", f.Collection)
	}
	if !f.School.Open || f.Form.Open || f.Type.Open || f.Period.Open {
		t.Fatalf("default disclosure state = %#v", f)
	}
	if strings.Contains(view.ClearUrl, "tone=") {
		t.Fatalf("tone leaked into clear URL: %q", view.ClearUrl)
	}
}

func TestTask71YearRangeNormalisation(t *testing.T) {
	malformed := buildFilters(url.Values{"year_from": {"abc"}, "year_to": {"1700"}})
	if malformed.YearFrom != "" || malformed.YearTo != "1700" {
		t.Fatalf("malformed year_from did not become the default: %#v", malformed)
	}

	clamped := buildFilters(url.Values{"year_from": {"50"}, "year_to": {"99999"}})
	if clamped.YearFrom != "" || clamped.YearTo != "" {
		t.Fatalf("out-of-range years did not clamp to defaults: %#v", clamped)
	}

	reversed := buildFilters(url.Values{"year_from": {"1700"}, "year_to": {"1500"}})
	if reversed.YearFrom != "1500" || reversed.YearTo != "1700" {
		t.Fatalf("reversed bounds were not normalised: %#v", reversed)
	}
	filter, params := reversed.BuildFilter()
	if params["year_from"] != "1500" || params["year_to"] != "1700" {
		t.Fatalf("year predicate disagrees with normalised state: %q %#v", filter, params)
	}
	path := reversed.BuildPath("/artworks")
	for _, part := range []string{"year_from=1500", "year_to=1700"} {
		if !strings.Contains(path, part) {
			t.Errorf("canonical path %q missing %q", path, part)
		}
	}
}

func TestTask71YearRangeDisplayAndSummaryAgree(t *testing.T) {
	app := newArtworkSearchApp(t)
	view, canonical, err := buildArtworkSearchView(app, url.Values{"year_from": {"1700"}, "year_to": {"1500"}}, 1, 16)
	if err != nil {
		t.Fatalf("build view: %v", err)
	}
	if view.Facets.Year.Summary != "1500–1700" {
		t.Fatalf("year summary = %q, want 1500–1700", view.Facets.Year.Summary)
	}
	if view.Facets.YearRange.FromValue != 1500 || view.Facets.YearRange.ToValue != 1700 {
		t.Fatalf("year range display = %#v, want 1500–1700", view.Facets.YearRange)
	}
	for _, part := range []string{"year_from=1500", "year_to=1700"} {
		if !strings.Contains(canonical, part) {
			t.Errorf("canonical %q missing %q", canonical, part)
		}
	}
}
