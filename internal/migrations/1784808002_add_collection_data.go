package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// This migration shares the 1784808002 bootstrap timestamp with the synthetic
// seed migration. Its filename sorts before "seed_synthetic_data" (and after
// "add_artist_selections") so the source-owned locations taxonomy and the
// artwork archive/date/location/period fields exist before seed.Import
// records the producer values against them.
//
// The date-span fields are also declared here, before the seed migration, so
// the seed importer can carry the authoritative producer values; the later
// timeline date-span migration (1787529600) remains idempotent and still owns
// the timeline's dedicated date-range index.
const (
	collectionDataIndexSourceRow       = "pbx_artwork_source_row"
	collectionDataIndexCurrentLocation = "pbx_artwork_current_location"
	collectionDataIndexArtPeriod       = "pbx_artwork_art_period"
)

func init() {
	m.Register(addCollectionData, removeCollectionData)
}

// addCollectionData creates the producer locations taxonomy and carries the
// serial artwork archive order, creation-date span, current-location relation,
// art-period relation onto the artworks collection. The
// locations collection and artwork fields carry no API rules, so they remain
// operationally private like artists and artworks.
func addCollectionData(app core.App) error {
	if _, err := app.FindCollectionByNameOrId("locations"); err != nil {
		locations := core.NewBaseCollection("Locations")
		locations.Id = "locations"
		locations.MarkAsNew()
		locations.Fields.Add(
			textField("name", true, true),
			textField("city", false, false),
			textField("country", false, false),
			boolField("museum", false),
			boolField("is_public", false),
		)
		locations.Fields.Add(
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
		)
		if err := app.Save(locations); err != nil {
			return err
		}
	}

	artworks, err := app.FindCollectionByNameOrId("artworks")
	if err != nil {
		return err
	}

	if artworks.Fields.GetByName("source_row") == nil {
		artworks.Fields.Add(numberField("source_row", false))
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
	if artworks.Fields.GetByName("current_location_id") == nil {
		artworks.Fields.Add(relationField("current_location_id", "locations", 0, 1, false, false))
	}
	if artworks.Fields.GetByName("art_period_id") == nil {
		artworks.Fields.Add(relationField("art_period_id", "art_periods", 0, 1, false, false))
	}
	artworks.Indexes = appendIndex(artworks.Indexes,
		"CREATE INDEX `"+collectionDataIndexSourceRow+"` ON `Artworks` (`published`, `source_row`, `id`)",
	)
	artworks.Indexes = appendIndex(artworks.Indexes,
		"CREATE INDEX `"+collectionDataIndexCurrentLocation+"` ON `Artworks` (`published`, `current_location_id`)",
	)
	artworks.Indexes = appendIndex(artworks.Indexes,
		"CREATE INDEX `"+collectionDataIndexArtPeriod+"` ON `Artworks` (`published`, `art_period_id`)",
	)

	return app.Save(artworks)
}

// removeCollectionData drops the catalogue/timeline query indexes but keeps the
// source-owned fields and the locations taxonomy. Those fields are
// authoritative once the real-data release carries them, so rollback must not
// destroy source-backed data; operators disable the affected public routes
// until the active implementation re-adds the indexes.
func removeCollectionData(app core.App) error {
	artworks, err := app.FindCollectionByNameOrId("artworks")
	if err != nil {
		return err
	}

	artworks.Indexes = removeIndex(artworks.Indexes,
		collectionDataIndexSourceRow,
		collectionDataIndexCurrentLocation,
		collectionDataIndexArtPeriod,
	)

	return app.Save(artworks)
}
