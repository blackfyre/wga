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
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/tools/cron"
	"github.com/pocketbase/pocketbase/tools/mailer"
)

// sendPostcard sends a postcard to the recipients specified in the given record.
// It takes a pointer to a models.Record, a pointer to a pocketbase.PocketBase, and a mailer.Mailer as parameters.
// The recipients are extracted from the "recipients" field of the record and split into a slice of mail.Address.
// For each recipient, a postcard message is rendered using the renderMessage function.
// The message is then sent using the mailClient.
// If there is an error sending the postcard, an error message is logged and the function returns.
// Finally, the postcard record is updated using the updatePostcardRecord function.
func sendPostcard(r *models.Record, app *pocketbase.PocketBase, mailClient mailer.Mailer) {

	recipientsRaw := r.GetString("recipients")
	recipientsSlice := strings.Split(recipientsRaw, ",")
    var recipients []mail.Address

	for _, recipient := range recipientsSlice {
		recipients = append(recipients, mail.Address{Address: recipient})
	}

	for i, rec := range recipients {

		app.Logger().Info("Sending postcard to", fmt.Sprintf("%d", i), r.GetId())

		message := renderMessage(r, rec, app)

		if err := mailClient.Send(message); err != nil {
			app.Logger().Error("Error sending postcard", "recipient", rec.Address, "record_id", r.GetId(), "error", err.Error())
			return
		}
	}
}

// renderMessage renders the email message for a postcard notification.
// It takes a pointer to a models.Record, a mail.Address, and a pointer to a pocketbase.PocketBase as input.
// It returns a pointer to a mailer.Message.
func renderMessage(r *models.Record, rec mail.Address, app *pocketbase.PocketBase) *mailer.Message {
	html, err := assets.RenderEmail("postcard:notification", map[string]any{
		"SenderName": r.GetString("sender_name"),
		"PickUpUrl":  utils.AssetUrl("/postcards?p=" + r.GetString("id")),
		"Title":      "",
		"LogoUrl":    utils.AssetUrl("/assets/images/logo.png"),
	})

	if err != nil {
		app.Logger().Error("Error rendering postcard email", "error", err.Error())
		return &mailer.Message{}
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

	return message
}

// updatePostcardRecord updates the postcard record with the given status and sent_at timestamp.
// It takes a pointer to a models.Record object and a pointer to a pocketbase.PocketBase object as parameters.
// The function sets the "status" field of the record to "sent" and the "sent_at" field to the current Unix timestamp.
// It then saves the updated record using the SaveRecord method of the pocketbase.PocketBase object.
// If there is an error during the update, it logs the error using the Logger method of the pocketbase.PocketBase object.
func updatePostcardRecord(r *models.Record, app *pocketbase.PocketBase) {
	r.Set("status", "sent")
	r.Set("sent_at", time.Now().Unix())

	if err := app.Dao().SaveRecord(r); err != nil {
		app.Logger().Error("Error updating postcard record", "record_id", r.GetId(), "error", err.Error())
	}
}

// sendPostcards sends postcards based on a specified frequency.
// It retrieves postcard records with a status of 'queued' from the database,
// sends each postcard using the mail client, and updates the postcard record.
// The frequency can be customized by setting the environment variable WGA_POSTCARD_FREQUENCY.
// If the environment variable is not set, the default frequency is "*/1 * * * *".
//
// Parameters:
// - app: A pointer to the PocketBase application instance.
// - scheduler: A pointer to the cron scheduler instance.
//
// Example usage:
//
//	sendPostcards(app, scheduler)
//
// Note: The sendPostcards function assumes that the necessary dependencies are already imported.
func sendPostcards(app *pocketbase.PocketBase, scheduler *cron.Cron) {

	var frequency = os.Getenv("WGA_POSTCARD_FREQUENCY")

	if frequency == "" {
		frequency = "*/1 * * * *"
	}

	scheduler.MustAdd("postcards", frequency, func() {
		records, err := app.Dao().FindRecordsByFilter(
			"postcards",         // collection
			"status = 'queued'", // filter
			"",                  // sort
			0,                   // limit
			0,                   // offset
		)

		if err != nil {
			app.Logger().Error("Error fetching postcards", "error", err.Error())
			return
		}

		mailClient := app.NewMailClient()

		for _, r := range records {
			sendPostcard(r, app, mailClient)
			updatePostcardRecord(r, app)
		}
	})

}
