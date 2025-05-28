package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {

	tId := "feedbacks"
	tName := "Feedbacks"

	m.Register(func(app core.App) error {

		collection := core.NewBaseCollection(tName)

		collection.Name = tName
		collection.Id = tId
		collection.System = false
		collection.MarkAsNew()

		collection.Fields.Add(
			&core.TextField{
				Id:          tId + "_name",
				Name:        "name",
				Presentable: true,
				Required:    true,
			},
			&core.EmailField{
				Id:       tId + "_email",
				Name:     "email",
				Required: true,
			},
			&core.URLField{
				Id:   tId + "_refer_to",
				Name: "refer_to",
				// OnlyDomains: []string{hostname},
				Required: true,
			},
			&core.EditorField{
				Id:          tId + "_message",
				Name:        "message",
				Required:    true,
				ConvertURLs: true,
			},
			&core.BoolField{
				Id:          tId + "_handled",
				Name:        "handled",
				Presentable: true,
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

		return app.Save(collection)

	}, func(app core.App) error {
		c, err := app.FindCollectionByNameOrId(tId)
		if err != nil {
			return nil
		}

		return app.Delete(c)
	})
}
