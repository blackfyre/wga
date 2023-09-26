package migrations

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/models/schema"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		dao := daos.New(db)

		collection := &models.Collection{}

		collection.Name = "strings"
		collection.Type = models.CollectionTypeBase
		collection.System = false
		collection.MarkAsNew()
		collection.Schema = schema.NewSchema(
			&schema.SchemaField{
				Id:          "strings_name",
				Name:        "field_name",
				Type:        schema.FieldTypeText,
				Options:     &schema.TextOptions{},
				Presentable: true,
			},
			&schema.SchemaField{
				Id:      "strings_content",
				Name:    "content",
				Type:    schema.FieldTypeText,
				Options: &schema.TextOptions{},
			},
		)

		return dao.SaveCollection(collection)

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
