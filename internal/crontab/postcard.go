package crontab

import (
	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/logging"
	"github.com/blackfyre/wga/internal/postcards"
	"github.com/pocketbase/pocketbase/core"
)

func sendPostcards(app core.App, postcardConfig config.Postcards) {
	app.Logger().Info("Postcard delivery schedule registered", "event", "postcard.delivery.schedule_registered")

	app.Cron().MustAdd("postcards", postcardConfig.Expression(), func() {
		runID := logging.NewRunID()
		logger := logging.RunLogger(app, runID)
		logger.Info("Postcard delivery run started", "event", "postcard.delivery.run", "outcome", "started")

		if err := postcards.ProcessDue(app, app.NewMailClient(), postcardConfig, runID); err != nil {
			logger.Error("Postcard delivery run failed",
				"event", "postcard.delivery.run",
				"outcome", "failed",
				"error_type", logging.ErrorType(err),
				"error", logging.Redact(err),
			)
			return
		}

		logger.Info("Postcard delivery run completed", "event", "postcard.delivery.run", "outcome", "completed")
	})
}
