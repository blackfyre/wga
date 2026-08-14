package migrations

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(addArtistPortrait, func(core.App) error {
		return fmt.Errorf("artist portrait migration cannot be rolled back safely")
	})
}

func addArtistPortrait(app core.App) error {
	artists, err := app.FindCollectionByNameOrId("artists")
	if err != nil {
		return err
	}
	if artists.Fields.GetByName("portrait") != nil {
		return nil
	}

	artists.Fields.Add(&core.FileField{
		Name:      "portrait",
		MaxSelect: 1,
		MimeTypes: []string{"image/jpeg", "image/png"},
		MaxSize:   5 * 1024 * 1024,
		Thumbs:    []string{"320x320"},
	})

	return app.Save(artists)
}
