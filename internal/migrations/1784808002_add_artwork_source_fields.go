package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// artworkColourImageHashIndex lets the four-basis related-work resolver quickly
// select published artworks that carry an image-derived colour profile.
const artworkColourImageHashIndex = "pbx_artwork_colour_image_hash"

func init() {
	m.Register(addArtworkSourceFields, removeArtworkSourceFields)
}

// addArtworkSourceFields carries the producer's source URL/path, raw and
// enriched commentary provenance, and image-derived colour profile onto the
// artworks collection. Every field is optional and stays empty until the
// real-data release supplies it; the embedded synthetic source and older
// exports leave them unset.
//
// It shares the 1784808002 bootstrap timestamp and sorts before
// "seed_synthetic_data" so the fields exist before seed.Import records producer
// values against them.
func addArtworkSourceFields(app core.App) error {
	artworks, err := app.FindCollectionByNameOrId("artworks")
	if err != nil {
		return err
	}

	if artworks.Fields.GetByName("source_url") == nil {
		artworks.Fields.Add(textField("source_url", false, false))
	}
	if artworks.Fields.GetByName("source_path") == nil {
		artworks.Fields.Add(textField("source_path", false, false))
	}
	if artworks.Fields.GetByName("source_comment") == nil {
		artworks.Fields.Add(textField("source_comment", false, false))
	}
	if artworks.Fields.GetByName("colour_palette") == nil {
		artworks.Fields.Add(jsonField("colour_palette", false))
	}
	if artworks.Fields.GetByName("colour_signature") == nil {
		artworks.Fields.Add(jsonField("colour_signature", false))
	}
	if artworks.Fields.GetByName("colour_profile_version") == nil {
		artworks.Fields.Add(textField("colour_profile_version", false, false))
	}
	if artworks.Fields.GetByName("colour_image_hash") == nil {
		artworks.Fields.Add(textField("colour_image_hash", false, false))
	}

	artworks.Indexes = appendIndex(artworks.Indexes,
		"CREATE INDEX `"+artworkColourImageHashIndex+"` ON `Artworks` (`published`, `colour_image_hash`)",
	)

	return app.Save(artworks)
}

// removeArtworkSourceFields drops the colour-profile query index but keeps the
// source-owned fields. Those fields are authoritative once the real-data release
// carries them, so rollback must not destroy source-backed data; operators
// disable the affected public routes until the active implementation re-adds the
// index.
func removeArtworkSourceFields(app core.App) error {
	artworks, err := app.FindCollectionByNameOrId("artworks")
	if err != nil {
		return err
	}

	artworks.Indexes = removeIndex(artworks.Indexes, artworkColourImageHashIndex)

	return app.Save(artworks)
}
