package pages

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
)

func renderInspiration(t *testing.T, content dto.ImageGrid) string {
	t.Helper()
	var output bytes.Buffer
	if err := InspirationContent(content).Render(context.Background(), &output); err != nil {
		t.Fatalf("render inspiration: %v", err)
	}
	return output.String()
}

func TestInspirationRendersShuffledCollectionAndJourneyLinks(t *testing.T) {
	rendered := renderInspiration(t, dto.ImageGrid{{
		Title: "A Work", Image: "/art.jpg", Url: "/artists/artist-123/a-work-456", Artist: dto.Artist{Name: "An Artist"},
	}})
	for _, expected := range []string{
		"07 — INSPIRATION",
		"A shuffled slice of the collection",
		"There is no prescribed order here.",
		`href="/inspire"`,
		"ANOTHER SET",
		`hx-get="/inspire"`,
		`hx-target="#inspiration"`,
		`hx-select="#inspiration"`,
		`hx-swap="outerHTML"`,
		"IF YOU WANT SOMETHING ORDERED",
		`href="/tours"`,
		"editor-maintained routes",
		`href="/itineraries"`,
		"journeys made by visitors",
		`href="/artists/artist-123/a-work-456"`,
	} {
		if !strings.Contains(rendered, expected) {
			t.Errorf("rendered inspiration does not contain %q", expected)
		}
	}
	if strings.Count(rendered, "<h1") != 1 {
		t.Errorf("h1 count = %d, want 1", strings.Count(rendered, "<h1"))
	}
	if strings.Contains(rendered, "data-viewer") {
		t.Fatal("inspiration must not mount a viewer hook")
	}
}

func TestInspirationRendersHonestEmptyState(t *testing.T) {
	rendered := renderInspiration(t, nil)
	if !strings.Contains(rendered, "There are no published works to explore here yet.") {
		t.Error("missing honest empty state")
	}
	if !strings.Contains(rendered, `href="/artworks"`) {
		t.Error("missing artworks recovery link")
	}
	if strings.Contains(rendered, "grid grid-cols-1") {
		t.Error("empty state must not render artwork grid")
	}
}
