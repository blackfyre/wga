package hooks

import (
	"github.com/blackfyre/wga/internal/constants"
	"github.com/blackfyre/wga/internal/repositories"
	"github.com/pocketbase/pocketbase/core"
)

func collectionHoldingsCacheHook(app core.App) {
	invalidate := func(e *core.RecordEvent) error {
		repositories.InvalidateCollectionHoldings(e.App)
		return e.Next()
	}

	for _, collection := range []string{constants.CollectionLocations, constants.CollectionArtists} {
		app.OnRecordAfterCreateSuccess(collection).BindFunc(invalidate)
		app.OnRecordAfterUpdateSuccess(collection).BindFunc(invalidate)
		app.OnRecordAfterDeleteSuccess(collection).BindFunc(invalidate)
	}
}
