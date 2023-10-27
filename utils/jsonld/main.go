package jsonld

import (
	"os"

	"blackfyre.ninja/wga/utils"
	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase/models"
)

// generateArtistJsonLdContent generates a JSON-LD content for an artist record.
// It takes a pointer to a models.Record and an echo.Context as input and returns a map[string]any.
// The returned map contains the JSON-LD content for the artist record, including the artist's name, URL, profession,
// birth and death dates, and birth and death places (if available).
func GenerateArtistJsonLdContent(r *models.Record, c echo.Context) map[string]any {

	fullUrl := os.Getenv("WGA_PROTOCOL") + "://" + c.Request().Host + "/artists/" + r.GetString("slug")

	d := map[string]any{
		"@context":      "https://schema.org",
		"@type":         "Person",
		"name":          r.GetString("name"),
		"url":           fullUrl,
		"hasOccupation": r.GetString("profession"),
	}

	if r.GetInt("year_of_birth") > 0 {
		d["birthDate"] = r.GetString("year_of_birth")
	}

	if r.GetInt("year_of_death") > 0 {
		d["deathDate"] = r.GetString("year_of_death")
	}

	if r.GetString("place_of_birth") != "" {
		d["birthPlace"] = map[string]string{
			"@type": "Place",
			"name":  r.GetString("place_of_birth"),
		}
	}

	if r.GetString("place_of_death") != "" {
		d["deathPlace"] = map[string]string{
			"@type": "Place",
			"name":  r.GetString("place_of_death"),
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
