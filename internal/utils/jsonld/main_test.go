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

func TestArtistJsonLdResolvesRecordProfilePortrait(t *testing.T) {
	artist := testArtistRecord("portrait.jpg")
	artist.Set("biography_image_width", 601)
	if got, want := ArtistJsonLd(artist).Image, utils.AssetUrl(urlutils.GenerateDeliveryURL(constants.CollectionArtists, artist.Id, "portrait.jpg", 601, urlutils.DeliveryProfilePortraitRecordAndWorkFallback, "")); got != want {
		t.Fatalf("JSON-LD image = %q, want %q", got, want)
	}
}

func TestArtistJsonLdOmitsMissingPortrait(t *testing.T) {
	artist := testArtistRecord("")
	if got := ArtistJsonLd(artist).Image; got != "" {
		t.Fatalf("JSON-LD image = %q, want empty", got)
	}
}

func TestArtistJsonLdOmitsUnknownDatesAndEmptyFields(t *testing.T) {
	artist := testArtistRecord("")
	artist.Set("year_of_birth", 0)
	artist.Set("year_of_death", 0)

	person := ArtistJsonLd(artist)
	if person.BirthDate != "" {
		t.Errorf("BirthDate = %q, want empty for unknown birth year", person.BirthDate)
	}
	if person.DeathDate != "" {
		t.Errorf("DeathDate = %q, want empty for unknown death year", person.DeathDate)
	}
	if person.PlaceOfBirth.Name != "" {
		t.Errorf("PlaceOfBirth = %q, want empty", person.PlaceOfBirth.Name)
	}
	if person.HasOccupation.Name != "" {
		t.Errorf("HasOccupation = %q, want empty", person.HasOccupation.Name)
	}
}

func TestArtistJsonLdKeepsTruthfulDates(t *testing.T) {
	artist := testArtistRecord("")
	artist.Set("year_of_birth", 1606)
	artist.Set("year_of_death", 1669)

	person := ArtistJsonLd(artist)
	if person.BirthDate != "1606" {
		t.Errorf("BirthDate = %q, want 1606", person.BirthDate)
	}
	if person.DeathDate != "1669" {
		t.Errorf("DeathDate = %q, want 1669", person.DeathDate)
	}
}

func testArtistRecord(portrait string) *core.Record {
	artists := core.NewBaseCollection("Artists")
	artists.Id = constants.CollectionArtists
	artists.Fields.Add(
		&core.TextField{Name: "name"},
		&core.TextField{Name: "slug"},
		&core.TextField{Name: "portrait"},
		&core.NumberField{Name: "year_of_birth"},
		&core.NumberField{Name: "year_of_death"},
		&core.TextField{Name: "place_of_birth"},
		&core.TextField{Name: "place_of_death"},
		&core.TextField{Name: "profession"},
	)

	artist := core.NewRecord(artists)
	artist.Id = "artist"
	artist.Set("name", "Portrait Artist")
	artist.Set("slug", "portrait-artist")
	artist.Set("portrait", portrait)
	return artist
}
