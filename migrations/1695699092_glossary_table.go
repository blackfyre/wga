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

type Glossary struct {
	Id         string `db:"id" json:"id"`
	Expression string `db:"expression" json:"expression"`
	Definition string `db:"definition" json:"definition"`
}

func init() {
	m.Register(func(db dbx.Builder) error {
		dao := daos.New(db)

		collection := &models.Collection{}

		collection.Name = "glossary"
		collection.Type = models.CollectionTypeBase
		collection.System = false
		collection.MarkAsNew()
		collection.Schema = schema.NewSchema(
			&schema.SchemaField{
				Id:          "glossary_expression",
				Name:        "expression",
				Type:        schema.FieldTypeText,
				Options:     &schema.TextOptions{},
				Presentable: true,
			},
			&schema.SchemaField{
				Id:      "glorssary_definition",
				Name:    "definition",
				Type:    schema.FieldTypeText,
				Options: &schema.TextOptions{},
			},
		)

		err := dao.SaveCollection(collection)

		if err != nil {
			return err
		}

		// read the file at ../reference/glossary_stage_1.json
		// unmarshal the json into a []Glossary
		// loop through the []Glossary
		// create a up query for each Glossary
		// execute the up query

		data, err := assets.InternalFiles.ReadFile("reference/glossary_stage_1.json")

		if err != nil {
			return err
		}

		var glossary []Glossary

		err = json.Unmarshal(data, &glossary)

		if err != nil {
			return err
		}

		for _, g := range glossary {
			q := db.Insert("glossary", dbx.Params{
				"id":         g.Id,
				"expression": g.Expression,
				"definition": g.Definition,
			})

			_, err = q.Execute()

			if err != nil {
				return err
			}

		}

		return nil
	}, func(db dbx.Builder) error {
		q := db.DropTable("glossary")
		_, err := q.Execute()

		return err
	})
}
