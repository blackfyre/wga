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

		collection.Name = "guestbook"
		collection.Type = models.CollectionTypeBase
		collection.System = false
		collection.MarkAsNew()
		collection.Schema = schema.NewSchema(
			&schema.SchemaField{
				Id:      "guestbooks_message",
				Name:    "message",
				Type:    schema.FieldTypeText,
				Options: &schema.TextOptions{},
			},
			&schema.SchemaField{
				Id:          "guestbooks_name",
				Name:        "name",
				Type:        schema.FieldTypeText,
				Options:     &schema.TextOptions{},
				Presentable: true,
			},
			&schema.SchemaField{
				Id:          "guestbooks_email",
				Name:        "email",
				Type:        schema.FieldTypeEmail,
				Options:     &schema.EmailOptions{},
				Presentable: true,
			},
			&schema.SchemaField{
				Id:      "guestbooks_location",
				Name:    "location",
				Type:    schema.FieldTypeText,
				Options: &schema.TextOptions{},
			},
		)

		return dao.SaveCollection(collection)
	}, func(db dbx.Builder) error {
		// add down queries...

		q := db.DropTable("guestbook")
		_, err := q.Execute()

		return err
	})
}
