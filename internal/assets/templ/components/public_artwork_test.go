package components

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
)

func renderPublicArtworkCard(t *testing.T, image dto.Image) string {
	t.Helper()
	var output bytes.Buffer
	if err := PublicArtworkCard(image).Render(context.Background(), &output); err != nil {
		t.Fatalf("render public artwork card: %v", err)
	}
	return output.String()
}

func TestPublicArtworkCardRendersStoredLocation(t *testing.T) {
	rendered := renderPublicArtworkCard(t, dto.Image{
		Id:       "artwork12345678",
		Title:    "A Painting",
		Url:      "/artists/artist-artistone000001/a-painting-artwork12345678",
		Location: "Kunsthistorisches Museum, Vienna & Annex",
		Artist:   dto.Artist{Name: "Portrait Artist"},
	})

	if !strings.Contains(rendered, `data-plate-location>Kunsthistorisches Museum, Vienna &amp; Annex</p>`) {
		t.Errorf("expected the escaped stored location line, got %s", rendered)
	}
	title := strings.Index(rendered, "A Painting</h2>")
	artist := strings.Index(rendered, "Portrait Artist</p>")
	location := strings.Index(rendered, "data-plate-location")
	if title < 0 || artist < title || location < artist {
		t.Error("the location line must follow the title and the artist")
	}
}

func TestPublicArtworkCardOmitsAbsentLocation(t *testing.T) {
	rendered := renderPublicArtworkCard(t, dto.Image{
		Id:     "artwork12345678",
		Title:  "A Painting",
		Url:    "/artists/artist-artistone000001/a-painting-artwork12345678",
		Artist: dto.Artist{Name: "Portrait Artist"},
	})

	if strings.Contains(rendered, "data-plate-location") {
		t.Error("a plate without a stored location must render no location line")
	}
}

func TestPublicArtworkCardPinsActionsToPlateBottom(t *testing.T) {
	rendered := renderPublicArtworkCard(t, dto.Image{
		Id:    "artwork12345678",
		Title: "A Painting",
		Url:   "/artists/artist-artistone000001/a-painting-artwork12345678",
	})

	if !strings.Contains(rendered, `<article class="group row-span-3 grid grid-rows-subgrid gap-y-0`) {
		t.Error("the plate must adopt three grid rows so plates in a row share row heights")
	}
	// The record link, VIEW WORK, and the itinerary action must be the plate's
	// three direct children, one per subgrid row.
	link := strings.Index(rendered, `class="wga-record-card"`)
	viewWork := strings.Index(rendered, `data-plate-view-work>VIEW WORK →</a>`)
	itinerary := strings.Index(rendered, `<div class="mt-3" data-plate-itinerary>`)
	if link < 0 || viewWork < link || itinerary < viewWork {
		t.Error("the record link, VIEW WORK, and itinerary action must follow each other as subgrid rows")
	}
	if between := rendered[link:viewWork]; strings.Count(between, "</a>") != 1 {
		t.Error("VIEW WORK must be a direct child of the plate, not nested in a wrapper")
	}
}
