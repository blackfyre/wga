package jsonld

import (
	"fmt"

	wgaModels "github.com/blackfyre/wga/models"
	"github.com/blackfyre/wga/utils"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
)

// generateArtistJsonLdContent generates a JSON-LD content for an artist record.
// It takes a pointer to a models.Record and an echo.Context as input and returns a map[string]any.
// The returned map contains the JSON-LD content for the artist record, including the artist's name, URL, profession,
// birth and death dates, and birth and death places (if available).
// Deprecated: Use ArtistJsonLd instead.
func GenerateArtistJsonLdContent(r *core.Record, c echo.Context) map[string]any {

	fullUrl := c.Scheme() + "://" + c.Request().Host + "/artists/" + r.Slug + "-" + r.Id

	d := map[string]any{
		"@context":      "https://schema.org",
		"@type":         "Person",
		"name":          r.Name,
		"url":           fullUrl,
		"hasOccupation": r.Profession,
	}

	if r.YearOfBirth > 0 {
		d["birthDate"] = r.YearOfBirth
	}

	if r.YearOfDeath > 0 {
		d["deathDate"] = r.YearOfDeath
	}

	if r.PlaceOfBirth != "" {
		d["birthPlace"] = map[string]string{
			"@type": "Place",
			"name":  r.PlaceOfBirth,
		}
	}

	if r.PlaceOfDeath != "" {
		d["deathPlace"] = map[string]string{
			"@type": "Place",
			"name":  r.PlaceOfDeath,
		}
	}

	return d
}

// ArtistJsonLd generates a JSON-LD representation of an artist.
// It takes an instance of wgaModels.Artist and an echo.Context as input.
// It returns a Person struct representing the artist in JSON-LD format.
func ArtistJsonLd(r *wgaModels.Artist, c echo.Context) Person {
	return newPerson(Person{
		Name:      r.Name,
		Url:       c.Scheme() + "://" + c.Request().Host + "/artists/" + r.Slug + "-" + r.Id,
		BirthDate: fmt.Sprint(r.YearOfBirth),
		DeathDate: fmt.Sprint(r.YearOfDeath),
		PlaceOfBirth: newPlace(Place{
			Name: r.PlaceOfBirth,
		}),
		PlaceOfDeath: newPlace(Place{
			Name: r.PlaceOfDeath,
		}),
		HasOccupation: newOccupation(Occupation{
			Name: r.Profession,
		}),
		Description: utils.StrippedHTML(r.Bio),
	})
}

// generateVisualArtworkJsonLdContent generates a map containing JSON-LD content for a visual artwork record.
// It takes a models.Record pointer and an echo.Context as input and returns a map[string]any.
func GenerateVisualArtworkJsonLdContent(r *models.Record, c echo.Context) map[string]any {

	d := map[string]any{
		"@context":    "https://schema.org",
		"@type":       "VisualArtwork",
		"name":        r.GetString("name"),
		"description": utils.StrippedHTML(r.GetString("comment")),
		"artform":     r.GetString("technique"),
	}

	return d
}

func ArtworkJsonLd(r *models.Record, a *wgaModels.Artist, c echo.Context) VisualArtwork {
	return VisualArtwork{
		Name:        r.GetString("name"),
		Description: utils.StrippedHTML(r.GetString("comment")),
		Artform:     r.GetString("technique"),
		Url:         c.Scheme() + "://" + c.Request().Host + "/artworks/" + r.GetString("slug") + "-" + r.GetId(),
		Artist:      ArtistJsonLd(a, c),
		ArtMedium:   r.GetString("medium"),
		Image: ImageObject{
			Image: c.Scheme() + "://" + c.Request().Host + "/images/" + r.GetString("image"),
		},
	}

}
