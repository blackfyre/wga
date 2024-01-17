package crontab

import (
	"net/mail"
	"os"
	"strings"
	"time"

	"github.com/blackfyre/wga/assets"
	"github.com/blackfyre/wga/utils"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/tools/cron"
	"github.com/pocketbase/pocketbase/tools/mailer"
)

func sendPostcards(app *pocketbase.PocketBase, scheduler *cron.Cron) {

	scheduler.MustAdd("hello", "*/1 * * * *", func() {
		records, err := app.Dao().FindRecordsByFilter(
			"postcards",         // collection
			"status = 'queued'", // filter
			"",                  // sort
			0,                   // limit
			0,                   // offset
		)

		if err != nil {
			panic(err)
		}

		mailCleint := app.NewMailClient()

		for _, r := range records {

			recipientsRaw := r.GetString("recipients")
			recipientsSlice := strings.Split(recipientsRaw, ",")
			recipients := []mail.Address{}

			for _, recipient := range recipientsSlice {
				recipients = append(recipients, mail.Address{Address: recipient})
			}

			for _, rec := range recipients {

				html, err := assets.RenderEmail("postcard:notification", map[string]any{
					"SenderName": r.GetString("sender_name"),
					"PickUpUrl":  utils.AssetUrl("/postcards?p=" + r.GetString("id")),
					"Title":      "",
					"LogoUrl":    utils.AssetUrl("/assets/images/logo.png"),
				})

				if err != nil {
					panic(err)
				}

				message := &mailer.Message{
					From: mail.Address{
						Name:    os.Getenv("WGA_SENDER_NAME"),
						Address: os.Getenv("WGA_SENDER_ADDRESS"),
					},
					To:      []mail.Address{rec},
					Subject: "You got a postcard from " + r.GetString("sender_name") + "!",
					HTML:    html,
				}

				if err := mailCleint.Send(message); err != nil {
					panic(err)
				}
			}

			r.Set("status", "sent")
			r.Set("sent_at", time.Now().Unix())

			if err := app.Dao().SaveRecord(r); err != nil {
				panic(err)
			}

		}
	})

}
