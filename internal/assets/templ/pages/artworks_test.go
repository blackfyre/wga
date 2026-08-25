package pages

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
)

func renderArtworkSearchResults(t *testing.T, view ArtworkSearchResultsView) string {
	t.Helper()

	var output bytes.Buffer
	if err := ArtworkSearchResults(view).Render(context.Background(), &output); err != nil {
		t.Fatalf("render artwork search results: %v", err)
	}

	return output.String()
}

func renderArtworkFilterBlock(t *testing.T, view ArtworkSearchView) string {
	t.Helper()

	var output bytes.Buffer
	if err := ArtworkSeachFilterBlock(view).Render(context.Background(), &output); err != nil {
		t.Fatalf("render artwork filter block: %v", err)
	}

	return output.String()
}

func sampleArtworkSearchResults() ArtworkSearchResultsView {
	return ArtworkSearchResultsView{
		ResultCount: 1,
		View:        "grid",
		GridUrl:     "/artworks/results",
		ListUrl:     "/artworks/results?view=list",
		ResetUrl:    "/artworks",
		SortOptions: []ArtworkSortOption{
			{Key: "catalogue", Label: "CATALOGUE", Href: "/artworks/results", Active: true},
			{Key: "date", Label: "DATE", Href: "/artworks/results?sort=date"},
			{Key: "artist", Label: "ARTIST", Href: "/artworks/results?sort=artist"},
			{Key: "title", Label: "TITLE", Href: "/artworks/results?sort=title"},
		},
		SortDirLabel:  "↑ ARCHIVE ORDER",
		SortToggleUrl: "/artworks/results?dir=desc",
	}
}

func sampleArtworkSearchView() ArtworkSearchView {
	return ArtworkSearchView{
		NameField: dto.Field{
			ID:          "artwork-query",
			Name:        "q",
			Label:       "TITLE OR ARTIST",
			Type:        "search",
			Placeholder: "e.g. milkmaid",
		},
		TechniqueField: dto.Field{
			ID:          "artwork-technique",
			Name:        "technique",
			Label:       "TECHNIQUE",
			Type:        "search",
			Placeholder: "e.g. oil on canvas",
		},
		SchoolGroup:   dto.ChipGroup{Legend: "SCHOOL", Name: "art_school", Options: []dto.ChipOption{{Label: "ALL", Value: "", Checked: true}}},
		FormGroup:     dto.ChipGroup{Legend: "FORM", Name: "art_form", Options: []dto.ChipOption{{Label: "ALL", Value: "", Checked: true}}},
		TypeGroup:     dto.ChipGroup{Legend: "TYPE", Name: "art_type", Options: []dto.ChipOption{{Label: "ALL", Value: "", Checked: true}}},
		PeriodGroup:   dto.ChipGroup{Legend: "PERIOD", Name: "period", Note: "No values are recorded for this filter yet."},
		LocationGroup: dto.ChipGroup{Legend: "LOCATION", Name: "location", Note: "No values are recorded for this filter yet."},
		YearFrom:      "200",
		YearTo:        "1900",
		ClearUrl:      "/artworks",
		HxTarget:      "#artwork-search-results",
		Results:       sampleArtworkSearchResults(),
	}
}

func TestArtworkMatchCount(t *testing.T) {
	for _, test := range []struct {
		total int
		want  string
	}{
		{total: 0, want: "0 WORKS MATCH"},
		{total: 1, want: "1 WORK MATCHES"},
		{total: 5, want: "5 WORKS MATCH"},
	} {
		if got := artworkMatchCount(test.total); got != test.want {
			t.Errorf("artworkMatchCount(%d) = %q, want %q", test.total, got, test.want)
		}
	}
}

func TestArtworkSearchResultsRendersSortCriteriaAndDirection(t *testing.T) {
	rendered := renderArtworkSearchResults(t, sampleArtworkSearchResults())

	for _, label := range []string{"CATALOGUE", "DATE", "ARTIST", "TITLE"} {
		if !strings.Contains(rendered, ">"+label+"</a>") {
			t.Errorf("expected %s sort chip", label)
		}
	}
	if !strings.Contains(rendered, "ARCHIVE ORDER") {
		t.Error("expected the criterion-specific direction label, not an abstract ASC/DESC label")
	}
	if !strings.Contains(rendered, `aria-pressed="true"`) {
		t.Error("expected the active sort criterion to carry aria-pressed")
	}
}

func TestArtworkSearchResultsRendersGridView(t *testing.T) {
	view := sampleArtworkSearchResults()
	view.Artworks = dto.ImageGrid{{
		Url:       "/artworks/sample-work-123",
		Thumb:     "/api/files/artworks/123/image.jpg",
		Title:     "Sample Work",
		Technique: "Oil on canvas",
		Artist:    dto.Artist{Name: "Sample Artist"},
	}}
	rendered := renderArtworkSearchResults(t, view)

	if !strings.Contains(rendered, `data-view="grid"`) {
		t.Error("expected the grid view marker")
	}
	if !strings.Contains(rendered, `href="/artworks/sample-work-123"`) {
		t.Error("expected the artwork record link")
	}
}

func TestArtworkSearchResultsRendersListView(t *testing.T) {
	view := sampleArtworkSearchResults()
	view.View = "list"
	view.Artworks = dto.ImageGrid{{
		Url:       "/artworks/sample-work-123",
		Thumb:     "/api/files/artworks/123/image.jpg",
		Title:     "Sample Work",
		Technique: "Oil on canvas",
		Artist:    dto.Artist{Name: "Sample Artist"},
	}}
	rendered := renderArtworkSearchResults(t, view)

	if !strings.Contains(rendered, `data-view="list"`) {
		t.Error("expected the list view marker")
	}
	if !strings.Contains(rendered, `data-kbd-cols="1"`) {
		t.Error("expected the single-column list container")
	}
}

func TestArtworkSearchResultsRendersNoMatchingEmptyState(t *testing.T) {
	view := sampleArtworkSearchResults()
	view.ActiveFiltering = true
	view.ResultCount = 0
	view.ResetUrl = "/artworks"
	rendered := renderArtworkSearchResults(t, view)

	if !strings.Contains(rendered, "NO MATCHING WORKS") {
		t.Error("expected the NO MATCHING WORKS empty state")
	}
	if !strings.Contains(rendered, `href="/artworks"`) || !strings.Contains(rendered, "RESET FILTERS") {
		t.Error("expected a reset link to the clear URL")
	}
}

func TestArtworkSearchResultsRendersBrowseEmptyState(t *testing.T) {
	view := sampleArtworkSearchResults()
	view.ActiveFiltering = false
	view.ResultCount = 0
	rendered := renderArtworkSearchResults(t, view)

	if !strings.Contains(rendered, "SET FILTERS TO BROWSE THE COLLECTION") {
		t.Error("expected the browse-prompt empty state")
	}
}

func TestArtworkFilterBlockRendersCatalogueFilters(t *testing.T) {
	rendered := renderArtworkFilterBlock(t, sampleArtworkSearchView())

	for _, label := range []string{"TITLE OR ARTIST", "TECHNIQUE", "SCHOOL", "FORM", "TYPE", "PERIOD", "LOCATION"} {
		if !strings.Contains(rendered, label) {
			t.Errorf("expected %s filter label", label)
		}
	}
	if !strings.Contains(rendered, `name="technique"`) {
		t.Error("expected the technique search input")
	}
}

func TestArtworkFilterBlockRendersHonestUnavailableFilters(t *testing.T) {
	rendered := renderArtworkFilterBlock(t, sampleArtworkSearchView())

	count := strings.Count(rendered, "No values are recorded for this filter yet.")
	if count != 2 {
		t.Errorf("expected 2 honest unavailable notes (period and location), got %d", count)
	}
}

func TestArtworkFilterBlockPreservesNoJsForm(t *testing.T) {
	rendered := renderArtworkFilterBlock(t, sampleArtworkSearchView())

	if !strings.Contains(rendered, `action="/artworks/results"`) || !strings.Contains(rendered, `method="GET"`) {
		t.Error("expected an ordinary GET form")
	}
	if !strings.Contains(rendered, `<noscript>`) || !strings.Contains(rendered, "APPLY FILTERS") {
		t.Error("expected a no-JavaScript apply button")
	}
}
