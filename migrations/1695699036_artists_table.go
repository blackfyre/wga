package migrations

import (
	"encoding/json"

	"blackfyre.ninja/wga/assets"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/models/schema"
	"github.com/pocketbase/pocketbase/tools/types"
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
	m.Register(func(db dbx.Builder) error {
		dao := daos.New(db)

		collection := &models.Collection{}

		collection.Name = "artists"
		collection.Id = "artists"
		collection.Type = models.CollectionTypeBase
		collection.System = false
		collection.MarkAsNew()
		collection.Schema = schema.NewSchema(
			&schema.SchemaField{
				Id:          "artiust_name",
				Name:        "name",
				Type:        schema.FieldTypeText,
				Options:     &schema.TextOptions{},
				Presentable: true,
			},
			&schema.SchemaField{
				Id:      "artist_slug",
				Name:    "slug",
				Type:    schema.FieldTypeText,
				Options: &schema.TextOptions{},
			},
			&schema.SchemaField{
				Id:      "artist_bio",
				Name:    "bio",
				Type:    schema.FieldTypeEditor,
				Options: &schema.EditorOptions{},
			},
			&schema.SchemaField{
				Id:      "artist_yob",
				Name:    "year_of_birth",
				Type:    schema.FieldTypeNumber,
				Options: &schema.NumberOptions{},
			},
			&schema.SchemaField{
				Id:      "artist_yod",
				Name:    "year_of_death",
				Type:    schema.FieldTypeNumber,
				Options: &schema.NumberOptions{},
			},
			&schema.SchemaField{
				Id:      "artist_place_of_birth",
				Name:    "place_of_birth",
				Type:    schema.FieldTypeText,
				Options: &schema.TextOptions{},
			},
			&schema.SchemaField{
				Id:      "artist_place_of_death",
				Name:    "place_of_death",
				Type:    schema.FieldTypeText,
				Options: &schema.TextOptions{},
			},
			&schema.SchemaField{
				Id:       "artist_exact_year_of_birth",
				Name:     "exact_year_of_birth",
				Type:     schema.FieldTypeSelect,
				Options:  &schema.SelectOptions{Values: []string{Yes, No, NotApplicable}, MaxSelect: 1},
				Required: true,
			},
			&schema.SchemaField{
				Id:       "artist_exact_year_of_death",
				Name:     "exact_year_of_death",
				Type:     schema.FieldTypeSelect,
				Options:  &schema.SelectOptions{Values: []string{Yes, No, NotApplicable}, MaxSelect: 1},
				Required: true,
			},
			&schema.SchemaField{
				Id:       "artist_known_place_of_birth",
				Name:     "known_place_of_birth",
				Type:     schema.FieldTypeSelect,
				Options:  &schema.SelectOptions{Values: []string{Yes, No, NotApplicable}, MaxSelect: 1},
				Required: true,
			},
			&schema.SchemaField{
				Id:       "artist_known_place_of_death",
				Name:     "known_place_of_death",
				Type:     schema.FieldTypeSelect,
				Options:  &schema.SelectOptions{Values: []string{Yes, No, NotApplicable}, MaxSelect: 1},
				Required: true,
			},
			&schema.SchemaField{
				Id:      "artist_profession",
				Name:    "profession",
				Type:    schema.FieldTypeText,
				Options: &schema.TextOptions{},
			},
			&schema.SchemaField{
				Id:   "artist_school",
				Name: "school",
				Type: schema.FieldTypeRelation,
				Options: &schema.RelationOptions{
					CollectionId: "schools",
					MinSelect:    Ptr(1),
				},
			},
			&schema.SchemaField{
				Id:   "artist_aka",
				Name: "also_known_as",
				Type: schema.FieldTypeRelation,
				Options: &schema.RelationOptions{
					CollectionId: "artists",
				},
			},
			&schema.SchemaField{
				Id:      "artist_published",
				Name:    "published",
				Type:    schema.FieldTypeBool,
				Options: &schema.BoolOptions{},
			},
		)

		collection.Indexes = types.JsonArray[string]{
			"CREATE UNIQUE INDEX `pbx_artist_slug` ON `artists` (`slug`)",
		}

		err := dao.SaveCollection(collection)

		if err != nil {
			return err
		}

		// read the file at ../reference/glossary_stage_1.json
		// unmarshal the json into a []Glossary
		// loop through the []Glossary
		// create a up query for each Glossary
		// execute the up query

		data, err := assets.InternalFiles.ReadFile("reference/artists_with_bio_stage_2.json")

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
				"exact_year_of_birth": i.Meta.ExactYearOfBirth,
				"exact_year_of_death": i.Meta.ExactYearOfDeath,
				"school":              i.School,
				"published":           true,
			})

			_, err = q.Execute()

			if err != nil {
				return err
			}

		}

		return nil
	}, func(db dbx.Builder) error {
		// add down queries...

		return nil
	})
}
