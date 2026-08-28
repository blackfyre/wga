package pages

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/assets/templ/components"
	"github.com/blackfyre/wga/internal/assets/templ/dto"
)

func renderSelection(t *testing.T, view SelectionView) string {
	t.Helper()
	var output bytes.Buffer
	if err := SelectionContent(view).Render(context.Background(), &output); err != nil {
		t.Fatalf("render selection: %v", err)
	}
	return output.String()
}

func TestSelectionContentRendersTitleCommentaryAndWorks(t *testing.T) {
	rendered := renderSelection(t, SelectionView{
		ArtistFilingName: "Dürer, Albrecht",
		ArtistShortName:  "Dürer",
		ArtistURL:        "/artists/durer-artistone000001",
		DisplayTitle:     "Dürer: Paintings",
		Context:          "Dürer",
		Commentary:       "<p>An editorial lede.</p>",
		HasCommentary:    true,
		WorkCount:        1,
		ShowBreadcrumbs:  true,
		Works: dto.ImageGrid{{
			Id: "workone00000001", Title: "A Painting",
			Url:    "/artists/durer-artistone000001/a-painting-workone00000001",
			Image:  "/api/files/artworks/workone00000001/painting.jpg",
			Artist: dto.Artist{Name: "Dürer"},
		}},
		HoldingURL: "/artworks?artist=D%C3%BCrer",
	})

	for _, expected := range []string{
		"Dürer: Paintings",
		"An editorial lede.",
		"21 — SELECTION",
		"SELECTED WORKS",
		"A Painting",
		"VIEW FULL HOLDING",
		`href="/artists/durer-artistone000001"`,
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("expected selection to contain %q", expected)
		}
	}
	if count := strings.Count(rendered, "<h1"); count != 1 {
		t.Errorf("expected exactly one h1, got %d", count)
	}
}

func TestSelectionContentRendersSectionTwentyOne(t *testing.T) {
	rendered := renderSelection(t, SelectionView{DisplayTitle: "Dürer: Paintings"})

	if !strings.Contains(rendered, "21 — SELECTION") {
		t.Error("selection reading page must identify section 21")
	}
	if strings.Contains(rendered, "03 — SELECTION") {
		t.Error("selection reading page must not use the obsolete section 03 label")
	}
}

func TestSelectionContentRendersResponsiveWorkCardGrid(t *testing.T) {
	rendered := renderSelection(t, SelectionView{
		DisplayTitle: "Dürer: Paintings",
		WorkCount:    2,
		Works: dto.ImageGrid{
			{
				Id: "workone00000001", Title: "A Painting",
				Url:    "/artists/durer-artistone000001/a-painting-workone00000001",
				Image:  "/api/files/artworks/workone00000001/painting.jpg",
				Artist: dto.Artist{Name: "Dürer"},
			},
			{
				Id: "workone00000002", Title: "B Painting",
				Url:    "/artists/durer-artistone000001/b-painting-workone00000002",
				Image:  "/api/files/artworks/workone00000002/painting.jpg",
				Artist: dto.Artist{Name: "Dürer"},
			},
		},
	})

	if !strings.Contains(rendered, `class="grid grid-cols-2 md:grid-cols-4 gap-x-4 md:gap-x-6 gap-y-8"`) {
		t.Error("selection works must render in the reference two-column/four-column work-card grid")
	}
	for _, forbidden := range []string{"grid-cols-1", "sm:grid-cols-2", "md:grid-cols-2", "lg:grid-cols-3", "xl:grid-cols-4"} {
		if strings.Contains(rendered, forbidden) {
			t.Errorf("selection work grid must not expose shared-grid breakpoint %q", forbidden)
		}
	}
}

func TestSelectionContentRendersHonestMissingCommentary(t *testing.T) {
	rendered := renderSelection(t, SelectionView{
		DisplayTitle:  "Dürer: Paintings",
		HasCommentary: false,
	})

	if !strings.Contains(rendered, "Commentary is unavailable for this selection.") {
		t.Error("expected honest missing-commentary state")
	}
	if strings.Contains(rendered, "generated") {
		t.Error("must not substitute generated prose")
	}
}

func TestSelectionContentOmitsOtherSelectionsWhenNone(t *testing.T) {
	rendered := renderSelection(t, SelectionView{DisplayTitle: "Paintings"})

	if strings.Contains(rendered, "OTHER SELECTIONS") {
		t.Error("expected no other-selections block when none exist")
	}
}

func TestSelectionContentRendersOtherSelectionsAndHolding(t *testing.T) {
	rendered := renderSelection(t, SelectionView{
		DisplayTitle: "Dürer: Paintings",
		OtherSelections: []SelectionLink{
			{URL: "/artists/durer-artistone000001/selections/rselect00000002", Title: "Dürer: Studies"},
		},
		HoldingURL: "/artworks?artist=D%C3%BCrer",
	})

	for _, expected := range []string{
		"OTHER SELECTIONS",
		"Dürer: Studies",
		`href="/artists/durer-artistone000001/selections/rselect00000002"`,
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("expected other selections to contain %q", expected)
		}
	}
}

func TestSelectionContentRendersHoldingWithoutWorks(t *testing.T) {
	rendered := renderSelection(t, SelectionView{
		DisplayTitle: "Dürer: Paintings",
		HoldingURL:   "/artworks?artist=D%C3%BCrer",
	})

	if !strings.Contains(rendered, "VIEW FULL HOLDING") {
		t.Error("selection must expose the wider-holding link even with zero renderable works")
	}
	if !strings.Contains(rendered, `href="/artworks?artist=D%C3%BCrer"`) {
		t.Error("holding link must use the supplied HoldingURL")
	}
}

func TestSelectionContentEscapesDisplayTitle(t *testing.T) {
	rendered := renderSelection(t, SelectionView{DisplayTitle: "<script>alert(1)</script>"})

	if strings.Contains(rendered, "<script>alert(1)</script>") {
		t.Error("display title must be escaped")
	}
	if !strings.Contains(rendered, "&lt;script&gt;") {
		t.Error("expected escaped display title")
	}
}

func TestSelectionContentNeverBuildsThumbnailQuery(t *testing.T) {
	rendered := renderSelection(t, SelectionView{
		DisplayTitle: "Paintings",
		WorkCount:    1,
		Works: dto.ImageGrid{{
			Id: "workone00000001", Title: "A Painting", Image: "/api/files/artworks/workone00000001/painting.jpg",
		}},
	})

	if strings.Contains(rendered, "thumb=") {
		t.Error("template must not construct a thumbnail query parameter")
	}
}

func TestSelectionContentRendersCitation(t *testing.T) {
	rendered := renderSelection(t, SelectionView{
		DisplayTitle: "Dürer: Paintings",
		Citation: components.Citation{
			Key:   "wga-r2633f9d80c78f0",
			Title: "Dürer: Paintings (selection)",
			URL:   "https://gallery.example/artists/durer-artistone000001/selections/r2633f9d80c78f0",
		},
	})

	for _, expected := range []string{
		"CITE THIS RECORD — BIBTEX",
		"wga-r2633f9d80c78f0",
		"Dürer: Paintings (selection)",
		"https://gallery.example/artists/durer-artistone000001/selections/r2633f9d80c78f0",
		"<pre",
		"COPY BIBTEX",
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("expected selection citation to contain %q", expected)
		}
	}
}

func TestSelectionContentUsesOrdinaryLinks(t *testing.T) {
	rendered := renderSelection(t, SelectionView{
		DisplayTitle: "Dürer: Paintings",
		WorkCount:    1,
		OtherSelections: []SelectionLink{
			{URL: "/artists/durer-artistone000001/selections/rselect00000002", Title: "Dürer: Studies"},
		},
		HoldingURL: "/artworks?artist=D%C3%BCrer",
		Works: dto.ImageGrid{{
			Id: "workone00000001", Title: "A Painting",
			Url:    "/artists/durer-artistone000001/a-painting-workone00000001",
			Image:  "/api/files/artworks/workone00000001/painting.jpg",
			Artist: dto.Artist{Name: "Dürer"},
		}},
	})

	if strings.Contains(rendered, "hx-get") {
		t.Error("selection navigation links must be ordinary links, not hx-get")
	}
	if strings.Contains(rendered, "data-viewer") {
		t.Error("selection works must not carry artwork viewer hooks")
	}
	for _, expected := range []string{
		`href="/artists/durer-artistone000001/selections/rselect00000002"`,
		`href="/artworks?artist=D%C3%BCrer"`,
		`href="/artists/durer-artistone000001/a-painting-workone00000001"`,
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("expected ordinary link %q", expected)
		}
	}
}
