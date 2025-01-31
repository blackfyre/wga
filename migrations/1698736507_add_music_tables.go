package migrations

import (
	"log"

	"github.com/blackfyre/wga/handlers"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {

		composerCollection := core.NewBaseCollection("Music_composer")

		composerCollection.Name = "Music_composer"
		composerCollection.System = false
		composerCollection.Id = "music_composer"
		composerCollection.MarkAsNew()

		composerCollection.Fields.Add(
			&core.TextField{
				Id:          "music_composer_name",
				Name:        "name",
				Required:    true,
				Presentable: true,
			},
			&core.SelectField{
				Id:       "music_composer_century",
				Name:     "century",
				Values:   []string{"12", "13", "14", "15", "16", "17", "18", "19", "20", "21"},
				MaxSelect: 1,
				Required: true,
			},
			&core.TextField{
				Id: "music_composer_language",
				Name: "language",
				Presentable: true,
			},
		)


		err := app.Save(composerCollection)
		if err != nil {
			// Handle the error, for example log it and return
			log.Printf("Error saving collection: %v", err)
			return err
		}

		songCollection := core.NewBaseCollection("Music_song")

		songCollection.Name = "Music_song"
		songCollection.System = false
		songCollection.Id = "music_song"
		songCollection.MarkAsNew()

		songCollection.Fields.Add(
			&core.TextField{
				Id:          "music_song_title",
				Name:        "title",
				Required:    true,
				Presentable: true,
			},
			&core.RelationField{
				Id:           "music_song_composer",
				Name:         "composer",
				CollectionId: "music_composer",
				MinSelect:    1,
			},
			&core.FileField{
				Id:          "music_song_source",
				Name:        "source",
				MimeTypes:   []string{"audio/mpeg", "audio/mp3"},
				MaxSize:     1024 * 1024 * 5,
				Required: true,
			},
		)

		err = app.Save(songCollection)

		if err != nil {
			return err
		}

		composers := handlers.GetParsedMusics()

		for _, composer := range composers {

			composerRecord := core.NewRecord(composerCollection)
			composerRecord.Set("name", composer.Name)
			composerRecord.Set("century", composer.Century)
			composerRecord.Set("language", composer.Language)


			err = app.Save(composerRecord)

			if err != nil {
				app.Logger().Error("Error saving composer record: %v", err)
			}
		}

		return nil
	}, func(app core.App) error {

		err := deleteCollection(app, "Music_composer")

		if err != nil {
			return err
		}

		err = deleteCollection(app, "Music_song")

		if err != nil {
			return err
		}

		return nil
	})
}
