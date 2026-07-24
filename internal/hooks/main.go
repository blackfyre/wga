package hooks

import "github.com/pocketbase/pocketbase/core"

func RegisterHooks(app core.App) {
	app.Logger().Debug("Registering hooks...")
	fileDownloadHook(app)
	guestbookYearsCacheHook(app)
}
