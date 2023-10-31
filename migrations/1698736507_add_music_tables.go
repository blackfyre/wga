package migrations

import (
	"strings"

	"blackfyre.ninja/wga/handlers"
	shape "blackfyre.ninja/wga/models"
	"blackfyre.ninja/wga/utils"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/models/schema"
)

func init() {
	m.Register(func(db dbx.Builder) error {
		dao := daos.New(db)

		collection := &models.Collection{}

		collection.Name = "music_composers"
		collection.Type = models.CollectionTypeBase
		collection.System = false
		collection.Id = "music_composers"
		collection.MarkAsNew()
		collection.Schema = schema.NewSchema(
			&schema.SchemaField{
				Id:          "music_composers_name",
				Name:        "name",
				Type:        schema.FieldTypeText,
				Options:     &schema.TextOptions{},
				Presentable: true,
			},
			&schema.SchemaField{
				Id:          "music_composers_century",
				Name:        "century",
				Type:        schema.FieldTypeSelect,
				Options:     &schema.SelectOptions{
					Values:  []string{"12", "13", "14", "15", "16", "17", "18", "19", "20", "21"},
					MaxSelect: 1,
				},
				Presentable: true,
			},
			&schema.SchemaField{
				Id:          "music_composers_date",
				Name:        "date",
				Type:        schema.FieldTypeText,
				Options:     &schema.TextOptions{},
				Presentable: true,
			},
			&schema.SchemaField{
				Id:          "music_composers_language",
				Name:        "language",
				Type:        schema.FieldTypeText,
				Options:     &schema.TextOptions{},
				Presentable: true,
			},
		)

		err := dao.SaveCollection(collection)

		collection.Name = "music_songs"
		collection.Type = models.CollectionTypeBase
		collection.System = false
		collection.Id = "music_songs"
		collection.MarkAsNew()
		collection.Schema = schema.NewSchema(
			&schema.SchemaField{
				Id:          "music_songs_title",
				Name:        "title",
				Type:        schema.FieldTypeText,
				Options:     &schema.TextOptions{},
				Presentable: true,
			},
			&schema.SchemaField{
				Id:          "music_songs_url",
				Name:        "url",
				Type:        schema.FieldTypeText,
				Options:     &schema.TextOptions{},
				Presentable: true,
			},
			&schema.SchemaField{
				Id:          "music_songs_source",
				Name:        "source",
				Type:        schema.FieldTypeText,
				Options:     &schema.TextOptions{},
				Presentable: true,
			},
			&schema.SchemaField{
				Id:          "music_composer_name",
				Name:        "music_composer_name",
				Type: schema.FieldTypeRelation,
				Options: &schema.RelationOptions{
					CollectionId: "music_composers",
					MinSelect:    Ptr(1),
				},
			},
		)

		err = dao.SaveCollection(collection)

		if err != nil {
			return err
		}
		var composers []shape.Composer

		composers = handlers.GetParsedMusics(handlers.GetMusics())

		if err != nil {
			return err
		}

		for _, composer := range composers {

			q := db.Insert("music_composers", dbx.Params{
				"name": composer.Name,
				"date": composer.Date,
				"century": composer.Century,
				"language": composer.Language,
			})

			_, err = q.Execute()

			if err != nil {
				return err
			}

			for _, song := range composer.Songs {
				newSource := []string{}
				for _, source := range song.Source {
					newSource = append(newSource, utils.GetFileNameFromUrl(source, true))
				}
				newSourceStr := strings.Join(newSource, ",")
				q := db.Insert("music_songs", dbx.Params{
					"title": song.Title,
					"url": song.URL,
					"source": newSourceStr,
					"music_composer_name": composer.Name,
				})
				
				_, err = q.Execute()

				if err != nil {
					return err
				}
			}

		}

		return nil
	}, func(db dbx.Builder) error {
		q := db.DropTable("music_songs")
		_, err := q.Execute()

		q = db.DropTable("music_composers")
		_, err = q.Execute()

		return err
	})
}
