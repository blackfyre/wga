package hooks

import (
	"github.com/blackfyre/wga/internal/constants"
	"github.com/blackfyre/wga/internal/repositories"
	"github.com/pocketbase/pocketbase/core"
)

func collectionHoldingsCacheHook(app core.App) {
	invalidateHoldings := func(e *core.RecordEvent) error {
		repositories.InvalidateCollectionHoldings(e.App)
		return e.Next()
	}

	app.OnRecordAfterCreateSuccess(constants.CollectionLocations).BindFunc(invalidateHoldings)
	app.OnRecordAfterUpdateSuccess(constants.CollectionLocations).BindFunc(invalidateHoldings)
	app.OnRecordAfterDeleteSuccess(constants.CollectionLocations).BindFunc(invalidateHoldings)

	invalidateArtistProjections := func(e *core.RecordEvent) error {
		repositories.AdvanceArtworkCatalogueRevision(e.App)
		repositories.InvalidateCollectionHoldings(e.App)
		repositories.InvalidateArtistAvailability(e.App)
		return e.Next()
	}
	app.OnRecordAfterCreateSuccess(constants.CollectionArtists).BindFunc(invalidateArtistProjections)
	app.OnRecordAfterUpdateSuccess(constants.CollectionArtists).BindFunc(invalidateArtistProjections)
	app.OnRecordAfterDeleteSuccess(constants.CollectionArtists).BindFunc(invalidateArtistProjections)
}
