package hooks

import (
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func fileDownloadHook(app *pocketbase.PocketBase) {
	app.Logger().Debug("Registering file download hook...")
	app.OnFileDownloadRequest().BindFunc(func(e *core.FileDownloadRequestEvent) error {
		// e.App
		// e.Collection
		// e.Record
		// e.FileField
		// e.ServedPath
		// e.ServedName
		// and all RequestEvent fields...
		app.Logger().Debug("File download request", "event", e)

		return e.Next()
	})
}
