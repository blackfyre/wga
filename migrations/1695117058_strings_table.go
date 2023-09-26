package migrations

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		dao := daos.New(db)

		collection := &models.Collection{}

		collection.Name = "strings"
		collection.Type = models.CollectionTypeBase
		collection.System = false
		collection.Schema = models.Schema{
			"field_name": models.SchemaField{
				Type: "string",
			},
			"content": models.SchemaField{
				Type: "string",
			},
		}

		return dao.SaveCollection(collection)

		// add up queries...
		columns := map[string]string{
			"id":         "text",
			"created":    "text",
			"updated":    "text",
			"field_name": "text",
			"content":    "text",
		}

		q := db.CreateTable("strings", columns)
		_, err := q.Execute()

		return err
	}, func(db dbx.Builder) error {

		q := db.DropTable("guestbook")
		_, err := q.Execute()

		return err
	})
}
