package migrations

import (
	"encoding/json"

	"github.com/blackfyre/wga/assets"
	"github.com/blackfyre/wga/utils"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

type ArtType struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func init() {
	tName := "Art_types"
	tId := "art_types"
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
		)

		err := app.Save(collection)

		if err != nil {
			return err
		}

		data, err := assets.InternalFiles.ReadFile("reference/types.json")

		if err != nil {
			return err
		}

		var c []ArtType

		err = json.Unmarshal(data, &c)

		if err != nil {
			return err
		}

		for _, g := range c {

			record := core.NewRecord(collection)

			record.Set("id", g.ID)
			record.Set("name", g.Name)
			record.Set("slug", utils.Slugify(g.Name))

			err = app.Save(record)

			if err != nil {
				return err
			}
		}

		return nil
	}, func(app core.App) error {
		return deleteCollection(app, tId)
	})
}
