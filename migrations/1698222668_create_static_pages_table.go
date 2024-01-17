package migrations

import (
	"encoding/json"

	"github.com/blackfyre/wga/assets"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/models/schema"
)

type staticPage struct {
	Title   string `json:"title"`
	Slug    string `json:"slug"`
	Content string `json:"content"`
}

func init() {

	tId := "static_pages"
	tName := "Static_pages"

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
				Id:      tId + "_slug",
				Name:    "slug",
				Type:    schema.FieldTypeText,
				Options: &schema.TextOptions{},
			},
			&schema.SchemaField{
				Id:   tId + "_content",
				Name: "content",
				Type: schema.FieldTypeEditor,
				Options: &schema.EditorOptions{
					ConvertUrls: true,
				},
				Required: true,
			},
		)

		err := dao.SaveCollection(collection)

		if err != nil {
			return err
		}

		data, err := assets.InternalFiles.ReadFile("reference/static_content.json")

		if err != nil {
			return err
		}

		var c []staticPage

		err = json.Unmarshal(data, &c)

		if err != nil {
			return err
		}

		for _, g := range c {
			q := db.Insert(tId, dbx.Params{
				"title":   g.Title,
				"slug":    g.Slug,
				"content": g.Content,
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
