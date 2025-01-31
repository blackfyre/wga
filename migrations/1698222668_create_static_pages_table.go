package migrations

import (
	"encoding/json"

	"github.com/blackfyre/wga/assets"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

type staticPage struct {
	Title   string `json:"title"`
	Slug    string `json:"slug"`
	Content string `json:"content"`
}

func init() {

	tId := "static_pages"
	tName := "Static_pages"

	m.Register(func(app core.App) error {

		collection := core.NewBaseCollection(tName)

		collection.Name = tName
		collection.Id = tId
		collection.System = false
		collection.MarkAsNew()

		collection.Fields.Add(
			&core.TextField{
				Id:       tId + "_title",
				Name:     "title",
				Required: true,
				Presentable: true,
			},
			&core.TextField{
				Id:       tId + "_slug",
				Name:     "slug",
			},
			&core.EditorField{
				Id:       tId + "_content",
				Name:     "content",
				Required: true,
				ConvertURLs: true,
			},
		)

		err := app.Save(collection)

		if err != nil {
			return err
		}

		data, err := assets.InternalFiles.ReadFile("reference/static_content.json")

		if err != nil {
			return err
		}

		var c []staticPage

		err = json.Unmarshal(data, &c)

		if err != nil {
			return err
		}

		for _, g := range c {

			r := core.NewRecord(collection)

			r.Set("title", g.Title)
			r.Set("slug", g.Slug)
			r.Set("content", g.Content)

			err = app.Save(r)

			if err != nil {
				return err
			}

		}

		return nil

	}, func(app core.App) error {
		collection, err := app.FindCollectionByNameOrId(tId)
		if err != nil {
			return nil
		}

		return app.Delete(collection)
	})
}
