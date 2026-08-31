package crontab

import (
	"time"

	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/logging"
	"github.com/blackfyre/wga/internal/utils/sitemap"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func generateSiteMap(app *pocketbase.PocketBase, sitemapConfig config.Sitemap) {
	app.Logger().Debug("Registering cron job for sitemap generation...")
	app.Cron().MustAdd("sitemap", "0 0 * * *", func() {
		runSitemap(app, sitemapConfig, "scheduled")
	})
	app.OnServe().BindFunc(func(event *core.ServeEvent) error {
		runSitemap(app, sitemapConfig, "startup")
		return event.Next()
	})
}

func runSitemap(app core.App, sitemapConfig config.Sitemap, trigger string) error {
	runID := logging.NewRunID()
	logger := logging.RunLogger(app, runID)
	started := time.Now()
	result, err := sitemap.GenerateSiteMap(app, sitemapConfig)
	if err != nil {
		logger.Error("Sitemap generation failed",
			"event", "sitemap.generation.failed",
			"trigger", trigger,
			"duration", time.Since(started).String(),
			"error_type", logging.ErrorType(err),
		)
		return err
	}
	logger.Info("Sitemap generation completed",
		"event", "sitemap.generation.completed",
		"trigger", trigger,
		"duration", time.Since(started).String(),
		"url_count", result.URLCount,
		"excluded_count", result.ExcludedCount,
	)
	if result.CleanupErr != nil {
		logger.Warn("Sitemap cleanup failed",
			"event", "sitemap.generation.cleanup_failed",
			"error_type", logging.ErrorType(result.CleanupErr),
		)
	}

	return nil
}
