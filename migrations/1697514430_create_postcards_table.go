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

		collection.Name = "postcards"
		collection.Id = "postcards"
		collection.Type = models.CollectionTypeBase
		collection.System = false
		collection.MarkAsNew()
		collection.Schema = schema.NewSchema(
			&schema.SchemaField{
				Id:          "postcard_sender_name",
				Name:        "sender_name",
				Type:        schema.FieldTypeText,
				Options:     &schema.TextOptions{},
				Presentable: true,
				Required:    true,
			},
			&schema.SchemaField{
				Id:       "postcard_sender_email",
				Name:     "sender_email",
				Type:     schema.FieldTypeEmail,
				Options:  &schema.EmailOptions{},
				Required: true,
			},
			&schema.SchemaField{
				Id:       "postcard_recipients",
				Name:     "recipients",
				Type:     schema.FieldTypeText,
				Options:  &schema.TextOptions{},
				Required: true,
			},
			&schema.SchemaField{
				Id:       "postcard_message",
				Name:     "message",
				Type:     schema.FieldTypeEditor,
				Options:  &schema.EditorOptions{},
				Required: true,
			},
			&schema.SchemaField{
				Id:   "postcard_image_id",
				Name: "image_id",
				Type: schema.FieldTypeRelation,
				Options: &schema.RelationOptions{
					CollectionId: "artworks",
					MinSelect:    Ptr(1),
				},
			},
			&schema.SchemaField{
				Id:      "postcard_notify_sender",
				Name:    "notify_sender",
				Type:    schema.FieldTypeBool,
				Options: schema.BoolOptions{},
			},
			&schema.SchemaField{
				Id:   "postcard_status",
				Name: "status",
				Type: schema.FieldTypeSelect,
				Options: &schema.SelectOptions{
					Values: []string{"queued", "sent", "received"},
				},
				Presentable: true,
			},
			&schema.SchemaField{
				Id:      "postcard_sent_at",
				Name:    "sent_at",
				Type:    schema.FieldTypeDate,
				Options: &schema.DateOptions{},
			},
		)

		return dao.SaveCollection(collection)

	}, func(db dbx.Builder) error {
		q := db.DropTable("postcards")
		_, err := q.Execute()

		return err

	})
}
