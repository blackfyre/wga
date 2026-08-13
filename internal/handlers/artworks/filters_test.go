package artworks

import (
	"strings"
	"testing"
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

	if !strings.Contains(filterString, "(title ~ {:query} || author.name ~ {:query})") {
		t.Errorf("filter %q does not match title or artist", filterString)
	}

	if got := params["query"]; got != "milkmaid" {
		t.Errorf("parameter query = %q, want %q", got, "milkmaid")
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
