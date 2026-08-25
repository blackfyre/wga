package pages

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/assets/templ/components"
	"github.com/blackfyre/wga/internal/assets/templ/dto"
)

func renderArtistRecord(t *testing.T, view ArtistView) string {
	t.Helper()
	var output bytes.Buffer
	if err := ArtistRecordContent(view).Render(context.Background(), &output); err != nil {
		t.Fatalf("render artist record: %v", err)
	}
	return output.String()
}

func TestArtistRecordContentRendersHierarchyAndMetadata(t *testing.T) {
	rendered := renderArtistRecord(t, ArtistView{
		Name:        "Portrait Artist",
		LifeSummary: "b. 1606 Leiden, d. 1669 Amsterdam",
		Schools:     "Dutch",
		Period:      "Baroque",
		Profession:  "Painting, Graphics",
		Aliases:     "Rembrandt Harmenszoon van Rijn",
	})

	for _, expected := range []string{
		`<h1`,
		"Portrait Artist",
		"b. 1606 Leiden, d. 1669 Amsterdam",
		"Dutch",
		"Baroque",
		"Painting, Graphics",
		"Rembrandt Harmenszoon van Rijn",
		"LIFE",
		"SCHOOLS",
		"PERIOD",
		"PROFESSION",
		"ALIASES",
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("expected record to contain %q", expected)
		}
	}
	if count := strings.Count(rendered, "<h1"); count != 1 {
		t.Errorf("expected exactly one h1, got %d", count)
	}
}

func TestArtistRecordContentOmitsAbsentMetadata(t *testing.T) {
	rendered := renderArtistRecord(t, ArtistView{Name: "Solo Artist", LifeSummary: "b. 1600"})

	for _, absent := range []string{"SCHOOLS", "PERIOD", "PROFESSION", "ALIASES", "PERIOD MUSIC"} {
		if strings.Contains(rendered, absent) {
			t.Errorf("expected absent metadata %q to be omitted", absent)
		}
	}
}

func TestArtistRecordContentRendersPortrait(t *testing.T) {
	rendered := renderArtistRecord(t, ArtistView{
		Name:     "Portrait Artist",
		Portrait: "/api/files/artists/artist/portrait.jpg",
	})

	for _, expected := range []string{
		`src="/api/files/artists/artist/portrait.jpg"`,
		`alt="Portrait Artist"`,
		"object-cover",
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("expected rendered record to contain %q", expected)
		}
	}
	if strings.Contains(rendered, "PORTRAIT —") {
		t.Error("expected portrait fallback to be absent")
	}
}

func TestArtistRecordContentRendersPortraitFallback(t *testing.T) {
	rendered := renderArtistRecord(t, ArtistView{Name: "Portrait Artist"})

	if !strings.Contains(rendered, "PORTRAIT — Portrait Artist") {
		t.Error("expected labelled portrait fallback")
	}
	if strings.Contains(rendered, `<img`) {
		t.Error("expected no portrait image without a portrait")
	}
}

func TestArtistRecordContentNeverBuildsThumbnailQuery(t *testing.T) {
	rendered := renderArtistRecord(t, ArtistView{
		Name:     "Portrait Artist",
		Portrait: "/api/files/artists/artist/portrait.jpg",
	})

	if strings.Contains(rendered, "thumb=") {
		t.Error("template must not construct a thumbnail query parameter")
	}
}

func TestArtistRecordContentRendersGlossaryBiographyAndEscapesText(t *testing.T) {
	rendered := renderArtistRecord(t, ArtistView{
		Name: "<script>alert(1)</script>",
		Bio:  `<p>He used <dfn class="wga-term" role="note" tabindex="0" aria-label="chiaroscuro: A treatment of light and shade." data-bionic="off">chiaroscuro<span class="wga-term__tooltip" aria-hidden="true"><span class="wga-tooltip__meta">GLOSSARY</span><span class="wga-tooltip__body">A treatment of light and shade.</span></span></dfn>.</p>`,
	})

	if strings.Contains(rendered, `<script>alert(1)</script>`) {
		t.Error("dynamic name text must be escaped")
	}
	if !strings.Contains(rendered, "&lt;script&gt;") {
		t.Error("expected escaped script name")
	}
	for _, expected := range []string{
		`class="wga-term"`,
		`role="note"`,
		`tabindex="0"`,
		`aria-label="chiaroscuro: A treatment of light and shade."`,
		`chiaroscuro`,
		`<p>He used`,
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("expected glossary biography to contain %q", expected)
		}
	}
}

func TestArtistRecordContentRendersWorksAndWiderRoute(t *testing.T) {
	rendered := renderArtistRecord(t, ArtistView{
		Name:        "Portrait Artist",
		WorkCount:   5,
		WorksURL:    "/artworks?artist=Portrait+Artist",
		Works:       dto.ImageGrid{{Id: "artwork12345678", Title: "A Painting", Url: "/artists/portrait-artist-artist/a-painting-artwork12345678", Image: "/api/files/artworks/artwork12345678/p.jpg", Artist: dto.Artist{Name: "Portrait Artist"}}},
	})

	for _, expected := range []string{
		"WORKS IN ARCHIVE",
		"1 OF 5 RECORDS",
		"A Painting",
		`href="/artists/portrait-artist-artist/a-painting-artwork12345678"`,
		`href="/artworks?artist=Portrait+Artist"`,
		`hx-get="/artworks?artist=Portrait+Artist"`,
		"VIEW ALL WORKS",
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("expected works section to contain %q", expected)
		}
	}
}

func TestArtistRecordContentRendersHonestEmptyState(t *testing.T) {
	rendered := renderArtistRecord(t, ArtistView{Name: "Portrait Artist", WorkCount: 0})

	if !strings.Contains(rendered, "No published works are in the archive for this artist yet.") {
		t.Error("expected honest empty works state")
	}
	if strings.Contains(rendered, "VIEW ALL WORKS") {
		t.Error("expected no wider-catalogue link for an empty catalogue")
	}
}

func TestArtistRecordContentRendersPeriodMusicCard(t *testing.T) {
	rendered := renderArtistRecord(t, ArtistView{
		Name: "Portrait Artist",
		Music: components.MusicPeriodCard{
			SongID:    "song1234567890a",
			Piece:     "Fantasia chromatica",
			PlayerURL: "/player?song=song1234567890a",
		},
	})

	for _, expected := range []string{
		"PERIOD MUSIC",
		"Fantasia chromatica",
		`target="wga-period-music"`,
		`href="/player?song=song1234567890a"`,
		`data-wga-music="song1234567890a"`,
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("expected music card to contain %q", expected)
		}
	}
}

func TestArtistRecordContentOmitsPeriodMusicWithoutMatch(t *testing.T) {
	rendered := renderArtistRecord(t, ArtistView{Name: "Portrait Artist"})
	if strings.Contains(rendered, "PERIOD MUSIC") {
		t.Error("expected no period-music card without a match")
	}
}

func TestArtistRecordContentRendersCitation(t *testing.T) {
	rendered := renderArtistRecord(t, ArtistView{
		Name: "Portrait Artist",
		Citation: components.Citation{
			Key:   "wga-portrait-artist",
			Title: "Portrait Artist",
			URL:   "https://gallery.example/artists/portrait-artist-artist",
		},
	})

	for _, expected := range []string{
		"CITE THIS RECORD — BIBTEX",
		"wga-portrait-artist",
		"Portrait Artist",
		"https://gallery.example/artists/portrait-artist-artist",
		"<pre",
		"COPY BIBTEX",
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("expected citation to contain %q", expected)
		}
	}
}

func TestArtistRecordContentUsesHtmxEnhancedLinks(t *testing.T) {
	rendered := renderArtistRecord(t, ArtistView{
		Name:            "Portrait Artist",
		Url:             "/artists/portrait-artist-artist",
		Schools:         "Dutch",
		WorkCount:       1,
		WorksURL:        "/artworks?artist=Portrait+Artist",
		ShowBreadcrumbs: true,
		Works:           dto.ImageGrid{{Id: "artwork12345678", Title: "A Painting", Url: "/artists/portrait-artist-artist/a-painting-artwork12345678", Image: "/api/files/artworks/artwork12345678/p.jpg", Artist: dto.Artist{Name: "Portrait Artist"}}},
	})

	if !strings.Contains(rendered, `href="/artists"`) {
		t.Error("expected ARTISTS breadcrumb link")
	}
	if !strings.Contains(rendered, `hx-get="/artists"`) {
		t.Error("breadcrumb must carry hx-get so navigation is HTMX-enhanced, with the plain href as its no-JavaScript fallback")
	}
	if strings.Contains(rendered, "data-viewer") {
		t.Error("work preview must not carry viewer hooks")
	}
}

func TestArtistRecordContentRendersSelectionPreviews(t *testing.T) {
	rendered := renderArtistRecord(t, ArtistView{
		Name:      "Portrait Artist",
		WorkCount: 5,
		Selections: []SelectionPreview{
			{
				URL:             "/artists/portrait-artist-artistone000001/selections/rselect00000001",
				DisplayTitle:    "Portrait Artist: Paintings",
				SelectedCount:   2,
				CataloguedCount: 5,
				Commentary:      "<p>A supplied lede.</p>",
				HasCommentary:   true,
				Works: dto.ImageGrid{{
					Id: "artwork12345678", Title: "A Painting",
					Url: "/artists/portrait-artist-artistone000001/a-painting-artwork12345678",
					Image: "/api/files/artworks/artwork12345678/p.jpg",
					Artist: dto.Artist{Name: "Portrait Artist"},
				}},
			},
		},
	})

	for _, expected := range []string{
		"CURATED SELECTIONS",
		"Portrait Artist: Paintings",
		"2 SELECTED · 5 CATALOGUED",
		"A supplied lede.",
		"OPEN SELECTION",
		`href="/artists/portrait-artist-artistone000001/selections/rselect00000001"`,
		"A Painting",
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("expected selection preview to contain %q", expected)
		}
	}
	if !strings.Contains(rendered, `hx-get="/artists/portrait-artist-artistone000001/selections/rselect00000001"`) {
		t.Error("expected OPEN SELECTION link to carry hx-get")
	}
}

func TestArtistRecordContentRendersHonestMissingPreviewCommentary(t *testing.T) {
	rendered := renderArtistRecord(t, ArtistView{
		Name: "Portrait Artist",
		Selections: []SelectionPreview{
			{DisplayTitle: "Paintings", SelectedCount: 2, CataloguedCount: 5, HasCommentary: false},
		},
	})

	if !strings.Contains(rendered, "Commentary is unavailable for this selection.") {
		t.Error("expected honest missing-commentary state in the preview")
	}
}

func TestArtistRecordContentOmitsSelectionsWhenNone(t *testing.T) {
	rendered := renderArtistRecord(t, ArtistView{Name: "Portrait Artist", WorkCount: 2})

	if strings.Contains(rendered, "CURATED SELECTIONS") {
		t.Error("expected no selections section without previews")
	}
}

func TestArtistRecordBlockWrapsMainContentArea(t *testing.T) {
	var output bytes.Buffer
	if err := ArtistRecordBlock(ArtistView{Name: "Portrait Artist"}).Render(context.Background(), &output); err != nil {
		t.Fatalf("render artist record block: %v", err)
	}

	rendered := output.String()
	if !strings.Contains(rendered, `<main id="mc-area"`) {
		t.Error("HTMX block should wrap the main content area for hx-select/hx-swap")
	}
	if strings.Contains(rendered, "<!DOCTYPE html>") || strings.Contains(rendered, "<html") {
		t.Error("HTMX block should not render the full document")
	}
}

func TestArtistBlockRendersPortrait(t *testing.T) {
	var output bytes.Buffer
	artist := dto.Artist{Name: "Portrait Artist", Portrait: "/api/files/artists/artist/portrait.jpg"}
	if err := ArtistBlock(artist).Render(context.Background(), &output); err != nil {
		t.Fatalf("render artist block: %v", err)
	}

	rendered := output.String()
	for _, expected := range []string{
		`src="/api/files/artists/artist/portrait.jpg"`,
		`alt="Portrait Artist"`,
		"object-cover",
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("expected rendered artist block to contain %q", expected)
		}
	}
	if strings.Contains(rendered, "PORTRAIT —") {
		t.Error("expected portrait fallback to be absent")
	}
}

func TestArtistBlockRendersPortraitFallback(t *testing.T) {
	var output bytes.Buffer
	artist := dto.Artist{Name: "Portrait Artist"}
	if err := ArtistBlock(artist).Render(context.Background(), &output); err != nil {
		t.Fatalf("render artist block: %v", err)
	}

	rendered := output.String()
	if !strings.Contains(rendered, "PORTRAIT — Portrait Artist") {
		t.Error("expected portrait fallback")
	}
	if strings.Contains(rendered, `<img`) {
		t.Error("expected no portrait image")
	}
}
