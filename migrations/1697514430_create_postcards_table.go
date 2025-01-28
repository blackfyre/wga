package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		collection := core.NewBaseCollection("Postcards")

		collection.Name = "Postcards"
		collection.Id = "postcards"
		collection.MarkAsNew()

		collection.Fields.Add(
			&core.TextField{
				Id:       "postcard_sender_name",
				Name:     "sender_name",
				Required: true,
				Presentable: true,
			},
			&core.EmailField{
				Id:       "postcard_sender_email",
				Name:     "sender_email",
				Required: true,
			},
			&core.TextField{
				Id:       "postcard_recipients",
				Name:     "recipients",
				Required: true,
			},
			&core.EditorField{
				Id:       "postcard_message",
				Name:     "message",
				Required: true,
			},
			&core.RelationField{
				Id:           "postcard_image_id",
				Name:         "image_id",
				CollectionId: "artworks",
				MinSelect:    1,
				MaxSelect: 1,
			},
			&core.BoolField{
				Id:       "postcard_notify_sender",
				Name:     "notify_sender",
			},
			&core.SelectField{
				Id:       "postcard_status",
				Name:     "status",
				Values: []string{"queued", "sent", "received"},
				MaxSelect: 1,
				Required: true,
				Presentable: true,
			},
			&core.DateField{
				Id:       "postcard_sent_at",
				Name:     "sent_at",
			},
		)

		return app.Save(collection)

	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId("postcards")

		if err != nil {
			// collection not found, probably already deleted
			// so nothing to do
			return nil
		}

		return app.Delete(collection)

	})
}
