package artists

import (
	"testing"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
	"github.com/blackfyre/wga/internal/config"
	apputils "github.com/blackfyre/wga/internal/utils"
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
		Title: "Girl with a Pearl Earring",
		Artist: dto.Artist{
			Name: "Johannes Vermeer",
		},
	}
	populateArtworkCitation(&artwork)

	if artwork.CitationKey != "wga-girl-with-a-pearl-earring" {
		t.Errorf("CitationKey = %q", artwork.CitationKey)
	}
	if artwork.CitationTitle != "Girl with a Pearl Earring by Johannes Vermeer" {
		t.Errorf("CitationTitle = %q", artwork.CitationTitle)
	}
	if artwork.CitationURL != "https://gallery.example/artworks/girl-with-a-pearl-earring" {
		t.Errorf("CitationURL = %q", artwork.CitationURL)
	}
}
