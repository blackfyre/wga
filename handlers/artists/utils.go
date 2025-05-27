package artists

import (
	"strconv"
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

const (
	Yes           string = "yes"
	No            string = "no"
	NotApplicable string = "n/a"
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

// generateBioSection generates a bio section based on the provided parameters.
// It takes a prefix string, year integer, exactYear string, place string, and knownPlace string as input.
// It returns a string representing the generated bio section.
func generateBioSection(prefix string, year int, exactYear string, place string, knownPlace string) string {
	var c []string

	c = append(c, prefix)
	y := strconv.Itoa(year)

	if exactYear == No {
		y = "~" + y
	}

	c = append(c, y)

	if knownPlace == No {
		place += "?"
	}

	c = append(c, place)

	return strings.Join(c, " ")
}

// normalizedBioExcerpt returns a normalized biography excerpt for the given record.
// It includes the person's year and place of birth and death (if available).
func normalizedBioExcerpt(d BioExcerptDTO) string {
	var s []string

	s = append(s, generateBioSection("b.", d.YearOfBirth, d.ExactYearOfBirth, d.PlaceOfBirth, d.KnownPlaceOfBirth))
	s = append(s, generateBioSection("d.", d.YearOfDeath, d.ExactYearOfDeath, d.PlaceOfDeath, d.KnownPlaceOfDeath))

	return strings.Join(s, ", ")
}

// findArtworksByAuthorId retrieves a list of artworks by the given author ID.
// It uses the provided PocketBase instance to query the database and returns
// a slice of Record pointers and an error, if any.
func findArtworksByAuthorId(app *pocketbase.PocketBase, authorId string) ([]*core.Record, error) {
	return app.FindRecordsByFilter("artworks", "author = '"+authorId+"'", "+title", 100, 0)
}
