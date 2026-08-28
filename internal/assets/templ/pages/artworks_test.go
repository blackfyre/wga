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
		GridUrl:     "/artworks",
		ListUrl:     "/artworks?view=list",
		ResetUrl:    "/artworks",
		SortOptions: []ArtworkSortOption{
			{Key: "catalogue", Label: "CATALOGUE", Href: "/artworks", Active: true},
			{Key: "date", Label: "DATE", Href: "/artworks?sort=date"},
			{Key: "artist", Label: "ARTIST", Href: "/artworks?sort=artist"},
			{Key: "title", Label: "TITLE", Href: "/artworks?sort=title"},
		},
		SortDirLabel:  "↑ ARCHIVE ORDER",
		SortToggleUrl: "/artworks?dir=desc",
	}
}

func sampleArtworkSearchFacets() ArtworkSearchFacets {
	return ArtworkSearchFacets{
		Query:     ArtworkSearchFacet{Label: "TITLE OR ARTIST", Summary: "ANY", Open: true},
		Technique: ArtworkSearchFacet{Label: "TECHNIQUE", Summary: "ANY"},
		School:    ArtworkSearchFacet{Label: "SCHOOL", Summary: "ANY", Open: true},
		Form:      ArtworkSearchFacet{Label: "FORM", Summary: "ANY"},
		Type:      ArtworkSearchFacet{Label: "TYPE", Summary: "ANY"},
		Period:    ArtworkSearchFacet{Label: "PERIOD", Summary: "ANY"},
		Collection: ArtworkSearchCollectionFacet{
			Facet: ArtworkSearchFacet{Label: "COLLECTION", Summary: "ANY"},
			Name:  "venue",
			QueryField: dto.Field{
				ID:          "artwork-venue-query",
				Name:        "venue_q",
				Label:       "FILTER COLLECTIONS BY NAME",
				Type:        "search",
				Placeholder: "filter collections",
			},
		},
		Year: ArtworkSearchFacet{Label: "YEAR RANGE", Summary: "200–1900", Last: true},
		YearRange: dto.RangeField{
			Label:     "YEAR RANGE",
			FromID:    "year_from",
			FromName:  "year_from",
			FromValue: 200,
			ToID:      "year_to",
			ToName:    "year_to",
			ToValue:   1900,
			Min:       200,
			Max:       1900,
			Step:      10,
		},
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
		SchoolGroup: dto.ChipGroup{Legend: "SCHOOL", Name: "art_school", Options: []dto.ChipOption{{Label: "ALL", Value: "", Checked: true}}},
		FormGroup:   dto.ChipGroup{Legend: "FORM", Name: "art_form", Options: []dto.ChipOption{{Label: "ALL", Value: "", Checked: true}}},
		TypeGroup:   dto.ChipGroup{Legend: "TYPE", Name: "art_type", Options: []dto.ChipOption{{Label: "ALL", Value: "", Checked: true}}},
		PeriodGroup: dto.ChipGroup{Legend: "PERIOD", Name: "period", Note: "No values are recorded for this filter yet."},
		Facets:      sampleArtworkSearchFacets(),
		Sort:        "catalogue",
		Dir:         "asc",
		ClearUrl:    "/artworks",
		HxTarget:    "#artwork-search",
		Results:     sampleArtworkSearchResults(),
	}
}

// facetOpenStates returns whether each <details> disclosure, in document order,
// carries the open attribute.
func facetOpenStates(t *testing.T, rendered string) []bool {
	t.Helper()

	states := []bool{}
	rest := rendered
	for {
		idx := strings.Index(rest, "<details")
		if idx < 0 {
			break
		}
		end := strings.Index(rest[idx:], ">")
		if end < 0 {
			t.Fatal("unterminated <details> opening tag")
		}
		states = append(states, strings.Contains(rest[idx:idx+end], " open"))
		rest = rest[idx+end+1:]
	}

	return states
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
		Artist:    dto.Artist{FilingName: "Artist, Sample", ShortName: "Sample"},
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
		Artist:    dto.Artist{FilingName: "Artist, Sample", ShortName: "Sample"},
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

	for _, label := range []string{"TITLE OR ARTIST", "TECHNIQUE", "SCHOOL", "FORM", "TYPE", "PERIOD", "COLLECTION"} {
		if !strings.Contains(rendered, label) {
			t.Errorf("expected %s filter label", label)
		}
	}
	if !strings.Contains(rendered, `name="technique"`) {
		t.Error("expected the technique search input")
	}
}

func TestArtworkFilterBlockRendersSemanticFacetDisclosures(t *testing.T) {
	rendered := renderArtworkFilterBlock(t, sampleArtworkSearchView())

	if got := strings.Count(rendered, "<details"); got != 8 {
		t.Errorf("expected 8 <details> disclosures, got %d", got)
	}
	if got := strings.Count(rendered, "<summary"); got != 8 {
		t.Errorf("expected 8 <summary> elements, got %d", got)
	}
}

func TestArtworkFilterBlockOpensTitleAndSchoolByDefault(t *testing.T) {
	rendered := renderArtworkFilterBlock(t, sampleArtworkSearchView())

	states := facetOpenStates(t, rendered)
	want := []bool{true, false, true, false, false, false, false, false}
	if len(states) != len(want) {
		t.Fatalf("facet count = %d, want %d", len(states), len(want))
	}
	for i := range want {
		if states[i] != want[i] {
			t.Errorf("facet %d open = %v, want %v", i, states[i], want[i])
		}
	}
}

func TestArtworkFilterBlockReopensActiveFacets(t *testing.T) {
	view := sampleArtworkSearchView()
	view.Facets.Form = ArtworkSearchFacet{Label: "FORM", Summary: "PAINTING", Active: true, Open: true}
	view.Facets.Period = ArtworkSearchFacet{Label: "PERIOD", Summary: "BAROQUE", Active: true, Open: true}
	view.FormGroup.Options = []dto.ChipOption{
		{Label: "ALL", Value: "", Checked: false},
		{Label: "Painting", Value: "painting", Checked: true},
	}
	view.PeriodGroup.Options = []dto.ChipOption{
		{Label: "ALL", Value: "", Checked: false},
		{Label: "Baroque", Value: "baroque", Checked: true},
	}

	rendered := renderArtworkFilterBlock(t, view)

	if !strings.Contains(rendered, "PAINTING") || !strings.Contains(rendered, "BAROQUE") {
		t.Error("expected active facet summaries to state their values")
	}

	states := facetOpenStates(t, rendered)
	if len(states) != 8 {
		t.Fatalf("facet count = %d, want 8", len(states))
	}
	if !states[3] {
		t.Error("expected the active FORM facet to reopen")
	}
	if !states[5] {
		t.Error("expected the active PERIOD facet to reopen")
	}
}

func TestArtworkFilterBlockShowsActiveCount(t *testing.T) {
	base := renderArtworkFilterBlock(t, sampleArtworkSearchView())
	if strings.Contains(base, "ACTIVE") {
		t.Error("expected no active count when no filters are set")
	}

	view := sampleArtworkSearchView()
	view.Facets.ActiveCount = 3
	rendered := renderArtworkFilterBlock(t, view)
	if !strings.Contains(rendered, "3 ACTIVE") {
		t.Error("expected the heading to report 3 ACTIVE filters")
	}
}

func TestArtworkFilterBlockRendersVenueFacet(t *testing.T) {
	view := sampleArtworkSearchView()
	view.Facets.Collection = ArtworkSearchCollectionFacet{
		Facet: ArtworkSearchFacet{Label: "COLLECTION", Summary: "MAURITSHUIS", Active: true, Open: true},
		Name:  "venue",
		QueryField: dto.Field{
			ID:          "artwork-venue-query",
			Name:        "venue_q",
			Label:       "FILTER COLLECTIONS BY NAME",
			Type:        "search",
			Value:       "maurits",
			Placeholder: "filter collections",
		},
		Options: []ArtworkSearchCollectionOption{
			{Label: "Mauritshuis, The Hague", Value: "mauritshuis", Count: 12, Selected: true},
			{Label: "Rijksmuseum, Amsterdam", Value: "rijksmuseum", Count: 7},
		},
		Note:            "Showing 2 of 130 collections; omitted collections hold 41 works. Keep typing to narrow.",
		TotalOptions:    130,
		OmittedOptions:  128,
		OmittedHoldings: 41,
	}

	rendered := renderArtworkFilterBlock(t, view)

	if !strings.Contains(rendered, `name="venue"`) {
		t.Error("expected the venue radio group")
	}
	if !strings.Contains(rendered, `name="venue_q"`) {
		t.Error("expected the venue_q collection search input")
	}
	if !strings.Contains(rendered, `value="maurits"`) {
		t.Error("expected the venue_q search value to round-trip")
	}
	if !strings.Contains(rendered, "Mauritshuis, The Hague") {
		t.Error("expected the counted collection option label")
	}
	if !strings.Contains(rendered, ">12</span>") {
		t.Error("expected the collection holding count")
	}
	if !strings.Contains(rendered, "omitted collections hold 41 works") {
		t.Error("expected the hidden-holdings note from server fields")
	}
}

func TestArtworkFilterBlockRendersSharedYearRange(t *testing.T) {
	rendered := renderArtworkFilterBlock(t, sampleArtworkSearchView())

	if !strings.Contains(rendered, `name="year_from"`) {
		t.Error("expected the shared year_from range input")
	}
	if !strings.Contains(rendered, `name="year_to"`) {
		t.Error("expected the shared year_to range input")
	}
	if !strings.Contains(rendered, `value="200"`) || !strings.Contains(rendered, `value="1900"`) {
		t.Error("expected the year range bounds as input values")
	}
}

func TestArtworkFilterBlockAbsentToneAndLocationFields(t *testing.T) {
	rendered := renderArtworkFilterBlock(t, sampleArtworkSearchView())

	for _, forbidden := range []string{`name="tone"`, ">TONE<", `name="location"`, "LOCATION"} {
		if strings.Contains(rendered, forbidden) {
			t.Errorf("filter block must not expose %q", forbidden)
		}
	}
}

func TestArtworkFilterBlockRendersHonestUnavailableFilters(t *testing.T) {
	rendered := renderArtworkFilterBlock(t, sampleArtworkSearchView())

	count := strings.Count(rendered, "No values are recorded for this filter yet.")
	if count != 1 {
		t.Errorf("expected 1 honest unavailable note (period), got %d", count)
	}
}

func TestArtworkFilterBlockPreservesNoJsForm(t *testing.T) {
	rendered := renderArtworkFilterBlock(t, sampleArtworkSearchView())

	if !strings.Contains(rendered, `action="/artworks"`) || !strings.Contains(rendered, `method="GET"`) {
		t.Error("expected an ordinary GET form")
	}
	if !strings.Contains(rendered, `<noscript>`) || !strings.Contains(rendered, "APPLY FILTERS") {
		t.Error("expected a no-JavaScript apply button")
	}
}

func TestArtworkFilterBlockRendersVenueClearControl(t *testing.T) {
	rendered := renderArtworkFilterBlock(t, sampleArtworkSearchView())

	if !strings.Contains(rendered, `name="venue"`) {
		t.Error("expected the venue radio group")
	}
	if !strings.Contains(rendered, `value="" checked`) {
		t.Error("expected the ALL venue radio to be checked when no venue is selected")
	}
	if !strings.Contains(rendered, ">ALL</span>") {
		t.Error("expected a labelled ALL control to clear the collection")
	}
}

func TestArtworkFilterBlockPreservesSortDirViewState(t *testing.T) {
	view := sampleArtworkSearchView()
	view.Sort = "date"
	view.Dir = "desc"
	view.Results.View = "list"

	rendered := renderArtworkFilterBlock(t, view)

	for _, expected := range []string{`name="sort" value="date"`, `name="dir" value="desc"`, `name="view" value="list"`} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("filter form lost %q", expected)
		}
	}
}

func TestArtworkFilterBlockOmitsDefaultSortDirViewState(t *testing.T) {
	rendered := renderArtworkFilterBlock(t, sampleArtworkSearchView())

	for _, defaulted := range []string{`name="sort"`, `name="dir"`, `name="view"`} {
		if strings.Contains(rendered, defaulted) {
			t.Errorf("filter form must omit default %s state", defaulted)
		}
	}
}

func TestArtworkFilterBlockTriggerBubblesFromEverySearchInput(t *testing.T) {
	rendered := renderArtworkFilterBlock(t, sampleArtworkSearchView())

	if !strings.Contains(rendered, `hx-trigger="change, input changed delay:300ms"`) {
		t.Error("expected a form-bubbled change/input trigger")
	}
	if strings.Contains(rendered, "from:") {
		t.Error("expected no from: clause, which binds only the first matching input")
	}
	for _, input := range []string{`name="q"`, `name="technique"`, `name="venue_q"`} {
		if !strings.Contains(rendered, input) {
			t.Errorf("filter form missing search input %s", input)
		}
	}
	if got := strings.Count(rendered, `type="search"`); got != 3 {
		t.Errorf("expected all 3 search inputs inside the form, got %d", got)
	}
}

func TestArtworkSortAndViewLinksTargetFullBlock(t *testing.T) {
	rendered := renderArtworkSearchResults(t, sampleArtworkSearchResults())

	for _, link := range []string{
		`hx-get="/artworks"`,
		`hx-get="/artworks?sort=date"`,
		`hx-get="/artworks?view=list"`,
		`hx-get="/artworks?dir=desc"`,
	} {
		if !strings.Contains(rendered, link) {
			t.Errorf("results fragment missing canonical link %s", link)
		}
	}
	if strings.Contains(rendered, "/artworks/results") {
		t.Error("sort/view links must use the canonical /artworks path, not /artworks/results")
	}
	if got := strings.Count(rendered, `hx-target="#artwork-search"`); got != 7 {
		t.Errorf("expected all 7 sort/view controls to target #artwork-search, got %d", got)
	}
	if got := strings.Count(rendered, `hx-select="#artwork-search"`); got != 7 {
		t.Errorf("expected all 7 sort/view controls to select #artwork-search, got %d", got)
	}
	if strings.Contains(rendered, `hx-target="#artwork-search-results"`) {
		t.Error("sort/view controls must not remain result-local")
	}
}
