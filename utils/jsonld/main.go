package jsonld

import (
	"os"

	wgamodels "github.com/blackfyre/wga/models"
	"github.com/blackfyre/wga/utils"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/models"
)

// generateArtistJsonLdContent generates a JSON-LD content for an artist record.
// It takes a pointer to a models.Record and an echo.Context as input and returns a map[string]any.
// The returned map contains the JSON-LD content for the artist record, including the artist's name, URL, profession,
// birth and death dates, and birth and death places (if available).
func GenerateArtistJsonLdContent(r *wgamodels.Artist, c echo.Context) map[string]any {

	fullUrl := os.Getenv("WGA_PROTOCOL") + "://" + c.Request().Host + "/artists/" + r.Slug

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
