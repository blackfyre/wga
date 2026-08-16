package components

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
)

func TestImageBigUsesDedicatedZoomURL(t *testing.T) {
	var output bytes.Buffer
	if err := ImageBig("/api/files/artworks/work.jpg?thumb=1400x0", "/api/files/artworks/work.jpg?thumb=2000x0", "Work", "Artist").Render(context.Background(), &output); err != nil {
		t.Fatalf("render image: %v", err)
	}

	rendered := output.String()
	if !strings.Contains(rendered, `src="/api/files/artworks/work.jpg?thumb=1400x0"`) {
		t.Fatal("expected display URL")
	}
	if !strings.Contains(rendered, `data-zoom-url="/api/files/artworks/work.jpg?thumb=2000x0"`) {
		t.Fatal("expected dedicated zoom URL")
	}
}

func TestPublicArtworkGridDoesNotMountViewer(t *testing.T) {
	var output bytes.Buffer
	if err := PublicArtworkGrid(dto.ImageGrid{{Title: "Work", Image: "/api/files/artworks/work.jpg?thumb=500x0"}}).Render(context.Background(), &output); err != nil {
		t.Fatalf("render artwork grid: %v", err)
	}

	if strings.Contains(output.String(), "data-viewer") {
		t.Fatal("artwork grids must not mount ViewerJS")
	}
}
