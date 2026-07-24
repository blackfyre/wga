package crontab

import (
	"net/mail"
	"strings"
	"time"

	"github.com/blackfyre/wga/internal/assets"
	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/logging"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/mailer"
)

// sendPostcard sends a postcard to every recipient and reports whether all deliveries succeeded.
func sendPostcard(r *core.Record, app core.App, mailClient mailer.Mailer, postcards config.Postcards, runID string) bool {
	recipients := convertCommaSeparatedEmailsToMailAddresses(r.GetString("recipients"))
	allDelivered := true

	for index, rec := range recipients {
		message, err := renderMessage(r, rec, postcards)
		if err != nil {
			logPostcardDelivery(app, runID, index+1, len(recipients), "render_failed", err)
			allDelivered = false
			continue
		}

		if err := mailClient.Send(message); err != nil {
			logPostcardDelivery(app, runID, index+1, len(recipients), "failed", err)
			allDelivered = false
			continue
		}

		logPostcardDelivery(app, runID, index+1, len(recipients), "sent", nil)
	}

	return allDelivered
}

// convertCommaSeparatedEmailsToMailAddresses converts a comma-separated string of email addresses
// into a slice of mail.Address structs.
func convertCommaSeparatedEmailsToMailAddresses(emails string) []mail.Address {
	recipientsSlice := strings.Split(emails, ",")
	var recipients []mail.Address

	for _, recipient := range recipientsSlice {
		recipients = append(recipients, mail.Address{Address: recipient})
	}

	return recipients
}

// renderMessage renders the email message for a postcard notification.
func renderMessage(r *core.Record, rec mail.Address, postcards config.Postcards) (*mailer.Message, error) {
	html, err := assets.RenderEmail("postcard:notification", map[string]any{
		"SenderName": r.GetString("sender_name"),
		"PickUpUrl":  postcards.PublicURL.Resolve("/postcard?p=" + r.GetString("id")),
		"Title":      "",
		"LogoUrl":    postcards.PublicURL.Resolve("/assets/images/logo.png"),
	})

	if err != nil {
		return nil, err
	}

	message := &mailer.Message{
		From: mail.Address{
			Name:    postcards.Sender.Name,
			Address: postcards.Sender.Address.Address,
		},
		To:      []mail.Address{rec},
		Subject: "You got a postcard from " + r.GetString("sender_name") + "!",
		HTML:    html,
	}

	return message, nil
}

// updatePostcardRecord marks a successfully delivered postcard as sent.
func updatePostcardRecord(r *core.Record, app core.App, runID string) bool {
	r.Set("status", "sent")
	r.Set("sent_at", time.Now().Unix())

	if err := app.Save(r); err != nil {
		logging.RunLogger(app, runID).Error("Postcard delivery record update failed",
			"event", "postcard.delivery.record_update",
			"outcome", "failed",
			"error_type", logging.ErrorType(err),
			"error", logging.Redact(err),
		)
		return false
	}

	return true
}

func logPostcardDelivery(app core.App, runID string, recipientIndex int, recipientCount int, outcome string, err error) {
	attributes := []any{
		"event", "postcard.delivery.attempt",
		"recipient_index", recipientIndex,
		"recipient_count", recipientCount,
		"attempt", 1,
		"outcome", outcome,
	}
	logger := logging.RunLogger(app, runID)

	if err == nil {
		logger.Info("Postcard delivery completed", attributes...)
		return
	}

	attributes = append(attributes,
		"error_type", logging.ErrorType(err),
		"error", logging.Redact(err),
	)
	logger.Error("Postcard delivery failed", attributes...)
}

func processPostcard(r *core.Record, app core.App, mailClient mailer.Mailer, postcards config.Postcards, runID string) bool {
	allDelivered := sendPostcard(r, app, mailClient, postcards, runID)
	if !updatePostcardRecord(r, app, runID) {
		return false
	}

	return allDelivered
}

// sendPostcards sends postcards based on a specified frequency.
// It retrieves postcard records with a status of 'queued' from the database,
// sends each postcard using the mail client, and updates the postcard record.
// The frequency is provided by the application configuration.
//
// Parameters:
// - app: A pointer to the PocketBase application instance.
// - scheduler: A pointer to the cron scheduler instance.
//
// Example usage:
//
//	sendPostcards(app, postcards)
//
// Note: The sendPostcards function assumes that the necessary dependencies are already imported.
func sendPostcards(app core.App, postcards config.Postcards) {
	app.Logger().Info("Postcard delivery schedule registered", "event", "postcard.delivery.schedule_registered")

	app.Cron().MustAdd("postcards", postcards.Expression(), func() {
		runID := logging.NewRunID()
		logger := logging.RunLogger(app, runID)
		logger.Info("Postcard delivery run started",
			"event", "postcard.delivery.run",
			"outcome", "started",
		)

		records, err := app.FindRecordsByFilter(
			"postcards",         // collection
			"status = 'queued'", // filter
			"",                  // sort
			0,                   // limit
			0,                   // offset
		)

		if err != nil {
			logger.Error("Postcard delivery queue lookup failed",
				"event", "postcard.delivery.run",
				"outcome", "queue_lookup_failed",
				"error_type", logging.ErrorType(err),
				"error", logging.Redact(err),
			)
			return
		}

		mailClient := app.NewMailClient()
		deliveredCount := 0
		failedCount := 0

		for _, r := range records {
			if !processPostcard(r, app, mailClient, postcards, runID) {
				failedCount++
				continue
			}

			deliveredCount++
		}

		if failedCount > 0 {
			outcome := "partial_failure"
			if deliveredCount == 0 {
				outcome = "failed"
			}

			logger.Warn("Postcard delivery run completed",
				"event", "postcard.delivery.run",
				"postcard_count", len(records),
				"delivered_count", deliveredCount,
				"failed_count", failedCount,
				"outcome", outcome,
			)
			return
		}

		logger.Info("Postcard delivery run completed",
			"event", "postcard.delivery.run",
			"postcard_count", len(records),
			"delivered_count", deliveredCount,
			"outcome", "completed",
		)
	})

}
