package jsonld

import (
	"fmt"

	"github.com/blackfyre/wga/utils"
	"github.com/pocketbase/pocketbase/core"
)

// ArtistJsonLd generates a JSON-LD representation of an artist.
// It takes an instance of wgaModels.Artist and an echo.Context as input.
// It returns a Person struct representing the artist in JSON-LD format.
func ArtistJsonLd(r *core.Record) Person {
	return newPerson(Person{
		Name:      r.GetString("name"),
		Url:       utils.AssetUrl("/artists/" + r.GetString("slug") + "-" + r.GetString("id")),
		BirthDate: fmt.Sprint(r.GetString("year_of_birth")),
		DeathDate: fmt.Sprint(r.GetString("year_of_death")),
		PlaceOfBirth: newPlace(Place{
			Name: r.GetString("place_of_birth"),
		}),
		PlaceOfDeath: newPlace(Place{
			Name: r.GetString("place_of_death"),
		}),
		HasOccupation: newOccupation(Occupation{
			Name: r.GetString("profession"),
		}),
		Description: utils.StrippedHTML(r.GetString("bio")),
	})
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
