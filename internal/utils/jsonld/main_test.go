package jsonld

import (
	"testing"

	"github.com/blackfyre/wga/internal/constants"
	"github.com/blackfyre/wga/internal/utils"
	urlutils "github.com/blackfyre/wga/internal/utils/url"
	"github.com/pocketbase/pocketbase/core"
)

func TestArtistJsonLdIncludesPortrait(t *testing.T) {
	artist := testArtistRecord("portrait.jpg")
	if got, want := ArtistJsonLd(artist).Image, utils.AssetUrl(urlutils.GenerateFileUrl(constants.CollectionArtists, artist.Id, "portrait.jpg", "")); got != want {
		t.Fatalf("JSON-LD image = %q, want %q", got, want)
	}
}

func TestArtistJsonLdOmitsMissingPortrait(t *testing.T) {
	artist := testArtistRecord("")
	if got := ArtistJsonLd(artist).Image; got != "" {
		t.Fatalf("JSON-LD image = %q, want empty", got)
	}
}

func testArtistRecord(portrait string) *core.Record {
	artists := core.NewBaseCollection("Artists")
	artists.Id = constants.CollectionArtists
	artists.Fields.Add(
		&core.TextField{Name: "name"},
		&core.TextField{Name: "slug"},
		&core.TextField{Name: "portrait"},
	)

	artist := core.NewRecord(artists)
	artist.Id = "artist"
	artist.Set("name", "Portrait Artist")
	artist.Set("slug", "portrait-artist")
	artist.Set("portrait", portrait)
	return artist
}
