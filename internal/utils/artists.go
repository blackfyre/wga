package utils

import (
	"strconv"
	"strings"

	"github.com/blackfyre/wga/internal/constants"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

type BioExcerptDTO struct {
	YearOfBirth       int
	ExactYearOfBirth  string
	PlaceOfBirth      string
	KnownPlaceOfBirth string
	YearOfDeath       int
	ExactYearOfDeath  string
	PlaceOfDeath      string
	KnownPlaceOfDeath string
}

func generateBioSection(prefix string, year int, exactYear string, place string, knownPlace string) string {
	var components []string

	components = append(components, prefix)
	yearText := strconv.Itoa(year)

	if exactYear == "no" {
		yearText = "~" + yearText
	}

	components = append(components, yearText)

	if knownPlace == "no" {
		place += "?"
	}

	components = append(components, place)

	return strings.Join(components, " ")
}

func NormalizedBioExcerpt(d BioExcerptDTO) string {
	var sections []string

	sections = append(sections, generateBioSection("b.", d.YearOfBirth, d.ExactYearOfBirth, d.PlaceOfBirth, d.KnownPlaceOfBirth))
	sections = append(sections, generateBioSection("d.", d.YearOfDeath, d.ExactYearOfDeath, d.PlaceOfDeath, d.KnownPlaceOfDeath))

	return strings.Join(sections, ", ")
}

func FindArtworksByAuthorID(app *pocketbase.PocketBase, authorID string) ([]*core.Record, error) {
	return app.FindRecordsByFilter(constants.CollectionArtworks, "author ?~ {:authorId}", "+title", 0, 0, dbx.Params{
		"authorId": authorID,
	})
}
