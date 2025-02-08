package migrations

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/blackfyre/wga/assets"
	"github.com/blackfyre/wga/utils"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
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

func init() {
	tId := "artworks"
	tName := "Artworks"
	m.Register(func(app core.App) error {
		collection := core.NewBaseCollection(tName)

		collection.Name = tName
		collection.Id = tId
		collection.System = false
		collection.MarkAsNew()

		collection.Fields.Add(
			&core.TextField{
				Id:          tId + "_title",
				Name:        "title",
				Presentable: true,
				Required:    true,
			},
			&core.RelationField{
				Id:           tId + "_author",
				Name:         "author",
				CollectionId: "artists",
				MinSelect:    1,
				MaxSelect:    10,
			},
			&core.RelationField{
				Id:           tId + "_form",
				Name:         "form",
				CollectionId: "art_forms",
				MinSelect:    1,
				MaxSelect:    20,
			},
			&core.RelationField{
				Id:           tId + "_type",
				Name:         "type",
				CollectionId: "art_types",
				MinSelect:    1,
				MaxSelect:    20,
			},
			&core.TextField{
				Id:   tId + "_technique",
				Name: "technique",
			},
			&core.RelationField{
				Id:           tId + "_school",
				Name:         "school",
				CollectionId: "schools",
				MinSelect:    1,
				MaxSelect:    10,
			},
			&core.EditorField{
				Id:   tId + "_comment",
				Name: "comment",
			},
			&core.BoolField{
				Id:   tId + "_published",
				Name: "published",
			},
			&core.FileField{
				Id:   tId + "_image",
				Name: "image",
				MimeTypes: []string{
					"image/jpeg", "image/png",
				},
				MaxSize: 1024 * 1024 * 5,
				Thumbs:  []string{"100x100", "320x240"},
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

		err := app.Save(collection)

		if err != nil {
			return err
		}

		zstFile, err := assets.InternalFiles.ReadFile("reference/artworks_stage_2.json.zst")

		if err != nil {
			return err
		}

		var buf bytes.Buffer

		err = utils.Decompress(bytes.NewReader(zstFile), &buf)

		if err != nil {
			return err
		}

		var c []ArtworkStage1

		err = json.Unmarshal(buf.Bytes(), &c)

		if err != nil {
			return err
		}

		errorCounter := 0

		for _, g := range c {

			record := core.NewRecord(collection)

			record.Set("id", g.Id)
			record.Set("title", g.Title)
			record.Set("author", g.AuthorId)
			record.Set("form", g.FormId)
			record.Set("technique", g.Technique)
			record.Set("school", g.SchoolId)
			record.Set("comment", g.Comment)
			record.Set("published", true)
			// record.Set("image", g.Image)
			record.Set("type", g.TypeId)

			err = app.Save(record)

			if err != nil {
				errorCounter++
			}

		}

		if errorCounter > 0 {
			fmt.Println("Failed to insert", errorCounter, "artworks")
		}

		return nil
	}, func(app core.App) error {
		return deleteCollection(app, "artworks")
	})
}
