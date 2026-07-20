package crontab

import (
	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/utils/sitemap"
	"github.com/pocketbase/pocketbase"
)

func generateSiteMap(app *pocketbase.PocketBase, sitemapConfig config.Sitemap) {
	app.Logger().Debug("Registering cron job for sitemap generation...")
	app.Cron().MustAdd("sitemap", "0 0 * * *", func() {
		sitemap.GenerateSiteMap(app, sitemapConfig)
	})
}
