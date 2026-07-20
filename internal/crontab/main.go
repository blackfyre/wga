package crontab

import (
	"github.com/blackfyre/wga/internal/config"
	"github.com/pocketbase/pocketbase"
)

func RegisterCronJobs(app *pocketbase.PocketBase, postcards config.Postcards, sitemapConfig config.Sitemap) {
	app.Logger().Debug("Registering cron jobs...")
	sendPostcards(app, postcards)
	generateSiteMap(app, sitemapConfig)

}
