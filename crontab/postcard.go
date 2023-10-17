package crontab

import (
	"net/mail"
	"os"
	"strings"
	"time"

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
				message := &mailer.Message{
					From: mail.Address{
						Name:    os.Getenv("WGA_SENDER_NAME"),
						Address: os.Getenv("WGA_SENDER_ADDRESS"),
					},
					To:      []mail.Address{rec},
					Subject: "You got a postcard from " + r.GetString("sender_name") + "!",
					Text:    "Your WGA Postcard",
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
