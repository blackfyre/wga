package artists

import (
	"testing"
	"time"

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

func TestArtworkBibTeX(t *testing.T) {
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

	bibTeX := artworkBibTeX(dto.Artwork{
		Title: "Girl with a Pearl Earring",
		Artist: dto.Artist{
			Name: "Johannes Vermeer",
		},
	}, time.Date(2026, time.July, 31, 0, 0, 0, 0, time.UTC))
	want := `@online{wga-girl-with-a-pearl-earring,
  author       = {Krén, Emil and Marx, Daniel},
  title        = {Girl with a Pearl Earring by Johannes Vermeer},
  organization = {{Web Gallery of Art}},
  date         = {2026-07-31},
  url          = {https://gallery.example/artworks/girl-with-a-pearl-earring},
  urldate      = {2026-07-31},
  langid       = {english}
}`

	if bibTeX != want {
		t.Errorf("artworkBibTeX() = %q, want %q", bibTeX, want)
	}
}
