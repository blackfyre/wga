package dual

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	neturl "net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
	"github.com/blackfyre/wga/internal/assets/templ/pages"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func mustParseQuery(t *testing.T, raw string) neturl.Values {
	t.Helper()
	parsed, err := neturl.ParseQuery(raw)
	if err != nil {
		t.Fatalf("parse query %q: %v", raw, err)
	}
	return parsed
}

func assertDualPath(t *testing.T, got string, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("path = %q, want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// Parser
// ---------------------------------------------------------------------------

func TestParseDualStateDefaults(t *testing.T) {
	state := parseDualState(neturl.Values{})

	if state.wide {
		t.Fatal("wide should default to false")
	}
	if state.left.path != "" || state.right.path != "" {
		t.Fatal("both panes should default to the index")
	}
	if state.left.renderTo != "right" || state.right.renderTo != "left" {
		t.Fatalf("default link routing should target the other pane, got %q/%q", state.left.renderTo, state.right.renderTo)
	}
	if state.left.size != sizeMedium || state.right.size != sizeMedium {
		t.Fatal("both panes should default to the medium study image")
	}
	if state.left.index.view != viewList || state.left.index.sort != sortAZ {
		t.Fatalf("default index state = %+v", state.left.index)
	}
}

func TestParseDualStateReadsFullState(t *testing.T) {
	state := parseDualState(mustParseQuery(t,
		"left=/artists/left-123&right=/artists/right-456/artwork-789&"+
			"left_render_to=left&right_render_to=left&"+
			"l_letter=B&l_school=dutch&l_q=van&l_born_from=1600&l_born_to=1700&l_view=grid&l_sort=birth&l_size=large&"+
			"r_size=small&wide=1"))

	if !state.wide {
		t.Fatal("wide should be true")
	}
	if state.left.path != "/artists/left-123" {
		t.Fatalf("left path = %q", state.left.path)
	}
	if state.right.path != "/artists/right-456/artwork-789" {
		t.Fatalf("right path = %q", state.right.path)
	}
	if state.left.renderTo != "left" || state.right.renderTo != "left" {
		t.Fatalf("renderTo = %q/%q", state.left.renderTo, state.right.renderTo)
	}
	if state.left.size != sizeLarge || state.right.size != sizeSmall {
		t.Fatalf("sizes = %q/%q", state.left.size, state.right.size)
	}
	idx := state.left.index
	if idx.letter != "B" || idx.school != "dutch" || idx.query != "van" || idx.bornFrom != 1600 || idx.bornTo != 1700 || idx.view != viewGrid || idx.sort != sortBirth {
		t.Fatalf("left index = %+v", idx)
	}
}

func TestParseDualPath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "", want: ""},
		{input: "  ", want: ""},
		{input: "default", want: ""},
		{input: "/artists/example-123", want: "/artists/example-123"},
		{input: "artists/example-123", want: "/artists/example-123"},
		{input: "/artists/example-123/artwork-777", want: "/artists/example-123/artwork-777"},
		{input: "/artworks/artwork-777", want: "/artworks/artwork-777"},
		{input: "/pages/privacy-policy", want: ""},
		{input: strings.Repeat("a", 600), want: ""},
	}

	for _, test := range tests {
		if got := parseDualPath(test.input); got != test.want {
			t.Errorf("parseDualPath(%q) = %q, want %q", test.input, got, test.want)
		}
	}
}

func TestParseDualIndexBounds(t *testing.T) {
	tests := []struct {
		raw    string
		letter string
		view   string
		sort   string
	}{
		{raw: "l_letter=b", letter: "B"},
		{raw: "l_letter=BB", letter: ""},
		{raw: "l_letter=1", letter: ""},
		{raw: "l_view=list", view: viewList},
		{raw: "l_view=grid", view: viewGrid},
		{raw: "l_view=table", view: viewList},
		{raw: "l_sort=za", sort: sortZA},
		{raw: "l_sort=birth", sort: sortBirth},
		{raw: "l_sort=weird", sort: sortAZ},
	}

	for _, test := range tests {
		idx := parseDualIndex(mustParseQuery(t, test.raw), "l")
		if test.letter != "" && idx.letter != test.letter {
			t.Errorf("%q letter = %q, want %q", test.raw, idx.letter, test.letter)
		}
		if test.view != "" && idx.view != test.view {
			t.Errorf("%q view = %q, want %q", test.raw, idx.view, test.view)
		}
		if test.sort != "" && idx.sort != test.sort {
			t.Errorf("%q sort = %q, want %q", test.raw, idx.sort, test.sort)
		}
	}
}

func TestParseDualBornYearReordersInvertedRange(t *testing.T) {
	idx := parseDualIndex(mustParseQuery(t, "l_born_from=1700&l_born_to=1600"), "l")
	if idx.bornFrom != 1600 || idx.bornTo != 1700 {
		t.Fatalf("inverted range not reordered: %+v", idx)
	}
}

func TestParseDualSize(t *testing.T) {
	if got := parseDualSize("small"); got != sizeSmall {
		t.Fatalf("small = %q", got)
	}
	if got := parseDualSize("large"); got != sizeLarge {
		t.Fatalf("large = %q", got)
	}
	if got := parseDualSize("medium"); got != sizeMedium {
		t.Fatalf("medium = %q", got)
	}
	if got := parseDualSize("huge"); got != sizeMedium {
		t.Fatalf("invalid size should fall back to medium, got %q", got)
	}
	if got := parseDualSize(""); got != sizeMedium {
		t.Fatalf("empty size should fall back to medium, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// Serialization and canonicalisation
// ---------------------------------------------------------------------------

func TestDualStatePathDefaultIsBare(t *testing.T) {
	assertDualPath(t, parseDualState(neturl.Values{}).path(), "/dual-mode")
}

func TestDualStatePathOmitsDefaults(t *testing.T) {
	state := dualState{
		left: dualPaneState{
			path:     "/artists/aaa-bbb",
			renderTo: "right",
			index:    dualIndexState{view: viewList, sort: sortAZ},
			size:     sizeMedium,
		},
		right: dualPaneState{
			renderTo: "left",
			index:    dualIndexState{view: viewList, sort: sortAZ},
			size:     sizeMedium,
		},
	}

	assertDualPath(t, state.path(), "/dual-mode?left=%2Fartists%2Faaa-bbb")
}

func TestDualStatePathIsOrderedAndCanonical(t *testing.T) {
	state := dualState{
		left: dualPaneState{
			path:     "/artists/aaa-bbb",
			renderTo: "right",
			index:    dualIndexState{view: viewList, sort: sortAZ},
			size:     sizeMedium,
		},
		right: dualPaneState{
			renderTo: "left",
			index: dualIndexState{
				letter: "A",
				view:   viewGrid,
				sort:   sortAZ,
			},
			size: sizeLarge,
		},
	}

	assertDualPath(t, state.path(), "/dual-mode?left=%2Fartists%2Faaa-bbb&r_letter=A&r_view=grid&r_size=large")
}

func TestDualStatePathRoundTrip(t *testing.T) {
	states := []dualState{
		parseDualState(neturl.Values{}),
		parseDualState(mustParseQuery(t, "wide=1")),
		parseDualState(mustParseQuery(t,
			"left=/artists/left-123&right=/artists/right-456/artwork-789&"+
				"left_render_to=left&right_render_to=right&"+
				"l_letter=Z&l_school=dutch&l_period=period1&l_q=van gogh&l_born_from=1500&l_born_to=1700&l_view=grid&l_sort=birth&l_size=large&"+
				"r_size=small&wide=1")),
	}

	for i, state := range states {
		roundTripped := parseDualState(mustParseQuery(t, strings.TrimPrefix(state.path(), "/dual-mode?")))
		if !reflect.DeepEqual(roundTripped, state) {
			t.Errorf("state %d round trip mismatch:\n got %+v\nwant %+v", i, roundTripped, state)
		}
	}
}

func TestDualStateIgnoresInvalidInput(t *testing.T) {
	state := parseDualState(mustParseQuery(t,
		"left=/pages/privacy-policy&right=default&"+
			"left_render_to=centre&right_render_to=up&"+
			"l_size=huge&r_view=weird&l_born_from=abc&wide=yes"))

	assertDualPath(t, state.path(), "/dual-mode")
}

// ---------------------------------------------------------------------------
// Mutations and pane independence
// ---------------------------------------------------------------------------

func TestDualStateMutationsAreIndependent(t *testing.T) {
	base := parseDualState(neturl.Values{})

	leftChanged := base.withPanePath("left", "/artists/left-123").withPaneSize("left", sizeLarge)
	if leftChanged.left.path != "/artists/left-123" || leftChanged.left.size != sizeLarge {
		t.Fatalf("left mutation not applied: %+v", leftChanged.left)
	}
	if leftChanged.right.path != "" || leftChanged.right.size != sizeMedium {
		t.Fatal("left mutation leaked into the right pane")
	}

	rightChanged := base.withPaneRenderTo("right", "right").withPaneIndex("right", dualIndexState{letter: "A", view: viewGrid, sort: sortAZ})
	if rightChanged.right.renderTo != "right" || rightChanged.right.index.letter != "A" {
		t.Fatalf("right mutation not applied: %+v", rightChanged.right)
	}
	if rightChanged.left.renderTo != "right" || rightChanged.left.index.letter != "" {
		t.Fatal("right mutation leaked into the left pane")
	}
}

func TestDualStateSwapCarriesFullState(t *testing.T) {
	state := dualState{
		left: dualPaneState{
			path:     "/artists/left-123",
			renderTo: "left",
			index:    dualIndexState{letter: "A", view: viewGrid, sort: sortAZ},
			size:     sizeLarge,
		},
		right: dualPaneState{
			path:     "/artists/right-456",
			renderTo: "right",
			index:    dualIndexState{view: viewList, sort: sortAZ},
			size:     sizeSmall,
		},
		wide: true,
	}

	swapped := state.swapped()

	if !reflect.DeepEqual(swapped.left, state.right) || !reflect.DeepEqual(swapped.right, state.left) {
		t.Fatal("swap did not exchange full pane state")
	}
	if !swapped.wide {
		t.Fatal("swap must preserve the wide override")
	}
}

func TestDualStateResetReturnsBothToIndex(t *testing.T) {
	state := dualState{
		left:  dualPaneState{path: "/artists/left-123", renderTo: "left", index: dualIndexState{letter: "A"}, size: sizeLarge},
		right: dualPaneState{path: "/artists/right-456", renderTo: "right", index: dualIndexState{view: viewGrid}, size: sizeSmall},
		wide:  true,
	}

	reset := state.reset()

	if reset.left.path != "" || reset.right.path != "" {
		t.Fatal("reset should clear both pane paths")
	}
	if reset.left.renderTo != "right" || reset.right.renderTo != "left" {
		t.Fatal("reset should restore default link routing")
	}
	if reset.left.index.letter != "" || reset.right.index.view != viewList {
		t.Fatal("reset should restore default index state")
	}
	if !reset.wide {
		t.Fatal("reset must preserve the wide override")
	}
}

// ---------------------------------------------------------------------------
// Normalization
// ---------------------------------------------------------------------------

func testReference() dualReference {
	return dualReference{
		schoolSlugs: map[string]string{"dutch": "Dutch", "italian": "Italian"},
		schoolByID:  map[string]string{"s1": "Dutch", "s2": "Italian"},
		periodByID:  map[string]dualPeriod{"p1": {id: "p1", name: "Baroque", start: 1600, end: 1750}},
		bornMin:     1300,
		bornMax:     1900,
	}
}

func TestNormalizeDropsUnknownSchoolAndPeriod(t *testing.T) {
	state := dualState{
		left: dualPaneState{
			index: dualIndexState{school: "dutch", period: "p1"},
		},
		right: dualPaneState{
			index: dualIndexState{school: "greek", period: "missing"},
		},
	}

	normalized := state.normalize(testReference())

	if normalized.left.index.school != "dutch" || normalized.left.index.period != "p1" {
		t.Fatalf("known values were dropped: %+v", normalized.left.index)
	}
	if normalized.right.index.school != "" || normalized.right.index.period != "" {
		t.Fatalf("unknown values should be dropped: %+v", normalized.right.index)
	}
}

func TestNormalizeClampsBornYears(t *testing.T) {
	state := dualState{
		left: dualPaneState{
			index: dualIndexState{bornFrom: 900, bornTo: 2100},
		},
	}

	normalized := state.normalize(testReference())

	if normalized.left.index.bornFrom != 1300 || normalized.left.index.bornTo != 1900 {
		t.Fatalf("born years not clamped: %+v", normalized.left.index)
	}
}

func TestNormalizeDropsBornYearsWithoutBounds(t *testing.T) {
	state := dualState{
		left: dualPaneState{
			index: dualIndexState{bornFrom: 1500, bornTo: 1600},
		},
	}

	normalized := state.normalize(dualReference{})

	if normalized.left.index.bornFrom != 0 || normalized.left.index.bornTo != 0 {
		t.Fatal("born years should be discarded when no published bounds exist")
	}
}

// ---------------------------------------------------------------------------
// Hidden fields
// ---------------------------------------------------------------------------

func TestHiddenFieldsExcludeVisibleFilters(t *testing.T) {
	state := dualState{
		left: dualPaneState{
			path:     "/artists/left-123",
			renderTo: "right",
			index:    dualIndexState{school: "dutch", letter: "A", view: viewList, sort: sortAZ},
			size:     sizeMedium,
		},
		right: dualPaneState{
			path:     "/artists/right-456",
			renderTo: "right",
			index:    dualIndexState{period: "p1", view: viewGrid, sort: sortAZ},
			size:     sizeSmall,
		},
		wide: true,
	}

	hidden := state.hiddenFieldsFor("left")
	names := map[string]bool{}
	for _, field := range hidden {
		names[field.Name] = true
	}

	for _, visible := range []string{"l_school", "l_period", "l_q", "l_born_from", "l_born_to"} {
		if names[visible] {
			t.Errorf("visible filter %q should not be a hidden field", visible)
		}
	}
	for _, preserved := range []string{"left", "l_letter", "right", "right_render_to", "r_period", "r_view", "r_size", "wide"} {
		if !names[preserved] {
			t.Errorf("state param %q should be preserved as a hidden field", preserved)
		}
	}
	// Default-valued params are omitted from the canonical serialization and
	// must not be emitted as hidden fields either.
	if names["left_render_to"] {
		t.Error("default-valued left_render_to should be omitted, not preserved")
	}
}

// ---------------------------------------------------------------------------
// Image sizes
// ---------------------------------------------------------------------------

func TestDualArtworkLocationAndDimensions(t *testing.T) {
	location, dimensions := dualArtworkLocationAndDimensions("Painted in 1642 · Rijksmuseum · 363 x 437 cm")
	if location != "Rijksmuseum" || dimensions != "363 x 437 cm" {
		t.Fatalf("location/dimensions = %q/%q", location, dimensions)
	}

	if location, dimensions := dualArtworkLocationAndDimensions("short comment"); location != "" || dimensions != "" {
		t.Fatalf("short comment should yield no facts, got %q/%q", location, dimensions)
	}

	if got := dualTrimDimensions("Oil on canvas, 363 x 437 cm", "363 x 437 cm"); got != "Oil on canvas" {
		t.Fatalf("trim dimensions = %q", got)
	}
}

func TestDualSizeMappings(t *testing.T) {
	if got := dualSizeWidth(sizeSmall); got != 700 {
		t.Fatalf("small width = %d, want 700", got)
	}
	if got := dualSizeWidth(sizeMedium); got != 1100 {
		t.Fatalf("medium width = %d, want 1100", got)
	}
	if got := dualSizeWidth(sizeLarge); got != 1600 {
		t.Fatalf("large width = %d, want 1600", got)
	}

	if got := dualSizeProfile(sizeSmall); got != "700x0" {
		t.Fatalf("small profile = %q, want 700x0", got)
	}
	if got := dualSizeProfile(sizeMedium); got != "1100x0" {
		t.Fatalf("medium profile = %q, want 1100x0", got)
	}
	if got := dualSizeProfile(sizeLarge); got != "1600x0" {
		t.Fatalf("large profile = %q, want 1600x0", got)
	}

	if got := dualSizePlateClass(sizeSmall); got != "h-[300px]" {
		t.Fatalf("small plate class = %q", got)
	}
	if got := dualSizePlateClass(sizeMedium); got != "h-[460px]" {
		t.Fatalf("medium plate class = %q", got)
	}
	if got := dualSizePlateClass(sizeLarge); got != "h-[680px]" {
		t.Fatalf("large plate class = %q", got)
	}
}

// ---------------------------------------------------------------------------
// Template rendering
// ---------------------------------------------------------------------------

func renderDualBlock(t *testing.T, view pages.DualModeView) string {
	t.Helper()
	var buff bytes.Buffer
	if err := pages.DualModeBlock(view).Render(context.Background(), &buff); err != nil {
		t.Fatalf("render dual block: %v", err)
	}
	return buff.String()
}

func TestDualModeBlockRendersNarrowGateByDefault(t *testing.T) {
	view := pages.DualModeView{
		Windows:   [2]pages.DualWindow{{View: "index"}, {View: "index"}},
		WideHref:  "/dual-mode?wide=1",
		ExitHref:  "/artists",
		ForceWide: false,
	}

	markup := renderDualBlock(t, view)

	if !strings.Contains(markup, "wga-dual-narrow") {
		t.Error("narrow notice should render")
	}
	if !strings.Contains(markup, "This mode needs a wide screen") {
		t.Error("narrow notice headline should render")
	}
	if !strings.Contains(markup, "OPEN IT ANYWAY") {
		t.Error("wide override link should render")
	}
	if !strings.Contains(markup, `href="/dual-mode?wide=1"`) {
		t.Error("wide override link should point at ?wide=1")
	}
	if strings.Contains(markup, "data-wide") {
		t.Error("default state must not carry the data-wide override attribute")
	}
	if !strings.Contains(markup, `id="dual-area"`) {
		t.Error("root should carry the dual-area id for HTMX swaps")
	}
}

func TestDualModeBlockRendersWideOverride(t *testing.T) {
	view := pages.DualModeView{
		Windows:   [2]pages.DualWindow{{View: "index"}, {View: "index"}},
		WideHref:  "/dual-mode?wide=1",
		ExitHref:  "/artists",
		ForceWide: true,
	}

	markup := renderDualBlock(t, view)

	if !strings.Contains(markup, `data-wide`) {
		t.Error("wide override should render the data-wide attribute")
	}
	if !strings.Contains(markup, "wga-dual-split") {
		t.Error("split surface should render")
	}
}

func TestDualModeBlockRendersTwoIndependentPanes(t *testing.T) {
	view := pages.DualModeView{
		Windows: [2]pages.DualWindow{
			{Key: "left", Tag: "L", Label: "LEFT", View: "index", SelfSel: "#dual-left", TargetSel: "#dual-right"},
			{Key: "right", Tag: "R", Label: "RIGHT", View: "work", SelfSel: "#dual-right", TargetSel: "#dual-right"},
		},
		ForceWide: true,
	}

	markup := renderDualBlock(t, view)

	if !strings.Contains(markup, `id="dual-left"`) || !strings.Contains(markup, `id="dual-right"`) {
		t.Error("both panes should render with distinct ids")
	}
	if !strings.Contains(markup, "01 — ARTIST INDEX") {
		t.Error("index pane should render the artist index")
	}
}

// TestDualIndexTableUsesAccessibleScrollRegion locks the artist-index list table
// inside an accessible, labelled, keyboard-focusable horizontal scroll region so
// the wide override never clips its columns within a narrow pane.
func TestDualIndexTableUsesAccessibleScrollRegion(t *testing.T) {
	view := pages.DualModeView{
		Windows: [2]pages.DualWindow{
			{
				Key: "left", Tag: "L", Label: "LEFT", View: "index",
				SelfSel: "#dual-left", TargetSel: "#dual-right",
				Index: pages.DualIndexView{
					View: "list",
					Artists: []pages.DualArtistRow{
						{Name: "An Artist", Dates: "1564–1616", School: "English", Period: "Renaissance", Form: "painting"},
					},
				},
			},
			{Key: "right", Tag: "R", Label: "RIGHT", View: "index", SelfSel: "#dual-right", TargetSel: "#dual-right"},
		},
		ForceWide: true,
	}

	markup := renderDualBlock(t, view)

	for _, expected := range []string{
		`class="overflow-x-auto"`,
		`role="region"`,
		`aria-label="Artists in the collection"`,
		`tabindex="0"`,
		`data-dual-horizontal-scroll`,
		`aria-describedby="dual-scroll-hint-left"`,
		`id="dual-scroll-hint-left"`,
		"Scroll horizontally with the left and right arrow keys while this table is focused.",
		`<table class="table table-sm min-w-[640px]">`,
	} {
		if !strings.Contains(markup, expected) {
			t.Errorf("index table scroll region does not contain %q", expected)
		}
	}
}

func TestDualModeBlockSelectionsUseAriaCurrentNotPressed(t *testing.T) {
	view := pages.DualModeView{
		Windows: [2]pages.DualWindow{
			{
				Key: "right", Tag: "R", Label: "RIGHT", View: "work",
				SelfSel: "#dual-right", TargetSel: "#dual-left", OtherLabel: "LEFT WINDOW",
				RoutesToSelf: true,
				Work: pages.DualWorkRecord{
					Title: "A Work",
					Sizes: []pages.DualLink{
						{Label: "700", Href: "/dual-mode?r_size=small"},
						{Label: "1100", Href: "/dual-mode", Selected: true},
						{Label: "1600", Href: "/dual-mode?r_size=large"},
					},
				},
			},
			{Key: "left", Tag: "L", Label: "LEFT", View: "index", SelfSel: "#dual-left", TargetSel: "#dual-left", OtherLabel: "RIGHT WINDOW"},
		},
		ForceWide: true,
	}

	markup := renderDualBlock(t, view)

	if strings.Contains(markup, "aria-pressed") {
		t.Error("routing and image-size links must not use aria-pressed")
	}
	if !strings.Contains(markup, `aria-current="true"`) {
		t.Error("selected links should be marked with aria-current")
	}
}

// ---------------------------------------------------------------------------
// Pane path and routing helpers
// ---------------------------------------------------------------------------

func TestParsePanePath(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    panePathDto
		wantErr bool
	}{
		{name: "default", input: "default", want: panePathDto{Kind: "default", RelPath: "default"}},
		{name: "artist", input: "/artists/example-123", want: panePathDto{Kind: "artist", Id: "123", RelPath: "/artists/example-123"}},
		{name: "artist artwork", input: "/artists/example-123/artwork-777", want: panePathDto{Kind: "artwork", Id: "777", RelPath: "/artists/example-123/artwork-777"}},
		{name: "legacy artwork", input: "/artists/example-123/artworks/artwork-777", want: panePathDto{Kind: "artwork", Id: "777", RelPath: "/artists/example-123/artworks/artwork-777"}},
		{name: "artworks route", input: "/artworks/artwork-777", want: panePathDto{Kind: "artwork", Id: "777", RelPath: "/artworks/artwork-777"}},
		{name: "unsupported", input: "/pages/privacy-policy", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parsePanePath(test.input)
			if test.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q", test.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", test.input, err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("got %+v, want %+v", got, test.want)
			}
		})
	}
}

func TestReverseSide(t *testing.T) {
	if reverseSide("left") != "right" || reverseSide("right") != "left" || reverseSide("centre") != "" {
		t.Fatal("reverseSide behaviour incorrect")
	}
}

func TestResolvePaneTarget(t *testing.T) {
	tests := []struct {
		side      string
		requested string
		want      string
	}{
		{side: "left", requested: "left", want: "left"},
		{side: "left", requested: "right", want: "right"},
		{side: "right", requested: "right", want: "right"},
		{side: "right", requested: "left", want: "left"},
		{side: "left", requested: "", want: "right"},
		{side: "right", requested: "centre", want: "left"},
	}

	for _, test := range tests {
		if got := resolvePaneTarget(test.side, test.requested); got != test.want {
			t.Errorf("resolvePaneTarget(%q, %q) = %q, want %q", test.side, test.requested, got, test.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Lookup
// ---------------------------------------------------------------------------

func TestGetDualLookupResultsRequiresTwoRunes(t *testing.T) {
	content, err := getDualLookupResults(nil, "artwork", " é ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content.Kind != "artwork" || content.Query != "é" || !content.QueryTooShort {
		t.Fatalf("unexpected short lookup result: %#v", content)
	}
}

func TestResolveDualLookupKind(t *testing.T) {
	if resolveDualLookupKind("artwork") != "artwork" {
		t.Fatal("artwork kind not resolved")
	}
	if resolveDualLookupKind("artist") != "artist" {
		t.Fatal("artist kind not resolved")
	}
	if resolveDualLookupKind("weird") != "artist" {
		t.Fatal("unknown kind should default to artist")
	}
}

// ---------------------------------------------------------------------------
// Test app and record-building integration
// ---------------------------------------------------------------------------

func newDualTestApp(t *testing.T) *pocketbase.PocketBase {
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

	saveDualCollection(t, app, "schools",
		&core.TextField{Name: "name", Required: true},
		&core.TextField{Name: "slug"},
	)
	saveDualCollection(t, app, "art_periods",
		&core.TextField{Name: "name", Required: true},
		&core.NumberField{Name: "start"},
		&core.NumberField{Name: "end"},
	)
	saveDualCollection(t, app, "art_types",
		&core.TextField{Name: "name", Required: true},
	)

	artists := core.NewBaseCollection("Artists")
	artists.Id = "artists"
	artists.MarkAsNew()
	artists.Fields.Add(
		&core.TextField{Name: "name", Required: true},
		&core.TextField{Name: "slug"},
		&core.EditorField{Name: "bio"},
		&core.NumberField{Name: "year_of_birth"},
		&core.NumberField{Name: "year_of_death"},
		&core.TextField{Name: "profession"},
		&core.RelationField{Name: "school", CollectionId: "schools", MinSelect: 0, MaxSelect: 10},
		&core.TextField{Name: "portrait"},
		&core.NumberField{Name: "biography_image_width"},
		&core.BoolField{Name: "published"},
	)
	if err := app.Save(artists); err != nil {
		t.Fatalf("save artists collection: %v", err)
	}
	artists.Fields.Add(&core.RelationField{Name: "also_known_as", CollectionId: "artists", MinSelect: 0, MaxSelect: 0})
	if err := app.Save(artists); err != nil {
		t.Fatalf("save artists also_known_as field: %v", err)
	}

	saveDualCollection(t, app, "artworks",
		&core.TextField{Name: "title", Required: true},
		&core.RelationField{Name: "author", CollectionId: "artists", MinSelect: 1, MaxSelect: 10},
		&core.TextField{Name: "technique"},
		&core.EditorField{Name: "comment"},
		&core.BoolField{Name: "published"},
		&core.TextField{Name: "image"},
		&core.NumberField{Name: "image_width"},
		&core.NumberField{Name: "year"},
		&core.RelationField{Name: "type", CollectionId: "art_types", MinSelect: 0, MaxSelect: 10},
	)
	saveDualCollection(t, app, "glossary",
		&core.TextField{Name: "expression", Required: true},
		&core.TextField{Name: "definition", Required: true},
	)
	saveDualCollection(t, app, "music_composer",
		&core.TextField{Name: "name", Required: true},
		&core.SelectField{Name: "century", Values: []string{"16", "17", "18", "19"}, MaxSelect: 1},
		&core.BoolField{Name: "published"},
	)
	saveDualCollection(t, app, "music_song",
		&core.TextField{Name: "title", Required: true},
		&core.RelationField{Name: "composer", CollectionId: "music_composer", MinSelect: 1, MaxSelect: 20},
		&core.TextField{Name: "source"},
		&core.BoolField{Name: "published"},
	)

	return app
}

func saveDualCollection(t *testing.T, app *pocketbase.PocketBase, id string, fields ...core.Field) {
	t.Helper()
	collection := core.NewBaseCollection(id)
	collection.Id = id
	collection.MarkAsNew()
	collection.Fields.Add(fields...)
	if err := app.Save(collection); err != nil {
		t.Fatalf("save %s collection: %v", id, err)
	}
}

func saveDualRecord(t *testing.T, app *pocketbase.PocketBase, collection string, id string, fields map[string]any) {
	t.Helper()
	coll, err := app.FindCollectionByNameOrId(collection)
	if err != nil {
		t.Fatalf("find %s: %v", collection, err)
	}
	record := core.NewRecord(coll)
	record.Id = id
	for key, value := range fields {
		record.Set(key, value)
	}
	if err := app.Save(record); err != nil {
		t.Fatalf("save %s %s: %v", collection, id, err)
	}
}

func seedDualArtistAndWork(t *testing.T, app *pocketbase.PocketBase) {
	t.Helper()
	saveDualRecord(t, app, "schools", "schooldutch0001", map[string]any{"name": "Dutch", "slug": "dutch"})
	saveDualRecord(t, app, "art_periods", "periodbaroque01", map[string]any{"name": "Baroque", "start": 1600, "end": 1750})
	saveDualRecord(t, app, "artists", "artistone000001", map[string]any{
		"name":          "Rembrandt",
		"slug":          "rembrandt",
		"bio":           "<p>A master of chiaroscuro.</p>",
		"year_of_birth": 1606,
		"year_of_death": 1669,
		"profession":    "painter",
		"school":        []string{"schooldutch0001"},
		"portrait":      "portrait.jpg",
		"biography_image_width": 800,
		"published":     true,
	})
	saveDualRecord(t, app, "artworks", "artworkone00001", map[string]any{
		"title":       "The Night Watch",
		"author":      []string{"artistone000001"},
		"technique":   "Oil on canvas, 363 x 437 cm",
		"comment":     "Painted in 1642 · Rijksmuseum · 363 x 437 cm",
		"published":   true,
		"image":       "night-watch.jpg",
		"image_width": 4000,
		"year":        1642,
	})
}

func TestBuildDualWorkRecordImageRenditionAndCitation(t *testing.T) {
	app := newDualTestApp(t)
	seedDualArtistAndWork(t, app)

	ref, err := loadDualReference(app)
	if err != nil {
		t.Fatalf("load reference: %v", err)
	}

	state := parseDualState(neturl.Values{})
	pane := state.right
	pane.path = "/artists/rembrandt-artistone000001/the-night-watch-artworkone00001"
	pane.renderTo = "left"

	record, artistName, artistPath, err := buildDualWorkRecord(app, "right", pane, state, ref)
	if err != nil {
		t.Fatalf("build work record: %v", err)
	}

	if artistName != "Rembrandt" {
		t.Fatalf("artist name = %q", artistName)
	}
	if artistPath != "/artists/rembrandt-artistone000001" {
		t.Fatalf("artist path = %q", artistPath)
	}

	// Medium (default) size must resolve to the 1100 profile.
	if !strings.Contains(record.Image, "thumb=1100x0") {
		t.Fatalf("medium image should use 1100x0 profile, got %q", record.Image)
	}
	// The deliberate viewer must resolve to the 2000 profile.
	if !strings.Contains(record.Zoom, "thumb=2000x0") {
		t.Fatalf("viewer should use 2000x0 profile, got %q", record.Zoom)
	}
	// The citation must point at the canonical artwork URL, never /dual-mode.
	if strings.Contains(record.Citation.URL, "/dual-mode") {
		t.Fatalf("citation must not reference the interactive route, got %q", record.Citation.URL)
	}
	if !strings.Contains(record.Citation.URL, "/artists/rembrandt-artistone000001/the-night-watch-artworkone00001") {
		t.Fatalf("citation URL should be canonical artwork URL, got %q", record.Citation.URL)
	}

	// Large size renders the 1600 profile.
	largePane := pane
	largePane.size = sizeLarge
	largeRecord, _, _, err := buildDualWorkRecord(app, "right", largePane, state, ref)
	if err != nil {
		t.Fatalf("build large work record: %v", err)
	}
	if !strings.Contains(largeRecord.Image, "thumb=1600x0") {
		t.Fatalf("large image should use 1600x0 profile, got %q", largeRecord.Image)
	}
	if largeRecord.PlateClass != "h-[680px]" {
		t.Fatalf("large plate class = %q", largeRecord.PlateClass)
	}
}

func TestBuildDualWorkRecordFallsBackToOriginalWhenNarrower(t *testing.T) {
	app := newDualTestApp(t)
	seedDualArtistAndWork(t, app)
	// A source narrower than every study profile must use the original file.
	saveDualRecord(t, app, "artworks", "artworktwo00001", map[string]any{
		"title": "Small Source", "author": []string{"artistone000001"}, "published": true, "image": "small.jpg", "image_width": 400, "year": 1600,
	})

	ref, err := loadDualReference(app)
	if err != nil {
		t.Fatalf("load reference: %v", err)
	}

	state := parseDualState(neturl.Values{})
	pane := state.right
	pane.path = "/artists/rembrandt-artistone000001/small-source-artworktwo00001"

	record, _, _, err := buildDualWorkRecord(app, "right", pane, state, ref)
	if err != nil {
		t.Fatalf("build work record: %v", err)
	}
	if strings.Contains(record.Image, "thumb=") {
		t.Fatalf("narrow source should use the original file, got %q", record.Image)
	}
	if !strings.Contains(record.Image, "/api/files/artworks/artworktwo00001/small.jpg") {
		t.Fatalf("image should be the original file URL, got %q", record.Image)
	}
}

func TestBuildDualArtistRecordCitationIsCanonical(t *testing.T) {
	app := newDualTestApp(t)
	seedDualArtistAndWork(t, app)

	ref, err := loadDualReference(app)
	if err != nil {
		t.Fatalf("load reference: %v", err)
	}

	state := parseDualState(neturl.Values{})
	pane := state.left
	pane.path = "/artists/rembrandt-artistone000001"

	record, err := buildDualArtistRecord(app, "left", pane, state, ref)
	if err != nil {
		t.Fatalf("build artist record: %v", err)
	}

	if record.Name != "Rembrandt" {
		t.Fatalf("artist name = %q", record.Name)
	}
	if strings.Contains(record.Citation.URL, "/dual-mode") {
		t.Fatalf("citation must not reference the interactive route, got %q", record.Citation.URL)
	}
	if !strings.Contains(record.Citation.URL, "/artists/rembrandt-artistone000001") {
		t.Fatalf("citation URL should be canonical artist URL, got %q", record.Citation.URL)
	}
	if record.Citation.Key != "wga-rembrandt" {
		t.Fatalf("citation key = %q", record.Citation.Key)
	}
	if len(record.Works) == 0 || record.Works[0].Title != "The Night Watch" {
		t.Fatalf("artist works grid should include the published work, got %+v", record.Works)
	}
}

func TestBuildWindowFallsBackToIndexForMissingRecord(t *testing.T) {
	app := newDualTestApp(t)
	seedDualArtistAndWork(t, app)

	ref, err := loadDualReference(app)
	if err != nil {
		t.Fatalf("load reference: %v", err)
	}

	state := parseDualState(neturl.Values{})
	pane := state.left
	pane.path = "/artists/missing-artistgone000001"

	window, err := buildWindow(app, "left", pane, state, ref)
	if err != nil {
		t.Fatalf("build window: %v", err)
	}
	if window.View != "index" {
		t.Fatalf("missing record should fall back to the index, got view %q", window.View)
	}
	if len(window.Crumb) != 1 || window.Crumb[0].Label != "ARTISTS" {
		t.Fatalf("index fallback crumb = %+v", window.Crumb)
	}
}

// ---------------------------------------------------------------------------
// Handler routes
// ---------------------------------------------------------------------------

func serveDualRequests(t *testing.T, app *pocketbase.PocketBase, requests []recordRequest) []*httptest.ResponseRecorder {
	t.Helper()
	RegisterHandlers(app)

	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatalf("create router: %v", err)
	}
	serveEvent := &core.ServeEvent{App: app, Router: router}

	recorders := make([]*httptest.ResponseRecorder, len(requests))
	if err := app.OnServe().Trigger(serveEvent, func(event *core.ServeEvent) error {
		mux, err := event.Router.BuildMux()
		if err != nil {
			return err
		}
		for i, req := range requests {
			request := httptest.NewRequest(http.MethodGet, req.path, nil)
			if req.htmx {
				request.Header.Set("HX-Request", "true")
			}
			recorders[i] = httptest.NewRecorder()
			mux.ServeHTTP(recorders[i], request)
		}
		return nil
	}); err != nil {
		t.Fatalf("trigger serve event: %v", err)
	}

	return recorders
}

type recordRequest struct {
	path string
	htmx bool
}

func TestDualModeRouteRendersFullAndHTMX(t *testing.T) {
	app := newDualTestApp(t)
	seedDualArtistAndWork(t, app)

	path := "/dual-mode?left=/artists/rembrandt-artistone000001&right=/artists/rembrandt-artistone000001/the-night-watch-artworkone00001"
	recorders := serveDualRequests(t, app, []recordRequest{
		{path: path, htmx: false},
		{path: path, htmx: true},
	})
	full := recorders[0]
	partial := recorders[1]

	if full.Code != http.StatusOK {
		t.Fatalf("full status = %d, want 200", full.Code)
	}
	body := full.Body.String()
	if !strings.Contains(body, "<html") {
		t.Error("full response should render the complete document")
	}
	if !strings.Contains(body, "Rembrandt") || !strings.Contains(body, "The Night Watch") {
		t.Error("full response should render both records")
	}
	if !strings.Contains(body, `id="dual-left"`) || !strings.Contains(body, `id="dual-right"`) {
		t.Error("full response should render both panes")
	}
	canonical := parseDualState(mustParseQuery(t, strings.TrimPrefix(path, "/dual-mode?"))).path()
	if got := full.Header().Get("HX-Push-Url"); got != canonical {
		t.Errorf("HX-Push-Url = %q, want %q", got, canonical)
	}

	if partial.Code != http.StatusOK {
		t.Fatalf("partial status = %d, want 200", partial.Code)
	}
	if strings.Contains(partial.Body.String(), "<html") {
		t.Error("HTMX response should not render the full document")
	}
	if !strings.Contains(partial.Body.String(), `id="dual-area"`) {
		t.Error("HTMX response should render the dual block fragment")
	}
}

func TestDualModeRoutePushesCanonicalURL(t *testing.T) {
	app := newDualTestApp(t)
	seedDualArtistAndWork(t, app)

	// Non-canonical input (leading spaces, default sentinel, invalid render_to)
	// should be normalised into the canonical push URL.
	recorders := serveDualRequests(t, app, []recordRequest{
		{path: "/dual-mode?right=default&left_render_to=centre", htmx: false},
	})
	recorder := recorders[0]

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if got := recorder.Header().Get("HX-Push-Url"); got != "/dual-mode" {
		t.Errorf("HX-Push-Url = %q, want canonical /dual-mode", got)
	}
}

func TestDualModeLookupRoute(t *testing.T) {
	app := newDualTestApp(t)
	seedDualArtistAndWork(t, app)

	recorders := serveDualRequests(t, app, []recordRequest{
		{path: "/dual-mode/lookup?kind=artist&q=rembrandt", htmx: false},
	})
	recorder := recorders[0]

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "Rembrandt") {
		t.Error("lookup should return matching artists")
	}
}

func TestDualModeRouteFallsBackForUnpublishedArtwork(t *testing.T) {
	app := newDualTestApp(t)
	seedDualArtistAndWork(t, app)
	saveDualRecord(t, app, "artworks", "artworkhid00001", map[string]any{
		"title": "Hidden Work", "author": []string{"artistone000001"}, "published": false, "image": "hidden.jpg", "image_width": 4000, "year": 1650,
	})

	recorders := serveDualRequests(t, app, []recordRequest{
		{path: "/dual-mode?left=" + neturl.QueryEscape("/artists/rembrandt-artistone000001/hidden-work-artworkhid00001"), htmx: false},
	})
	recorder := recorders[0]

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "01 — ARTIST INDEX") {
		t.Error("unpublished artwork should fall back to the pane index")
	}
	if strings.Contains(body, "Hidden Work") {
		t.Error("unpublished artwork must not render")
	}
}

func TestDualModeRouteFallsBackForUnpublishedAuthor(t *testing.T) {
	app := newDualTestApp(t)
	saveDualRecord(t, app, "artists", "artisthid000001", map[string]any{"name": "Hidden Artist", "slug": "hidden-artist", "published": false})
	saveDualRecord(t, app, "artworks", "artworkauth0001", map[string]any{
		"title": "Hidden Author Work", "author": []string{"artisthid000001"}, "published": true, "image": "ha.jpg", "image_width": 4000, "year": 1650,
	})

	recorders := serveDualRequests(t, app, []recordRequest{
		{path: "/dual-mode?left=" + neturl.QueryEscape("/artists/hidden-artist-artisthid000001/hidden-author-work-artworkauth0001"), htmx: false},
	})
	recorder := recorders[0]

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "01 — ARTIST INDEX") {
		t.Error("work with an unpublished author should fall back to the pane index")
	}
	if strings.Contains(body, "Hidden Author Work") {
		t.Error("work with an unpublished author must not render")
	}
}

func TestDualModeRouteCanonicalisesMismatchedArtistSlug(t *testing.T) {
	app := newDualTestApp(t)
	seedDualArtistAndWork(t, app)

	recorders := serveDualRequests(t, app, []recordRequest{
		{path: "/dual-mode?left=" + neturl.QueryEscape("/artists/wrong-slug-artistone000001"), htmx: false},
	})
	recorder := recorders[0]

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	want := parseDualState(mustParseQuery(t, "left=/artists/rembrandt-artistone000001")).path()
	if got := recorder.Header().Get("HX-Push-Url"); got != want {
		t.Errorf("HX-Push-Url = %q, want canonical %q", got, want)
	}
}

func TestDualModeRouteCanonicalisesMismatchedArtistSegment(t *testing.T) {
	app := newDualTestApp(t)
	seedDualArtistAndWork(t, app)
	saveDualRecord(t, app, "artists", "artisttwo000001", map[string]any{"name": "Vermeer", "slug": "vermeer", "published": true})

	// The artwork belongs to Rembrandt, but the requested artist segment points
	// at Vermeer. Canonicalisation must use the work's published author.
	recorders := serveDualRequests(t, app, []recordRequest{
		{path: "/dual-mode?left=" + neturl.QueryEscape("/artists/vermeer-artisttwo000001/the-night-watch-artworkone00001"), htmx: false},
	})
	recorder := recorders[0]

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	want := parseDualState(mustParseQuery(t, "left=/artists/rembrandt-artistone000001/the-night-watch-artworkone00001")).path()
	if got := recorder.Header().Get("HX-Push-Url"); got != want {
		t.Errorf("HX-Push-Url = %q, want canonical %q", got, want)
	}
}

func TestDualAnnotatedHTMLStripsScriptsHandlersAndUnsafeURLs(t *testing.T) {
	input := `<p onclick="alert(1)">Hello<script>alert(2)</script> <a href="javascript:alert(3)">link</a></p>`

	got := dualAnnotatedHTML(input, nil)

	if strings.Contains(got, "<script") {
		t.Errorf("script element not stripped: %q", got)
	}
	if strings.Contains(got, "onclick") {
		t.Errorf("event handler attribute not stripped: %q", got)
	}
	if strings.Contains(got, "javascript:") {
		t.Errorf("unsafe URL scheme not stripped: %q", got)
	}
	if !strings.Contains(got, "Hello") {
		t.Errorf("safe text not preserved: %q", got)
	}
	if !strings.Contains(got, "link") {
		t.Errorf("safe link text not preserved: %q", got)
	}
}

func TestBuildDualWorkRecordSanitisesComment(t *testing.T) {
	app := newDualTestApp(t)
	seedDualArtistAndWork(t, app)
	saveDualRecord(t, app, "artworks", "artworkmal00001", map[string]any{
		"title":       "Malicious Work",
		"author":      []string{"artistone000001"},
		"published":   true,
		"image":       "m.jpg",
		"image_width": 4000,
		"year":        1650,
		"comment":     `<p onclick="x()">Nice<script>bad()</script> <a href="javascript:bad()">click</a></p>`,
	})

	ref, err := loadDualReference(app)
	if err != nil {
		t.Fatalf("load reference: %v", err)
	}

	state := parseDualState(neturl.Values{})
	pane := state.left
	pane.path = "/artists/rembrandt-artistone000001/malicious-work-artworkmal00001"

	record, _, _, err := buildDualWorkRecord(app, "left", pane, state, ref)
	if err != nil {
		t.Fatalf("build work record: %v", err)
	}

	for _, forbidden := range []string{"<script", "onclick", "javascript:"} {
		if strings.Contains(record.Comment, forbidden) {
			t.Errorf("commentary contains %q after sanitisation: %q", forbidden, record.Comment)
		}
	}
	if !strings.Contains(record.Comment, "Nice") {
		t.Errorf("safe commentary text not preserved: %q", record.Comment)
	}
}

// ---------------------------------------------------------------------------
// Both-pane state and wide-override serialisation
// ---------------------------------------------------------------------------

// TestParseDualStateReadsBothPanesIndependently proves each pane carries its own
// full index/filter/view/sort/size state, so the two windows never share state.
func TestParseDualStateReadsBothPanesIndependently(t *testing.T) {
	state := parseDualState(mustParseQuery(t,
		"l_letter=B&l_school=dutch&l_view=grid&l_sort=za&l_size=large&"+
			"r_letter=M&r_school=italian&r_period=p1&r_q=vermeer&r_born_from=1500&r_born_to=1600&r_view=list&r_sort=birth&r_size=small"))

	left := state.left.index
	if left.letter != "B" || left.school != "dutch" || left.view != viewGrid || left.sort != sortZA || left.period != "" {
		t.Fatalf("left index = %+v", left)
	}
	if state.left.size != sizeLarge {
		t.Fatalf("left size = %q, want large", state.left.size)
	}

	right := state.right.index
	if right.letter != "M" || right.school != "italian" || right.period != "p1" || right.query != "vermeer" ||
		right.bornFrom != 1500 || right.bornTo != 1600 || right.view != viewList || right.sort != sortBirth {
		t.Fatalf("right index = %+v", right)
	}
	if state.right.size != sizeSmall {
		t.Fatalf("right size = %q, want small", state.right.size)
	}
}

// TestDualStatePathSerializesWideOverride pins the ?wide=1 flag and its leading
// position in the canonical serialisation.
func TestDualStatePathSerializesWideOverride(t *testing.T) {
	assertDualPath(t, parseDualState(mustParseQuery(t, "wide=1")).path(), "/dual-mode?wide=1")

	state := parseDualState(mustParseQuery(t, "left=/artists/aaa-bbb&wide=1"))
	assertDualPath(t, state.path(), "/dual-mode?wide=1&left=%2Fartists%2Faaa-bbb")
}

// ---------------------------------------------------------------------------
// Filter/view/sort/image/link-target transitions
// ---------------------------------------------------------------------------

// TestDualIndexTransitionsProduceCanonicalURLs verifies every per-pane transition
// serialises to exactly the canonical URL for that change.
func TestDualIndexTransitionsProduceCanonicalURLs(t *testing.T) {
	state := parseDualState(neturl.Values{})

	assertDualPath(t, state.withPaneIndex("left", state.left.index.withLetter("A")).path(), "/dual-mode?l_letter=A")
	assertDualPath(t, state.withPaneIndex("left", dualIndexState{school: "dutch", view: viewList, sort: sortAZ}).path(), "/dual-mode?l_school=dutch")
	assertDualPath(t, state.withPaneIndex("left", dualIndexState{period: "p1", view: viewList, sort: sortAZ}).path(), "/dual-mode?l_period=p1")
	assertDualPath(t, state.withPaneIndex("left", dualIndexState{query: "van gogh", view: viewList, sort: sortAZ}).path(), "/dual-mode?l_q=van+gogh")
	assertDualPath(t, state.withPaneIndex("left", dualIndexState{bornFrom: 1600, bornTo: 1700, view: viewList, sort: sortAZ}).path(), "/dual-mode?l_born_from=1600&l_born_to=1700")
	assertDualPath(t, state.withPaneIndex("left", state.left.index.withView(viewGrid)).path(), "/dual-mode?l_view=grid")
	assertDualPath(t, state.withPaneIndex("left", state.left.index.withSort(sortZA)).path(), "/dual-mode?l_sort=za")
	assertDualPath(t, state.withPaneSize("left", sizeLarge).path(), "/dual-mode?l_size=large")
	assertDualPath(t, state.withPaneRenderTo("left", "left").path(), "/dual-mode?left_render_to=left")
}

// TestDualIndexTransitionPreservesOtherPane confirms a transition on one pane
// never rewrites the other pane's record state.
func TestDualIndexTransitionPreservesOtherPane(t *testing.T) {
	base := dualState{
		left:  dualPaneState{renderTo: "right", index: dualIndexState{view: viewList, sort: sortAZ}, size: sizeMedium},
		right: dualPaneState{path: "/artists/right-456", renderTo: "left", index: dualIndexState{view: viewList, sort: sortAZ}, size: sizeMedium},
	}

	changed := base.withPaneIndex("left", base.left.index.withView(viewGrid))
	assertDualPath(t, changed.path(), "/dual-mode?l_view=grid&right=%2Fartists%2Fright-456")
}

func TestNextDualSortAndLabel(t *testing.T) {
	if got := nextDualSort(sortAZ); got != sortZA {
		t.Fatalf("nextDualSort(az) = %q, want za", got)
	}
	if got := nextDualSort(sortZA); got != sortBirth {
		t.Fatalf("nextDualSort(za) = %q, want birth", got)
	}
	if got := nextDualSort(sortBirth); got != sortAZ {
		t.Fatalf("nextDualSort(birth) = %q, want az", got)
	}

	if got := dualSortLabel(sortAZ); got != "A–Z" {
		t.Fatalf("dualSortLabel(az) = %q", got)
	}
	if got := dualSortLabel(sortZA); got != "Z–A" {
		t.Fatalf("dualSortLabel(za) = %q", got)
	}
	if got := dualSortLabel(sortBirth); got != "BIRTH YEAR" {
		t.Fatalf("dualSortLabel(birth) = %q", got)
	}
}

func TestDualFilterNote(t *testing.T) {
	ref := testReference()

	if got := dualFilterNote(dualIndexState{letter: "B", school: "dutch", period: "p1", bornFrom: 1600, bornTo: 1700}, ref); got != "· B · DUTCH · BAROQUE · BORN 1600–1700" {
		t.Fatalf("full filter note = %q", got)
	}
	if got := dualFilterNote(dualIndexState{bornFrom: 1600}, ref); got != "· BORN 1600–1900" {
		t.Fatalf("born-only filter note = %q", got)
	}
	if got := dualFilterNote(dualIndexState{}, ref); got != "" {
		t.Fatalf("empty filter note = %q", got)
	}
}

// ---------------------------------------------------------------------------
// No-JavaScript links
// ---------------------------------------------------------------------------

// TestDualModeBlockRendersNoJavascriptControls proves the whole comparison
// surface is ordinary GET navigation: the filter form submits without JS and
// every state/navigation control is a real link carrying a plain href.
func TestDualModeBlockRendersNoJavascriptControls(t *testing.T) {
	view := pages.DualModeView{
		Windows: [2]pages.DualWindow{
			{
				Key: "left", Tag: "L", Label: "LEFT", View: "index",
				SelfSel: "#dual-left", TargetSel: "#dual-right", OtherLabel: "RIGHT WINDOW",
				IndexHref: "/dual-mode",
				RouteHref: "/dual-mode?left_render_to=left",
				Index: pages.DualIndexView{
					View:     "list",
					AllUrl:   "/dual-mode",
					GridHref: "/dual-mode?l_view=grid",
					ListHref: "/dual-mode",
					SortHref: "/dual-mode?l_sort=za",
					SortLabel: "A–Z",
					Letters:   []pages.DualLetter{{Label: "A", Href: "/dual-mode?l_letter=A", Enabled: true}},
					SchoolGroup: dto.ChipGroup{
						Legend: "SCHOOL", Name: "l_school", Inline: true,
						Options: []dto.ChipOption{{Label: "ALL", Value: "", Checked: true}},
					},
					PeriodGroup: dto.ChipGroup{
						Legend: "PERIOD", Name: "l_period", Inline: true,
						Options: []dto.ChipOption{{Label: "ALL", Value: "", Checked: true}},
					},
					NameField: dto.Field{ID: "l-name", Name: "l_q", Label: "NAME CONTAINS", Type: "search"},
					Artists: []pages.DualArtistRow{
						{Name: "An Artist", Dates: "1564–1616", Href: "/dual-mode?right=%2Fartists%2Fan-artist-artistid000001"},
					},
				},
			},
			{
				Key: "right", Tag: "R", Label: "RIGHT", View: "work",
				SelfSel: "#dual-right", TargetSel: "#dual-left", OtherLabel: "LEFT WINDOW",
				RoutesToSelf: true,
				IndexHref:    "/dual-mode",
				RouteHref:    "/dual-mode?right_render_to=right",
				BackHref:     "/dual-mode",
				Work: pages.DualWorkRecord{
					Title:      "A Work",
					Byline:     "An Artist, 1600 →",
					ArtistHref: "/dual-mode?left=%2Fartists%2Fan-artist-artistid000001",
					Sizes: []pages.DualLink{
						{Label: "700", Href: "/dual-mode?r_size=small"},
						{Label: "1100", Href: "/dual-mode", Selected: true},
						{Label: "1600", Href: "/dual-mode?r_size=large"},
					},
					SizeCaption: "REPRODUCTION AT 1100PX WIDE",
				},
				Send: pages.DualLink{Label: "SEND TO LEFT WINDOW →", Href: "/dual-mode?left=%2Fartists%2Fan-artist-artistid000001"},
			},
		},
		SwapHref:  "/dual-mode",
		ResetHref: "/dual-mode",
		ExitHref:  "/artists",
		WideHref:  "/dual-mode?wide=1",
		ForceWide: true,
	}

	markup := renderDualBlock(t, view)

	for _, expected := range []string{
		`action="/dual-mode"`,
		`method="GET"`,
		`<noscript>`,
		"APPLY FILTERS",
	} {
		if !strings.Contains(markup, expected) {
			t.Errorf("no-JS filter form missing %q", expected)
		}
	}

	for _, expected := range []string{
		`<a href="/dual-mode?l_letter=A"`,
		`<a href="/dual-mode?l_view=grid"`,
		`<a href="/dual-mode?l_sort=za"`,
		`<a href="/dual-mode?left_render_to=left"`,
		`<a href="/dual-mode?r_size=small"`,
		`<a href="/dual-mode?right=%2Fartists%2Fan-artist-artistid000001"`,
		`<a href="/artists"`,
		`<a href="/dual-mode?wide=1"`,
	} {
		if !strings.Contains(markup, expected) {
			t.Errorf("no-JS control missing link %q", expected)
		}
	}

	// The artist row and the work byline must be real anchors, not JS-only.
	if !strings.Contains(markup, ">An Artist</a>") {
		t.Error("available artist row must render as an anchor")
	}
	if !strings.Contains(markup, ">An Artist, 1600 →</a>") {
		t.Error("work byline must render as an anchor to the artist")
	}
}

// ---------------------------------------------------------------------------
// Record/index parity
// ---------------------------------------------------------------------------

// TestDualIndexAndRecordParity proves the index, the resolved pane path, and the
// built record agree: an available index row links to a canonical artist URL
// whose record reproduces the row's name and published works.
func TestDualIndexAndRecordParity(t *testing.T) {
	app := newDualTestApp(t)
	seedDualArtistAndWork(t, app)

	ref, err := loadDualReference(app)
	if err != nil {
		t.Fatalf("load reference: %v", err)
	}

	state := parseDualState(neturl.Values{})
	state.left.renderTo = "left"

	indexView, err := buildDualIndexView(app, "left", state.left, state, ref)
	if err != nil {
		t.Fatalf("build index: %v", err)
	}

	var row *pages.DualArtistRow
	for i := range indexView.Artists {
		if indexView.Artists[i].Name == "Rembrandt" {
			row = &indexView.Artists[i]
			break
		}
	}
	if row == nil {
		t.Fatal("index should list the seeded artist")
	}
	if row.Unavailable {
		t.Fatal("seeded artist should be available")
	}
	if row.Href == "" {
		t.Fatal("available artist row must carry a href")
	}

	rowState := parseDualState(mustParseQuery(t, strings.TrimPrefix(row.Href, "/dual-mode?")))
	canonical, err := resolvePaneCanonicalPath(app, rowState.left)
	if err != nil {
		t.Fatalf("resolve row href: %v", err)
	}
	if canonical != "/artists/rembrandt-artistone000001" {
		t.Fatalf("row href resolves to %q, want canonical artist path", canonical)
	}

	pane := state.left
	pane.path = canonical
	record, err := buildDualArtistRecord(app, "left", pane, state, ref)
	if err != nil {
		t.Fatalf("build artist record: %v", err)
	}
	if record.Name != "Rembrandt" {
		t.Fatalf("record name = %q, want index name", record.Name)
	}
	if len(record.Works) != 1 || record.Works[0].Title != "The Night Watch" {
		t.Fatalf("record works = %+v, want the single published work", record.Works)
	}
	if !strings.Contains(record.Works[0].Href, "the-night-watch-artworkone00001") {
		t.Fatalf("work card href %q should reference the canonical artwork", record.Works[0].Href)
	}
}

// ---------------------------------------------------------------------------
// Index view filter/sort and cross-pane state retention
// ---------------------------------------------------------------------------

// TestBuildDualIndexViewAppliesPaneFilterAndSort builds the left pane's index
// over a two-artist fixture with its own query/sort state and proves the right
// pane's canonical state survives in every generated link URL.
func TestBuildDualIndexViewAppliesPaneFilterAndSort(t *testing.T) {
	app := newDualTestApp(t)
	seedDualArtistAndWork(t, app)
	saveDualRecord(t, app, "artists", "artisttwo000001", map[string]any{
		"name": "Vermeer", "slug": "vermeer", "year_of_birth": 1632, "profession": "painter", "published": true,
	})

	ref, err := loadDualReference(app)
	if err != nil {
		t.Fatalf("load reference: %v", err)
	}

	state := parseDualState(neturl.Values{})
	state.left.index = dualIndexState{query: "ver", sort: sortZA, view: viewGrid}
	state.right.path = "/artists/rembrandt-artistone000001"

	indexView, err := buildDualIndexView(app, "left", state.left, state, ref)
	if err != nil {
		t.Fatalf("build index: %v", err)
	}

	// The left pane's query must narrow the result set to the single match.
	if indexView.Total != 1 {
		t.Fatalf("total = %d, want 1", indexView.Total)
	}
	if len(indexView.Artists) != 1 || indexView.Artists[0].Name != "Vermeer" {
		t.Fatalf("filtered artists = %+v, want exactly [Vermeer]", indexView.Artists)
	}
	if indexView.SortLabel != dualSortLabel(sortZA) {
		t.Fatalf("sort label = %q, want %q", indexView.SortLabel, dualSortLabel(sortZA))
	}

	// The right pane's canonical path must be retained by every generated link
	// URL the left pane emits.
	for name, href := range map[string]string{
		"AllUrl":    indexView.AllUrl,
		"GridHref":  indexView.GridHref,
		"ListHref":  indexView.ListHref,
		"SortHref":  indexView.SortHref,
		"ResetHref": indexView.ResetHref,
	} {
		values := mustParseQuery(t, strings.TrimPrefix(href, "/dual-mode?"))
		if got := values.Get("right"); got != "/artists/rembrandt-artistone000001" {
			t.Errorf("%s = %q, right pane path was not retained", name, href)
		}
	}

	// The filter form's hidden fields must carry the other pane's state so a
	// subsequent GET never loses it.
	preserved := false
	for _, field := range indexView.Hidden {
		if field.Name == "right" && field.Value == "/artists/rembrandt-artistone000001" {
			preserved = true
		}
	}
	if !preserved {
		t.Errorf("hidden fields should preserve the right pane path, got %+v", indexView.Hidden)
	}
}

// TestBuildDualIndexViewSortsDeterministically proves the pane's sort state
// changes the ordering of the two seeded artists deterministically.
func TestBuildDualIndexViewSortsDeterministically(t *testing.T) {
	app := newDualTestApp(t)
	seedDualArtistAndWork(t, app)
	saveDualRecord(t, app, "artists", "artisttwo000001", map[string]any{
		"name": "Vermeer", "slug": "vermeer", "year_of_birth": 1632, "profession": "painter", "published": true,
	})

	ref, err := loadDualReference(app)
	if err != nil {
		t.Fatalf("load reference: %v", err)
	}

	names := func(sort string) []string {
		t.Helper()
		state := parseDualState(neturl.Values{})
		state.left.index = dualIndexState{sort: sort, view: viewList}
		view, err := buildDualIndexView(app, "left", state.left, state, ref)
		if err != nil {
			t.Fatalf("build index (sort %q): %v", sort, err)
		}
		result := make([]string, 0, len(view.Artists))
		for _, row := range view.Artists {
			result = append(result, row.Name)
		}
		return result
	}

	az := names(sortAZ)
	if len(az) != 2 || az[0] != "Rembrandt" || az[1] != "Vermeer" {
		t.Fatalf("name-ascending order = %v, want [Rembrandt Vermeer]", az)
	}

	za := names(sortZA)
	if len(za) != 2 || za[0] != "Vermeer" || za[1] != "Rembrandt" {
		t.Fatalf("name-descending order = %v, want [Vermeer Rembrandt]", za)
	}
}

// ---------------------------------------------------------------------------
// Unpublished records
// ---------------------------------------------------------------------------

// TestDualModeRouteFallsBackForUnpublishedArtist covers a direct unpublished
// artist pane path, which must resolve to the pane index rather than render.
func TestDualModeRouteFallsBackForUnpublishedArtist(t *testing.T) {
	app := newDualTestApp(t)
	saveDualRecord(t, app, "artists", "artisthid000001", map[string]any{"name": "Hidden Artist", "slug": "hidden-artist", "published": false})

	recorders := serveDualRequests(t, app, []recordRequest{
		{path: "/dual-mode?left=" + neturl.QueryEscape("/artists/hidden-artist-artisthid000001"), htmx: false},
	})
	recorder := recorders[0]

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, "01 — ARTIST INDEX") {
		t.Error("unpublished artist should fall back to the pane index")
	}
	if strings.Contains(body, "Hidden Artist") {
		t.Error("unpublished artist must not render")
	}
}
