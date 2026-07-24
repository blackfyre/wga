package postcards

import (
	"database/sql"
	"errors"
	"net"
	"net/mail"
	"net/textproto"
	"time"

	"github.com/blackfyre/wga/internal/assets"
	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/logging"
	"github.com/google/uuid"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/mailer"
	"github.com/pocketbase/pocketbase/tools/types"
)

const (
	deliveryLease     = 5 * time.Minute
	deliveryRetryBase = time.Minute
	deliveryRetryMax  = time.Hour
	deliveryBatchSize = 20
)

type ClaimedAttempt struct {
	Attempt  *core.Record
	Delivery *core.Record
	Postcard *core.Record
	Token    string
}

type deliveryFailure struct {
	class     string
	retryable bool
	ambiguous bool
}

func ProcessDue(app core.App, mailClient mailer.Mailer, postcards config.Postcards, runID string) error {
	if err := expandLegacyQueuedPostcards(app); err != nil {
		return err
	}
	for range deliveryBatchSize {
		claim, err := claimDue(app, types.NowDateTime())
		if err != nil {
			return err
		}
		if claim == nil {
			return nil
		}
		if err := deliver(app, mailClient, postcards, claim, runID); err != nil {
			return err
		}
	}

	return nil
}

func claimDue(app core.App, now types.DateTime) (*ClaimedAttempt, error) {
	if err := recoverExpiredClaims(app, now); err != nil {
		return nil, err
	}

	token := uuid.NewString()
	expires := now.Add(deliveryLease)
	var row struct {
		ID string `db:"id"`
	}
	err := app.DB().NewQuery(`
		UPDATE postcardDeliveryAttempts
		SET status = 'processing', claim_token = {:token}, claim_expires_at = {:expires},
			transport_started_at = ''
		WHERE id = (
			SELECT id FROM postcardDeliveryAttempts
			WHERE status = 'queued' AND available_at <= {:now}
			ORDER BY available_at, id LIMIT 1
		)
		AND status = 'queued' AND available_at <= {:now}
		RETURNING id
	`).Bind(dbx.Params{"token": token, "expires": expires, "now": now}).One(&row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	attempt, err := app.FindRecordById(collectionDeliveryAttempts, row.ID)
	if err != nil {
		return nil, err
	}
	delivery, err := app.FindRecordById(collectionDeliveries, attempt.GetString("delivery"))
	if err != nil {
		return nil, err
	}
	postcard, err := app.FindRecordById(collectionPostcards, delivery.GetString("postcard"))
	if err != nil {
		return nil, err
	}

	return &ClaimedAttempt{Attempt: attempt, Delivery: delivery, Postcard: postcard, Token: token}, nil
}

func recoverExpiredClaims(app core.App, now types.DateTime) error {
	_, err := app.DB().NewQuery(`
		UPDATE postcardDeliveryAttempts
		SET status = 'queued', claim_token = '', claim_expires_at = '', available_at = {:now}
		WHERE status = 'processing' AND claim_expires_at <= {:now} AND transport_started_at = ''
	`).Bind(dbx.Params{"now": now}).Execute()
	if err != nil {
		return err
	}
	_, err = app.DB().NewQuery(`
		UPDATE postcardDeliveryAttempts
		SET status = 'dead_lettered', dead_lettered_at = {:now}, claim_token = '', claim_expires_at = '',
			last_error_class = 'ambiguous_transport_outcome', last_error_retryable = false
		WHERE status = 'processing' AND claim_expires_at <= {:now} AND transport_started_at != ''
	`).Bind(dbx.Params{"now": now}).Execute()

	return err
}

func deliver(app core.App, mailClient mailer.Mailer, postcards config.Postcards, claim *ClaimedAttempt, runID string) error {
	message, err := renderMessage(claim.Postcard, claim.Delivery.GetString("recipient"), postcards)
	if err != nil {
		err = deadLetter(app, claim, deliveryFailure{class: "render_failed"}, types.NowDateTime())
		if err == nil {
			logDelivery(app, runID, claim, "render_failed")
		}
		return err
	}
	if err := startTransport(app, claim, types.NowDateTime()); err != nil {
		return err
	}
	if err := mailClient.Send(message); err != nil {
		failure := classifyDeliveryError(err)
		if failure.retryable && claim.Attempt.GetInt("attempt_count") < claim.Attempt.GetInt("max_attempts") {
			err = retry(app, claim, failure, types.NowDateTime())
			if err == nil {
				logDelivery(app, runID, claim, "retry_scheduled")
			}
			return err
		}
		err = deadLetter(app, claim, failure, types.NowDateTime())
		if err == nil {
			logDelivery(app, runID, claim, failure.class)
		}
		return err
	}

	err = complete(app, claim, types.NowDateTime())
	if err == nil {
		logDelivery(app, runID, claim, "sent")
	}
	return err
}

func logDelivery(app core.App, runID string, claim *ClaimedAttempt, outcome string) {
	logging.RunLogger(app, runID).Info("Postcard delivery attempt completed",
		"event", "postcard.delivery.attempt",
		"correlation_id", claim.Attempt.GetString("correlation_id"),
		"delivery_id", claim.Delivery.Id,
		"attempt_id", claim.Attempt.Id,
		"attempt", claim.Attempt.GetInt("attempt_count"),
		"outcome", outcome,
	)
}

func renderMessage(postcard *core.Record, recipient string, postcards config.Postcards) (*mailer.Message, error) {
	html, err := assets.RenderEmail("postcard:notification", map[string]any{
		"SenderName": postcard.GetString("sender_name"),
		"PickUpUrl":  postcards.PublicURL.Resolve("/postcard?p=" + postcard.Id),
		"Title":      "",
		"LogoUrl":    postcards.PublicURL.Resolve("/assets/images/logo.png"),
	})
	if err != nil {
		return nil, err
	}

	return &mailer.Message{
		From:    mail.Address{Name: postcards.Sender.Name, Address: postcards.Sender.Address.Address},
		To:      []mail.Address{{Address: recipient}},
		Subject: "You got a postcard from " + postcard.GetString("sender_name") + "!",
		HTML:    html,
	}, nil
}

func startTransport(app core.App, claim *ClaimedAttempt, now types.DateTime) error {
	result, err := app.DB().NewQuery(`
		UPDATE postcardDeliveryAttempts
		SET transport_started_at = {:now}, claim_expires_at = {:expires},
			attempt_count = attempt_count + 1, last_attempt_at = {:now}
		WHERE id = {:id} AND status = 'processing' AND claim_token = {:token}
	`).Bind(dbx.Params{"id": claim.Attempt.Id, "token": claim.Token, "now": now, "expires": now.Add(deliveryLease)}).Execute()
	if err != nil {
		return err
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if updated != 1 {
		return errors.New("postcard delivery claim lost")
	}
	claim.Attempt.Set("attempt_count", claim.Attempt.GetInt("attempt_count")+1)

	return nil
}

func complete(app core.App, claim *ClaimedAttempt, now types.DateTime) error {
	return app.RunInTransaction(func(txApp core.App) error {
		if err := updateOwnedAttempt(txApp, claim, `
			status = 'processed', processed_at = {:now}, result_code = 'smtp_accepted',
			claim_token = '', claim_expires_at = ''`, dbx.Params{"now": now}); err != nil {
			return err
		}
		delivery, err := txApp.FindRecordById(collectionDeliveries, claim.Delivery.Id)
		if err != nil {
			return err
		}
		delivery.Set("status", "sent")
		delivery.Set("sent_at", now)
		if err := txApp.Save(delivery); err != nil {
			return err
		}
		return markPostcardSent(txApp, claim.Postcard.Id, now)
	})
}

func retry(app core.App, claim *ClaimedAttempt, failure deliveryFailure, now types.DateTime) error {
	delay := deliveryRetryBase * time.Duration(1<<min(claim.Attempt.GetInt("attempt_count")-1, 5))
	if delay > deliveryRetryMax {
		delay = deliveryRetryMax
	}
	return updateOwnedAttempt(app, claim, `
		status = 'queued', available_at = {:available_at}, claim_token = '', claim_expires_at = '',
		last_error_class = {:error_class}, last_error_retryable = true`, dbx.Params{
		"available_at": now.Add(delay), "error_class": failure.class,
	})
}

func deadLetter(app core.App, claim *ClaimedAttempt, failure deliveryFailure, now types.DateTime) error {
	return updateOwnedAttempt(app, claim, `
		status = 'dead_lettered', dead_lettered_at = {:now}, claim_token = '', claim_expires_at = '',
		last_error_class = {:error_class}, last_error_retryable = {:retryable}`, dbx.Params{
		"now": now, "error_class": failure.class, "retryable": failure.retryable,
	})
}

func updateOwnedAttempt(app core.App, claim *ClaimedAttempt, changes string, params dbx.Params) error {
	params["id"] = claim.Attempt.Id
	params["token"] = claim.Token
	result, err := app.DB().NewQuery(`UPDATE postcardDeliveryAttempts SET ` + changes + `
		WHERE id = {:id} AND status = 'processing' AND claim_token = {:token}`).Bind(params).Execute()
	if err != nil {
		return err
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if updated != 1 {
		return errors.New("postcard delivery claim lost")
	}

	return nil
}

func markPostcardSent(app core.App, postcardID string, now types.DateTime) error {
	var totals struct {
		Pending int `db:"pending"`
	}
	if err := app.DB().NewQuery(`SELECT COUNT(*) AS pending FROM postcardDeliveries WHERE postcard = {:postcard} AND status != 'sent'`).Bind(dbx.Params{"postcard": postcardID}).One(&totals); err != nil {
		return err
	}
	if totals.Pending != 0 {
		return nil
	}
	postcard, err := app.FindRecordById(collectionPostcards, postcardID)
	if err != nil {
		return err
	}
	if postcard.GetString("status") != "queued" {
		return nil
	}
	postcard.Set("status", "sent")
	postcard.Set("sent_at", now)
	return app.Save(postcard)
}

func classifyDeliveryError(err error) deliveryFailure {
	var dnsError *net.DNSError
	if errors.As(err, &dnsError) {
		return deliveryFailure{class: "dns_failed", retryable: true}
	}
	var operationError *net.OpError
	if errors.As(err, &operationError) && operationError.Op == "dial" {
		return deliveryFailure{class: "dial_failed", retryable: true}
	}
	var smtpError *textproto.Error
	if errors.As(err, &smtpError) {
		if smtpError.Code >= 400 && smtpError.Code < 500 {
			return deliveryFailure{class: "smtp_temporary_failure", retryable: true}
		}
		return deliveryFailure{class: "smtp_permanent_failure"}
	}

	return deliveryFailure{class: "ambiguous_transport_outcome", ambiguous: true}
}
