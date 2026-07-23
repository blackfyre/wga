package migrations

import (
	"database/sql"
	"errors"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(addSyntheticSourceFields, removeSyntheticSourceFields)
}

func addSyntheticSourceFields(app core.App) error {
	if err := addSyntheticReferenceCollections(app); err != nil {
		return err
	}

	if err := addSyntheticArtistFields(app); err != nil {
		return err
	}
	if err := addSyntheticArtworkFields(app); err != nil {
		return err
	}
	if err := addSyntheticGlossaryFields(app); err != nil {
		return err
	}

	return addSyntheticMusicFields(app)
}

func addSyntheticReferenceCollections(app core.App) error {
	professions := core.NewBaseCollection("Professions")
	professions.Id = "professions"
	professions.MarkAsNew()
	professions.Fields.Add(
		&core.TextField{Id: "professions_name", Name: "name", Required: true, Presentable: true},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	professions.AddIndex("pbx_professions_name", true, "name", "")
	if err := app.Save(professions); err != nil {
		return err
	}

	biographies := core.NewBaseCollection("Biographies")
	biographies.Id = "biographies"
	biographies.MarkAsNew()
	biographies.Fields.Add(
		&core.RelationField{Id: "biographies_artist", Name: "artist", CollectionId: "artists", MinSelect: 1, MaxSelect: 1},
		&core.TextField{Id: "biographies_raw_life_detail", Name: "raw_life_detail", Required: true},
		&core.EditorField{Id: "biographies_raw_html", Name: "raw_biography_html", Required: true},
		&core.EditorField{Id: "biographies_html", Name: "biography_html", Required: true},
		&core.TextField{Id: "biographies_text", Name: "biography_text", Required: true},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	biographies.AddIndex("pbx_biographies_artist", true, "artist", "")
	if err := app.Save(biographies); err != nil {
		return err
	}

	biographyLinks := core.NewBaseCollection("Biography_links")
	biographyLinks.Id = "biography_links"
	biographyLinks.MarkAsNew()
	biographyLinks.Fields.Add(
		&core.RelationField{Id: "biography_links_biography", Name: "biography", CollectionId: "biographies", MinSelect: 1, MaxSelect: 1},
		&core.TextField{Id: "biography_links_type", Name: "link_type", Required: true},
		&core.TextField{Id: "biography_links_target", Name: "target_path", Required: true},
		&core.TextField{Id: "biography_links_text", Name: "link_text", Required: true},
		&core.NumberField{Id: "biography_links_order", Name: "link_order"},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	if err := app.Save(biographyLinks); err != nil {
		return err
	}

	attributions := core.NewBaseCollection("Source_attributions")
	attributions.Id = "source_attributions"
	attributions.MarkAsNew()
	attributions.Fields.Add(
		&core.NumberField{Id: "source_attributions_order", Name: "attribution_order", Required: true},
		&core.TextField{Id: "source_attributions_category", Name: "category", Required: true},
		&core.TextField{Id: "source_attributions_subcategory", Name: "subcategory"},
		&core.TextField{Id: "source_attributions_title", Name: "title", Required: true},
		&core.TextField{Id: "source_attributions_citation", Name: "citation", Required: true},
		&core.TextField{Id: "source_attributions_url", Name: "source_url", Required: true},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	attributions.AddIndex("pbx_source_attributions_order", true, "attribution_order", "")

	return app.Save(attributions)
}

func addSyntheticArtistFields(app core.App) error {
	artists, err := app.FindCollectionByNameOrId("artists")
	if err != nil {
		return err
	}

	artists.Fields.Add(
		&core.TextField{Id: "artists_source_path", Name: "source_path"},
		&core.TextField{Id: "artists_source_hash", Name: "source_hash"},
		&core.TextField{Id: "artists_debug_hash", Name: "debug_hash"},
		&core.TextField{Id: "artists_source_display_name", Name: "source_display_name"},
		&core.TextField{Id: "artists_source_url", Name: "artist_url"},
		&core.TextField{Id: "artists_activity_text", Name: "activity_text"},
		&core.NumberField{Id: "artists_activity_start", Name: "activity_start_year"},
		&core.NumberField{Id: "artists_activity_end", Name: "activity_end_year"},
		&core.TextField{Id: "artists_index_path", Name: "artist_index_path"},
		&core.TextField{Id: "artists_biography_image_path", Name: "biography_image_path"},
		&core.RelationField{Id: "artists_professions", Name: "professions", CollectionId: "professions", MaxSelect: 20},
	)

	return app.Save(artists)
}

func addSyntheticArtworkFields(app core.App) error {
	artworks, err := app.FindCollectionByNameOrId("artworks")
	if err != nil {
		return err
	}

	artworks.Fields.Add(
		&core.TextField{Id: "artworks_date_text", Name: "date_text"},
		&core.NumberField{Id: "artworks_date_start", Name: "date_start"},
		&core.NumberField{Id: "artworks_date_end", Name: "date_end"},
		&core.BoolField{Id: "artworks_is_circa", Name: "is_circa"},
		&core.TextField{Id: "artworks_date_qualifier", Name: "date_qualifier"},
		&core.TextField{Id: "artworks_technique_without_dimensions", Name: "technique_without_dimensions"},
		&core.TextField{Id: "artworks_dimensions", Name: "dimensions"},
		&core.TextField{Id: "artworks_location", Name: "location"},
		&core.TextField{Id: "artworks_source_url", Name: "source_url"},
		&core.TextField{Id: "artworks_source_image_path", Name: "source_image_path"},
		&core.TextField{Id: "artworks_output_image_path", Name: "output_image_path"},
		&core.NumberField{Id: "artworks_source_row", Name: "source_row"},
	)

	return app.Save(artworks)
}

func addSyntheticGlossaryFields(app core.App) error {
	glossary, err := app.FindCollectionByNameOrId("glossary")
	if err != nil {
		return err
	}

	glossary.Fields.Add(
		&core.TextField{Id: "glossary_anchor", Name: "anchor"},
		&core.TextField{Id: "glossary_source_page", Name: "source_page"},
		&core.NumberField{Id: "glossary_sort_order", Name: "sort_order"},
	)

	return app.Save(glossary)
}

func addSyntheticMusicFields(app core.App) error {
	songs, err := app.FindCollectionByNameOrId("music_song")
	if err != nil {
		return err
	}

	songs.Fields.Add(
		&core.NumberField{Id: "music_song_track_order", Name: "track_order"},
		&core.TextField{Id: "music_song_period", Name: "period"},
		&core.TextField{Id: "music_song_origin", Name: "origin"},
		&core.TextField{Id: "music_song_playback_url", Name: "playback_url"},
		&core.SelectField{Id: "music_song_media_format", Name: "media_format", Values: []string{"mp3", "m3u"}, MaxSelect: 1},
		&core.TextField{Id: "music_song_player_url", Name: "player_url"},
		&core.TextField{Id: "music_song_local_path", Name: "local_path"},
		&core.NumberField{Id: "music_song_part", Name: "part"},
		&core.NumberField{Id: "music_song_part_count", Name: "part_count"},
	)

	return app.Save(songs)
}

func removeSyntheticSourceFields(app core.App) error {
	for _, collectionName := range []string{
		"biography_links",
		"biographies",
		"source_attributions",
	} {
		if err := deleteSyntheticCollection(app, collectionName); err != nil {
			return err
		}
	}

	if err := removeSyntheticFields(app, "artists", []string{
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
	if err := removeSyntheticFields(app, "artworks", []string{
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
	if err := removeSyntheticFields(app, "glossary", []string{
		"anchor",
		"source_page",
		"sort_order",
	}); err != nil {
		return err
	}
	if err := removeSyntheticFields(app, "music_song", []string{
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

	return deleteSyntheticCollection(app, "professions")
}

func deleteSyntheticCollection(app core.App, name string) error {
	collection, err := app.FindCollectionByNameOrId(name)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}

	return app.Delete(collection)
}

func removeSyntheticFields(app core.App, collectionName string, names []string) error {
	collection, err := app.FindCollectionByNameOrId(collectionName)
	if err != nil {
		return err
	}

	for _, name := range names {
		collection.Fields.RemoveByName(name)
	}

	return app.Save(collection)
}
