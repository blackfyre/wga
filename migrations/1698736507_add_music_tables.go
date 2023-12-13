package migrations

import (
	"log"

	"blackfyre.ninja/wga/handlers"
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

		collection.Name = "Music_composer"
		collection.Type = models.CollectionTypeBase
		collection.System = false
		collection.Id = "music_composer"
		collection.MarkAsNew()
		collection.Schema = schema.NewSchema(
			&schema.SchemaField{
				Id:          "music_composer_id",
				Name:        "id",
				Type:        schema.FieldTypeText,
				Options:     &schema.TextOptions{},
			},
			&schema.SchemaField{
				Id:          "music_composer_name",
				Name:        "name",
				Type:        schema.FieldTypeText,
				Options:     &schema.TextOptions{},
				Presentable: true,
			},
			&schema.SchemaField{
				Id:          "music_composer_century",
				Name:        "century",
				Type:        schema.FieldTypeSelect,
				Options:     &schema.SelectOptions{
					Values:  []string{"12", "13", "14", "15", "16", "17", "18", "19", "20", "21"},
					MaxSelect: 1,
				},
				Presentable: true,
			},
			&schema.SchemaField{
				Id:          "music_composer_date",
				Name:        "date",
				Type:        schema.FieldTypeText,
				Options:     &schema.TextOptions{},
				Presentable: true,
			},
			&schema.SchemaField{
				Id:          "music_composer_language",
				Name:        "language",
				Type:        schema.FieldTypeText,
				Options:     &schema.TextOptions{},
				Presentable: true,
			},
		)

		err := dao.SaveCollection(collection)
		if err != nil {
			// Handle the error, for example log it and return
			log.Printf("Error saving collection: %v", err)
			return err
		}

		collection.Name = "Music_song"
		collection.Type = models.CollectionTypeBase
		collection.System = false
		collection.Id = "music_song"
		collection.MarkAsNew()
		collection.Schema = schema.NewSchema(
			&schema.SchemaField{
				Id:          "composer_id",
				Name:        "composer_id",
				Type:        schema.FieldTypeText,
				Options:     &schema.TextOptions{},
				Presentable: true,
			},
			&schema.SchemaField{
				Id:          "music_song_title",
				Name:        "title",
				Type:        schema.FieldTypeText,
				Options:     &schema.TextOptions{},
				Presentable: true,
			},
			&schema.SchemaField{
				Id:          "music_song_url",
				Name:        "url",
				Type:        schema.FieldTypeText,
				Options:     &schema.TextOptions{},
			},
			&schema.SchemaField{
				Id:          "music_song_source",
				Name:        "source",
				Type:        schema.FieldTypeFile,
				Options:     &schema.FileOptions{},
				Presentable: true,
			},
		)

		err = dao.SaveCollection(collection)

		if err != nil {
			return err
		}
		composers, err := handlers.GetParsedMusics()

		if err != nil {
			return err
		}

		for _, composer := range composers {

			q := db.Insert("music_composer", dbx.Params{
				"id": composer.ID,
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
				q := db.Insert("music_song", dbx.Params{
					"composer_id": song.ComposerID,
					"title": song.Title,
					"url": song.URL,
					"source": song.Source,
				})
				
				_, err = q.Execute()

				if err != nil {
					return err
				}
			}

		}

		return nil
	}, func(db dbx.Builder) error {
		q := db.DropTable("music_song")
		_, err := q.Execute()

		if err != nil {
			log.Printf("Error executing drop music_song query: %v", err)
			return err
		}

		q = db.DropTable("music_composer")
		_, err = q.Execute()

		if err != nil {
			log.Printf("Error executing drop music_composer query: %v", err)
			return err
		}

		return err
	})
}
