package crontab

import (
	"github.com/blackfyre/wga/utils/sitemap"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/tools/cron"
)

func generateSiteMap(app *pocketbase.PocketBase, scheduler *cron.Cron) {
	scheduler.MustAdd("sitemap", "0 0 * * *", func() {
		sitemap.GenerateSiteMap(app)
	})
}
