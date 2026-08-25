package crontab

import (
	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/contributors"
	"github.com/blackfyre/wga/internal/itineraries"
	"github.com/pocketbase/pocketbase"
)

func RegisterCronJobs(app *pocketbase.PocketBase, postcards config.Postcards, sitemapConfig config.Sitemap, contributorRefresh contributors.RefreshJob) {
	app.Logger().Debug("Registering cron jobs...")
	sendPostcards(app, postcards)
	refreshContributors(app, contributorRefresh)
	generateSiteMap(app, sitemapConfig)
	itineraries.RegisterPurgeJob(app)
	registerParticipationPurge(app)
}
