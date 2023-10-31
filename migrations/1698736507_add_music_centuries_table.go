package migrations

import (
	"strings"

	"blackfyre.ninja/wga/handlers"
	"blackfyre.ninja/wga/utils"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/daos"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/models/schema"
)

type Song struct {
	Title  string
	URL    string
	Source []string
}

type Composer struct {
	Name     string
	Date     string
	Language string
	Songs    []Song
}

type Century struct {
	Century   string
	Composers []Composer
}


func init() {
	m.Register(func(db dbx.Builder) error {
		dao := daos.New(db)

		collection := &models.Collection{}
		
		collection.Name = "music_centuries"
		collection.Type = models.CollectionTypeBase
		collection.System = false
		collection.Id = "music_centuries"
		collection.MarkAsNew()
		collection.Schema = schema.NewSchema(
			&schema.SchemaField{
				Id:          "music_centuries_century",
				Name:        "century",
				Type:        schema.FieldTypeText,
				Options:     &schema.TextOptions{},
				Presentable: true,
			},
			&schema.SchemaField{
				Id:      "music_centuries_composers",
				Name:    "composers",
				Type:    schema.FieldTypeRelation,
				Options: &schema.RelationOptions{},
			},
		)

		err := dao.SaveCollection(collection)

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
			&schema.SchemaField{
				Id:      "music_composers_songs",
				Name:    "songs",
				Type:    schema.FieldTypeRelation,
				Options: &schema.RelationOptions{},
			},
		)

		err = dao.SaveCollection(collection)

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
		)

		err = dao.SaveCollection(collection)

		if err != nil {
			return err
		}

		data := handlers.GetMusics()

		if err != nil {
			return err
		}

		for _, century := range data {
			q := db.Insert("music_centuries", dbx.Params{
				"century": century.Century,
			})

			_, err = q.Execute()

			if err != nil {
				return err
			}

			for _, composer := range century.Composers {
				q := db.Insert("music_composers", dbx.Params{
					"name": composer.Name,
					"date": composer.Date,
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
					})
					
					_, err = q.Execute()

					if err != nil {
						return err
					}
				}
			}
		}

		return nil
	}, func(db dbx.Builder) error {
		q := db.DropTable("music_songs")
		_, err := q.Execute()

		q = db.DropTable("music_composers")
		_, err = q.Execute()

		q = db.DropTable("music_centuries")
		_, err = q.Execute()

		return err
	})
}
