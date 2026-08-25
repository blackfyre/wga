package timeline

import (
	"net/url"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/assets/templ/pages"
)

// spyTimelineStore records which persistence reads the workflow performs so
// tests can prove the paged artwork query is skipped for empty windows.
type spyTimelineStore struct {
	periods []artPeriod
	artMin  int
	artMax  int
	total   int
	spans   []dateSpan
	works   []artworkRow

	countWorksCalls     int
	pagedListWorksCalls int
	listWorksLimits     []int
}

func (s *spyTimelineStore) listPeriods() ([]artPeriod, error) {
	return s.periods, nil
}

func (s *spyTimelineStore) artworkBounds() (int, int, error) {
	return s.artMin, s.artMax, nil
}

func (s *spyTimelineStore) countWorks(from int, to int) (int, error) {
	s.countWorksCalls++

	return s.total, nil
}

func (s *spyTimelineStore) dateSpans(from int, to int) ([]dateSpan, error) {
	return s.spans, nil
}

func (s *spyTimelineStore) listWorks(from int, to int, limit int, offset int) ([]artworkRow, error) {
	if limit == worksPageSize {
		s.pagedListWorksCalls++
	}
	s.listWorksLimits = append(s.listWorksLimits, limit)

	return s.works, nil
}

func TestBuildTimelineViewEmptyWindowNormalisesPageAndSkipsPagedQuery(t *testing.T) {
	store := &spyTimelineStore{
		periods: []artPeriod{{name: "Early", start: 1000, end: 1200}},
		total:   0,
	}

	view, canonicalURL, err := buildTimelineView(store, url.Values{"page": {"999"}})
	if err != nil {
		t.Fatalf("buildTimelineView: %v", err)
	}

	if canonicalURL != "/timeline" {
		t.Errorf("canonicalURL = %q, want /timeline (page=999 canonicalised away)", canonicalURL)
	}
	if view.Page != 1 {
		t.Errorf("view.Page = %d, want 1", view.Page)
	}
	if !view.HasRange {
		t.Error("period-only window must keep HasRange true")
	}
	if view.HasWorks {
		t.Error("period-only window must keep HasWorks false")
	}
	if store.pagedListWorksCalls != 0 {
		t.Errorf("paged listWorks called %d times, want 0", store.pagedListWorksCalls)
	}
}

func TestBuildTimelineViewEmptyChronologyNormalisesPage(t *testing.T) {
	store := &spyTimelineStore{}

	view, canonicalURL, err := buildTimelineView(store, url.Values{"page": {"999"}})
	if err != nil {
		t.Fatalf("buildTimelineView: %v", err)
	}

	if canonicalURL != "/timeline" {
		t.Errorf("canonicalURL = %q, want /timeline (page=999 canonicalised away)", canonicalURL)
	}
	if view.Page != 1 {
		t.Errorf("view.Page = %d, want 1", view.Page)
	}
	if view.HasRange {
		t.Error("empty chronology must keep HasRange false")
	}
	if store.countWorksCalls != 0 {
		t.Errorf("countWorks called %d times, want 0 (early return)", store.countWorksCalls)
	}
	if store.pagedListWorksCalls != 0 {
		t.Errorf("paged listWorks called %d times, want 0", store.pagedListWorksCalls)
	}
}

func TestBuildTimelineViewPagesOnlyWhenWorksExist(t *testing.T) {
	store := &spyTimelineStore{
		periods: []artPeriod{{name: "Baroque", start: 1500, end: 1750}},
		artMin:  1500,
		artMax:  1800,
		total:   20,
		works:   make([]artworkRow, worksPageSize),
	}

	view, canonicalURL, err := buildTimelineView(store, url.Values{"from": {"1600"}, "to": {"1700"}, "page": {"2"}})
	if err != nil {
		t.Fatalf("buildTimelineView: %v", err)
	}

	if canonicalURL != "/timeline?from=1600&to=1700&page=2" {
		t.Errorf("canonicalURL = %q, want /timeline?from=1600&to=1700&page=2", canonicalURL)
	}
	if view.Page != 2 || view.PageCount != 3 {
		t.Errorf("page = %d, pageCount = %d, want 2 and 3", view.Page, view.PageCount)
	}
	if store.pagedListWorksCalls != 1 {
		t.Errorf("paged listWorks called %d times, want 1", store.pagedListWorksCalls)
	}
}

func TestBuildTimelineViewCapsMarksAndWorks(t *testing.T) {
	store := &spyTimelineStore{
		periods: []artPeriod{{name: "Baroque", start: 1500, end: 1750}},
		artMin:  1500,
		artMax:  1800,
		total:   52866,
		spans:   []dateSpan{{Start: 1500, End: 0}},
		works:   make([]artworkRow, markCap),
	}

	_, _, err := buildTimelineView(store, url.Values{})
	if err != nil {
		t.Fatalf("buildTimelineView: %v", err)
	}

	// The workflow must request the mark lane at the 48-mark cap and the card
	// panel at the 8-card page size, never the full catalogue.
	foundMarkCap := false
	foundPageSize := false
	for _, limit := range store.listWorksLimits {
		if limit == markCap {
			foundMarkCap = true
		}
		if limit == worksPageSize {
			foundPageSize = true
		}
	}
	if !foundMarkCap {
		t.Errorf("listWorks never requested the %d-mark cap; limits = %v", markCap, store.listWorksLimits)
	}
	if !foundPageSize {
		t.Errorf("listWorks never requested the %d-card page size; limits = %v", worksPageSize, store.listWorksLimits)
	}
}

func densityByDecade(bars []pages.TimelineDensityBar) map[string]pages.TimelineDensityBar {
	byDecade := make(map[string]pages.TimelineDensityBar, len(bars))
	for _, bar := range bars {
		byDecade[bar.Decade] = bar
	}

	return byDecade
}

func TestDensityBinsCountsSpanningWorksAcrossDecades(t *testing.T) {
	// A work dated 1409–1410 touches both the 1400s and the 1410s bins.
	spans := []dateSpan{{Start: 1409, End: 1410}}

	bars := densityBins(spans, 1400, 1420)
	if len(bars) != 3 {
		t.Fatalf("densityBins length = %d, want 3 (1400, 1410, 1420)", len(bars))
	}

	byDecade := densityByDecade(bars)
	if byDecade["1400"].Count != 1 {
		t.Errorf("1400 count = %d, want 1 (straddles upper boundary)", byDecade["1400"].Count)
	}
	if byDecade["1410"].Count != 1 {
		t.Errorf("1410 count = %d, want 1 (straddles lower boundary)", byDecade["1410"].Count)
	}
	if byDecade["1420"].Count != 0 {
		t.Errorf("1420 count = %d, want 0", byDecade["1420"].Count)
	}
}

func TestDensityBinsClipsToSelectedWindow(t *testing.T) {
	// A work dated 1401–1402 falls inside the full 1400s decade but before the
	// selected window start of 1405, so the clipped first bin must not count it.
	spans := []dateSpan{{Start: 1401, End: 1402}}

	bars := densityBins(spans, 1405, 1415)
	if len(bars) != 2 {
		t.Fatalf("densityBins length = %d, want 2 (1400, 1410)", len(bars))
	}

	byDecade := densityByDecade(bars)
	if byDecade["1400"].Count != 0 {
		t.Errorf("1400 count = %d, want 0 (work is before the clipped bin)", byDecade["1400"].Count)
	}
	if byDecade["1410"].Count != 0 {
		t.Errorf("1410 count = %d, want 0", byDecade["1410"].Count)
	}
}

func TestDensityBinsCountsWorksCrossingIntoWindow(t *testing.T) {
	// A work started before the window but ending inside it must count, matching
	// the overlap semantics used by marks and counts.
	spans := []dateSpan{{Start: 1403, End: 1406}}

	bars := densityBins(spans, 1405, 1415)
	byDecade := densityByDecade(bars)
	if byDecade["1400"].Count != 1 {
		t.Errorf("1400 count = %d, want 1 (ends inside the clipped window)", byDecade["1400"].Count)
	}
}

func TestDensityBinsExcludesWorksOutsideWindow(t *testing.T) {
	spans := []dateSpan{{Start: 1390, End: 1395}}

	bars := densityBins(spans, 1400, 1410)
	byDecade := densityByDecade(bars)
	for _, decade := range []string{"1400", "1410"} {
		if byDecade[decade].Count != 0 {
			t.Errorf("%s count = %d, want 0", decade, byDecade[decade].Count)
		}
	}
}

func TestDensityBinsNormalisesAndZeroFills(t *testing.T) {
	spans := []dateSpan{
		{Start: 1500, End: 0},
		{Start: 1505, End: 0},
		{Start: 1510, End: 0},
	}

	bars := densityBins(spans, 1500, 1519)
	byDecade := densityByDecade(bars)

	if byDecade["1500"].Count != 2 || byDecade["1500"].HeightPct != 100 {
		t.Errorf("1500 density = count %d height %d, want count 2 height 100", byDecade["1500"].Count, byDecade["1500"].HeightPct)
	}
	if byDecade["1510"].Count != 1 || byDecade["1510"].HeightPct != 50 {
		t.Errorf("1510 density = count %d height %d, want count 1 height 50", byDecade["1510"].Count, byDecade["1510"].HeightPct)
	}
}

func TestDensityBinsEmptySpansAllZero(t *testing.T) {
	bars := densityBins(nil, 1400, 1430)
	for _, bar := range bars {
		if bar.HeightPct != 0 || bar.Count != 0 {
			t.Errorf("bar %q = count %d height %d, want 0", bar.Decade, bar.Count, bar.HeightPct)
		}
	}
}

func TestPeriodBounds(t *testing.T) {
	periods := []artPeriod{
		{name: "Early", start: 1000, end: 1200},
		{name: "Late", start: 1900, end: 2100},
		{name: "Reversed", start: 1600, end: 1400},
		{name: "Empty", start: 0, end: 0},
	}

	min, max := periodBounds(periods)
	if min != 1000 || max != 2100 {
		t.Errorf("periodBounds = (%d, %d), want (1000, 2100)", min, max)
	}
}

func TestPeriodBoundsEmpty(t *testing.T) {
	if min, max := periodBounds(nil); min != 0 || max != 0 {
		t.Errorf("periodBounds(nil) = (%d, %d), want (0, 0)", min, max)
	}
	if min, max := periodBounds([]artPeriod{{start: 0, end: 0}}); min != 0 || max != 0 {
		t.Errorf("periodBounds(invalid) = (%d, %d), want (0, 0)", min, max)
	}
}

func TestUnionBounds(t *testing.T) {
	cases := []struct {
		name                 string
		artMin, artMax       int
		periodMin, periodMax int
		wantMin, wantMax     int
	}{
		{"period extends earlier", 1500, 1994, 1000, 1979, 1000, 1994},
		{"outlying period extends later", 1000, 1500, 1800, 2500, 1000, 2500},
		{"only periods", 0, 0, 1000, 1200, 1000, 1200},
		{"only artworks", 1500, 1800, 0, 0, 1500, 1800},
		{"both empty", 0, 0, 0, 0, 0, 0},
	}

	for _, tc := range cases {
		min, max := unionBounds(tc.artMin, tc.artMax, tc.periodMin, tc.periodMax)
		if min != tc.wantMin || max != tc.wantMax {
			t.Errorf("unionBounds(%s) = (%d, %d), want (%d, %d)", tc.name, min, max, tc.wantMin, tc.wantMax)
		}
	}
}

func TestBuildBandsClipsPeriods(t *testing.T) {
	periods := []artPeriod{
		{name: "Before", start: 1000, end: 1200, description: "early"},
		{name: "Middle", start: 1400, end: 1600, description: "middle"},
		{name: "After", start: 1800, end: 1900, description: "late"},
	}

	bands := buildBands(periods, 1450, 1550)

	if len(bands) != 1 {
		t.Fatalf("bands length = %d, want 1", len(bands))
	}
	if bands[0].Name != "Middle" {
		t.Errorf("band name = %q, want Middle", bands[0].Name)
	}
	if bands[0].SpanLabel != "1450–1550" {
		t.Errorf("band span = %q, want 1450–1550", bands[0].SpanLabel)
	}
	if bands[0].LeftPct != 0 || bands[0].WidthPct != 100 {
		t.Errorf("band geometry = (%v, %v), want (0, 100)", bands[0].LeftPct, bands[0].WidthPct)
	}
}

func TestBuildBandsExcludesInvalidSpans(t *testing.T) {
	periods := []artPeriod{
		{name: "Reversed", start: 1600, end: 1400},
		{name: "NoRange", start: 0, end: 1500},
	}

	if bands := buildBands(periods, 1400, 1700); len(bands) != 0 {
		t.Errorf("bands length = %d, want 0", len(bands))
	}
}

func TestBuildMarksAndWorksShareRows(t *testing.T) {
	rows := []artworkRow{
		{ID: "a1", Title: "Alpha", DateStart: 1500, DateEnd: 0, ArtistID: "ar1", ArtistName: "An Artist"},
		{ID: "b2", Title: "Beta", DateStart: 1600, DateEnd: 1610, Qualifier: "after", ArtistID: "ar2", ArtistName: "Another"},
	}

	marks := buildMarks(rows, 1400, 1700)
	if len(marks) != 2 {
		t.Fatalf("marks length = %d, want 2", len(marks))
	}
	if !almostEqual(marks[0].LeftPct, 100.0/3.0) {
		t.Errorf("marks[0].LeftPct = %v, want %v", marks[0].LeftPct, 100.0/3.0)
	}
	if marks[1].Label != "Beta — after 1600–1610" {
		t.Errorf("marks[1].Label = %q", marks[1].Label)
	}
	if !strings.Contains(marks[0].Href, "/artists/") {
		t.Errorf("marks[0].Href = %q, want artist record path", marks[0].Href)
	}

	works := buildWorks(rows, 1400, 1700)
	if len(works) != 2 {
		t.Fatalf("works length = %d, want 2", len(works))
	}
	if works[1].Metadata != "after 1600–1610" {
		t.Errorf("works[1].Metadata = %q, want after 1600–1610", works[1].Metadata)
	}
	if works[0].Artist != "An Artist" {
		t.Errorf("works[0].Artist = %q", works[0].Artist)
	}
}

func TestBuildWorksProjectsPageRows(t *testing.T) {
	rows := []artworkRow{
		{ID: "a1", Title: "Alpha", DateStart: 1500, ArtistName: "An Artist"},
		{ID: "b2", Title: "Beta", DateStart: 1600, ArtistName: "Another"},
	}

	works := buildWorks(rows, 1400, 1700)
	if len(works) != 2 {
		t.Errorf("works length = %d, want 2 (one page)", len(works))
	}
}

func TestWorksRangeLabel(t *testing.T) {
	cases := []struct {
		total int
		page  int
		want  string
	}{
		{0, 1, "0 WORKS"},
		{3, 1, "3 WORKS"},
		{worksPageSize, 1, "8 WORKS"},
		{20, 1, "WORKS 1–8 OF 20"},
		{20, 2, "WORKS 9–16 OF 20"},
		{20, 3, "WORKS 17–20 OF 20"},
	}
	for _, tc := range cases {
		if got := worksRangeLabel(tc.total, tc.page, worksPageSize); got != tc.want {
			t.Errorf("worksRangeLabel(%d, %d) = %q, want %q", tc.total, tc.page, got, tc.want)
		}
	}
}

func TestMarkLabelWithoutDate(t *testing.T) {
	if got := markLabel(artworkRow{Title: "Untitled"}); got != "Untitled" {
		t.Errorf("markLabel(no date) = %q, want Untitled", got)
	}
}

func TestMarksNote(t *testing.T) {
	if got := marksNote(10); got != "" {
		t.Errorf("marksNote(10) = %q, want empty", got)
	}
	if got := marksNote(markCap); got != "" {
		t.Errorf("marksNote(markCap) = %q, want empty (exactly at cap)", got)
	}
	want := "MARKS SHOW THE FIRST 48 OF 49 WORKS, ORDERED BY DATE"
	if got := marksNote(markCap + 1); got != want {
		t.Errorf("marksNote(markCap+1) = %q, want %q", got, want)
	}
}

func TestBuildPresetsIncludesFullAndPeriods(t *testing.T) {
	periods := []artPeriod{
		{name: "Early", start: 1000, end: 1200},
		{name: "Late", start: 1500, end: 1700},
		{name: "Bad", start: 0, end: 100},
	}

	presets := buildPresets(periods, window{from: 1500, to: 1700}, 1000, 1700)

	if len(presets) != 3 {
		t.Fatalf("presets length = %d, want 3 (full + 2 valid periods)", len(presets))
	}
	if presets[0].Label != "FULL RANGE" || presets[0].Href != "/timeline" {
		t.Errorf("presets[0] = %+v, want FULL RANGE -> /timeline", presets[0])
	}
	if presets[1].Label != "Early" || presets[1].Href != "/timeline?to=1200" {
		t.Errorf("presets[1] = %+v", presets[1])
	}
	if !presets[2].Active {
		t.Errorf("presets[2] should be active for matching window")
	}
	for _, preset := range presets {
		if preset.HxTarget != "" {
			t.Errorf("preset %q must not lean on the shared NavChips HxTarget field", preset.Label)
		}
	}
}

func TestPeriodPathOmitsDefault(t *testing.T) {
	if got := periodPath(1000, 1700, 1000, 1700); got != "/timeline" {
		t.Errorf("periodPath(full) = %q, want /timeline", got)
	}
	if got := periodPath(1500, 1700, 1000, 1700); got != "/timeline?from=1500" {
		t.Errorf("periodPath(partial) = %q", got)
	}
}

func TestArtworkImageURLFallsBackWhenMissing(t *testing.T) {
	if got := artworkImageURL(artworkRow{Image: ""}); got != "" {
		t.Errorf("artworkImageURL(empty) = %q, want empty", got)
	}
	if got := artworkImageURL(artworkRow{Image: "x.jpg", ImageWidth: 200, ID: "a1"}); !strings.HasPrefix(got, "/api/files/artworks/a1/x.jpg") {
		t.Errorf("artworkImageURL(original) = %q", got)
	}
}
