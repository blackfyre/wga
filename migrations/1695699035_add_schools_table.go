package migrations

import (
	"encoding/json"

	"blackfyre.ninja/wga/assets"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/models/schema"
)

type School struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

func init() {
	m.Register(func(db dbx.Builder) error {
		dao := daos.New(db)

		collection := &models.Collection{}

		collection.Name = "schools"
		collection.Id = "schools"
		collection.Type = models.CollectionTypeBase
		collection.System = false
		collection.MarkAsNew()
		collection.Schema = schema.NewSchema(
			&schema.SchemaField{
				Id:          "schools_name",
				Name:        "name",
				Type:        schema.FieldTypeText,
				Options:     &schema.TextOptions{},
				Presentable: true,
			},
		)

		err := dao.SaveCollection(collection)

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
			q := db.Insert("schools", dbx.Params{
				"id":   i.Id,
				"name": i.Name,
			})

			_, err = q.Execute()

			if err != nil {
				return err
			}

		}

		return nil
	}, func(db dbx.Builder) error {
		// add down queries...

		return nil
	})
}
