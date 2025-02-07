package migrations

import (
	"bytes"
	"cmp"
	"encoding/json"
	"fmt"

	"github.com/blackfyre/wga/assets"
	"github.com/blackfyre/wga/utils"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

type Artist struct {
	Id     string `json:"id"`
	Name   string `json:"name"`
	Source struct {
		Artist     string `json:"artist"`
		BirthData  string `json:"birth_data"`
		Profession string `json:"profession"`
		School     string `json:"school"`
		URL        string `json:"url"`
	} `json:"source"`
	Slug               string     `json:"slug"`
	RelativePath       string     `json:"relativePath"`
	WgaRelativePath    string     `json:"wgaRelativePath"`
	WgaID              string     `json:"wgaId"`
	Meta               ArtistMeta `json:"meta"`
	PossibleInfluences any        `json:"possibleInfluences"`
	Bio                string     `json:"bio"`
	School             string     `json:"school"`
}

type ArtistMeta struct {
	YearOfBirth          int    `json:"year_of_birth"`
	PlaceOfBirth         string `json:"place_of_birth"`
	YearOfDeath          int    `json:"year_of_death"`
	PlaceOfDeath         string `json:"place_of_death"`
	YearActiveStart      int    `json:"year_active_start"`
	YearActiveEnd        int    `json:"year_active_end"`
	ExactYearOfBirth     string `json:"exact_year_of_birth"`
	ExactYearOfDeath     string `json:"exact_year_of_death"`
	PlaceOfActivityStart string `json:"place_of_activity_start"`
	PlaceOfActivityEnd   string `json:"place_of_activity_end"`
	Normalized           string `json:"normalized"`
	ExactActiveStart     string `json:"exact_active_start"`
	ExactActiveEnd       string `json:"exact_active_end"`
	KnownPlaceOfBirth    string `json:"known_place_of_birth"`
	KnownPlaceOfDeath    string `json:"known_place_of_death"`
	Parsed               bool   `json:"parsed"`
}

func Ptr[T any](v T) *T {
	return &v
}

const (
	Yes           string = "yes"
	No            string = "no"
	NotApplicable string = "n/a"
)

func init() {
	m.Register(func(app core.App) error {

		collection := core.NewBaseCollection("Artists")

		collection.Name = "Artists"
		collection.Id = "artists"
		collection.MarkAsNew()

		collection.Fields.Add(
			&core.TextField{
				Id:          "artist_name",
				Name:        "name",
				Required:    true,
				Presentable: true,
			},
			&core.TextField{
				Id:       "artist_slug",
				Name:     "slug",
				Required: true,
			},
			&core.EditorField{
				Id:   "artist_bio",
				Name: "bio",
			},
			&core.NumberField{
				Id:   "artist_yob",
				Name: "year_of_birth",
			},
			&core.NumberField{
				Id:   "artist_yod",
				Name: "year_of_death",
			},
			&core.TextField{
				Id:   "artist_place_of_birth",
				Name: "place_of_birth",
			},
			&core.TextField{
				Id:   "artist_place_of_death",
				Name: "place_of_death",
			},
			&core.BoolField{
				Id:   "artist_exact_year_of_birth",
				Name: "exact_year_of_birth",
			},
			&core.BoolField{
				Id:   "artist_exact_year_of_death",
				Name: "exact_year_of_death",
			},
			&core.TextField{
				Id:   "artist_profession",
				Name: "profession",
			},
			&core.SelectField{
				Id:        "artist_known_place_of_birth",
				Name:      "known_place_of_birth",
				Values:    []string{Yes, No, NotApplicable},
				Required:  true,
				MaxSelect: 1,
			},
			&core.SelectField{
				Id:        "artist_known_place_of_death",
				Name:      "known_place_of_death",
				Values:    []string{Yes, No, NotApplicable},
				Required:  true,
				MaxSelect: 1,
			},
			&core.TextField{
				Id:   "artist_profession",
				Name: "profession",
			},
			&core.RelationField{
				Id:           "artist_school",
				Name:         "school",
				Required:     true,
				CollectionId: "schools",
				Presentable:  true,
				MinSelect:    1,
				MaxSelect:    10,
			},
			&core.BoolField{
				Id:       "artist_published",
				Name:     "published",
				Required: true,
			},
			&core.AutodateField{
				Name:     "created",
				OnCreate: true,
			},
			&core.AutodateField{
				Name:     "updated",
				OnCreate: true,
				OnUpdate: true,
			},
		)

		collection.AddIndex("pbx_artist_slug", true, "slug", "")

		err := app.Save(collection)

		if err != nil {
			return err
		}

		collection, err = app.FindCollectionByNameOrId("artists")

		if err != nil {
			return err
		}

		collection.Fields.Add(

			&core.RelationField{
				Id:           "artist_aka",
				Name:         "also_known_as",
				CollectionId: "artists",
				Presentable:  true,
			},
		)

		err = app.Save(collection)

		if err != nil {
			return err
		}

		// read the file at ../reference/glossary_stage_1.json
		// unmarshal the json into a []Glossary
		// loop through the []Glossary
		// create a up query for each Glossary
		// execute the up query

		zstFile, err := assets.InternalFiles.ReadFile("reference/artists_with_bio_stage_2.json.zst")

		if err != nil {
			return err
		}

		var buf bytes.Buffer

		err = utils.Decompress(bytes.NewReader(zstFile), &buf)

		if err != nil {
			return err
		}

		var c []Artist

		err = json.Unmarshal(buf.Bytes(), &c)

		if err != nil {
			return err
		}

		for _, i := range c {
			r := core.NewRecord(collection)
			r.Set("id", i.Id)
			r.Set("name", i.Name)
			r.Set("bio", i.Bio)
			r.Set("slug", i.Slug)
			r.Set("year_of_birth", i.Meta.YearOfBirth)
			r.Set("year_of_death", i.Meta.YearOfDeath)
			r.Set("place_of_birth", i.Meta.PlaceOfBirth)
			r.Set("place_of_death", i.Meta.PlaceOfDeath)
			r.Set("profession", i.Source.Profession)
			r.Set("exact_year_of_birth", i.Meta.ExactYearOfBirth)
			r.Set("exact_year_of_death", i.Meta.ExactYearOfDeath)
			r.Set("school", i.School)
			r.Set("published", true)
			r.Set("known_place_of_birth", cmp.Or(i.Meta.KnownPlaceOfBirth, NotApplicable))
			r.Set("known_place_of_death", cmp.Or(i.Meta.KnownPlaceOfDeath, NotApplicable))

			err = app.Save(r)

			if err != nil {
				app.Logger().Error("Error saving record", err)
				fmt.Printf("Record data: %+v", i)
				return err
			}

		}

		return nil
	}, func(app core.App) error {
		return deleteCollection(app, "artists")
	})
}
