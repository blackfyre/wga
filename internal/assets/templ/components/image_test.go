package components

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
)

func TestPublicArtworkGridDoesNotMountViewer(t *testing.T) {
	var output bytes.Buffer
	if err := PublicArtworkGrid(dto.ImageGrid{{Title: "Work", Image: "/api/files/artworks/work.jpg?thumb=500x0"}}).Render(context.Background(), &output); err != nil {
		t.Fatalf("render artwork grid: %v", err)
	}

	if strings.Contains(output.String(), "data-viewer") {
		t.Fatal("artwork grids must not mount ViewerJS")
	}
}
