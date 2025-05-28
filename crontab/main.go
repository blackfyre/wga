package crontab

import (
	"github.com/pocketbase/pocketbase"
)

func RegisterCronJobs(app *pocketbase.PocketBase) {
	app.Logger().Debug("Registering cron jobs...")
	sendPostcards(app)
	generateSiteMap(app)

}
