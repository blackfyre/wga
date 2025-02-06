package migrations

import (
	"encoding/json"

	"github.com/blackfyre/wga/assets"
	"github.com/blackfyre/wga/utils"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

type ArtPeriod struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Start       int    `json:"start"`
	End         int    `json:"end"`
	Description string `json:"description"`
}

func init() {
	tName := "Art_periods"
	tId := "art_periods"
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
			},
			&core.TextField{
				Id:   tId + "_slug",
				Name: "slug",
			},
			&core.NumberField{
				Id:   tId + "_start",
				Name: "start",
			},
			&core.NumberField{
				Id:   tId + "_end",
				Name: "end",
			},
			&core.TextField{
				Id:   tId + "_description",
				Name: "description",
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

		data, err := assets.InternalFiles.ReadFile("reference/art_periods.json")

		if err != nil {
			return err
		}

		var c []ArtPeriod

		err = json.Unmarshal(data, &c)

		if err != nil {
			return err
		}

		for _, g := range c {
			r := core.NewRecord(collection)

			r.Set("id", g.ID)
			r.Set("start", g.Start)
			r.Set("end", g.End)
			r.Set("name", g.Name)
			r.Set("description", g.Description)
			r.Set("slug", utils.Slugify(g.Name))

			err = app.Save(r)

			if err != nil {
				return err
			}

		}

		return nil
	}, func(app core.App) error {
		return deleteCollection(app, tId)
	})
}
