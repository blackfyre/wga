package artists

import (
	"fmt"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/constants"
	"github.com/blackfyre/wga/internal/repositories"
)

func TestTask73ArtworkFileEvidenceIsSourceBacked(t *testing.T) {
	app, request := newArtworkRouteApp(t)
	artwork, err := app.FindRecordById(constants.CollectionArtworks, "workone00000001")
	if err != nil {
		t.Fatalf("find artwork: %v", err)
	}
	artwork.Set("image", "original.JPEG")
	artwork.Set("image_width", 1200)
	artwork.Set("image_height", 800)
	artwork.Set("image_size_bytes", 987654)

	if got := artworkReproductionFile(artwork); got != "1200 × 800 px · JPEG" {
		t.Fatalf("reproduction file = %q, want recorded dimensions and filename format", got)
	}
	content := dto.Artwork{}
	populateArtworkSourceData(app, artwork, &content, config.EnvironmentDevelopment)
	if content.OriginalFileBytes != 987654 {
		t.Errorf("original bytes = %d, want exact recorded byte count", content.OriginalFileBytes)
	}
	body := request("/artists/synthetic-artist-artistone000001/a-painting-workone00000001").Body.String()
	if !strings.Contains(body, `href="/api/files/artworks/workone00000001/painting.jpg"`) {
		t.Error("record should link to the original file without thumbnail parameters")
	}

	artwork.Set("image_width", 0)
	artwork.Set("image_height", 0)
	if got := artworkReproductionFile(artwork); got != "JPEG" {
		t.Errorf("missing dimensions still exposed filename format as %q, want JPEG", got)
	}
	artwork.Set("image", "")
	artwork.Set("image_size_bytes", 0)
	content = dto.Artwork{}
	populateArtworkSourceData(app, artwork, &content, config.EnvironmentDevelopment)
	if got := artworkReproductionFile(artwork); got != "" || content.OriginalFileBytes != 0 {
		t.Errorf("missing file evidence = (%q, %d), want empty and zero", got, content.OriginalFileBytes)
	}
}

func TestTask73NoPublicSourceOrLicenceClaimsInAnyEnvironment(t *testing.T) {
	for _, environment := range []config.Environment{
		config.EnvironmentDevelopment,
		config.EnvironmentTest,
		config.EnvironmentStaging,
		config.EnvironmentProduction,
	} {
		t.Run(string(environment), func(t *testing.T) {
			app, request := newArtworkRouteAppWithEnvironment(t, environment)
			artwork, err := app.FindRecordById(constants.CollectionArtworks, "workone00000001")
			if err != nil {
				t.Fatalf("find artwork: %v", err)
			}
			artwork.Set("source_url", "html/a/artist/painting.html")
			if err := app.Save(artwork); err != nil {
				t.Fatalf("save source URL: %v", err)
			}
			body := request("/artists/synthetic-artist-artistone000001/a-painting-workone00000001").Body.String()
			recordSection := body
			if start := strings.Index(body, "<figure"); start >= 0 {
				recordSection = body[start:]
			}
			if end := strings.Index(recordSection, "</figure>"); end >= 0 {
				recordSection = recordSection[:end]
			}
			if strings.Contains(strings.ToUpper(recordSection), "LICENCE") {
				t.Error("public artwork response must not make a licence claim")
			}
			if strings.Contains(body, "VIEW ORIGINAL AT WEB GALLERY OF ART") {
				t.Errorf("environment %s must not expose a public source claim", environment)
			}
		})
	}
}

func TestTask73RelatedBasesAreCanonicalAndPaletteHasNoHolding(t *testing.T) {
	base := "/artists/synthetic-artist-artistone000001/a-painting-workone00000001"
	bases := relatedWorkBases(base, repositories.RelatedByPeriod)
	if len(bases) != 4 {
		t.Fatalf("basis count = %d, want 4", len(bases))
	}
	want := map[string]string{
		"artist":     base,
		"collection": base + "?basis=collection",
		"period":     base + "?basis=period",
		"palette":    base + "?basis=palette",
	}
	for _, basis := range bases {
		if basis.URL != want[basis.Value] {
			t.Errorf("%s URL = %q, want %q", basis.Value, basis.URL, want[basis.Value])
		}
		if basis.Value == "period" != basis.Active {
			t.Errorf("%s active = %t, want period to be active", basis.Value, basis.Active)
		}
	}

	app, _ := newArtworkRouteApp(t)
	artwork, err := app.FindRecordById(constants.CollectionArtworks, "workone00000001")
	if err != nil {
		t.Fatalf("find artwork: %v", err)
	}
	result, err := repositories.NewRelatedWorkResolver(app).Resolve(artwork, repositories.RelatedByPalette)
	if err != nil {
		t.Fatalf("resolve palette: %v", err)
	}
	if result.Holding != nil {
		t.Fatal("palette similarity must not expose a filterable holding")
	}
}

func TestTask73RelatedArtistCapsEightCandidatesToClosestFourAndCountsCurrent(t *testing.T) {
	app, _ := newArtworkRouteApp(t)
	artist, err := app.FindRecordById(constants.CollectionArtists, "artistone000001")
	if err != nil {
		t.Fatalf("find artist: %v", err)
	}
	artist.Set("filing_name", "Artist, Synthetic")
	if err := app.Save(artist); err != nil {
		t.Fatalf("save artist: %v", err)
	}
	for i := 1; i <= 10; i++ {
		id := fmt.Sprintf("task73w%08d", i)
		saveRecordRecord(t, app, constants.CollectionArtworks, id, map[string]any{
			"title": fmt.Sprintf("Related %c", 'A'+i), "author": []string{"artistone000001"}, "published": true,
			"date_start": 1600 + i,
		})
	}
	current, err := app.FindRecordById(constants.CollectionArtworks, "workone00000001")
	if err != nil {
		t.Fatalf("find current: %v", err)
	}
	current.Set("date_start", 1600)
	if err := app.Save(current); err != nil {
		t.Fatalf("save current: %v", err)
	}
	result, err := repositories.NewRelatedWorkResolver(app).Resolve(current, repositories.RelatedByArtist)
	if err != nil {
		t.Fatalf("resolve artist: %v", err)
	}
	if len(result.Works) != 4 {
		t.Fatalf("sample size = %d, want closest four from bounded candidates", len(result.Works))
	}
	if result.Holding == nil || result.Holding.QueryKey != "artist" || result.Holding.Count != 11 {
		t.Fatalf("holding = %+v, want artist count 11 including current", result.Holding)
	}
}
