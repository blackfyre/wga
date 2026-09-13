package artworks

import (
	"net/url"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/assets/templ/pages"
)

func TestArtworkMultiFacetsUseORWithinAndANDBetweenFacets(t *testing.T) {
	app := newArtworkSearchApp(t)
	saveSearchArtist(t, app, "facetartist0001", "Facet Artist")
	saveSearchTaxonomy(t, app, "schools", "facetschoold001", "dutch", "Dutch")
	saveSearchTaxonomy(t, app, "schools", "facetschooli001", "italian", "Italian")
	saveSearchTaxonomy(t, app, "art_forms", "facetformpain01", "painting", "Painting")
	saveSearchTaxonomy(t, app, "art_forms", "facetformscul01", "sculpture", "Sculpture")
	saveSearchArtwork(t, app, searchArtworkSeed{id: "facetwork000001", title: "Dutch Painting", authors: []string{"facetartist0001"}, school: "facetschoold001", form: "facetformpain01", published: true})
	saveSearchArtwork(t, app, searchArtworkSeed{id: "facetwork000002", title: "Italian Painting", authors: []string{"facetartist0001"}, school: "facetschooli001", form: "facetformpain01", published: true})
	saveSearchArtwork(t, app, searchArtworkSeed{id: "facetwork000003", title: "Italian Sculpture", authors: []string{"facetartist0001"}, school: "facetschooli001", form: "facetformscul01", published: true})

	view, canonical, err := buildArtworkSearchView(app, url.Values{
		"art_school": {"dutch", "italian"},
		"art_form":   {"painting"},
		"view":       {"list"},
	}, 1, 16)
	if err != nil {
		t.Fatalf("build artwork search view: %v", err)
	}
	assertTitles(t, view, []string{"Dutch Painting", "Italian Painting"})
	if canonical != "/artworks?art_form=painting&art_school=dutch&art_school=italian&view=list" {
		t.Fatalf("canonical = %q", canonical)
	}
	if view.Facets.ActiveCount != 2 || view.Facets.School.Summary != "2 SELECTED" || !view.Facets.Form.Active {
		t.Fatalf("facet state = %#v", view.Facets)
	}
	if got := multiFacetOptionCount(view.Facets.SchoolMulti, "dutch"); got != 1 {
		t.Fatalf("Dutch candidate count = %d, want 1 with form filter retained", got)
	}
	if got := multiFacetOptionCount(view.Facets.SchoolMulti, "italian"); got != 1 {
		t.Fatalf("Italian candidate count = %d, want 1 with form filter retained", got)
	}
	if got := multiFacetOptionCount(view.Facets.FormMulti, "painting"); got != 2 {
		t.Fatalf("Painting candidate count = %d, want 2 with both schools retained", got)
	}
	if got := multiFacetOptionCount(view.Facets.FormMulti, "sculpture"); got != 1 {
		t.Fatalf("Sculpture candidate count = %d, want 1 with both schools retained", got)
	}
}

func TestArtworkMultiFacetCollapsedRosterRetainsSelectedOption(t *testing.T) {
	app := newArtworkSearchApp(t)
	options := map[string]string{"": "Any"}
	for _, value := range []string{"alpha", "bravo", "charlie", "delta", "echo", "foxtrot", "golf", "hotel", "india", "zulu"} {
		options[value] = strings.ToUpper(value[:1]) + value[1:]
	}
	f := buildFilters(url.Values{"art_school": {"zulu"}, "view": {"list"}})
	facet, err := buildCountedMultiFacet(app, f, nil, schoolMultiFacet, options)
	if err != nil {
		t.Fatalf("build counted facet: %v", err)
	}
	if facet.Total != 10 || len(facet.Options) != 9 {
		t.Fatalf("collapsed roster total/visible = %d/%d, want 10/9", facet.Total, len(facet.Options))
	}
	option := multiFacetOption(facet, "zulu")
	if !option.Selected || option.Disabled {
		t.Fatalf("selected zero-count option = %#v, want selected and enabled", option)
	}
	if facet.ToggleLabel != "SHOW ALL 10" || !strings.Contains(facet.ToggleURL, "school_all=1") || !strings.Contains(facet.ToggleURL, "view=list") {
		t.Fatalf("toggle state = %q %q", facet.ToggleLabel, facet.ToggleURL)
	}
	if strings.Contains(facet.ClearURL, "art_school=") || !strings.Contains(facet.ClearURL, "view=list") {
		t.Fatalf("clear URL = %q", facet.ClearURL)
	}
}

func multiFacetOptionCount(facet pages.ArtworkSearchMultiFacet, value string) int {
	return multiFacetOption(facet, value).Count
}

func multiFacetOption(facet pages.ArtworkSearchMultiFacet, value string) pages.ArtworkSearchMultiOption {
	for _, option := range facet.Options {
		if option.Value == value {
			return option
		}
	}
	return pages.ArtworkSearchMultiOption{}
}
