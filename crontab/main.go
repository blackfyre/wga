package crontab

import (
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/cron"
)

func RegisterCronJobs(app *pocketbase.PocketBase) {
	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		scheduler := cron.New()

		sendPostcards(app, scheduler)
		generateSiteMap(app, scheduler)

		scheduler.Start()

		return nil
	})
}
