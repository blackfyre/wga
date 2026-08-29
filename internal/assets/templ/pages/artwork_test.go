package pages

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
	tmplUtils "github.com/blackfyre/wga/internal/assets/templ/utils"
)

const artworkTestID = "aw0000000000001"

func renderArtworkBlock(t *testing.T, aw dto.Artwork, ctx context.Context) string {
	t.Helper()

	var output bytes.Buffer
	if err := ArtworkBlock(aw).Render(ctx, &output); err != nil {
		t.Fatalf("render artwork block: %v", err)
	}

	return output.String()
}

func sampleArtwork() dto.Artwork {
	return dto.Artwork{
		Id:    artworkTestID,
		Title: "Girl with a Pearl Earring",
		Image: dto.Image{
			Image: "/api/files/artworks/" + artworkTestID + "/work.jpg?thumb=1400x0",
			Zoom:  "/api/files/artworks/" + artworkTestID + "/work.jpg?thumb=2000x0",
			Title: "Girl with a Pearl Earring",
		},
		Artist: dto.Artist{
			FilingName: "Vermeer, Johannes",
			ShortName:  "Vermeer",
			Url:        "/artists/johannes-vermeer-artist00000001",
		},
		ReproFile: "4,095 × 4,801 px · JPEG · 12.4 MB",
		SourceURL: "/api/files/artworks/" + artworkTestID + "/work.jpg",
	}
}

// TestArtworkBlockRendersPlateNotLegacyWell locks the record composition to the
// shared reference Plate: a natural-aspect, contained image well with the
// dedicated zoom URL and no legacy fixed-height shadow well.
func TestArtworkBlockRendersPlateNotLegacyWell(t *testing.T) {
	rendered := renderArtworkBlock(t, sampleArtwork(), context.Background())

	for _, expected := range []string{
		`src="/api/files/artworks/` + artworkTestID + `/work.jpg?thumb=1400x0"`,
		`data-zoom-url="/api/files/artworks/` + artworkTestID + `/work.jpg?thumb=2000x0"`,
		`href="/api/files/artworks/` + artworkTestID + `/work.jpg?thumb=2000x0"`,
		`data-viewer`,
		"CLICK TO ZOOM",
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("artwork block does not contain %q", expected)
		}
	}

	for _, legacy := range []string{"h-[340px]", "md:h-[560px]", "shadow", "hidden-caption"} {
		if strings.Contains(rendered, legacy) {
			t.Errorf("artwork block must not carry legacy image-well class %q", legacy)
		}
	}
}

// TestArtworkBlockRendersNoImageFallbackInsideWell proves the no-image state
// renders the plate's bounded text placeholder rather than an <img>, so the
// fallback never escapes the well.
func TestArtworkBlockRendersNoImageFallbackInsideWell(t *testing.T) {
	aw := sampleArtwork()
	aw.Image = dto.Image{Title: aw.Title}

	rendered := renderArtworkBlock(t, aw, context.Background())

	if strings.Contains(rendered, "<img") {
		t.Error("no-image artwork must not render an <img> element")
	}
	if !strings.Contains(rendered, "IMAGE — Girl with a Pearl Earring") {
		t.Error("no-image artwork must render the plate placeholder caption")
	}
}

// TestArtworkBlockAddToItineraryUsesUnsetSentinel proves the artwork record's
// add control clears the shared body-level hx-select via the htmx "unset"
// sentinel so the tray/OOB response is processed, while keeping the ordinary
// POST fallback and its explicit local target/swap.
func TestArtworkBlockAddToItineraryUsesUnsetSentinel(t *testing.T) {
	ctx := tmplUtils.WithItineraryProjection(context.Background(), "csrf-token", dto.ItineraryTrayView{}, map[string]bool{})

	rendered := renderArtworkBlock(t, sampleArtwork(), ctx)

	for _, expected := range []string{
		`action="/itineraries/draft/add"`,
		`method="post"`,
		`hx-post="/itineraries/draft/add"`,
		`hx-target="#itinerary-tray"`,
		`hx-swap="outerHTML"`,
		`hx-select="unset"`,
		`name="artwork_id" value="` + artworkTestID + `"`,
		"ADD TO AN ITINERARY +",
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("artwork add control does not contain %q", expected)
		}
	}

	if strings.Contains(rendered, `hx-disinherit="hx-select"`) {
		t.Error("add control must not use hx-disinherit; the unset sentinel preserves the inherited tray/OOB swap")
	}
}

// TestArtworkBlockRendersReproductionFigcaption locks the reference artwork
// composition: a "THIS REPRODUCTION" caption with a full-file download link and
// a FILE metadata cell rendered from the recorded reproduction fields only.
func TestArtworkBlockRendersReproductionFigcaption(t *testing.T) {
	rendered := renderArtworkBlock(t, sampleArtwork(), context.Background())

	for _, expected := range []string{
		"THIS REPRODUCTION",
		"DOWNLOAD THE FULL FILE ↓",
		`href="/api/files/artworks/` + artworkTestID + `/work.jpg"`,
		"FILE",
		"4,095 × 4,801 px · JPEG",
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("artwork figcaption does not contain %q", expected)
		}
	}

	if !strings.Contains(rendered, `<figcaption>`) {
		t.Error("reproduction caption must be a figcaption inside the figure")
	}

	// The plate must precede the figcaption, which precedes the record article.
	plate := strings.Index(rendered, "CLICK TO ZOOM")
	caption := strings.Index(rendered, "THIS REPRODUCTION")
	title := strings.Index(rendered, "03 — ARTWORK")
	if plate < 0 || caption < 0 || title < 0 || !(plate < caption && caption < title) {
		t.Error("artwork composition order must be plate, then reproduction figcaption, then record article")
	}
}

// TestArtworkBlockDownloadUsesSourceNotRendition proves the visible download
// link resolves to the original source URL, never a relabelled thumbnail
// rendition.
func TestArtworkBlockDownloadUsesSourceNotRendition(t *testing.T) {
	rendered := renderArtworkBlock(t, sampleArtwork(), context.Background())

	if !strings.Contains(rendered, `href="/api/files/artworks/`+artworkTestID+`/work.jpg" target="_blank" rel="noopener noreferrer"`) {
		t.Error("download link must resolve to the original source URL with target=_blank")
	}
}

// TestArtworkBlockNeverRendersReproductionSourceOrLicence proves the record
// reproduction caption exposes no public source or licence claim. The source
// URL and licence are deliberately absent from the DTO, so no such presentation
// can exist.
func TestArtworkBlockNeverRendersReproductionSourceOrLicence(t *testing.T) {
	rendered := renderArtworkBlock(t, sampleArtwork(), context.Background())

	for _, absent := range []string{
		"VIEW ORIGINAL AT WEB GALLERY OF ART",
		"wga.hu",
		"LICENCE",
		"LICENSE",
		"SOURCE",
	} {
		if strings.Contains(rendered, absent) {
			t.Errorf("artwork reproduction caption must not expose %q", absent)
		}
	}
}

// TestArtworkBlockOmitsSourceControlsWithoutImage proves the download link and
// reproduction caption details are omitted when no source image exists.
func TestArtworkBlockOmitsSourceControlsWithoutImage(t *testing.T) {
	aw := sampleArtwork()
	aw.SourceURL = ""
	aw.ReproFile = ""

	rendered := renderArtworkBlock(t, aw, context.Background())

	if strings.Contains(rendered, "DOWNLOAD THE FULL FILE") {
		t.Error("download link must be omitted without a source image")
	}
	if strings.Contains(rendered, `>FILE</dt>`) {
		t.Error("FILE cell must be omitted without reproduction dimensions")
	}
}

// TestArtworkBlockEscapesSourceURL proves the source URL is HTML-escaped in the
// rendered href so a filename with special characters cannot inject markup.
func TestArtworkBlockEscapesSourceURL(t *testing.T) {
	aw := sampleArtwork()
	aw.SourceURL = `/api/files/artworks/` + artworkTestID + `/work&copy.jpg`

	rendered := renderArtworkBlock(t, aw, context.Background())

	if strings.Contains(rendered, `href="/api/files/artworks/`+artworkTestID+`/work&copy.jpg"`) {
		t.Error("source URL must be HTML-escaped in the href")
	}
	if !strings.Contains(rendered, `work&amp;copy.jpg`) {
		t.Error("source URL ampersand must render as &amp;")
	}
}

// TestArtworkBlockOmitsReproductionFileWhenUnknown proves the FILE cell is not
// rendered when no supported reproduction evidence is recorded (no fabricated
// content); independently absent facts may still be omitted from the summary.
func TestArtworkBlockOmitsReproductionFileWhenUnknown(t *testing.T) {
	aw := sampleArtwork()
	aw.ReproFile = ""

	rendered := renderArtworkBlock(t, aw, context.Background())

	if strings.Contains(rendered, `>FILE</dt>`) {
		t.Error("artwork must not render a FILE cell when no reproduction file is known")
	}
	if !strings.Contains(rendered, "THIS REPRODUCTION") {
		t.Error("reproduction caption heading must still render")
	}
}

// TestArtworkBlockRendersPostcardBeforeItinerary proves the reference action
// order: SEND AS POSTCARD precedes ADD TO AN ITINERARY +.
func TestArtworkBlockRendersPostcardBeforeItinerary(t *testing.T) {
	ctx := tmplUtils.WithItineraryProjection(context.Background(), "csrf-token", dto.ItineraryTrayView{}, map[string]bool{})
	rendered := renderArtworkBlock(t, sampleArtwork(), ctx)

	postcard := strings.Index(rendered, "SEND AS POSTCARD")
	itinerary := strings.Index(rendered, "ADD TO AN ITINERARY +")
	if postcard < 0 || itinerary < 0 {
		t.Fatal("expected both postcard and itinerary controls")
	}
	if postcard > itinerary {
		t.Error("SEND AS POSTCARD must render before ADD TO ITINERARY")
	}
}

func TestArtworkBlockStacksRecordActionsAtReferenceSpacing(t *testing.T) {
	ctx := tmplUtils.WithItineraryProjection(context.Background(), "csrf-token", dto.ItineraryTrayView{}, map[string]bool{})
	rendered := renderArtworkBlock(t, sampleArtwork(), ctx)

	for _, expected := range []string{`class="mt-7 flex flex-col gap-2.5"`, `h-[50px] w-full items-center justify-center`, `h-[50px]`} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("record actions must contain %q", expected)
		}
	}
}

// TestReproRuleEscapesValue proves the reproduction metadata cell escapes HTML
// so a hostile or malformed value cannot inject markup.
func TestReproRuleEscapesValue(t *testing.T) {
	var output bytes.Buffer
	if err := reproRule("FILE", `<script>alert(1)</script> & "quoted"`).Render(context.Background(), &output); err != nil {
		t.Fatalf("render repro rule: %v", err)
	}

	rendered := output.String()
	if strings.Contains(rendered, "<script>") {
		t.Error("repro rule must escape script-like input")
	}
	if !strings.Contains(rendered, "&lt;script&gt;") {
		t.Error("repro rule must HTML-escape the value")
	}
}

func relatedSampleArtwork() dto.Artwork {
	aw := sampleArtwork()
	aw.Related = dto.RelatedWorkState{
		ActiveBasis:    "collection",
		Connection:     "SAME COLLECTION",
		Sparse:         true,
		SparseNote:     "The archive catalogues no further works from this collection.",
		Alternative:    "BY ARTIST",
		AlternativeURL: "/artists/johannes-vermeer-artist00000001/girl-with-a-pearl-earring-aw0000000000001",
		Bases: []dto.RelatedWorkBasis{
			{Value: "artist", Label: "BY ARTIST", URL: "/artists/johannes-vermeer-artist00000001/girl-with-a-pearl-earring-aw0000000000001", Active: false},
			{Value: "collection", Label: "SAME COLLECTION", URL: "/artists/johannes-vermeer-artist00000001/girl-with-a-pearl-earring-aw0000000000001?basis=collection", Active: true},
			{Value: "period", Label: "SAME PERIOD", URL: "/artists/johannes-vermeer-artist00000001/girl-with-a-pearl-earring-aw0000000000001?basis=period", Active: false},
		},
	}
	return aw
}

func TestArtworkBlockRendersPalette(t *testing.T) {
	aw := sampleArtwork()
	aw.Palette = []dto.ColourSwatch{
		{Name: "Prussian Blue", Hex: "#1a2b3c", Weight: 5000},
		{Name: "Slate Blue", Hex: "#4d5e6f", Weight: 3000},
	}

	rendered := renderArtworkBlock(t, aw, context.Background())

	for _, expected := range []string{
		"PALETTE",
		"background:#1a2b3c",
		"background:#4d5e6f",
		"#1a2b3c",
		"#4d5e6f",
		"Prussian Blue",
		"63%",
		"SAMPLED FROM THE IMAGE",
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("artwork palette does not contain %q", expected)
		}
	}
}

func TestArtworkBlockRendersUnnamedPaletteSwatches(t *testing.T) {
	aw := sampleArtwork()
	aw.Palette = []dto.ColourSwatch{{Hex: "#1a2b3c", Weight: 5000}}

	rendered := renderArtworkBlock(t, aw, context.Background())

	if !strings.Contains(rendered, `aria-label="#1a2b3c, 100% of the surface"`) {
		t.Errorf("unnamed artwork palette has incorrect label: %s", rendered)
	}
	if strings.Contains(rendered, "Not recorded.") {
		t.Error("unnamed artwork palette must not be treated as unavailable")
	}
}

func TestArtworkBlockStatesUnavailablePaletteWhenEmpty(t *testing.T) {
	rendered := renderArtworkBlock(t, sampleArtwork(), context.Background())

	if !strings.Contains(rendered, "PALETTE") {
		t.Error("artwork must render the palette section even when no palette is recorded")
	}
	if !strings.Contains(rendered, "Not recorded.") {
		t.Error("artwork must state the unavailable palette state explicitly")
	}
	if strings.Contains(rendered, "SAMPLED FROM THE IMAGE") {
		t.Error("artwork must not render the palette disclaimer without a palette")
	}
}

func TestArtworkPaletteBlockUsesContainedInteractiveSwatches(t *testing.T) {
	aw := dto.Artwork{
		Palette: []dto.ColourSwatch{
			{Name: "Prussian Blue", Hex: "#1a2b3c", Weight: 5000},
			{Name: "Slate Blue", Hex: "#4d5e6f", Weight: 3000},
		},
	}

	var output bytes.Buffer
	if err := artworkPaletteBlock(aw).Render(context.Background(), &output); err != nil {
		t.Fatalf("render palette block: %v", err)
	}
	rendered := output.String()

	for _, expected := range []string{"data-wga-palette-bar", "Prussian Blue, #1a2b3c, 63% of the surface", "background:#1a2b3c;flex-grow:5000;flex-basis:0", "Prussian Blue", "#1a2b3c"} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("palette must include interactive swatch contract %q", expected)
		}
	}
}

func TestArtworkPaletteBlockUnavailableStateUnchanged(t *testing.T) {
	var output bytes.Buffer
	if err := artworkPaletteBlock(dto.Artwork{}).Render(context.Background(), &output); err != nil {
		t.Fatalf("render palette block: %v", err)
	}
	rendered := output.String()

	if !strings.Contains(rendered, "Not recorded.") {
		t.Error("palette unavailable state must remain")
	}
	if strings.Contains(rendered, "SAMPLED FROM THE IMAGE") {
		t.Error("palette disclaimer must not render without a palette")
	}
}

func TestArtworkBlockRendersPeriodMusic(t *testing.T) {
	aw := sampleArtwork()
	aw.Music = dto.MusicPeriod{
		Available: true,
		SongID:    "songone00000000",
		Piece:     "Fantasia chromatica",
		Composer:  "Sweelinck",
		PlayerURL: "/player?song=songone00000000",
	}

	rendered := renderArtworkBlock(t, aw, context.Background())

	for _, expected := range []string{
		"PERIOD MUSIC",
		"Sweelinck — Fantasia chromatica",
		`href="/player?song=songone00000000"`,
		`target="wga-period-music"`,
		`data-wga-music="songone00000000"`,
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("artwork music card does not contain %q", expected)
		}
	}
}

func TestArtworkBlockOmitsMusicWhenUnavailable(t *testing.T) {
	rendered := renderArtworkBlock(t, sampleArtwork(), context.Background())

	if strings.Contains(rendered, "PERIOD MUSIC") {
		t.Error("artwork must omit the music card when no published match exists")
	}
}

func TestArtworkBlockRendersCommentaryHonestly(t *testing.T) {
	withCommentary := sampleArtwork()
	withCommentary.HasCommentary = true
	withCommentary.SourceComment = "<p>A truthful source commentary.</p>"
	withCommentary.Comment = "<p>A divergent enriched commentary.</p>"

	rendered := renderArtworkBlock(t, withCommentary, context.Background())
	if !strings.Contains(rendered, "A truthful source commentary.") {
		t.Error("artwork must render the source commentary when present")
	}
	if strings.Contains(rendered, "A divergent enriched commentary.") {
		t.Error("artwork must not render the divergent enriched comment field")
	}
	if strings.Contains(rendered, "Commentary is unavailable") {
		t.Error("artwork must not render the unavailable state when commentary exists")
	}

	without := sampleArtwork()
	rendered = renderArtworkBlock(t, without, context.Background())
	if !strings.Contains(rendered, "Commentary is unavailable for this artwork.") {
		t.Error("artwork must render the honest unavailable commentary state")
	}
	if strings.Contains(rendered, "A truthful source commentary.") {
		t.Error("artwork must not invent commentary when none exists")
	}
}

func TestArtworkBlockRendersRelatedBasisControls(t *testing.T) {
	rendered := renderArtworkBlock(t, relatedSampleArtwork(), context.Background())

	for _, expected := range []string{
		"SAME COLLECTION",
		"BY ARTIST",
		"SAME PERIOD",
		`href="/artists/johannes-vermeer-artist00000001/girl-with-a-pearl-earring-aw0000000000001?basis=collection"`,
		"The archive catalogues no further works from this collection.",
		"TRY BY ARTIST →",
		`href="/artists/johannes-vermeer-artist00000001/girl-with-a-pearl-earring-aw0000000000001"`,
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("artwork related block does not contain %q", expected)
		}
	}
	if strings.Contains(rendered, "SIMILAR PALETTE") {
		t.Error("artwork related block must omit the palette similarity basis")
	}
	// The active basis carries aria-current.
	if !strings.Contains(rendered, `aria-current="page"`) {
		t.Error("active related basis must carry aria-current")
	}
}

func TestArtworkRelatedLinksCarryHtmxEnhancement(t *testing.T) {
	rendered := renderArtworkBlock(t, relatedSampleArtwork(), context.Background())

	// The basis links carry the ?basis= query; they must render both the
	// plain href (the no-JavaScript fallback) and hx-get (the HTMX
	// enhancement) on the same element.
	if !strings.Contains(rendered, `hx-get="/artists/johannes-vermeer-artist00000001/girl-with-a-pearl-earring-aw0000000000001?basis=collection"`) {
		t.Error("related basis links must carry hx-get")
	}
	if !strings.Contains(rendered, `href="/artists/johannes-vermeer-artist00000001/girl-with-a-pearl-earring-aw0000000000001?basis=collection"`) {
		t.Error("related basis links must also render ordinary hrefs")
	}
}

func TestArtworkBlockOmitsRelatedWithoutBasis(t *testing.T) {
	rendered := renderArtworkBlock(t, sampleArtwork(), context.Background())

	for _, absent := range []string{"BY ARTIST", "SAME COLLECTION", "SAME PERIOD"} {
		if strings.Contains(rendered, absent) {
			t.Errorf("artwork must omit related basis controls without basis state, found %q", absent)
		}
	}
}

func TestArtworkBlockRendersRelatedCards(t *testing.T) {
	aw := relatedSampleArtwork()
	aw.Related.Sparse = false
	aw.Related.SparseNote = ""
	aw.Related.AlternativeURL = ""
	aw.RelatedWorks = dto.ImageGrid{{
		Id: "related00000001", Title: "A Related Work",
		Url:      "/artists/johannes-vermeer-artist00000001/a-related-work-related00000001",
		Image:    "/api/files/artworks/related00000001/r.jpg?thumb=400x0",
		Metadata: "1665",
		Artist:   dto.Artist{Name: "Johannes Vermeer"},
	}}

	rendered := renderArtworkBlock(t, aw, context.Background())

	if !strings.Contains(rendered, "A Related Work") {
		t.Error("artwork related block must render related-work cards")
	}
	for _, expected := range []string{"1665", `grid-cols-2`, `md:grid-cols-3`, `lg:grid-cols-4`, `aspect-[4/5]`} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("related-work grid must contain %q", expected)
		}
	}
	if strings.Contains(rendered, "VIEW WORK →") {
		t.Error("related-work grid must not use the shared catalogue card action")
	}
	if strings.Contains(rendered, "The archive catalogues no further works") {
		t.Error("artwork must not render a sparse note when the basis is not sparse")
	}
}

func TestArtworkBlockRendersCountedHoldingLink(t *testing.T) {
	aw := relatedSampleArtwork()
	aw.Related.Sparse = false
	aw.Related.Holding = &dto.RelatedWorkHoldingLink{
		Label: "FIND MORE 11 IN THE ARTWORK SEARCH →",
		URL:   "/artworks?artist=Vermeer%2C+Johannes",
	}

	rendered := renderArtworkBlock(t, aw, context.Background())

	for _, expected := range []string{
		"FIND MORE 11 IN THE ARTWORK SEARCH →",
		`href="/artworks?artist=Vermeer%2C+Johannes"`,
		`hx-get="/artworks?artist=Vermeer%2C+Johannes"`,
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("artwork related block does not contain holding link %q", expected)
		}
	}
}

func TestArtworkBlockOmitsHoldingWithoutState(t *testing.T) {
	aw := relatedSampleArtwork()
	rendered := renderArtworkBlock(t, aw, context.Background())

	if strings.Contains(rendered, "IN THE ARTWORK SEARCH") {
		t.Error("artwork must not render a holding link without holding state")
	}
}

func TestArtworkBlockRendersReproductionFileWeight(t *testing.T) {
	rendered := renderArtworkBlock(t, sampleArtwork(), context.Background())

	if !strings.Contains(rendered, "4,095 × 4,801 px · JPEG · 12.4 MB") {
		t.Error("artwork FILE cell must render dimensions, format, and decimal-SI weight")
	}
}
