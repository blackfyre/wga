package hooks

import (
	"github.com/blackfyre/wga/internal/constants"
	"github.com/blackfyre/wga/internal/utils"
	"github.com/pocketbase/pocketbase/core"
)

func guestbookYearsCacheHook(app core.App) {
	invalidate := func(e *core.RecordEvent) error {
		utils.DeleteCachedValue(e.App, constants.CacheGuestbookYears)
		return e.Next()
	}

	app.OnRecordAfterCreateSuccess(constants.CollectionGuestbook).BindFunc(invalidate)
	app.OnRecordAfterUpdateSuccess(constants.CollectionGuestbook).BindFunc(invalidate)
	app.OnRecordAfterDeleteSuccess(constants.CollectionGuestbook).BindFunc(invalidate)
}
