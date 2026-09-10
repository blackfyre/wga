package hooks

import (
	"github.com/blackfyre/wga/internal/constants"
	"github.com/blackfyre/wga/internal/repositories"
	"github.com/pocketbase/pocketbase/core"
)

func artworkAvailabilityCacheHook(app core.App) {
	invalidate := func(e *core.RecordEvent) error {
		repositories.InvalidateArtistAvailability(e.App)
		return e.Next()
	}

	app.OnRecordAfterCreateSuccess(constants.CollectionArtworks).BindFunc(invalidate)
	app.OnRecordAfterUpdateSuccess(constants.CollectionArtworks).BindFunc(invalidate)
	app.OnRecordAfterDeleteSuccess(constants.CollectionArtworks).BindFunc(invalidate)
}
