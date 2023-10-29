package migrations

import (
	"encoding/json"

	"blackfyre.ninja/wga/assets"
	"blackfyre.ninja/wga/utils"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/models/schema"
)

type ArtPeriod struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Start       int    `json:"start"`
	End         int    `json:"end"`
	Description string `json:"description"`
}

func init() {
	tName := "art_periods"
	m.Register(func(db dbx.Builder) error {
		dao := daos.New(db)

		collection := &models.Collection{}

		collection.Name = tName
		collection.Id = tName
		collection.Type = models.CollectionTypeBase
		collection.System = false
		collection.MarkAsNew()
		collection.Schema = schema.NewSchema(
			&schema.SchemaField{
				Id:          tName + "_name",
				Name:        "name",
				Type:        schema.FieldTypeText,
				Options:     &schema.TextOptions{},
				Presentable: true,
			},
			&schema.SchemaField{
				Id:      "schools_slug",
				Name:    "slug",
				Type:    schema.FieldTypeText,
				Options: &schema.TextOptions{},
			},
			&schema.SchemaField{
				Id:      tName + "_start",
				Name:    "start",
				Type:    schema.FieldTypeNumber,
				Options: &schema.NumberOptions{},
			},
			&schema.SchemaField{
				Id:      tName + "_end",
				Name:    "end",
				Type:    schema.FieldTypeNumber,
				Options: &schema.NumberOptions{},
			},
			&schema.SchemaField{
				Id:      tName + "_description",
				Name:    "description",
				Type:    schema.FieldTypeText,
				Options: &schema.TextOptions{},
			},
		)

		err := dao.SaveCollection(collection)

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
			q := db.Insert(tName, dbx.Params{
				"id":          g.ID,
				"start":       g.Start,
				"end":         g.End,
				"name":        g.Name,
				"description": g.Description,
				"slug":        utils.Slugify(g.Name),
			})

			_, err = q.Execute()

			if err != nil {
				return err
			}

		}

		return nil
	}, func(db dbx.Builder) error {
		q := db.DropTable(tName)
		_, err := q.Execute()

		return err
	})
}
