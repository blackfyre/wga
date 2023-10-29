package crontab

import (
	"blackfyre.ninja/wga/utils/sitemap"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/tools/cron"
)

func generateSiteMap(app *pocketbase.PocketBase, scheduler *cron.Cron) {
	scheduler.MustAdd("hello", "0 0 * * *", func() {
		sitemap.GenerateSiteMap(app)
	})
}
