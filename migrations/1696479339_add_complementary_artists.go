package migrations

import (
	"encoding/json"
	"strings"

	"blackfyre.ninja/wga/assets"
	"github.com/pocketbase/dbx"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		data, err := assets.InternalFiles.ReadFile("reference/complementary_artists.json")

		if err != nil {
			return err
		}

		var c []Artist

		err = json.Unmarshal(data, &c)

		if err != nil {
			return err
		}

		for _, i := range c {
			q := db.Insert("artists", dbx.Params{
				"id":                  i.Id,
				"name":                i.Name,
				"bio":                 i.Bio,
				"slug":                i.Slug,
				"year_of_birth":       i.Meta.YearOfBirth,
				"year_of_death":       i.Meta.YearOfDeath,
				"place_of_birth":      i.Meta.PlaceOfBirth,
				"place_of_death":      i.Meta.PlaceOfDeath,
				"profession":          i.Source.Profession,
				"school":              i.School,
				"published":           true,
				"exact_year_of_birth": i.Meta.ExactYearOfBirth,
				"exact_year_of_death": i.Meta.ExactYearOfDeath,
			})

			_, err = q.Execute()

			if err != nil {
				errString := err.Error()

				// if errString contains "UNIQUE constraint failed: artists.slug" then ignore
				// otherwise return error

				if !strings.Contains(errString, "UNIQUE constraint failed: Artists.slug") {
					return err
				}

			}

		}

		return nil
	}, func(db dbx.Builder) error {
		// add down queries...

		return nil
	})
}
