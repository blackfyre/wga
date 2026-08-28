package pages

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
)

func renderTimelineBlock(t *testing.T, view TimelineView) string {
	t.Helper()
	var output bytes.Buffer
	if err := TimelineBlock(view).Render(context.Background(), &output); err != nil {
		t.Fatalf("render timeline block: %v", err)
	}
	return output.String()
}

func populatedTimelineView() TimelineView {
	return TimelineView{
		From:       1500,
		To:         1600,
		HasRange:   true,
		RangeLabel: "1500–1600",
		SpanLabel:  "101 YEARS",
		HitLabel:   "3 WORKS",
		Presets: []dto.NavChip{
			{Label: "FULL RANGE", Href: "/timeline", Active: true},
			{Label: "Baroque", Href: "/timeline?from=1500&to=1750"},
		},
		WindowRange: dto.RangeField{
			Label: "SET A WINDOW", FromID: "timeline-from", FromName: "from", FromValue: 1500,
			ToID: "timeline-to", ToName: "to", ToValue: 1600, Min: 1500, Max: 1800, Step: 1, Brush: true,
		},
		Density: []TimelineDensityBar{
			{Decade: "1500", Count: 2, HeightPct: 100},
			{Decade: "1510", Count: 1, HeightPct: 50},
		},
		Bands: []TimelineBand{
			{Name: "Baroque", SpanLabel: "1500–1600", LeftPct: 0, WidthPct: 100, Description: "a period"},
		},
		Marks: []TimelineMark{
			{Href: "/artists/a-1/w-1", Label: "Work One — 1500", LeftPct: 25},
		},
		Works: []dto.Work{
			{URL: "/artists/a-1/w-1", ImageURL: "/api/files/artworks/w-1/x.jpg", Title: "Work One", Artist: "An Artist", Metadata: "1500"},
		},
		WorksRangeLabel: "3 WORKS",
		HasWorks:        true,
		TotalWorks:      3,
		Page:            1,
		PageCount:       1,
		Lanes: []TimelineLane{
			{Key: LaneKeyArtists, Label: "ARTISTS", Count: 1, State: LaneStateShown},
			{Key: LaneKeyWorks, Label: "WORKS", Count: 3, State: LaneStateShown},
			{Key: LaneKeyMovements, Label: "MOVEMENTS", Count: 1, State: LaneStateShown},
			{Key: LaneKeyBuildings, Label: "BUILDINGS", Count: 0, State: LaneStateUnavailable},
			{Key: LaneKeyEvents, Label: "EVENTS", Count: 0, State: LaneStateUnavailable},
			{Key: LaneKeyMusic, Label: "MUSIC", Count: 0, State: LaneStateUnavailable},
		},
		Artists: []TimelineArtist{
			{Href: "/artists/a-1", Name: "An Artist", Meta: "b. 1600"},
		},
	}
}

func TestTimelineBlockRendersSemanticChronology(t *testing.T) {
	rendered := renderTimelineBlock(t, populatedTimelineView())

	for _, expected := range []string{
		"19 — TIMELINE EXPLORER",
		`<h1`,
		"1500–1600",
		"101 YEARS",
		"3 WORKS",
		"BARS SHOW CATALOGUE DENSITY PER DECADE",
		"ART PERIODS IN THIS WINDOW",
		"Baroque",
		"WORKS ACROSS THE WINDOW",
		"WORKS IN THIS WINDOW",
		`action="/timeline"`,
		`method="GET"`,
		`name="from"`,
		`name="to"`,
		`<noscript>`,
		`type="submit"`,
		"APPLY WINDOW",
		`href="/artists/a-1/w-1"`,
		`aria-live="polite"`,
		// Preset links carry the timeline-specific replacement contract.
		`hx-select="#timeline"`,
		`hx-swap="outerHTML"`,
		// The density strip exposes an assistive-technology table.
		`class="sr-only"`,
		"DECADE",
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("rendered timeline does not contain %q", expected)
		}
	}

	if strings.Count(rendered, "<h1") != 1 {
		t.Errorf("h1 count = %d, want 1", strings.Count(rendered, "<h1"))
	}

	for _, mark := range []string{`aria-label="Work One — 1500"`, `title="Work One — 1500"`} {
		if !strings.Contains(rendered, mark) {
			t.Errorf("rendered timeline marks do not contain %q", mark)
		}
	}
}

func TestTimelineBlockUsesTypedBlockAddControlForWorks(t *testing.T) {
	view := populatedTimelineView()
	view.Works[0].ArtworkID = "aw0000000000001"
	ctx := tmplUtils.WithItineraryProjection(context.Background(), "csrf-token", dto.ItineraryTrayView{}, nil)
	var output bytes.Buffer
	if err := TimelineBlock(view).Render(ctx, &output); err != nil {
		t.Fatalf("render timeline block: %v", err)
	}
	rendered := output.String()
	for _, expected := range []string{`h-[50px]`, "ADD TO AN ITINERARY +", `hx-select="unset"`} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("timeline work card missing typed block contract %q", expected)
		}
	}
}

func TestTimelineBlockRendersDensityTableValues(t *testing.T) {
	rendered := renderTimelineBlock(t, populatedTimelineView())

	if !strings.Contains(rendered, `>1500</th>`) || !strings.Contains(rendered, `>2</td>`) {
		t.Error("density table did not render the decade and count values")
	}
	if !strings.Contains(rendered, `>1510</th>`) || !strings.Contains(rendered, `>1</td>`) {
		t.Error("density table did not render the second decade and count values")
	}
}

func TestTimelineBlockRendersNoChronologyEmptyState(t *testing.T) {
	view := populatedTimelineView()
	view.HasRange = false
	rendered := renderTimelineBlock(t, view)

	if !strings.Contains(rendered, "No chronology yet") {
		t.Error("missing no-chronology empty state")
	}
	if !strings.Contains(rendered, `href="/artworks"`) {
		t.Error("missing artwork recovery link")
	}
	if strings.Contains(rendered, "WORKS ACROSS THE WINDOW") {
		t.Error("empty state must not render the mark lane")
	}
	for _, expected := range []string{
		`aria-label="Timeline lanes"`,
		"BUILDINGS IN THIS WINDOW",
		"EVENTS IN THIS WINDOW",
		"MUSIC IN THIS WINDOW",
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("empty chronology must preserve %q", expected)
		}
	}
}

func TestTimelineBlockRendersNoWorksInWindowState(t *testing.T) {
	view := populatedTimelineView()
	view.HasWorks = false
	view.WorksRangeLabel = "0 WORKS"
	view.Works = nil

	rendered := renderTimelineBlock(t, view)

	if !strings.Contains(rendered, "No works in this window") {
		t.Error("missing no-works-in-window empty state")
	}
	if !strings.Contains(rendered, `href="/timeline"`) {
		t.Error("missing full-range recovery link")
	}
}

func TestTimelineBlockRendersPagination(t *testing.T) {
	view := populatedTimelineView()
	view.WorksRangeLabel = "WORKS 9–16 OF 20"
	view.TotalWorks = 20
	view.Page = 2
	view.PageCount = 3
	view.PrevUrl = "/timeline?from=1500&to=1600"
	view.NextUrl = "/timeline?from=1500&to=1600&page=3"

	rendered := renderTimelineBlock(t, view)

	if !strings.Contains(rendered, "PAGE 2 OF 3") {
		t.Error("missing pagination readout")
	}
	if !strings.Contains(rendered, "WORKS 9–16 OF 20") {
		t.Error("missing truthful range label")
	}
	if !strings.Contains(rendered, `hx-get="/timeline?from=1500&amp;to=1600"`) {
		t.Error("missing prev link")
	}
	if !strings.Contains(rendered, `hx-get="/timeline?from=1500&amp;to=1600&amp;page=3"`) {
		t.Error("missing next link")
	}
	if strings.Contains(rendered, "VIEW ALL WORKS") {
		t.Error("timeline must not render the unfiltered /artworks continuation")
	}
}

func TestTimelineBlockOmitsPaginationOnSinglePage(t *testing.T) {
	rendered := renderTimelineBlock(t, populatedTimelineView())

	if strings.Contains(rendered, "PAGE 1 OF 1") {
		t.Error("single-page timeline must not render pagination controls")
	}
	if strings.Contains(rendered, "← PREV") || strings.Contains(rendered, "NEXT →") {
		t.Error("single-page timeline must not render prev/next links")
	}
}

func TestTimelineBlockRendersResponsiveMarkup(t *testing.T) {
	rendered := renderTimelineBlock(t, populatedTimelineView())

	// Preset controls wrap rather than forcing a fixed intrinsic width.
	if !strings.Contains(rendered, `class="mt-6 flex flex-wrap gap-1.5"`) {
		t.Error("preset nav must wrap via flex flex-wrap")
	}
	// Every preset anchor may shrink below its content width and break long
	// labels, so 33 presets can never overflow a narrow container.
	if !strings.Contains(rendered, `class="min-w-0 max-w-full break-words border border-primary`) {
		t.Error("active preset anchor must carry min-w-0 max-w-full break-words")
	}
	if !strings.Contains(rendered, `class="min-w-0 max-w-full break-words border border-control`) {
		t.Error("inactive preset anchor must carry min-w-0 max-w-full break-words")
	}
	// Density bars may shrink to zero so ~190 decade bins cannot overflow.
	if !strings.Contains(rendered, `class="min-w-0 flex-1 bg-primary/60"`) {
		t.Error("density bars must shrink via min-w-0 flex-1")
	}
	// The works grid uses one column until 400px so the full-width itinerary
	// action remains usable at enlarged text; it then restores two mobile columns
	// before the four-column md layout.
	if !strings.Contains(rendered, `grid min-w-0 grid-cols-1 gap-x-4 gap-y-10 min-[400px]:grid-cols-2 md:grid-cols-4`) {
		t.Error("works grid must shrink from one to two to four columns")
	}
	if !strings.Contains(rendered, `<li class="min-w-0">`) {
		t.Error("works grid items must shrink at enlarged text")
	}
	// Band rows reflow at narrow/zoomed widths: the row wraps and the fixed
	// name/date children cap and break rather than forcing a 640px minimum.
	if !strings.Contains(rendered, `class="flex flex-wrap items-center gap-4 border-b border-base-content/10 py-2"`) {
		t.Error("band row must wrap via flex flex-wrap")
	}
	if !strings.Contains(rendered, `class="w-44 shrink-0 max-w-full break-words text-(length:--t-14) font-semibold"`) {
		t.Error("band name must cap and break via max-w-full break-words")
	}
	if !strings.Contains(rendered, `class="relative h-5 flex-1 border border-base-content/10 bg-base-200/40"`) {
		t.Error("band lane must flex via flex-1")
	}
	if !strings.Contains(rendered, `class="w-28 shrink-0 max-w-full break-words text-right font-mono text-(length:--t-11) text-muted"`) {
		t.Error("band date must cap and break via max-w-full break-words")
	}
}

func TestTimelineBlockWrapsAccessibleDensityTable(t *testing.T) {
	rendered := renderTimelineBlock(t, populatedTimelineView())

	// The accessible density table must not carry sr-only directly on the table:
	// a table honours width:1px as a minimum and expands to its content width,
	// overflowing the viewport. A block wrapper clips it instead.
	if strings.Contains(rendered, `<table class="sr-only">`) {
		t.Error("density table must not place sr-only directly on the table element")
	}
	if !strings.Contains(rendered, `<div class="sr-only">`) || !strings.Contains(rendered, `<table>`) {
		t.Error("density table must be wrapped in a sr-only block container")
	}
	if !strings.Contains(rendered, `>DECADE</th>`) || !strings.Contains(rendered, `>WORKS</th>`) {
		t.Error("density table must remain present for assistive technology")
	}
}

func TestTimelineBlockOmitsHistoricalEvents(t *testing.T) {
	rendered := renderTimelineBlock(t, populatedTimelineView())

	// The timeline renders approved art periods and published artworks only; it
	// must not invent historical-event entries, names, dates, or prose. The
	// EVENTS lane remains visible but states its honest unavailability.
	for _, marker := range []string{
		"HISTORICAL EVENTS",
		">Events<",
		"war broke out",
		"revolution",
		"battle of",
		"the discovery of",
		"Columbus",
		"Napoleon",
		"1492",
		"1789",
	} {
		if strings.Contains(rendered, marker) {
			t.Errorf("timeline rendered fabricated historical-event content %q", marker)
		}
	}
}

func TestTimelineBlockRendersSixLabelledLanes(t *testing.T) {
	rendered := renderTimelineBlock(t, populatedTimelineView())

	for _, expected := range []string{
		`aria-label="Timeline lanes"`,
		">ARTISTS<",
		">WORKS<",
		">MOVEMENTS<",
		">BUILDINGS<",
		">EVENTS<",
		">MUSIC<",
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("rendered timeline lanes do not contain %q", expected)
		}
	}

	// Each lane exposes its honest state.
	for _, expected := range []string{
		">SHOWN<",
		">UNAVAILABLE<",
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("rendered timeline lanes do not contain state %q", expected)
		}
	}

	// Unsupported lanes carry a precise, visible availability explanation rather
	// than invented content.
	for _, expected := range []string{
		"BUILDINGS IN THIS WINDOW",
		"No approved source-backed building records have been supplied",
		"EVENTS IN THIS WINDOW",
		"No approved source-backed historical-event records have been supplied",
		"MUSIC IN THIS WINDOW",
		"The music collection has no approved art-period mapping",
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("rendered timeline does not contain unavailable-lane explanation %q", expected)
		}
	}
}

func TestTimelineBlockRendersTruthfulArtists(t *testing.T) {
	rendered := renderTimelineBlock(t, populatedTimelineView())

	if !strings.Contains(rendered, "ARTISTS IN THIS WINDOW") {
		t.Error("missing artists lane heading")
	}
	if !strings.Contains(rendered, `href="/artists/a-1"`) {
		t.Error("missing artist record link")
	}
	// The artist entry exposes the backend-supplied name and its stored birth
	// year; the template derives neither.
	for _, expected := range []string{">An Artist</a>", "b. 1600"} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("artists lane entry does not contain %q", expected)
		}
	}
}

func TestTimelineBlockRendersEmptyArtistLane(t *testing.T) {
	view := populatedTimelineView()
	view.Artists = nil

	rendered := renderTimelineBlock(t, view)

	if !strings.Contains(rendered, "ARTISTS IN THIS WINDOW") {
		t.Error("missing artists lane heading")
	}
	if !strings.Contains(rendered, "No artists are dated inside this window.") {
		t.Error("missing honest empty-artists explanation")
	}
}

func TestTimelineBlockRendersEmptyLaneState(t *testing.T) {
	view := populatedTimelineView()
	for i := range view.Lanes {
		if view.Lanes[i].Key == LaneKeyMovements {
			view.Lanes[i].Count = 0
			view.Lanes[i].State = LaneStateEmpty
		}
	}

	rendered := renderTimelineBlock(t, view)

	// An approved lane that resolves nothing inside the window is EMPTY, not
	// UNAVAILABLE and not SHOWN.
	if !strings.Contains(rendered, ">EMPTY<") {
		t.Error("empty movements lane did not render the EMPTY state")
	}
}
