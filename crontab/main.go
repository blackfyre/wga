package crontab

import (
	"github.com/pocketbase/pocketbase"
)

func RegisterCronJobs(app *pocketbase.PocketBase) {

	sendPostcards(app)
	generateSiteMap(app)

}
