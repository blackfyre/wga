package hooks

import "github.com/pocketbase/pocketbase"

func RegisterHooks(app *pocketbase.PocketBase) {
	app.Logger().Debug("Registering hooks...")
	fileDownloadHook(app)
}
