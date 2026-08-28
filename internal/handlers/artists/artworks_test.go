package artists

import (
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/repositories"
	apputils "github.com/blackfyre/wga/internal/utils"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func TestArtworkLocationAndDimensions(t *testing.T) {
	location, dimensions := artworkLocationAndDimensions("<p>1902 · Synthetic Gallery, Test City · 101 x 201 cm</p>")
	if location != "Synthetic Gallery, Test City" {
		t.Errorf("location = %q, want %q", location, "Synthetic Gallery, Test City")
	}
	if dimensions != "101 x 201 cm" {
		t.Errorf("dimensions = %q, want %q", dimensions, "101 x 201 cm")
	}
}

func TestArtworkLocationAndDimensionsWithoutCatalogueSummary(t *testing.T) {
	location, dimensions := artworkLocationAndDimensions("<p>Commentary without catalogue metadata.</p>")
	if location != "" || dimensions != "" {
		t.Errorf("artworkLocationAndDimensions() = %q, %q; want empty values", location, dimensions)
	}
}

func TestPopulateArtworkMetadataUsesDateEnd(t *testing.T) {
	collection := core.NewBaseCollection("Artworks")
	collection.Fields.Add(
		&core.NumberField{Name: "year"},
		&core.NumberField{Name: "date_end"},
	)
	artwork := core.NewRecord(collection)
	artwork.Set("year", 1500)
	artwork.Set("date_end", 1502)

	content := dto.Artwork{}
	populateArtworkMetadata(nil, artwork, &content)

	if content.Year != "1502" {
		t.Errorf("year = %q, want date_end 1502", content.Year)
	}

	artwork.Set("date_end", 0)
	content = dto.Artwork{}
	populateArtworkMetadata(nil, artwork, &content)
	if content.Year != "1500" {
		t.Errorf("year = %q, want legacy year 1500", content.Year)
	}
}

func TestBuildRelatedWorkImagesUsesBasisAppropriateMetadata(t *testing.T) {
	app := newArtistRecordApp(t)
	seedPublishedArtist(t, app)
	work, err := app.FindRecordById("artworks", "artworkone00001")
	if err != nil {
		t.Fatalf("find related work: %v", err)
	}

	byArtist := buildRelatedWorkImages(app, repositories.RelatedByArtist, []*core.Record{work}, "")
	if byArtist[0].Metadata != "1610" {
		t.Errorf("by-artist metadata = %q, want creation date 1610", byArtist[0].Metadata)
	}

	byCollection := buildRelatedWorkImages(app, repositories.RelatedByCollection, []*core.Record{work}, "")
	if byCollection[0].Metadata != "Artist, Synthetic" {
		t.Errorf("by-collection metadata = %q, want artist identity", byCollection[0].Metadata)
	}
}

func TestPopulateArtworkCitation(t *testing.T) {
	configuration := config.LoadFrom(func(key string) string {
		return map[string]string{
			"WGA_ENV":                "development",
			"WGA_PROTOCOL":           "https",
			"WGA_HOSTNAME":           "gallery.example",
			"WGA_SENDER_NAME":        "WGA",
			"WGA_SENDER_ADDRESS":     "sender@example.com",
			"WGA_POSTCARD_FREQUENCY": "*/1 * * * *",
		}[key]
	})
	server, err := configuration.Server()
	if err != nil {
		t.Fatalf("load server configuration: %v", err)
	}
	apputils.ConfigurePublicURL(server.PublicURL)
	t.Cleanup(func() {
		apputils.ConfigurePublicURL(config.PublicURL{})
	})

	artwork := dto.Artwork{
		Id:    "07561d2efd0a6db",
		Title: "Girl with a Pearl Earring",
		Url:   "/artists/johannes-vermeer-artist00001/girl-with-a-pearl-earring-07561d2efd0a6db",
		Artist: dto.Artist{
			FilingName: "Vermeer, Johannes",
		},
	}
	populateArtworkCitation(&artwork)

	if artwork.CitationKey != "wga-07561d2efd0a6db" {
		t.Errorf("CitationKey = %q", artwork.CitationKey)
	}
	if artwork.CitationTitle != "Girl with a Pearl Earring by Vermeer, Johannes" {
		t.Errorf("CitationTitle = %q", artwork.CitationTitle)
	}
	if artwork.CitationURL != "https://gallery.example/artists/johannes-vermeer-artist00001/girl-with-a-pearl-earring-07561d2efd0a6db" {
		t.Errorf("CitationURL = %q", artwork.CitationURL)
	}
}

func TestRelatedWorkURL(t *testing.T) {
	base := "/artists/durer-artistone000001/work-workone00000001"

	if got := relatedWorkURL(base, repositories.RelatedByArtist); got != base {
		t.Errorf("default basis URL = %q, want %q (no query)", got, base)
	}
	if got := relatedWorkURL(base, repositories.RelatedByCollection); got != base+"?basis=collection" {
		t.Errorf("collection basis URL = %q, want %q", got, base+"?basis=collection")
	}
	if got := relatedWorkURL(base, repositories.RelatedByPeriod); got != base+"?basis=period" {
		t.Errorf("period basis URL = %q, want %q", got, base+"?basis=period")
	}
	if got := relatedWorkURL(base, repositories.RelatedByPalette); got != base+"?basis=palette" {
		t.Errorf("palette basis URL = %q, want %q", got, base+"?basis=palette")
	}
}

func TestRelatedWorkBases(t *testing.T) {
	base := "/artists/durer-artistone000001/work-workone00000001"
	bases := relatedWorkBases(base, repositories.RelatedByCollection)

	wantLabels := []string{"BY ARTIST", "SAME COLLECTION", "SAME PERIOD", "SIMILAR PALETTE"}
	wantValues := []string{"artist", "collection", "period", "palette"}
	if len(bases) != 4 {
		t.Fatalf("bases = %d, want 4", len(bases))
	}
	for i, basis := range bases {
		if basis.Label != wantLabels[i] {
			t.Errorf("bases[%d] label = %q, want %q", i, basis.Label, wantLabels[i])
		}
		if basis.Value != wantValues[i] {
			t.Errorf("bases[%d] value = %q, want %q", i, basis.Value, wantValues[i])
		}
		if basis.Active != (basis.Value == "collection") {
			t.Errorf("bases[%d] active = %v, want collection active", i, basis.Active)
		}
		if basis.Value == "artist" && basis.URL != base {
			t.Errorf("artist URL = %q, want %q (no query)", basis.URL, base)
		}
		if basis.Value == "collection" && basis.URL != base+"?basis=collection" {
			t.Errorf("collection URL = %q", basis.URL)
		}
	}
}

func TestRelatedConnection(t *testing.T) {
	cases := []struct {
		basis     repositories.RelatedWorkBasis
		artist    string
		dateStart int
		want      string
	}{
		{repositories.RelatedByArtist, "Dürer", 0, "OTHER WORKS BY DÜRER"},
		{repositories.RelatedByCollection, "Dürer", 0, "SAME COLLECTION"},
		{repositories.RelatedByPalette, "Dürer", 0, "WORKS WITH A SIMILAR PALETTE"},
		{repositories.RelatedByPeriod, "Dürer", 1600, "ARTISTS WORKING 1560–1640"},
		{repositories.RelatedByPeriod, "Dürer", 0, "ARTISTS FROM THE SAME PERIOD"},
	}
	for _, c := range cases {
		if got := relatedConnection(c.basis, c.artist, c.dateStart); got != c.want {
			t.Errorf("relatedConnection(%q, %q, %d) = %q, want %q", c.basis, c.artist, c.dateStart, got, c.want)
		}
	}
}

func TestRelatedAlternative(t *testing.T) {
	base := "/artists/durer-artistone000001/work-workone00000001"

	label, u := relatedAlternative(repositories.RelatedByPalette, base)
	if label != "BY ARTIST" || u != base {
		t.Errorf("palette alternative = %q, %q; want BY ARTIST, %q", label, u, base)
	}

	label, u = relatedAlternative(repositories.RelatedByCollection, base)
	if label != "SIMILAR PALETTE" || u != base+"?basis=palette" {
		t.Errorf("collection alternative = %q, %q; want SIMILAR PALETTE", label, u)
	}
}

func TestParseArtworkPalette(t *testing.T) {
	collection := core.NewBaseCollection("Artworks")
	collection.Fields.Add(&core.JSONField{Name: "colour_palette"})

	valid := core.NewRecord(collection)
	valid.Set("colour_palette", []map[string]any{
		{"hex": "#1a2b3c", "weight": 5000},
		{"hex": "#4d5e6f", "weight": 3000},
	})
	swatches := parseArtworkPalette(valid)
	if len(swatches) != 2 {
		t.Fatalf("palette = %d swatches, want 2", len(swatches))
	}
	if swatches[0].Hex != "#1a2b3c" || swatches[0].Weight != 5000 {
		t.Errorf("swatch[0] = %+v", swatches[0])
	}
	if swatches[1].Hex != "#4d5e6f" || swatches[1].Weight != 3000 {
		t.Errorf("swatch[1] = %+v", swatches[1])
	}

	if got := parseArtworkPalette(core.NewRecord(collection)); len(got) != 0 {
		t.Errorf("empty palette = %v, want 0 swatches", got)
	}

	malformed := core.NewRecord(collection)
	malformed.Set("colour_palette", "not-json")
	if got := parseArtworkPalette(malformed); len(got) != 0 {
		t.Errorf("malformed palette = %v, want 0 swatches", got)
	}
}

func TestParseArtworkPaletteSkipsEmptyHex(t *testing.T) {
	collection := core.NewBaseCollection("Artworks")
	collection.Fields.Add(&core.JSONField{Name: "colour_palette"})

	record := core.NewRecord(collection)
	record.Set("colour_palette", []map[string]any{
		{"hex": "#1a2b3c", "weight": 5000},
		{"hex": "", "weight": 100},
	})
	swatches := parseArtworkPalette(record)
	if len(swatches) != 1 || swatches[0].Hex != "#1a2b3c" {
		t.Errorf("palette = %+v, want single #1a2b3c swatch", swatches)
	}
}

func TestArtworkCommentaryHTML(t *testing.T) {
	if got := artworkCommentaryHTML(""); got != "" {
		t.Errorf("empty = %q, want empty", got)
	}
	if got := artworkCommentaryHTML("One\n\nTwo"); got != "<p>One</p><p>Two</p>" {
		t.Errorf("paragraphs = %q, want <p>One</p><p>Two</p>", got)
	}
	if got := artworkCommentaryHTML("<script>alert(1)</script>"); !strings.Contains(got, "&lt;script&gt;") {
		t.Errorf("escaping = %q, want escaped script", got)
	}
}

func TestPopulateArtworkSourceDataUsesSourceComment(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})

	collection := core.NewBaseCollection("Artworks")
	collection.Fields.Add(
		&core.TextField{Name: "source_path"},
		&core.TextField{Name: "source_url"},
		&core.TextField{Name: "source_comment"},
		&core.TextField{Name: "comment"},
		&core.JSONField{Name: "colour_palette"},
		&core.NumberField{Name: "date_start"},
	)
	record := core.NewRecord(collection)
	record.Set("source_comment", "Raw source.")
	record.Set("source_url", "html/v/vermeer/girl.html")
	record.Set("comment", "Divergent enriched.")

	content := dto.Artwork{}
	populateArtworkSourceData(app, record, &content, config.EnvironmentDevelopment)

	if !content.HasCommentary {
		t.Error("HasCommentary must derive from source_comment")
	}
	if content.SourceComment != "<p>Raw source.</p>" {
		t.Errorf("SourceComment = %q, want <p>Raw source.</p>", content.SourceComment)
	}
	// No reproduction source or licence claim is ever populated; the DTO no
	// longer carries such a field, so the persisted source_url cannot surface
	// a public provenance link.

	// The enriched comment field must not drive HasCommentary.
	record.Set("source_comment", "")
	record.Set("comment", "Divergent enriched.")
	content = dto.Artwork{}
	populateArtworkSourceData(app, record, &content, config.EnvironmentDevelopment)
	if content.HasCommentary {
		t.Error("HasCommentary must not derive from the enriched comment field")
	}
	if content.SourceComment != "" {
		t.Errorf("SourceComment = %q, want empty without source_comment", content.SourceComment)
	}
}

// TestPopulateArtworkSourceDataAppendsFileWeight proves the reproduction FILE
// summary gains the human-readable decimal-SI weight only when both the byte
// count and a reproduction file are recorded, and never invents either.
func TestPopulateArtworkSourceDataAppendsFileWeight(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})

	collection := core.NewBaseCollection("Artworks")
	collection.Fields.Add(
		&core.TextField{Name: "source_path"},
		&core.TextField{Name: "source_url"},
		&core.TextField{Name: "source_comment"},
		&core.TextField{Name: "comment"},
		&core.JSONField{Name: "colour_palette"},
		&core.NumberField{Name: "date_start"},
		&core.NumberField{Name: "image_size_bytes"},
	)
	record := core.NewRecord(collection)
	record.Set("image_size_bytes", 1_400_000)

	content := dto.Artwork{ReproFile: "1,861 × 2,663 px · JPEG"}
	populateArtworkSourceData(app, record, &content, config.EnvironmentDevelopment)
	if content.OriginalFileBytes != 1_400_000 {
		t.Errorf("OriginalFileBytes = %d, want exact 1400000", content.OriginalFileBytes)
	}
	if content.ReproFile != "1,861 × 2,663 px · JPEG · 1.4 MB" {
		t.Errorf("ReproFile = %q, want dimensions · format · 1.4 MB", content.ReproFile)
	}

	record.Set("image_size_bytes", 0)
	content = dto.Artwork{ReproFile: "1,861 × 2,663 px · JPEG"}
	populateArtworkSourceData(app, record, &content, config.EnvironmentDevelopment)
	if content.ReproFile != "1,861 × 2,663 px · JPEG" {
		t.Errorf("zero weight must not append a size, got %q", content.ReproFile)
	}
}

// TestPopulateArtworkSourceDataAssemblesFileEvidenceIndependently proves the
// reproduction FILE summary assembles dimensions, recognised format, and the
// positive exact byte weight independently: a missing fact must never suppress
// another factual field.
func TestPopulateArtworkSourceDataAssemblesFileEvidenceIndependently(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})

	collection := core.NewBaseCollection("Artworks")
	collection.Fields.Add(&core.NumberField{Name: "image_size_bytes"})
	record := core.NewRecord(collection)

	for _, test := range []struct {
		name    string
		bytes   int
		summary string
		want    string
	}{
		{name: "weight only", bytes: 1_400_000, summary: "", want: "1.4 MB"},
		{name: "format only", bytes: 1_400_000, summary: "JPEG", want: "JPEG · 1.4 MB"},
		{name: "dimensions only", bytes: 1_400_000, summary: "1200 × 800 px", want: "1200 × 800 px · 1.4 MB"},
		{name: "all facts", bytes: 1_400_000, summary: "1200 × 800 px · JPEG", want: "1200 × 800 px · JPEG · 1.4 MB"},
		{name: "zero bytes adds nothing", bytes: 0, summary: "1200 × 800 px · JPEG", want: "1200 × 800 px · JPEG"},
		{name: "absent all omits file", bytes: 0, summary: "", want: ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			record.Set("image_size_bytes", test.bytes)
			content := dto.Artwork{ReproFile: test.summary}
			populateArtworkSourceData(app, record, &content, config.EnvironmentDevelopment)
			if content.OriginalFileBytes != test.bytes {
				t.Errorf("OriginalFileBytes = %d, want exact %d", content.OriginalFileBytes, test.bytes)
			}
			if content.ReproFile != test.want {
				t.Errorf("ReproFile = %q, want %q", content.ReproFile, test.want)
			}
		})
	}
}

func TestFormatFileSize(t *testing.T) {
	for _, test := range []struct {
		bytes int
		want  string
	}{
		{999, "999 B"},
		{1_000, "1.0 kB"},
		{1_400_000, "1.4 MB"},
		{12_400_000, "12.4 MB"},
		{987_654, "987.7 kB"},
		{1_400_000_000, "1.4 GB"},
	} {
		if got := formatFileSize(test.bytes); got != test.want {
			t.Errorf("formatFileSize(%d) = %q, want %q", test.bytes, got, test.want)
		}
	}
}

func TestRelatedHoldingURL(t *testing.T) {
	if got := relatedHoldingURL(&repositories.RelatedWorkHolding{QueryKey: "artist", QueryValue: "Vermeer, Johannes"}); got != "/artworks?artist=Vermeer%2C+Johannes" {
		t.Errorf("artist holding URL = %q", got)
	}
	if got := relatedHoldingURL(&repositories.RelatedWorkHolding{QueryKey: "venue", QueryValue: "location0001"}); got != "/artworks?venue=location0001" {
		t.Errorf("venue holding URL = %q", got)
	}
	if got := relatedHoldingURL(&repositories.RelatedWorkHolding{QueryKey: "period", QueryValue: "period00001"}); got != "/artworks?period=period00001" {
		t.Errorf("period holding URL = %q", got)
	}
	if got := relatedHoldingURL(&repositories.RelatedWorkHolding{QueryKey: "palette", QueryValue: "x"}); got != "" {
		t.Errorf("palette holding URL = %q, want empty (ranking-only)", got)
	}
	if got := relatedHoldingURL(&repositories.RelatedWorkHolding{QueryKey: "artist", QueryValue: "a&b=c"}); got != "/artworks?artist=a%26b%3Dc" {
		t.Errorf("holding URL must URL-encode the value, got %q", got)
	}
}

func TestRelatedHoldingLabel(t *testing.T) {
	if got := relatedHoldingLabel(11); got != "FIND MORE 11 IN THE ARTWORK SEARCH →" {
		t.Errorf("relatedHoldingLabel(11) = %q", got)
	}
}

func TestResolveWorkArtistChoosesPublishedAuthor(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})

	saveRecordCollection(t, app, core.NewBaseCollection("Artists"), "artists",
		&core.TextField{Name: "name"},
		&core.TextField{Name: "filing_name"},
		&core.TextField{Name: "short_name"},
		&core.TextField{Name: "slug"},
		&core.BoolField{Name: "published"},
	)
	saveRecordRecord(t, app, "artists", "artistunpub0001", map[string]any{
		"name": "Unpublished", "filing_name": "Unpublished, Filing", "short_name": "Unpublished", "slug": "unpublished", "published": false,
	})
	saveRecordRecord(t, app, "artists", "artistpub000001", map[string]any{
		"name": "Published Route", "filing_name": "Published, Filing", "short_name": "Published", "slug": "published", "published": true,
	})

	works := core.NewBaseCollection("Artworks")
	works.Fields.Add(&core.RelationField{Name: "author", CollectionId: "artists", MinSelect: 1, MaxSelect: 10})
	work := core.NewRecord(works)
	work.Set("author", []string{"artistunpub0001", "artistpub000001"})

	routeName, filingName, shortName, id := resolveWorkArtist(app, work)
	if routeName != "Published Route" || filingName != "Published, Filing" || shortName != "Published" || id != "artistpub000001" {
		t.Errorf("resolveWorkArtist = %q/%q/%q/%q, want route/filing/short/id", routeName, filingName, shortName, id)
	}

	// When every author is unpublished, no byline or link is produced.
	workOnlyUnpublished := core.NewRecord(works)
	workOnlyUnpublished.Set("author", []string{"artistunpub0001"})
	if routeName, filingName, shortName, id := resolveWorkArtist(app, workOnlyUnpublished); routeName != "" || filingName != "" || shortName != "" || id != "" {
		t.Errorf("resolveWorkArtist(only unpublished) = %q/%q/%q/%q, want empty", routeName, filingName, shortName, id)
	}
}
