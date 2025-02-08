package migrations

import (
	"cmp"
	"encoding/json"
	"strings"

	"github.com/blackfyre/wga/assets"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
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

			collection, err := app.FindCollectionByNameOrId("artists")

			if err != nil {
				return err
			}

			record := core.NewRecord(collection)

			record.Set("id", i.Id)
			record.Set("name", i.Name)
			record.Set("bio", i.Bio)
			record.Set("slug", i.Slug)
			record.Set("year_of_birth", i.Meta.YearOfBirth)
			record.Set("year_of_death", i.Meta.YearOfDeath)
			record.Set("place_of_birth", i.Meta.PlaceOfBirth)
			record.Set("place_of_death", i.Meta.PlaceOfDeath)
			record.Set("profession", i.Source.Profession)
			record.Set("school", i.School)
			record.Set("published", true)
			record.Set("exact_year_of_birth", i.Meta.ExactYearOfBirth)
			record.Set("exact_year_of_death", i.Meta.ExactYearOfDeath)
			record.Set("known_place_of_birth", cmp.Or(i.Meta.KnownPlaceOfBirth, NotApplicable))
			record.Set("known_place_of_death", cmp.Or(i.Meta.KnownPlaceOfDeath, NotApplicable))

			err = app.Save(record)

			if err != nil {
				errString := err.Error()

				// if errString contains "UNIQUE constraint failed: artists.slug" then ignore
				// otherwise return error

				if !strings.Contains(errString, "slug: Value must be unique.") {
					app.Logger().Error(errString)
					return err
				}
			}

		}

		return nil
	}, func(app core.App) error {
		// add down queries...

		return nil
	})
}
