package migrations

import (
	"encoding/json"

	"blackfyre.ninja/wga/assets"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/models/schema"
)

type PublicString struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

func init() {
	m.Register(func(db dbx.Builder) error {
		dao := daos.New(db)

		collection := &models.Collection{}

		collection.Name = "Strings"
		collection.Type = models.CollectionTypeBase
		collection.System = false
		collection.Id = "strings"
		collection.MarkAsNew()
		collection.Schema = schema.NewSchema(
			&schema.SchemaField{
				Id:          "strings_name",
				Name:        "name",
				Type:        schema.FieldTypeText,
				Options:     &schema.TextOptions{},
				Presentable: true,
			},
			&schema.SchemaField{
				Id:      "strings_content",
				Name:    "content",
				Type:    schema.FieldTypeEditor,
				Options: &schema.EditorOptions{},
			},
		)

		err := dao.SaveCollection(collection)

		if err != nil {
			return err
		}

		data, err := assets.InternalFiles.ReadFile("reference/strings.json")

		if err != nil {
			return err
		}

		var c []PublicString

		err = json.Unmarshal(data, &c)

		if err != nil {
			return err
		}

		for _, i := range c {
			q := db.Insert("strings", dbx.Params{
				"name":    i.Name,
				"content": i.Content,
			})

			_, err = q.Execute()

			if err != nil {
				return err
			}

		}

		return nil

		// add up queries...
		// columns := map[string]string{
		// 	"id":         "text",
		// 	"created":    "text",
		// 	"updated":    "text",
		// 	"field_name": "text",
		// 	"content":    "text",
		// }

		// q := db.CreateTable("strings", columns)
		// _, err := q.Execute()

		// return err
	}, func(db dbx.Builder) error {

		q := db.DropTable("strings")
		_, err := q.Execute()

		return err
	})
}
