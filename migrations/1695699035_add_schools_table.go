package migrations

import (
	"encoding/json"

	"github.com/blackfyre/wga/assets"
	"github.com/blackfyre/wga/utils"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

type School struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

func init() {
	m.Register(func(app core.App) error {
		collection := core.NewBaseCollection("Schools")

		collection.Name = "Schools"
		collection.Id = "schools"
		collection.System = false
		collection.MarkAsNew()

		collection.Fields.Add(
			&core.TextField{
				Id:          "schools_name",
				Name:        "name",
				Required:    true,
				Presentable: true,
			},
			&core.TextField{
				Id:       "schools_slug",
				Name:     "slug",
				Required: true,
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

		data, err := assets.InternalFiles.ReadFile("reference/schools.json")

		if err != nil {
			return err
		}

		var c []School

		err = json.Unmarshal(data, &c)

		if err != nil {
			return err
		}

		for _, i := range c {
			r := core.NewRecord(collection)
			r.Set("name", i.Name)
			r.Set("slug", utils.Slugify(i.Name))
			err = app.Save(r)

			if err != nil {
				return err
			}

		}

		return nil
	}, func(app core.App) error {
		return deleteCollection(app, "schools")
	})
}
