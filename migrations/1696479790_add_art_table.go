package migrations

import (
	"encoding/json"
	"strings"

	"blackfyre.ninja/wga/assets"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/models/schema"
)

type ArtworkStage0 struct {
	Author    string `json:"AUTHOR"`
	BORNDIED  string `json:"BORN-DIED"`
	Title     string `json:"TITLE"`
	Date      string `json:"DATE"`
	Technique string `json:"TECHNIQUE"`
	Location  string `json:"LOCATION"`
	URL       string `json:"URL"`
	Form      string `json:"FORM"`
	Type      string `json:"TYPE"`
	School    string `json:"SCHOOL"`
	Timeframe string `json:"TIMEFRAME"`
}

type ArtworkStage1 struct {
	Id        string            `json:"id"`
	AuthorId  string            `json:"author_id"`
	Title     string            `json:"title"`
	Src       ArtworkStage0     `json:"src"`
	FormId    string            `json:"form_id"`
	Type      string            `json:"type"`
	SchoolId  string            `json:"school_id"`
	Meta      ArtworkStage1Meta `json:"meta"`
	Comment   string            `json:"comment"`
	Technique string            `json:"technique"`
	Image     string            `json:"image"`
	TypeId    string            `json:"type_id"`
}

type ArtworkStage1Meta struct {
	WgaId string `json:"wga_id"`
}

func readArtworkStage1Files() ([]ArtworkStage1, error) {
	var artworks []ArtworkStage1

	fileList, err := assets.InternalFiles.ReadDir("reference")

	if err != nil {
		return nil, err
	}

	files := []string{}

	for _, file := range fileList {

		//if file name contains `artworks_stage_1_` then add to files
		if strings.Contains(file.Name(), "artworks_stage_2_") {
			files = append(files, "reference/"+file.Name())
		}
	}

	for _, file := range files {
		data, err := assets.InternalFiles.ReadFile(file)
		if err != nil {
			return nil, err
		}

		var c []ArtworkStage1
		err = json.Unmarshal(data, &c)
		if err != nil {
			return nil, err
		}

		artworks = append(artworks, c...)
	}

	return artworks, nil
}

func init() {
	tId := "artworks"
	tName := "Artworks"
	m.Register(func(db dbx.Builder) error {
		dao := daos.New(db)

		collection := &models.Collection{}

		collection.Name = tName
		collection.Id = tId
		collection.Type = models.CollectionTypeBase
		collection.System = false
		collection.MarkAsNew()
		collection.Schema = schema.NewSchema(
			&schema.SchemaField{
				Id:          tId + "_title",
				Name:        "title",
				Type:        schema.FieldTypeText,
				Options:     &schema.TextOptions{},
				Presentable: true,
				Required:    true,
			},
			&schema.SchemaField{
				Id:   tId + "_author",
				Name: "author",
				Type: schema.FieldTypeRelation,
				Options: &schema.RelationOptions{
					CollectionId: "artists",
					MinSelect:    Ptr(1),
				},
			},
			&schema.SchemaField{
				Id:   tId + "_form",
				Name: "form",
				Type: schema.FieldTypeRelation,
				Options: &schema.RelationOptions{
					CollectionId: "art_forms",
					MinSelect:    Ptr(1),
				},
			},
			&schema.SchemaField{
				Id:   tId + "_type",
				Name: "type",
				Type: schema.FieldTypeRelation,
				Options: &schema.RelationOptions{
					CollectionId: "art_types",
					MinSelect:    Ptr(1),
				},
			},
			&schema.SchemaField{
				Id:      tId + "_technique",
				Name:    "technique",
				Type:    schema.FieldTypeText,
				Options: &schema.TextOptions{},
			},
			&schema.SchemaField{
				Id:   tId + "_school",
				Name: "school",
				Type: schema.FieldTypeRelation,
				Options: &schema.RelationOptions{
					CollectionId: "schools",
					MinSelect:    Ptr(1),
				},
			},
			&schema.SchemaField{
				Id:      tId + "_comment",
				Name:    "comment",
				Type:    schema.FieldTypeEditor,
				Options: &schema.EditorOptions{},
			},
			&schema.SchemaField{
				Id:   tId + "_published",
				Name: "published",
				Type: schema.FieldTypeBool,
			},
			&schema.SchemaField{
				Id:   tId + "_image",
				Name: "image",
				Type: schema.FieldTypeFile,
				Options: &schema.FileOptions{
					MimeTypes: []string{
						"image/jpeg", "image/png",
					},
					Thumbs:  []string{"100x100", "320x240"},
					MaxSize: 1024 * 1024 * 5,
				},
				Required: true,
			},
		)

		err := dao.SaveCollection(collection)

		if err != nil {
			return err
		}

		data, err := readArtworkStage1Files()

		if err != nil {
			return err
		}

		for _, g := range data {
			q := db.Insert(tId, dbx.Params{
				"id":        g.Id,
				"title":     g.Title,
				"author":    g.AuthorId,
				"form":      g.FormId,
				"technique": g.Technique,
				"school":    g.SchoolId,
				"comment":   g.Comment,
				"published": true,
				"image":     g.Image,
				"type":      g.TypeId,
			})

			_, err = q.Execute()

			if err != nil {
				return err
			}

		}

		return nil
	}, func(db dbx.Builder) error {
		q := db.DropTable(tId)
		_, err := q.Execute()

		return err
	})
}
