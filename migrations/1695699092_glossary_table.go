package migrations

import (
	"encoding/json"

	"github.com/blackfyre/wga/assets"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

type Glossary struct {
	Id         string `db:"id" json:"id"`
	Expression string `db:"expression" json:"expression"`
	Definition string `db:"definition" json:"definition"`
}

func init() {
	m.Register(func(app core.App) error {
		collection := core.NewBaseCollection("Glossary")

		collection.Name = "Glossary"
		collection.Id = "glossary"

		collection.MarkAsNew()

		collection.Fields.Add(
			&core.TextField{
				Id:          "glossary_expression",
				Name:        "expression",
				Required:    true,
				Presentable: true,
			},
			&core.TextField{
				Id:       "glossary_definition",
				Name:     "definition",
				Required: true,
			},
		)

		err := app.Save(collection)

		if err != nil {
			return err
		}

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
			r := core.NewRecord(collection)
			r.Set("expression", g.Expression)
			r.Set("definition", g.Definition)
			err = app.Save(r)
			if err != nil {
				return err
			}
		}

		return nil
	}, func(app core.App) error {
		return deleteCollection(app, "glossary")
	})
}
