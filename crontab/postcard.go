package crontab

import (
	"fmt"
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

	scheduler.MustAdd("postcards", "*/1 * * * *", func() {
		records, err := app.Dao().FindRecordsByFilter(
			"postcards",         // collection
			"status = 'queued'", // filter
			"",                  // sort
			0,                   // limit
			0,                   // offset
		)

		if err != nil {
			app.Logger().Error("Error fetching postcards", err)
			return
		}

		mailClient := app.NewMailClient()

		for _, r := range records {

			recipientsRaw := r.GetString("recipients")
			recipientsSlice := strings.Split(recipientsRaw, ",")
			recipients := []mail.Address{}

			for _, recipient := range recipientsSlice {
				recipients = append(recipients, mail.Address{Address: recipient})
			}

			for i, rec := range recipients {

				app.Logger().Info("Sending postcard to", fmt.Sprintf("%d", i), r.GetId())

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

				if err := mailClient.Send(message); err != nil {
					fmt.Println("Error sending postcard", err)
					app.Logger().Error("Error sending postcard", err)
					return
				}
			}

			r.Set("status", "sent")
			r.Set("sent_at", time.Now().Unix())

			if err := app.Dao().SaveRecord(r); err != nil {
				app.Logger().Error("Error updating postcard record", err)
			}

		}
	})

}
