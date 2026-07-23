package migrations

import (
	"database/sql"
	"errors"

	"github.com/blackfyre/wga/internal/utils/seed"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const syntheticMusicSourceMaxSize = 15 * 1024 * 1024

func init() {
	m.Register(seedSyntheticData, func(core.App) error {
		return errors.New("synthetic bootstrap data cannot be rolled back safely")
	})
}

func seedSyntheticData(app core.App) error {
	if err := seed.RequireEmptyApplicationDatabase(app); err != nil {
		if errors.Is(err, seed.ErrApplicationRecords) {
			return nil
		}

		return err
	}

	if err := removeLegacySyntheticSourceSchema(app); err != nil {
		return err
	}
	if err := increaseMusicSourceMaxSize(app); err != nil {
		return err
	}

	err := seed.ImportEmbedded(app)
	if errors.Is(err, seed.ErrApplicationRecords) {
		return nil
	}

	return err
}

func removeLegacySyntheticSourceSchema(app core.App) error {
	for _, collectionName := range []string{
		"biography_links",
		"biographies",
		"source_attributions",
	} {
		if err := deleteLegacySyntheticCollection(app, collectionName); err != nil {
			return err
		}
	}

	if err := removeLegacySyntheticFields(app, "artists", []string{
		"source_path",
		"source_hash",
		"debug_hash",
		"source_display_name",
		"artist_url",
		"activity_text",
		"activity_start_year",
		"activity_end_year",
		"artist_index_path",
		"biography_image_path",
		"professions",
	}); err != nil {
		return err
	}
	if err := removeLegacySyntheticFields(app, "artworks", []string{
		"date_text",
		"date_start",
		"date_end",
		"is_circa",
		"date_qualifier",
		"technique_without_dimensions",
		"dimensions",
		"location",
		"source_url",
		"source_image_path",
		"output_image_path",
		"source_row",
	}); err != nil {
		return err
	}
	if err := removeLegacySyntheticFields(app, "glossary", []string{
		"anchor",
		"source_page",
		"sort_order",
	}); err != nil {
		return err
	}
	if err := removeLegacySyntheticFields(app, "music_song", []string{
		"track_order",
		"period",
		"origin",
		"playback_url",
		"media_format",
		"player_url",
		"local_path",
		"part",
		"part_count",
	}); err != nil {
		return err
	}

	return deleteLegacySyntheticCollection(app, "professions")
}

func deleteLegacySyntheticCollection(app core.App, name string) error {
	collection, err := app.FindCollectionByNameOrId(name)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}

	return app.Delete(collection)
}

func removeLegacySyntheticFields(app core.App, collectionName string, names []string) error {
	collection, err := app.FindCollectionByNameOrId(collectionName)
	if err != nil {
		return err
	}

	for _, name := range names {
		collection.Fields.RemoveByName(name)
	}

	return app.Save(collection)
}

func increaseMusicSourceMaxSize(app core.App) error {
	collection, err := app.FindCollectionByNameOrId("music_song")
	if err != nil {
		return err
	}

	field, ok := collection.Fields.GetByName("source").(*core.FileField)
	if !ok {
		return errors.New("music_song source field is not a file field")
	}
	if field.MaxSize >= syntheticMusicSourceMaxSize {
		return nil
	}

	field.MaxSize = syntheticMusicSourceMaxSize

	return app.Save(collection)
}
