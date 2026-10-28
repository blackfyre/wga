package pages

import (
	"strings"
	"testing"
)

func largeHoldingView() ArtistView {
	return ArtistView{
		FilingName: "Portrait Artist",
		ShortName:  "Portrait",
		WorkCount:  40,
		SelectionForms: []SelectionIndexForm{
			{
				Name:           "Drawings",
				AnchorID:       "form-rselect00000001",
				SelectionCount: 2,
				WorkCount:      27,
				Selections: []SelectionTile{
					{URL: "/artists/portrait-artist-artistone000001/selections/rselect00000001", AnchorID: "selection-rselect00000001", Label: "1484–1499", WorkCount: 16, Cover: "/api/files/artworks/artwork1/cover.jpg"},
					{URL: "/artists/portrait-artist-artistone000001/selections/rselect00000002", AnchorID: "selection-rselect00000002", Label: "1500–1505", WorkCount: 11, Cover: "/api/files/artworks/artwork2/cover.jpg"},
				},
			},
			{
				Name:           "The Navicella mosaic",
				AnchorID:       "form-rselect00000003",
				SelectionCount: 1,
				WorkCount:      1,
				Selections: []SelectionTile{
					{URL: "/artists/portrait-artist-artistone000001/selections/rselect00000003", AnchorID: "selection-rselect00000003", Label: "The Navicella mosaic", WorkCount: 1},
				},
			},
		},
	}
}

func TestArtistRecordContentRendersLargeHoldingFormBlocks(t *testing.T) {
	rendered := renderArtistRecord(t, largeHoldingView())

	for _, expected := range []string{
		`<section id="form-rselect00000001"`,
		`>Drawings</h2>`,
		"2 SELECTIONS · 27 WORKS",
		"1 SELECTION · 1 WORK",
		`<li id="selection-rselect00000001"`,
		`href="/artists/portrait-artist-artistone000001/selections/rselect00000001" hx-get="/artists/portrait-artist-artistone000001/selections/rselect00000001"`,
		`src="/api/files/artworks/artwork1/cover.jpg"`,
		"1484–1499",
		"16 works",
		"11 works",
		"1 work<",
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("expected large-holding record to contain %q", expected)
		}
	}
	for _, unexpected := range []string{"CURATED SELECTIONS", "OPEN SELECTION", "Commentary is unavailable", "SELECTED ·", "FIND MORE BY"} {
		if strings.Contains(rendered, unexpected) {
			t.Errorf("large-holding record must not render %q", unexpected)
		}
	}
}

func TestArtistRecordTOCOmittedWithoutCuratedSelections(t *testing.T) {
	for name, view := range map[string]ArtistView{
		"zero": {FilingName: "Artist, Portrait", ShortName: "Portrait", WorkCount: 2},
		"one":  {FilingName: "Artist, Portrait", ShortName: "Portrait", WorkCount: 2, Selections: []SelectionPreview{{AnchorID: "selection-one", DisplayTitle: "Only selection"}}},
	} {
		t.Run(name, func(t *testing.T) {
			rendered := renderArtistRecord(t, view)
			for _, unexpected := range []string{"ON THIS PAGE", "data-selection-index", "CURATED SELECTIONS"} {
				if strings.Contains(rendered, unexpected) {
					t.Errorf("record with %s selections must omit %q", name, unexpected)
				}
			}
		})
	}
}

func TestArtistRecordContentLargeHoldingTileFallsBackToFormName(t *testing.T) {
	rendered := renderArtistRecord(t, largeHoldingView())

	start := strings.Index(rendered, `<li id="selection-rselect00000003"`)
	if start < 0 {
		t.Fatal("expected the coverless tile")
	}
	tile := rendered[start:]
	tile = tile[:strings.Index(tile, "</li>")]
	if strings.Contains(tile, "<img") {
		t.Error("coverless tile must not render an image")
	}
	if !strings.Contains(tile, ">The Navicella mosaic</span>") {
		t.Error("coverless tile must show the form name in place of an image")
	}
	if !strings.Contains(tile, `href="/artists/portrait-artist-artistone000001/selections/rselect00000003"`) {
		t.Error("coverless tile must remain a link to its selection route")
	}
}

func TestArtistRecordTOCRendersLargeHoldingFormRows(t *testing.T) {
	rendered := renderArtistRecord(t, largeHoldingView())

	for _, expected := range []string{
		`data-selection-index`,
		`href="#biography"`,
		`href="#form-rselect00000001" class="min-w-0 font-mono text-(length:--t-11) tracking-[1px] hover:text-wga-accent aria-[current=location]:text-wga-accent">DRAWINGS · 2 SEL · 27</a>`,
		`href="#form-rselect00000003" class="min-w-0 font-mono text-(length:--t-11) tracking-[1px] hover:text-wga-accent aria-[current=location]:text-wga-accent">THE NAVICELLA MOSAIC · 1 SEL · 1</a>`,
		`<ul id="form-rselect00000001-entries" class="mt-2 mb-1 grid grid-cols-2`,
		`href="#selection-rselect00000001" class="wrap-anywhere hover:text-wga-accent">1484–1499</a>`,
		`href="#selection-rselect00000002" class="wrap-anywhere hover:text-wga-accent">1500–1505</a>`,
		`href="#cite-this-record"`,
		`md:sticky md:top-6 md:max-h-[calc(100dvh-var(--wga-bottom-stack-height,0px)-3rem)] md:overflow-y-auto`,
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("expected large-holding TOC to contain %q", expected)
		}
	}
	// Without JavaScript the rail offers no fold control and keeps every
	// foldable list open; a single-selection form has no list.
	start := strings.Index(rendered, `<nav class="mt-7`)
	if start < 0 {
		t.Fatal("expected the ON THIS PAGE rail")
	}
	rail := rendered[start:]
	rail = rail[:strings.Index(rail, "</nav>")]
	for _, unexpected := range []string{"<button", "aria-expanded", " hidden", `id="form-rselect00000003-entries"`} {
		if strings.Contains(rail, unexpected) {
			t.Errorf("server-rendered large-holding TOC must not contain %q", unexpected)
		}
	}
	if strings.Contains(rendered, "· 16 works</a>") {
		t.Error("large-holding TOC must not render per-selection preview rows")
	}
}
