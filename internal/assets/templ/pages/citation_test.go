package pages

import (
	"strings"
	"testing"
	"time"

	"github.com/blackfyre/wga/internal/assets/templ/dto"
)

func TestArtistBibTeXUsesCanonicalURL(t *testing.T) {
	citation := ArtistBibTeX(dto.Artist{
		Id:          "artist-id",
		Name:        "Artist Name",
		CitationUrl: "https://www.wga.hu/artists/artist-name-artist-id",
	})

	date := time.Now().UTC().Format(time.DateOnly)
	for _, expected := range []string{
		"@online{wga-artist-artist-id,",
		"author       = {{Web Gallery of Art}}",
		"organization = {{Web Gallery of Art}}",
		"date         = {" + date + "}",
		"url          = {https://www.wga.hu/artists/artist-name-artist-id}",
		"urldate      = {" + date + "}",
		"langid       = {english}",
	} {
		if !strings.Contains(citation, expected) {
			t.Errorf("expected citation to contain %q\ngot: %s", expected, citation)
		}
	}
}

func TestArtworkBibTeXUsesCanonicalURL(t *testing.T) {
	citation := ArtworkBibTeX(dto.Artwork{
		Id:          "artwork-id",
		Title:       "Artwork Title",
		CitationUrl: "https://www.wga.hu/artists/artist-name-artist-id/artwork-title-artwork-id",
		Artist:      dto.Artist{Name: "Artist Name"},
	})

	date := time.Now().UTC().Format(time.DateOnly)
	for _, expected := range []string{
		"@online{wga-artwork-artwork-id,",
		"author       = {Artist Name}",
		"title        = {Artwork Title by Artist Name}",
		"organization = {{Web Gallery of Art}}",
		"date         = {" + date + "}",
		"url          = {https://www.wga.hu/artists/artist-name-artist-id/artwork-title-artwork-id}",
		"urldate      = {" + date + "}",
		"langid       = {english}",
	} {
		if !strings.Contains(citation, expected) {
			t.Errorf("expected citation to contain %q\ngot: %s", expected, citation)
		}
	}
}
