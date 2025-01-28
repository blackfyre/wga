package crontab

import (
	"github.com/blackfyre/wga/utils/sitemap"
	"github.com/pocketbase/pocketbase"
)

func generateSiteMap(app *pocketbase.PocketBase) {
	app.Cron().MustAdd("sitemap", "0 0 * * *", func() {
		sitemap.GenerateSiteMap(app)
	})
}
