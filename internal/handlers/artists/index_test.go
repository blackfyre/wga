package artists

import (
	neturl "net/url"
	"testing"
)

func TestParseIndexQueryDefaults(t *testing.T) {
	query := parseIndexQuery(neturl.Values{}, 1500, 1900)

	if query.query != "" || query.letter != "" || query.school != "" || query.period != "" {
		t.Errorf("unexpected populated filters: %#v", query)
	}
	if query.view != viewGrid {
		t.Errorf("view = %q, want %q", query.view, viewGrid)
	}
	if query.sort != sortAZ {
		t.Errorf("sort = %q, want %q", query.sort, sortAZ)
	}
	if query.page != 1 {
		t.Errorf("page = %d, want 1", query.page)
	}
	if query.bornFrom != 0 || query.bornTo != 0 {
		t.Errorf("born range should be unset, got (%d, %d)", query.bornFrom, query.bornTo)
	}
}

func TestParseIndexQueryAllowListsValues(t *testing.T) {
	values := neturl.Values{
		"q":        {"  Van Gogh  "},
		"letter":   {"s"},
		"school":   {"dutch"},
		"period":   {"periodid123"},
		"view":     {"list"},
		"sort":     {"birth"},
		"page":     {"3"},
		"born_from": {"1600"},
		"born_to":   {"1700"},
	}

	query := parseIndexQuery(values, 1500, 1900)

	if query.query != "Van Gogh" {
		t.Errorf("query = %q, want trimmed %q", query.query, "Van Gogh")
	}
	if query.letter != "S" {
		t.Errorf("letter = %q, want uppercased S", query.letter)
	}
	if query.view != viewList {
		t.Errorf("view = %q, want list", query.view)
	}
	if query.sort != sortBirth {
		t.Errorf("sort = %q, want birth", query.sort)
	}
	if query.page != 3 {
		t.Errorf("page = %d, want 3", query.page)
	}
	if query.bornFrom != 1600 || query.bornTo != 1700 {
		t.Errorf("born range = (%d, %d), want (1600, 1700)", query.bornFrom, query.bornTo)
	}
}

func TestParseIndexQueryRejectsInvalidValues(t *testing.T) {
	values := neturl.Values{
		"letter":    {"AB"},
		"view":      {"carousel"},
		"sort":      {"descending"},
		"page":      {"not-a-number"},
		"born_from": {"not-a-number"},
		"born_to":   {"very-large"},
	}

	query := parseIndexQuery(values, 1500, 1900)

	if query.letter != "" {
		t.Errorf("letter = %q, want empty for invalid input", query.letter)
	}
	if query.view != viewGrid {
		t.Errorf("view = %q, want grid for invalid input", query.view)
	}
	if query.sort != sortAZ {
		t.Errorf("sort = %q, want az for invalid input", query.sort)
	}
	if query.page != 1 {
		t.Errorf("page = %d, want 1 for invalid input", query.page)
	}
	if query.bornFrom != 0 || query.bornTo != 0 {
		t.Errorf("born range = (%d, %d), want unset for invalid input", query.bornFrom, query.bornTo)
	}
}

func TestParseIndexQueryClampsBornRange(t *testing.T) {
	values := neturl.Values{
		"born_from": {"1000"},
		"born_to":   {"9999"},
	}

	query := parseIndexQuery(values, 1500, 1900)

	if query.bornFrom != 1500 {
		t.Errorf("born_from = %d, want clamped 1500", query.bornFrom)
	}
	if query.bornTo != 1900 {
		t.Errorf("born_to = %d, want clamped 1900", query.bornTo)
	}
}

func TestParseIndexQueryReordersReversedBornRange(t *testing.T) {
	values := neturl.Values{
		"born_from": {"1800"},
		"born_to":   {"1600"},
	}

	query := parseIndexQuery(values, 1500, 1900)

	if query.bornFrom != 1600 || query.bornTo != 1800 {
		t.Errorf("born range = (%d, %d), want reordered (1600, 1800)", query.bornFrom, query.bornTo)
	}
}

func TestClampBornYearHandlesMissingAndInvalid(t *testing.T) {
	for _, input := range []string{"", "abc", "  ", "12.5"} {
		if value, ok := clampBornYear(input, 1500, 1900); value != 0 || ok {
			t.Errorf("clampBornYear(%q) = (%d, %t), want (0, false)", input, value, ok)
		}
	}
}

func TestClampBornYearDiscardsWithoutKnownBounds(t *testing.T) {
	// When the published set has no known birth years there is no range to
	// clamp against, so a supplied bound must be treated as unset rather than
	// leaking into the repository filter and hiding every artist.
	for _, bounds := range [][2]int{{0, 0}, {1500, 0}, {0, 1900}} {
		if value, ok := clampBornYear("1600", bounds[0], bounds[1]); value != 0 || ok {
			t.Errorf("clampBornYear(\"1600\", %d, %d) = (%d, %t), want (0, false)", bounds[0], bounds[1], value, ok)
		}
	}
}

func TestParsePage(t *testing.T) {
	for input, want := range map[string]int{"": 1, "0": 1, "-3": 1, "junk": 1, "1": 1, "42": 42} {
		if got := parsePage(input); got != want {
			t.Errorf("parsePage(%q) = %d, want %d", input, got, want)
		}
	}
}

func TestIndexQueryPathOmitsDefaults(t *testing.T) {
	query := indexQuery{view: viewGrid, sort: sortAZ, page: 1, bornMin: 1500, bornMax: 1900}

	if got := query.path(); got != "/artists" {
		t.Errorf("path() = %q, want /artists", got)
	}
}

func TestIndexQueryPathUsesCanonicalOrder(t *testing.T) {
	query := indexQuery{
		query:    "van",
		letter:   "V",
		school:   "dutch",
		period:   "periodbaroque",
		bornFrom: 1600,
		bornTo:   1700,
		view:     viewList,
		sort:     sortBirth,
		page:     2,
	}

	want := "/artists?q=van&letter=V&school=dutch&period=periodbaroque&born_from=1600&born_to=1700&view=list&sort=birth&page=2"
	if got := query.path(); got != want {
		t.Errorf("path() = %q, want %q", got, want)
	}
}

func TestIndexQueryPathOmitsDefaultViewSortPage(t *testing.T) {
	query := indexQuery{
		query:  "van",
		letter: "V",
		view:   viewGrid,
		sort:   sortAZ,
		page:   1,
	}

	want := "/artists?q=van&letter=V"
	if got := query.path(); got != want {
		t.Errorf("path() = %q, want %q", got, want)
	}
}

func TestIndexQueryOverridesResetPage(t *testing.T) {
	query := indexQuery{letter: "V", sort: sortAZ, view: viewGrid, page: 4}

	if got := query.withLetter("S").path(); got != "/artists?letter=S" {
		t.Errorf("withLetter path = %q, want /artists?letter=S (page reset)", got)
	}
	if got := query.withView(viewList).path(); got != "/artists?letter=V&view=list" {
		t.Errorf("withView path = %q, want page reset and view=list", got)
	}
	if got := query.withPage(2).path(); got != "/artists?letter=V&page=2" {
		t.Errorf("withPage path = %q, want page=2 preserved", got)
	}
}

func TestSortLabelAndNextSort(t *testing.T) {
	if got := sortLabel(sortAZ); got != "A–Z" {
		t.Errorf("sortLabel(az) = %q", got)
	}
	if got := sortLabel(sortZA); got != "Z–A" {
		t.Errorf("sortLabel(za) = %q", got)
	}
	if got := sortLabel(sortBirth); got != "BIRTH YEAR" {
		t.Errorf("sortLabel(birth) = %q", got)
	}

	cycle := []string{sortAZ, sortZA, sortBirth, sortAZ}
	for i := 0; i < len(cycle)-1; i++ {
		if got := nextSort(cycle[i]); got != cycle[i+1] {
			t.Errorf("nextSort(%q) = %q, want %q", cycle[i], got, cycle[i+1])
		}
	}
}

func TestPeriodForBirth(t *testing.T) {
	periods := []artPeriod{
		{name: "Early", start: 100, end: 500},
		{name: "Middle", start: 500, end: 1000},
		{name: "Overlap A", start: 900, end: 1100},
		{name: "Overlap B", start: 950, end: 1200},
	}

	if got := periodForBirth(periods, 300); got != "Early" {
		t.Errorf("periodForBirth(300) = %q, want Early", got)
	}
	if got := periodForBirth(periods, 750); got != "Middle" {
		t.Errorf("periodForBirth(750) = %q, want Middle", got)
	}
	if got := periodForBirth(periods, 1000); got != "" {
		t.Errorf("periodForBirth(1000) = %q, want empty (ambiguous)", got)
	}
	if got := periodForBirth(periods, 2000); got != "" {
		t.Errorf("periodForBirth(2000) = %q, want empty (unmatched)", got)
	}
	if got := periodForBirth(periods, 0); got != "" {
		t.Errorf("periodForBirth(0) = %q, want empty (unknown)", got)
	}
}

func TestFormatArtistDates(t *testing.T) {
	if got := formatArtistDates(1600, 1660); got != "1600–1660" {
		t.Errorf("both = %q", got)
	}
	if got := formatArtistDates(1600, 0); got != "b. 1600" {
		t.Errorf("birth only = %q", got)
	}
	if got := formatArtistDates(0, 1660); got != "d. 1660" {
		t.Errorf("death only = %q", got)
	}
	if got := formatArtistDates(0, 0); got != "" {
		t.Errorf("neither = %q", got)
	}
}

func TestResolveSchoolNames(t *testing.T) {
	byID := map[string]string{"a": "Dutch", "b": "Italian"}

	if got := resolveSchoolNames([]string{"a", "b", "missing"}, byID); got != "Dutch, Italian" {
		t.Errorf("resolveSchoolNames = %q, want Dutch, Italian", got)
	}
	if got := resolveSchoolNames(nil, byID); got != "" {
		t.Errorf("resolveSchoolNames(nil) = %q, want empty", got)
	}
}

func TestFilterNote(t *testing.T) {
	query := indexQuery{letter: "V", school: "dutch", period: "baroque", bornFrom: 1600, bornTo: 1700}
	if got := filterNote(query, "Dutch", "Baroque"); got != "· V · DUTCH · BAROQUE · BORN 1600–1700" {
		t.Errorf("filterNote = %q", got)
	}

	empty := indexQuery{}
	if got := filterNote(empty, "", ""); got != "" {
		t.Errorf("empty filterNote = %q, want empty", got)
	}
}

func TestBuildIndexLettersOrderAndAvailability(t *testing.T) {
	letters := buildIndexLetters("B", []string{"A", "B", "Z"}, indexQuery{letter: "B"})

	if len(letters) != 26 {
		t.Fatalf("letters length = %d, want 26", len(letters))
	}
	if letters[0].Label != "A" || letters[25].Label != "Z" {
		t.Errorf("letters should be A through Z, got %q..%q", letters[0].Label, letters[25].Label)
	}

	for _, letter := range letters {
		switch letter.Label {
		case "A", "B", "Z":
			if !letter.Enabled || letter.Href == "" {
				t.Errorf("letter %q should be enabled with a href", letter.Label)
			}
		default:
			if letter.Enabled || letter.Href != "" {
				t.Errorf("letter %q should be disabled without a href", letter.Label)
			}
		}
	}

	if letters[1].Label != "B" || !letters[1].Selected {
		t.Errorf("letter B should be selected")
	}
	if letters[0].Label != "A" || letters[0].Selected {
		t.Errorf("letter A should not be selected")
	}
}
