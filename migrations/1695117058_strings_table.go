package migrations

import (
	"encoding/json"

	"github.com/blackfyre/wga/assets"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

type PublicString struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

func init() {
	m.Register(func(app core.App) error {

		collection := core.NewBaseCollection("Strings")

		collection.Name = "Strings"
		collection.System = false
		collection.Id = "strings"
		collection.MarkAsNew()

		collection.Fields.Add(
			&core.TextField{
				Id:          "strings_name",
				Name:        "name",
				Required:    true,
				Presentable: true,
			},
			&core.EditorField{
				Id:       "strings_content",
				Name:     "content",
				Required: true,
			},
		)

		err := app.Save(collection)

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
			r := core.NewRecord(collection)
			r.Set("name", i.Name)
			r.Set("content", i.Content)

			err = app.Save(r)

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
	}, func(app core.App) error {

		c, err := app.FindCollectionByNameOrId("strings")

		if err != nil {
			return err
		}

		if c == nil {
			return nil
		}

		return app.Delete(c)
	})
}
