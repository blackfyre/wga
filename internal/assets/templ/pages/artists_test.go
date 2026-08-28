package pages

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
)

func renderArtistsBlock(t *testing.T, view ArtistsView) string {
	t.Helper()

	var output bytes.Buffer
	if err := ArtistsBlock(view).Render(context.Background(), &output); err != nil {
		t.Fatalf("render artists block: %v", err)
	}

	return output.String()
}

func allLetters() []ArtistLetter {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	letters := make([]ArtistLetter, 0, len(alphabet))
	for _, letter := range alphabet {
		value := string(letter)
		letters = append(letters, ArtistLetter{Label: value, Href: "/artists?letter=" + value, Enabled: true})
	}

	return letters
}

func sampleView() ArtistsView {
	return ArtistsView{
		Letters: allLetters(),
		Schools: []dto.ChipOption{{Label: "ALL", Value: "", Checked: true}, {Label: "Dutch", Value: "dutch"}},
		Periods: []dto.ChipOption{{Label: "ALL", Value: "", Checked: true}, {Label: "Baroque", Value: "periodbaroque"}},
		NameField: dto.Field{
			ID:          "artist-name",
			Name:        "q",
			Label:       "NAME CONTAINS",
			Type:        "search",
			Placeholder: "e.g. van",
		},
		View:     "grid",
		Sort:     "az",
		SortLabel: "A–Z",
		GridUrl:   "/artists",
		ListUrl:   "/artists?view=list",
		SortUrl:   "/artists?sort=za",
		ResetUrl:  "/artists",
		Total:     1,
		Page:      1,
		PageCount: 1,
		Artists: []ArtistRow{{
			URL:       "/artists/rembrandt-van-rijn-123",
			Name:      "Rembrandt van Rijn",
			Dates:     "1606–1669",
			School:    "Dutch",
			Period:    "Baroque",
			Form:      "painter",
			Available: true,
		}},
	}
}

func TestArtistsBlockRendersSingleHeading(t *testing.T) {
	rendered := renderArtistsBlock(t, sampleView())

	if count := strings.Count(rendered, "<h1"); count != 1 {
		t.Errorf("expected exactly one h1, got %d", count)
	}
	if !strings.Contains(rendered, ">Artists</h1>") {
		t.Error("expected an Artists h1")
	}
}

func TestArtistsBlockRendersAllAndAlphabetInOrder(t *testing.T) {
	rendered := renderArtistsBlock(t, sampleView())

	allIndex := strings.Index(rendered, ">ALL</a>")
	if allIndex < 0 {
		t.Fatal("expected ALL link")
	}

	previous := allIndex
	for _, letter := range allLetters() {
		index := strings.Index(rendered, ">"+letter.Label+"</a>")
		if index < 0 {
			t.Fatalf("expected letter %q link", letter.Label)
		}
		if index <= previous {
			t.Errorf("letter %q out of order (index %d after %d)", letter.Label, index, previous)
		}
		previous = index
	}
}

func TestArtistsBlockAlphabetNavWrapsAndDoesNotClip(t *testing.T) {
	rendered := renderArtistsBlock(t, sampleView())

	// The alphabet nav must stay a wrapping flex container so every letter
	// reflows onto further rows at enlarged text. Masking it with an overflow
	// clip would hide the letters that do not fit one row instead of wrapping.
	const label = `aria-label="Filter artists by letter"`
	labelIndex := strings.Index(rendered, label)
	if labelIndex < 0 {
		t.Fatalf("expected an alphabet nav labelled %q", label)
	}

	openIndex := strings.LastIndex(rendered[:labelIndex], "<nav")
	if openIndex < 0 {
		t.Fatal("expected an alphabet <nav> opening tag before the label")
	}

	openTag := rendered[openIndex : labelIndex+strings.Index(rendered[labelIndex:], ">")+1]

	if !strings.Contains(openTag, "flex-wrap") {
		t.Errorf("alphabet nav must wrap at enlarged text; opening tag %q", openTag)
	}
	for _, mask := range []string{"overflow-hidden", "overflow-x-hidden", "overflow-x-auto", "overflow-x-clip"} {
		if strings.Contains(openTag, mask) {
			t.Errorf("alphabet nav must wrap rather than clip: %q appears in opening tag %q", mask, openTag)
		}
	}
}

func TestArtistsBlockDisablesEmptyLettersAndMarksActive(t *testing.T) {
	view := sampleView()
	view.Letters = []ArtistLetter{
		{Label: "A", Href: "/artists?letter=A", Enabled: true, Selected: true},
		{Label: "B", Enabled: false},
	}
	rendered := renderArtistsBlock(t, view)

	if !strings.Contains(rendered, `aria-current="page"`) {
		t.Error("expected the active letter to carry aria-current")
	}
	if !strings.Contains(rendered, `aria-disabled="true"`) || !strings.Contains(rendered, ">B</span>") {
		t.Error("expected the empty letter B to render disabled as a span")
	}
	if strings.Contains(rendered, `href="/artists?letter=B"`) {
		t.Error("empty letter B should not render a link")
	}
}

func TestArtistsBlockRendersLabelledRangeControl(t *testing.T) {
	view := sampleView()
	view.HasBirthBounds = true
	view.BornRange = dto.RangeField{
		Label:     "BORN BETWEEN",
		FromID:    "born-from",
		FromName:  "born_from",
		FromValue: 1600,
		ToID:      "born-to",
		ToName:    "born_to",
		ToValue:   1700,
		Min:       1500,
		Max:       1900,
		Step:      10,
	}
	rendered := renderArtistsBlock(t, view)

	if !strings.Contains(rendered, "BORN BETWEEN") {
		t.Error("expected BORN BETWEEN legend")
	}
	if !strings.Contains(rendered, `name="born_from"`) || !strings.Contains(rendered, `name="born_to"`) {
		t.Error("expected born_from and born_to range inputs")
	}
}

func TestArtistsBlockRendersGridAndListState(t *testing.T) {
	view := sampleView()
	view.View = "list"
	rendered := renderArtistsBlock(t, view)

	if !strings.Contains(rendered, `aria-current="page"`) {
		t.Error("expected the active view toggle to carry aria-current")
	}
	if !strings.Contains(rendered, `href="/artists"`) || !strings.Contains(rendered, `href="/artists?view=list"`) {
		t.Error("expected grid and list hrefs")
	}
}

func TestArtistsBlockListRendersTableWithHeaders(t *testing.T) {
	view := sampleView()
	view.View = "list"
	rendered := renderArtistsBlock(t, view)

	if !strings.Contains(rendered, "Artists in the collection") {
		t.Error("expected table caption")
	}
	for _, header := range []string{">NAME</th>", ">DATES</th>", ">SCHOOL</th>", ">PERIOD</th>", ">FORM</th>"} {
		if !strings.Contains(rendered, header) {
			t.Errorf("expected %s header", header)
		}
	}
	if !strings.Contains(rendered, ">painter</td>") {
		t.Error("expected FORM column to render the profession")
	}
}

func TestArtistsBlockRendersEmptyState(t *testing.T) {
	view := sampleView()
	view.Artists = nil
	view.Total = 0
	rendered := renderArtistsBlock(t, view)

	if !strings.Contains(rendered, "No artists match these filters.") {
		t.Error("expected empty state title")
	}
	if !strings.Contains(rendered, "RESET FILTERS") {
		t.Error("expected reset link")
	}
}

func TestArtistsBlockRendersUnavailableAsNonLink(t *testing.T) {
	view := sampleView()
	view.Artists = []ArtistRow{{
		URL:       "/artists/unknown-artist-123",
		Name:      "Unknown Artist",
		Dates:     "1600–1650",
		School:    "Dutch",
		Period:    "",
		Form:      "painter",
		Available: false,
	}}
	rendered := renderArtistsBlock(t, view)

	if !strings.Contains(rendered, "NOT DIGITISED") {
		t.Error("expected NOT DIGITISED marker")
	}
	if strings.Contains(rendered, `href="/artists/unknown-artist-123"`) {
		t.Error("unavailable artist should not render an artist-record link")
	}
	if !strings.Contains(rendered, "opacity-50") {
		t.Error("expected unavailable row/card to be subdued")
	}
}

func TestArtistsBlockRendersUnavailableListRowWithoutLink(t *testing.T) {
	view := sampleView()
	view.View = "list"
	view.Artists = []ArtistRow{{URL: "/artists/unknown-artist-123", Name: "Unknown Artist", Available: false}}
	rendered := renderArtistsBlock(t, view)

	if strings.Contains(rendered, `href="/artists/unknown-artist-123"`) {
		t.Error("unavailable list row should not link to the artist record")
	}
	if !strings.Contains(rendered, "NOT DIGITISED") {
		t.Error("expected NOT DIGITISED marker in list row")
	}
}

func TestArtistsBlockGridNameBreaksLongFilingNames(t *testing.T) {
	view := sampleView()
	view.Artists = []ArtistRow{
		{URL: "/artists/available-a-1", Name: "ABILDGAARD, Nicolai", Available: true},
		{URL: "/artists/unavailable-b-1", Name: "ADRIAENSSEN, Alexander", Available: false},
	}
	rendered := renderArtistsBlock(t, view)

	// The encyclopaedic filing form ("SURNAME, Given Names") produces long
	// unbreakable surname tokens that otherwise overflow the two-column grid at
	// enlarged text. Both the available and unavailable name spans must carry
	// break-words so the filing name wraps without altering its content.
	for _, name := range []string{"ABILDGAARD, Nicolai", "ADRIAENSSEN, Alexander"} {
		if !strings.Contains(rendered, `text-(length:--t-17) font-semibold break-words">`+name+"</span>") {
			t.Errorf("expected grid name %q to carry break-words wrapping", name)
		}
	}
}

func TestArtistNameEscapesScriptLikeContent(t *testing.T) {
	var output bytes.Buffer
	component := ArtistName(`<script>alert("x")</script>`, "<script>")
	if err := component.Render(context.Background(), &output); err != nil {
		t.Fatalf("render artist name: %v", err)
	}
	rendered := output.String()

	if strings.Contains(rendered, "<script>") {
		t.Errorf("script-like name was not escaped: %q", rendered)
	}
	if !strings.Contains(rendered, "&lt;script&gt;") {
		t.Errorf("expected escaped name content, got %q", rendered)
	}
	if !strings.Contains(rendered, "<mark>&lt;script&gt;</mark>") {
		t.Errorf("expected escaped match wrapped in mark, got %q", rendered)
	}
}

func TestSplitNameHighlightUnicodeSafe(t *testing.T) {
	// "İ" (U+0130) lowercases to a shorter byte sequence than the source rune,
	// so a byte index taken from the lowercased string must never be used to
	// slice the original. The old implementation sliced the original name by
	// such an index and produced invalid UTF-8 here.
	before, match, after := splitNameHighlight("İzmir", "zmir")
	if before != "İ" || match != "zmir" || after != "" {
		t.Errorf("splitNameHighlight = (%q, %q, %q), want (%q, %q, %q)", before, match, after, "İ", "zmir", "")
	}
	if !utf8.ValidString(before + match + after) {
		t.Errorf("splitNameHighlight output is not valid UTF-8: %q", before+match+after)
	}
}

func TestSplitNameHighlightCaseInsensitiveAndEscaped(t *testing.T) {
	before, match, after := splitNameHighlight("Rembrandt van Rijn", "VAN")
	if before != "Rembrandt " || match != "van" || after != " Rijn" {
		t.Errorf("splitNameHighlight = (%q, %q, %q), want case-insensitive (%q, %q, %q)", before, match, after, "Rembrandt ", "van", " Rijn")
	}
}

func TestArtistsBlockAllLinkClearsOnlyLetter(t *testing.T) {
	view := sampleView()
	view.SelectedLetter = "S"
	view.AllUrl = "/artists?school=dutch"
	rendered := renderArtistsBlock(t, view)

	if !strings.Contains(rendered, `href="/artists?school=dutch"`) {
		t.Error("expected the ALL link to preserve unrelated filters (clear only the letter)")
	}
}

func TestArtistsBlockPreservesOrdinaryNoJsControls(t *testing.T) {
	view := sampleView()
	view.SelectedLetter = "V"
	view.View = "list"
	view.Sort = "za"
	rendered := renderArtistsBlock(t, view)

	if !strings.Contains(rendered, `action="/artists"`) || !strings.Contains(rendered, `method="GET"`) {
		t.Error("expected ordinary GET form")
	}
	if !strings.Contains(rendered, `<noscript>`) || !strings.Contains(rendered, "APPLY FILTERS") {
		t.Error("expected no-JavaScript apply button")
	}
	if !strings.Contains(rendered, `name="letter" value="V"`) {
		t.Error("expected hidden letter input preserving state")
	}
	if !strings.Contains(rendered, `name="view" value="list"`) {
		t.Error("expected hidden view input preserving state")
	}
	if !strings.Contains(rendered, `name="sort" value="za"`) {
		t.Error("expected hidden sort input preserving state")
	}
	if !strings.Contains(rendered, `hx-select="#artists"`) {
		t.Error("expected HTMX select of #artists")
	}
}

func TestArtistsBlockPagination(t *testing.T) {
	view := sampleView()
	view.Total = 60
	view.Page = 2
	view.PageCount = 2
	view.PrevUrl = "/artists?page=1"
	view.NextUrl = "/artists?page=3"
	rendered := renderArtistsBlock(t, view)

	if !strings.Contains(rendered, "PAGE 2 OF 2") {
		t.Error("expected page readout")
	}
	if !strings.Contains(rendered, `href="/artists?page=1"`) || !strings.Contains(rendered, "← PREV") {
		t.Error("expected previous link")
	}
	if strings.Contains(rendered, `href="/artists?page=3"`) || !strings.Contains(rendered, "NEXT") {
		t.Error("expected disabled next on the last page")
	}
}

func TestArtistsBlockListKeyboardContainerAndIndexing(t *testing.T) {
	view := sampleView()
	view.View = "list"
	view.Artists = []ArtistRow{
		{URL: "/artists/available-a-1", Name: "Available A", Available: true},
		{URL: "/artists/unavailable-b-1", Name: "Unavailable B", Available: false},
		{URL: "/artists/available-c-1", Name: "Available C", Available: true},
	}
	rendered := renderArtistsBlock(t, view)

	if !strings.Contains(rendered, `data-kbd-list`) || !strings.Contains(rendered, `data-kbd-cols="1"`) {
		t.Error("expected the list table to be a single-column kbd-list container")
	}
	if !strings.Contains(rendered, `data-kbd-idx="0"`) || !strings.Contains(rendered, `data-kbd-idx="1"`) {
		t.Error("expected sequential data-kbd-idx for navigable rows")
	}
	if strings.Contains(rendered, `data-kbd-idx="2"`) {
		t.Error("the unavailable row must not advance the navigable index")
	}
	if !strings.Contains(rendered, `data-kbd-href="/artists/available-a-1"`) || !strings.Contains(rendered, `data-kbd-href="/artists/available-c-1"`) {
		t.Error("available rows must carry canonical kbd hrefs")
	}
	if strings.Contains(rendered, `data-kbd-href="/artists/unavailable-b-1"`) {
		t.Error("the unavailable row must not carry a kbd href")
	}
}

func TestArtistsBlockGridKeyboardHooksExcludeUnavailable(t *testing.T) {
	view := sampleView()
	view.Artists = []ArtistRow{
		{URL: "/artists/available-a-1", Name: "Available A", Available: true},
		{URL: "/artists/unavailable-b-1", Name: "Unavailable B", Available: false},
	}
	rendered := renderArtistsBlock(t, view)

	if !strings.Contains(rendered, `data-kbd-list`) || !strings.Contains(rendered, `data-kbd-cols="2"`) {
		t.Error("expected the grid to be a kbd-list container")
	}
	if !strings.Contains(rendered, `data-kbd-href="/artists/available-a-1"`) || !strings.Contains(rendered, `href="/artists/available-a-1"`) {
		t.Error("the available card must render its canonical record link and kbd href")
	}
	if strings.Contains(rendered, `data-kbd-href="/artists/unavailable-b-1"`) || strings.Contains(rendered, `href="/artists/unavailable-b-1"`) {
		t.Error("the unavailable card must not render a record link or kbd href")
	}
}
