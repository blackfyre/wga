package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// This migration shares the 1784808002 bootstrap timestamp with the synthetic
// seed migration. Its filename sorts after "add_collection_data" (the locations
// taxonomy it relates to) and before "seed_synthetic_data", so the canonical
// relationship collections and artwork relations exist before seed.Import
// records the producer values against them. An existing database applies it as
// an ordinary pending migration.
const (
	relationshipPathsIndexPrimaryAuthor   = "pbx_artwork_primary_author"
	relationshipPathsIndexAliasKey        = "pbx_location_alias_comparison_key"
	relationshipPathsIndexTechniqueKey    = "pbx_technique_comparison_key"
	relationshipPathsSeriesKindSeries     = "series"
	relationshipPathsSeriesKindAltarpiece = "altarpiece"
)

func init() {
	m.Register(addRelationshipPaths, removeRelationshipPaths)
}

// addRelationshipPaths creates the canonical subjects, series, techniques, and
// location-alias collections and the explicit artwork relations used for
// related-artwork paths. Current museums and original locations both relate to
// the producer-owned locations collection, whose museum and is_public flags
// classify institutions and private collections; current_location_id remains
// the current-museum relation, and the existing school and art_period_id
// relations remain the school/workshop and art-period relations.
//
// The legacy author relation is retained unchanged: primary_author is optional
// because a legacy record with several authors cannot be assigned one without
// curator review. Every collection and field carries no API rules, so they stay
// operationally private and editable only in the PocketBase dashboard.
func addRelationshipPaths(app core.App) error {
	if err := ensureRelationshipCollection(app, "Subjects", "subjects",
		textField("name", true, true),
	); err != nil {
		return err
	}
	if err := ensureRelationshipCollection(app, "Series", "series",
		textField("name", true, true),
		selectField("kind", []string{relationshipPathsSeriesKindSeries, relationshipPathsSeriesKindAltarpiece}, true),
	); err != nil {
		return err
	}
	if err := ensureRelationshipCollection(app, "Techniques", "techniques",
		textField("name", true, true),
		textField("comparison_key", true, false),
	); err != nil {
		return err
	}
	techniques, err := app.FindCollectionByNameOrId("techniques")
	if err != nil {
		return err
	}
	techniques.Indexes = appendIndex(techniques.Indexes,
		"CREATE UNIQUE INDEX `"+relationshipPathsIndexTechniqueKey+"` ON `Techniques` (`comparison_key`)",
	)
	if err := app.Save(techniques); err != nil {
		return err
	}

	if err := ensureRelationshipCollection(app, "Location_aliases", "location_aliases",
		textField("comparison_key", true, true),
		relationField("location", "locations", 1, 1, true, true),
	); err != nil {
		return err
	}
	aliases, err := app.FindCollectionByNameOrId("location_aliases")
	if err != nil {
		return err
	}
	aliases.Indexes = appendIndex(aliases.Indexes,
		"CREATE UNIQUE INDEX `"+relationshipPathsIndexAliasKey+"` ON `Location_aliases` (`comparison_key`)",
	)
	if err := app.Save(aliases); err != nil {
		return err
	}

	artworks, err := app.FindCollectionByNameOrId("artworks")
	if err != nil {
		return err
	}
	addFieldIfMissing(artworks, relationField("primary_author", "artists", 0, 1, false, false))
	addFieldIfMissing(artworks, relationField("co_authors", "artists", 0, 10, false, false))
	addFieldIfMissing(artworks, relationField("subjects", "subjects", 0, 0, false, false))
	addFieldIfMissing(artworks, relationField("series", "series", 0, 0, false, false))
	addFieldIfMissing(artworks, relationField("original_locations", "locations", 0, 0, false, false))
	addFieldIfMissing(artworks, relationField("techniques", "techniques", 0, 0, false, false))
	addFieldIfMissing(artworks, textField("source_location", false, false))
	artworks.Indexes = appendIndex(artworks.Indexes,
		"CREATE INDEX `"+relationshipPathsIndexPrimaryAuthor+"` ON `Artworks` (`published`, `primary_author`)",
	)

	return app.Save(artworks)
}

// ensureRelationshipCollection creates a base collection with the supplied
// fields unless it already exists.
func ensureRelationshipCollection(app core.App, name string, id string, fields ...core.Field) error {
	if _, err := app.FindCollectionByNameOrId(id); err == nil {
		return nil
	}

	return app.Save(currentCollection(name, id, fields...))
}

// removeRelationshipPaths drops only the query index. The canonical collections
// and artwork relations may hold curated or source-backed data, so rollback
// retains them for inspection rather than destroying it.
func removeRelationshipPaths(app core.App) error {
	artworks, err := app.FindCollectionByNameOrId("artworks")
	if err != nil {
		return err
	}

	artworks.Indexes = removeIndex(artworks.Indexes, relationshipPathsIndexPrimaryAuthor)

	return app.Save(artworks)
}
