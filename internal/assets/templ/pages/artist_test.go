package pages

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
)

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

func TestArtistsTableRendersPortraitThumbnail(t *testing.T) {
	var output bytes.Buffer
	artist := dto.Artist{Name: "Portrait Artist", Portrait: "/api/files/artists/artist/portrait.jpg?thumb=500x0"}
	if err := artistsTable([]dto.Artist{artist}, "").Render(context.Background(), &output); err != nil {
		t.Fatalf("render artists table: %v", err)
	}

	if !strings.Contains(output.String(), `src="/api/files/artists/artist/portrait.jpg?thumb=500x0"`) {
		t.Fatal("expected portrait thumbnail")
	}
}
