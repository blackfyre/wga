package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const artworkDateRangeIndex = "pbx_artwork_date_range"

func init() {
	m.Register(addArtworkDateSpans, removeArtworkDateSpans)
}

// addArtworkDateSpans carries the approved artwork creation-date range onto the
// artworks collection. The fields mirror the producer contract (`date_start`,
// `date_end`, `is_circa`, `date_qualifier`, `timeframe_text`) and remain absent
// from the WGA schema until the real-data release supplies them, so the timeline
// can derive its window, density, named spans, marks, and links without touching
// the shared seed importer.
func addArtworkDateSpans(app core.App) error {
	artworks, err := app.FindCollectionByNameOrId("artworks")
	if err != nil {
		return err
	}

	if artworks.Fields.GetByName("date_start") == nil {
		artworks.Fields.Add(numberField("date_start", false))
	}
	if artworks.Fields.GetByName("date_end") == nil {
		artworks.Fields.Add(numberField("date_end", false))
	}
	if artworks.Fields.GetByName("is_circa") == nil {
		artworks.Fields.Add(boolField("is_circa", false))
	}
	if artworks.Fields.GetByName("date_qualifier") == nil {
		artworks.Fields.Add(textField("date_qualifier", false, false))
	}
	if artworks.Fields.GetByName("timeframe_text") == nil {
		artworks.Fields.Add(textField("timeframe_text", false, false))
	}

	artworks.Indexes = appendIndex(artworks.Indexes,
		"CREATE INDEX `"+artworkDateRangeIndex+"` ON `Artworks` (`published`, `date_start`, `date_end`, `id`)",
	)

	return app.Save(artworks)
}

// removeArtworkDateSpans drops the timeline query index but keeps the date-span
// fields. The fields are authoritative once the real-data release carries them,
// so rollback must not destroy source-backed data; operators disable the public
// timeline route until the active implementation re-adds the index.
func removeArtworkDateSpans(app core.App) error {
	artworks, err := app.FindCollectionByNameOrId("artworks")
	if err != nil {
		return err
	}

	artworks.Indexes = removeIndex(artworks.Indexes, artworkDateRangeIndex)

	return app.Save(artworks)
}
