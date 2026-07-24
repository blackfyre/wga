package hooks

import (
	"github.com/blackfyre/wga/internal/logging"
	"github.com/pocketbase/pocketbase/core"
)

func fileDownloadHook(app core.App) {
	app.Logger().Debug("Registering file download hook...")
	app.OnFileDownloadRequest().BindFunc(func(e *core.FileDownloadRequestEvent) error {
		logFileDownload(app, e)

		return e.Next()
	})
}

func logFileDownload(app core.App, e *core.FileDownloadRequestEvent) {
	logging.RequestLogger(app, e.RequestEvent).Debug("File download served",
		"event", "file.download.served",
		"collection", e.Record.Collection().Name,
		"record_id", e.Record.Id,
		"file_field", e.FileField.Name,
		"outcome", "served",
	)
}
