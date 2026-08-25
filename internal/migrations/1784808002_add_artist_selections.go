package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// This migration shares the 1784808002 bootstrap timestamp with the synthetic
// seed migration. Its filename sorts before "seed_synthetic_data" so the
// source-owned art_selections collection exists before seed.Import records
// producer selections against it.
//
// The selection collection and its records are authoritative editorial data
// once the real-data release carries them. Rollback therefore drops only the
// query indexes and never deletes the collection or its seeded records; the
// up migration re-adds the indexes so a rollback is forward-compatible.
const (
	artistSelectionsIndexSourcePath = "pbx_artist_selections_source_path"
	artistSelectionsIndexPublished  = "pbx_artist_selections_published_id"
)

func init() {
	m.Register(addArtistSelections, removeArtistSelections)
}

// addArtistSelections creates the flat source-owned art_selections collection
// matching the producer's pb_schema.json contract, or re-adds its indexes when
// a prior rollback dropped them. The collection carries no API rules, so it
// remains operationally private like artists and artworks.
func addArtistSelections(app core.App) error {
	collection, err := app.FindCollectionByNameOrId("art_selections")
	if err != nil {
		collection = core.NewBaseCollection("Art_selections")
		collection.Id = "art_selections"
		collection.MarkAsNew()
		collection.Fields.Add(
			relationField("artist", "artists", 1, 1, true, false),
			textField("title", true, true),
			textField("context", false, false),
			textField("display_title", true, false),
			editorField("commentary", false),
			relationField("artworks", "artworks", 1, 1000, true, false),
			&core.TextField{Name: "source_path", Required: true, Hidden: true},
			&core.TextField{Name: "source_hash", Required: true, Hidden: true},
			&core.TextField{Name: "content_hash", Required: true, Hidden: true},
			boolField("published", true),
		)
		collection.Fields.Add(
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
		)
	}

	collection.Indexes = appendIndex(collection.Indexes,
		"CREATE UNIQUE INDEX `"+artistSelectionsIndexSourcePath+"` ON `Art_selections` (`source_path`)",
	)
	collection.Indexes = appendIndex(collection.Indexes,
		"CREATE INDEX `"+artistSelectionsIndexPublished+"` ON `Art_selections` (`published`, `id`)",
	)

	return app.Save(collection)
}

// removeArtistSelections drops the selection query indexes but keeps the
// source-owned art_selections collection and its seeded editorial records. The
// collection is authoritative once the real-data release carries it, so
// rollback must not destroy source-backed data; operators disable the affected
// public routes until the active implementation re-adds the indexes.
func removeArtistSelections(app core.App) error {
	collection, err := app.FindCollectionByNameOrId("art_selections")
	if err != nil {
		return err
	}

	collection.Indexes = removeIndex(collection.Indexes,
		artistSelectionsIndexSourcePath,
		artistSelectionsIndexPublished,
	)

	return app.Save(collection)
}
