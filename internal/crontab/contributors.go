package crontab

import (
	"context"

	"github.com/blackfyre/wga/internal/contributors"
	"github.com/blackfyre/wga/internal/logging"
	"github.com/pocketbase/pocketbase/core"
)

const contributorRefreshSchedule = "0 */6 * * *"

func refreshContributors(app core.App, job contributors.RefreshJob) {
	app.Logger().Info("Contributor refresh schedule registered", "event", "contributors.refresh.schedule_registered")
	app.Cron().MustAdd("contributors-refresh", contributorRefreshSchedule, func() {
		runID := logging.NewRunID()
		logger := logging.RunLogger(app, runID)
		logger.Info("Contributor refresh started", "event", "contributors.refresh.run", "outcome", "started")

		if err := job.Run(context.Background(), runID); err != nil {
			logger.Error("Contributor refresh failed",
				"event", "contributors.refresh.run",
				"outcome", "failed",
				"error_type", logging.ErrorType(err),
				"error", logging.Redact(err),
			)
			return
		}

		logger.Info("Contributor refresh completed", "event", "contributors.refresh.run", "outcome", "completed")
	})
}
