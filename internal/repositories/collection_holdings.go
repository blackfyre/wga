package repositories

import (
	"github.com/blackfyre/wga/internal/utils"
	"github.com/pocketbase/pocketbase/core"
)

const collectionHoldingsCacheKey = "artworks:search:collection-holdings"

// CollectionHolding is one location and its eligible published-work count.
// Zero-count locations are retained so selected values can be resolved from the
// complete projection without a request-specific lookup.
type CollectionHolding struct {
	Value string `db:"value"`
	Label string `db:"label"`
	Count int    `db:"holding_count"`
}

// LoadCollectionHoldings returns the application-scoped counted location
// projection. Counts match the artwork-search eligibility predicate.
func LoadCollectionHoldings(app core.App) ([]CollectionHolding, error) {
	return collectionHoldingsWithLoader(app, func() ([]CollectionHolding, error) {
		rows := []CollectionHolding{}
		err := app.DB().NewQuery(`
			SELECT
				locations.id AS value,
				locations.name AS label,
				COUNT(DISTINCT artworks.id) AS holding_count
			FROM locations
			LEFT JOIN artworks
				ON artworks.current_location_id = locations.id
				AND artworks.published = TRUE
				AND json_array_length(artworks.author) > 0
				AND EXISTS (
					SELECT 1 FROM artists
					WHERE artists.id = json_extract(artworks.author, '$[0]')
				)
			GROUP BY locations.id, locations.name`).All(&rows)
		return rows, err
	})
}

func collectionHoldingsWithLoader(app core.App, load func() ([]CollectionHolding, error)) ([]CollectionHolding, error) {
	return utils.GetOrLoadCachedValue(app, collectionHoldingsCacheKey, 0, load)
}

// InvalidateCollectionHoldings advances the projection generation so the next
// lookup observes current persisted locations, artworks, and first authors.
func InvalidateCollectionHoldings(app core.App) {
	utils.DeleteCachedValue(app, collectionHoldingsCacheKey)
}
