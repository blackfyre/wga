package jsonld

import (
	"strconv"

	"github.com/blackfyre/wga/internal/utils"
	"github.com/blackfyre/wga/internal/utils/url"
	"github.com/pocketbase/pocketbase/core"
)

// ArtistJsonLd generates a JSON-LD representation of an artist. Missing values
// are omitted rather than invented, and the portrait image resolves against the
// artist-record delivery profile so it matches the portrait rendered on the
// record page.
func ArtistJsonLd(r *core.Record) Person {
	person := Person{
		Name:        r.GetString("name"),
		Url:         utils.AssetUrl("/artists/" + r.GetString("slug") + "-" + r.GetString("id")),
		Description: utils.StrippedHTML(r.GetString("bio")),
	}

	if year := r.GetInt("year_of_birth"); year > 0 {
		person.BirthDate = strconv.Itoa(year)
	}
	if year := r.GetInt("year_of_death"); year > 0 {
		person.DeathDate = strconv.Itoa(year)
	}
	if place := r.GetString("place_of_birth"); place != "" {
		person.PlaceOfBirth = newPlace(Place{Name: place})
	}
	if place := r.GetString("place_of_death"); place != "" {
		person.PlaceOfDeath = newPlace(Place{Name: place})
	}
	if occupation := r.GetString("profession"); occupation != "" {
		person.HasOccupation = newOccupation(Occupation{Name: occupation})
	}
	if portraitURL := url.GenerateArtistPortraitImageURL(r, url.DeliveryProfilePortraitRecordAndWorkFallback, ""); portraitURL != "" {
		person.Image = utils.AssetUrl(portraitURL)
	}

	return newPerson(person)
}

// generateVisualArtworkJsonLdContent generates a map containing JSON-LD content for a visual artwork record.
// It takes a models.Record pointer and an echo.Context as input and returns a map[string]any.
func GenerateVisualArtworkJsonLdContent(r *core.Record) map[string]any {

	d := map[string]any{
		"@context":    "https://schema.org",
		"@type":       "VisualArtwork",
		"name":        r.GetString("name"),
		"description": utils.StrippedHTML(r.GetString("comment")),
		"artform":     r.GetString("technique"),
	}

	return d
}

func ArtworkJsonLd(artWork *core.Record, artist *core.Record) VisualArtwork {
	return VisualArtwork{
		Name:        artWork.GetString("name"),
		Description: utils.StrippedHTML(artWork.GetString("comment")),
		Artform:     artWork.GetString("technique"),
		Url:         utils.AssetUrl("/artworks/" + artWork.GetString("slug") + "-" + artWork.GetString("id")),
		Artist:      ArtistJsonLd(artist),
		ArtMedium:   artWork.GetString("medium"),
		Image: ImageObject{
			Image: utils.AssetUrl("/images/" + artWork.GetString("image")),
		},
	}

}
