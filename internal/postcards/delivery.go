package postcards

import (
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/textproto"
	"strings"
	"time"

	"github.com/blackfyre/wga/internal/assets"
	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/constants"
	"github.com/blackfyre/wga/internal/logging"
	"github.com/blackfyre/wga/internal/repositories"
	urlutils "github.com/blackfyre/wga/internal/utils/url"
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

// ClaimedAttempt is an attempt together with the records and lease token required to process it.
type ClaimedAttempt struct {
	Attempt  *core.Record
	Delivery *core.Record
	Postcard *core.Record
	Token    string
}

// deliveryFailure describes a classified mail delivery failure.
type deliveryFailure struct {
	class     string
	retryable bool
	ambiguous bool
}

// ProcessDue claims and processes a bounded batch of due postcard delivery attempts.
func ProcessDue(app core.App, mailClient mailer.Mailer, postcards config.Postcards, runID string) error {
	if err := expandLegacyQueuedPostcards(app); err != nil {
		return err
	}
	keyring := postcards.TokenKeyring()
	for range deliveryBatchSize {
		claim, err := claimDue(app, types.NowDateTime())
		if err != nil {
			return err
		}
		if claim == nil {
			return nil
		}
		if err := deliver(app, mailClient, postcards, keyring, claim, runID); err != nil {
			return err
		}
	}

	return nil
}

// claimDue atomically leases the next due delivery attempt.
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
		UPDATE postcard_delivery_attempts
		SET status = 'processing', claim_token = {:token}, claim_expires_at = {:expires},
			transport_started_at = ''
		WHERE id = (
			SELECT id FROM postcard_delivery_attempts
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

// recoverExpiredClaims requeues safe expired claims and dead-letters ambiguous ones.
func recoverExpiredClaims(app core.App, now types.DateTime) error {
	return app.RunInTransaction(func(txApp core.App) error {
		_, err := txApp.DB().NewQuery(`
		UPDATE postcard_delivery_attempts
		SET status = 'queued', claim_token = '', claim_expires_at = '', available_at = {:now}
		WHERE status = 'processing' AND claim_expires_at <= {:now} AND transport_started_at = ''
		`).Bind(dbx.Params{"now": now}).Execute()
		if err != nil {
			return err
		}
		var rows []struct {
			Delivery string `db:"delivery"`
		}
		err = txApp.DB().NewQuery(`
		UPDATE postcard_delivery_attempts
		SET status = 'dead_lettered', dead_lettered_at = {:now}, claim_token = '', claim_expires_at = '',
			last_error_class = 'ambiguous_transport_outcome', last_error_retryable = false
		WHERE status = 'processing' AND claim_expires_at <= {:now} AND transport_started_at != ''
		RETURNING delivery`).Bind(dbx.Params{"now": now}).All(&rows)
		if err != nil {
			return err
		}
		for _, row := range rows {
			delivery, err := txApp.FindRecordById(collectionDeliveries, row.Delivery)
			if err != nil {
				return err
			}
			delivery.Set("status", "failed")
			delivery.Set("failed_at", now)
			if err := txApp.Save(delivery); err != nil {
				return err
			}
			if err := finalizePostcard(txApp, delivery.GetString("postcard"), now); err != nil {
				return err
			}
		}
		return nil
	})
}

// deliver renders and sends one claimed recipient delivery.
func deliver(app core.App, mailClient mailer.Mailer, postcards config.Postcards, keyring config.PostcardTokenKeyring, claim *ClaimedAttempt, runID string) error {
	message, err := renderMessage(app, claim.Postcard, claim.Delivery, claim.Attempt.GetString("message_id"), postcards, keyring)
	if err != nil {
		err = deadLetter(app, claim, deliveryFailure{class: "invalid_message"}, types.NowDateTime())
		if err == nil {
			logDelivery(app, runID, claim, "invalid_message")
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

// logDelivery records a delivery outcome with safe execution identifiers.
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

// renderMessage produces the notification email for one postcard recipient.
func renderMessage(app core.App, postcard *core.Record, delivery *core.Record, messageID string, postcards config.Postcards, keyring config.PostcardTokenKeyring) (*mailer.Message, error) {
	token, err := recoverRecipientToken(keyring, delivery.Id, delivery.GetString("view_token_envelope"), delivery.GetString("view_token_hash"))
	if err != nil || delivery.GetString("recipient") == "" || !delivery.GetDateTime("view_expires_at").After(types.NowDateTime()) {
		return nil, errors.New("postcard delivery is missing recipient access material")
	}
	data := map[string]any{
		"SenderName":        postcard.GetString("sender_name"),
		"SenderNameUpper":   strings.ToUpper(postcard.GetString("sender_name")),
		"Message":           postcard.GetString("message"),
		"PickUpUrl":         postcards.PublicURL.Resolve("/postcard?token=" + token),
		"Title":             "A postcard is waiting for you",
		"LogoUrl":           postcards.PublicURL.Resolve("/assets/images/logo.png"),
		"HasArtwork":        false,
		"ArtworkTitle":      "",
		"ArtworkTitleUpper": "",
		"ArtworkDetails":    "",
		"ArtworkImageURL":   "",
		"MusicAvailable":    false,
		"MusicPiece":        "",
		"MusicComposer":     "",
	}
	artworkID := strings.Trim(strings.TrimSpace(fmt.Sprint(postcard.Get("image_id"))), "[]\"")
	if imageIDs := postcard.GetStringSlice("image_id"); len(imageIDs) > 0 {
		artworkID = imageIDs[0]
	}
	if artwork, artworkErr := app.FindFirstRecordByFilter(constants.CollectionArtworks, "id = {:id} && published = true", dbx.Params{"id": artworkID}); artworkErr == nil {
		title := artwork.GetString("title")
		if title != "" {
			data["HasArtwork"] = true
			data["ArtworkTitle"] = title
			data["ArtworkTitleUpper"] = strings.ToUpper(title)
			data["ArtworkDetails"] = artwork.GetString("date_text")
			data["ArtworkImageURL"] = postcards.PublicURL.Resolve(urlutils.GenerateArtworkImageURL(artwork, urlutils.DeliveryProfilePostcardSmallDualPlate, ""))
			if authors := artwork.GetStringSlice("author"); len(authors) > 0 {
				repository := repositories.NewArtistRecordRepository(app)
				if artist, artistErr := repository.FindPublishedArtist(authors[0]); artistErr == nil {
					data["ArtworkDetails"] = strings.TrimSpace(artist.GetString("filing_name") + " · " + artwork.GetString("date_text"))
					if postcard.GetBool("include_music") {
						if song, songErr := repository.MatchPeriodSong(artwork.GetInt("date_start")); songErr == nil && song != nil {
							data["MusicAvailable"] = true
							data["MusicPiece"] = song.Record.GetString("title")
							data["MusicComposer"] = song.Composer
						}
					}
				}
			}
		}
	}
	html, err := assets.RenderEmail("postcard:notification", data)
	if err != nil {
		return nil, err
	}

	return &mailer.Message{
		From:    mail.Address{Name: postcards.Sender.Name, Address: postcards.Sender.Address.Address},
		To:      []mail.Address{{Address: delivery.GetString("recipient")}},
		Subject: "You got a postcard from " + postcard.GetString("sender_name") + "!",
		HTML:    html,
		Headers: map[string]string{
			"Message-ID":        "<postcard-" + messageID + "@wga.invalid>",
			"X-WGA-Delivery-ID": messageID,
		},
	}, nil
}

// startTransport records the SMTP boundary and increments the transport attempt count.
func startTransport(app core.App, claim *ClaimedAttempt, now types.DateTime) error {
	result, err := app.DB().NewQuery(`
		UPDATE postcard_delivery_attempts
		SET transport_started_at = {:now}, claim_expires_at = {:expires},
			attempt_count = attempt_count + 1, last_attempt_at = {:now}
		WHERE id = {:id} AND status = 'processing' AND claim_token = {:token} AND transport_started_at = ''
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

// CancelPostcard cancels only recipient work that has not crossed the transport boundary.
type CancellationResult struct {
	CancelledDeliveries int
	StartedDeliveries   int
}

func CancelPostcard(app core.App, postcardID string, now types.DateTime) error {
	_, err := CancelPostcardWithResult(app, postcardID, now)
	return err
}

func CancelPostcardWithResult(app core.App, postcardID string, now types.DateTime) (CancellationResult, error) {
	result := CancellationResult{}
	err := app.RunInTransaction(func(txApp core.App) error {
		var started struct {
			Count int `db:"count"`
		}
		if err := txApp.DB().NewQuery(`
			SELECT COUNT(DISTINCT delivery) AS count FROM postcard_delivery_attempts
			WHERE delivery IN (SELECT id FROM postcard_deliveries WHERE postcard = {:postcard})
			AND transport_started_at != ''
		`).Bind(dbx.Params{"postcard": postcardID}).One(&started); err != nil {
			return err
		}
		result.StartedDeliveries = started.Count
		if _, err := txApp.DB().NewQuery(`
			UPDATE postcard_delivery_attempts
			SET status = 'cancelled', processed_at = {:now}, claim_token = '', claim_expires_at = ''
			WHERE delivery IN (SELECT id FROM postcard_deliveries WHERE postcard = {:postcard})
			AND (status = 'queued' OR (status = 'processing' AND transport_started_at = ''))
			AND NOT EXISTS (
				SELECT 1 FROM postcard_delivery_attempts started
				WHERE started.delivery = postcard_delivery_attempts.delivery AND started.transport_started_at != ''
			)
		`).Bind(dbx.Params{"postcard": postcardID, "now": now}).Execute(); err != nil {
			return err
		}
		update, err := txApp.DB().NewQuery(`
			UPDATE postcard_deliveries SET status = 'cancelled', cancelled_at = {:now}
			WHERE postcard = {:postcard} AND status = 'pending'
			AND NOT EXISTS (
				SELECT 1 FROM postcard_delivery_attempts a
				WHERE a.delivery = postcard_deliveries.id AND a.transport_started_at != ''
			)
		`).Bind(dbx.Params{"postcard": postcardID, "now": now}).Execute()
		if err != nil {
			return err
		}
		cancelled, err := update.RowsAffected()
		if err != nil {
			return err
		}
		result.CancelledDeliveries = int(cancelled)
		return finalizePostcard(txApp, postcardID, now)
	})
	return result, err
}

// complete records a successful transport and updates the recipient and postcard lifecycles.
func complete(app core.App, claim *ClaimedAttempt, now types.DateTime) error {
	return app.RunInTransaction(func(txApp core.App) error {
		if err := updateOwnedAttempt(txApp, claim, `
			status = 'processed', processed_at = {:now}, result_code = 'smtp_accepted',
			result_summary = 'accepted by configured mail transport', payload_purged_at = {:now},
			claim_token = '', claim_expires_at = ''`, dbx.Params{"now": now}); err != nil {
			return err
		}
		delivery, err := txApp.FindRecordById(collectionDeliveries, claim.Delivery.Id)
		if err != nil {
			return err
		}
		delivery.Set("status", "sent")
		delivery.Set("sent_at", now)
		delivery.Set("recipient", "purged:"+delivery.Id)
		delivery.Set("recipient_purged_at", now)
		delivery.Set("view_token_envelope", "")
		if err := txApp.Save(delivery); err != nil {
			return err
		}
		return finalizePostcard(txApp, claim.Postcard.Id, now)
	})
}

// retry returns an owned attempt to the queue with bounded exponential backoff.
func retry(app core.App, claim *ClaimedAttempt, failure deliveryFailure, now types.DateTime) error {
	delay := deliveryRetryBase * time.Duration(1<<min(claim.Attempt.GetInt("attempt_count")-1, 5))
	if delay > deliveryRetryMax {
		delay = deliveryRetryMax
	}
	return app.RunInTransaction(func(txApp core.App) error {
		if err := updateOwnedAttempt(txApp, claim, `
			status = 'processed', processed_at = {:now}, result_code = 'retry_scheduled',
			result_summary = 'retry scheduled after retryable transport failure', claim_token = '', claim_expires_at = '',
			last_error_class = {:error_class}, last_error_code = {:error_class}, last_error_retryable = true`, dbx.Params{
			"now": now, "error_class": failure.class,
		}); err != nil {
			return err
		}
		attempts, err := txApp.FindCollectionByNameOrId(collectionDeliveryAttempts)
		if err != nil {
			return err
		}
		messageID := claim.Attempt.GetString("message_id")
		next := core.NewRecord(attempts)
		next.Set("delivery", claim.Delivery.Id)
		next.Set("sequence", claim.Attempt.GetInt("sequence")+1)
		next.Set("status", "queued")
		next.Set("correlation_id", claim.Attempt.GetString("correlation_id"))
		next.Set("message_id", messageID)
		next.Set("attempt_count", claim.Attempt.GetInt("attempt_count"))
		next.Set("max_attempts", claim.Attempt.GetInt("max_attempts"))
		next.Set("available_at", now.Add(delay))
		setIntegrationMessageProfile(next, claim.Delivery.Id, messageID, claim.Delivery.GetDateTime("view_expires_at"))
		return txApp.Save(next)
	})
}

// deadLetter records a terminal failed outcome for an owned attempt.
func deadLetter(app core.App, claim *ClaimedAttempt, failure deliveryFailure, now types.DateTime) error {
	return app.RunInTransaction(func(txApp core.App) error {
		if err := updateOwnedAttempt(txApp, claim, `
		status = 'dead_lettered', dead_lettered_at = {:now}, claim_token = '', claim_expires_at = '',
		last_error_class = {:error_class}, last_error_code = {:error_class}, last_error_retryable = {:retryable}`, dbx.Params{
			"now": now, "error_class": failure.class, "retryable": failure.retryable,
		}); err != nil {
			return err
		}
		delivery, err := txApp.FindRecordById(collectionDeliveries, claim.Delivery.Id)
		if err != nil {
			return err
		}
		delivery.Set("status", "failed")
		delivery.Set("failed_at", now)
		if err := txApp.Save(delivery); err != nil {
			return err
		}
		return finalizePostcard(txApp, claim.Postcard.Id, now)
	})
}

// updateOwnedAttempt applies a transition only while the caller retains the lease token.
func updateOwnedAttempt(app core.App, claim *ClaimedAttempt, changes string, params dbx.Params) error {
	params["id"] = claim.Attempt.Id
	params["token"] = claim.Token
	result, err := app.DB().NewQuery(`UPDATE postcard_delivery_attempts SET ` + changes + `
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

// finalizePostcard applies the terminal parent status after every recipient delivery is resolved.
func finalizePostcard(app core.App, postcardID string, now types.DateTime) error {
	var totals struct {
		Pending   int `db:"pending"`
		Cancelled int `db:"cancelled"`
		Failed    int `db:"failed"`
	}
	if err := app.DB().NewQuery(`
		SELECT
			SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) AS pending,
			SUM(CASE WHEN status = 'cancelled' THEN 1 ELSE 0 END) AS cancelled,
			SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) AS failed
		FROM postcard_deliveries
		WHERE postcard = {:postcard}
	`).Bind(dbx.Params{"postcard": postcardID}).One(&totals); err != nil {
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
	postcard.Set("sender_email", "purged-"+postcard.Id+"@invalid.test")
	postcard.Set("sender_email_purged_at", now)
	postcard.Set("recipients", "purged")
	if totals.Failed != 0 {
		postcard.Set("status", "failed")
		postcard.Set("failed_at", now)
		return app.Save(postcard)
	}
	if totals.Cancelled != 0 {
		postcard.Set("status", "cancelled")
		return app.Save(postcard)
	}
	postcard.Set("sent_at", now)
	if postcard.GetString("received_at") != "" {
		postcard.Set("status", "received")
	} else {
		postcard.Set("status", "sent")
	}
	return app.Save(postcard)
}

// classifyDeliveryError determines whether a mail failure can safely be retried.
func classifyDeliveryError(err error) deliveryFailure {
	var dnsError *net.DNSError
	if errors.As(err, &dnsError) {
		return deliveryFailure{class: "transport_failure", retryable: dnsError.IsTemporary || dnsError.IsTimeout}
	}
	var operationError *net.OpError
	if errors.As(err, &operationError) && operationError.Op == "dial" {
		return deliveryFailure{class: "transport_failure", retryable: true}
	}
	var smtpError *textproto.Error
	if errors.As(err, &smtpError) {
		if smtpError.Code >= 400 && smtpError.Code < 500 {
			return deliveryFailure{class: "provider_unavailable", retryable: true}
		}
		return deliveryFailure{class: "invalid_request"}
	}

	return deliveryFailure{class: "ambiguous_transport_outcome", ambiguous: true}
}

// maxPurgeBatchSize bounds a single purge call so a pathological backlog cannot
// take an unbounded SQLite write lock.
const maxPurgeBatchSize = 1000

// PurgeCounts reports the number of expired delivery-access and postcard-content
// records removed by a single bounded purge call. A count equal to the requested
// limit signals that a full batch was removed and more work may remain.
type PurgeCounts struct {
	DeliveryAccess  int
	PostcardContent int
	SenderControls  int
}

// PurgeExpiredRecipientAccess removes expired bearer material only after work is
// terminal. It purges at most limit eligible delivery-access records and at most
// limit eligible postcard-content records per call, in deterministic ID order,
// and returns the exact per-kind counts. Active queued/processing messages and
// unresolved dead letters are deliberately retained, and already-purged rows are
// never rewritten, so repeated calls drain the backlog and terminate at zero.
func PurgeExpiredRecipientAccess(app core.App, now types.DateTime, limit int) (PurgeCounts, error) {
	if limit <= 0 || limit > maxPurgeBatchSize {
		return PurgeCounts{}, fmt.Errorf("postcard purge batch size must be between 1 and %d", maxPurgeBatchSize)
	}

	var counts PurgeCounts
	err := app.RunInTransaction(func(txApp core.App) error {
		result, err := txApp.DB().NewQuery(`
			UPDATE Postcards SET submission_key_hash = ''
			WHERE id IN (
				SELECT postcard FROM postcard_sender_controls
				WHERE expires_at <= {:now}
				ORDER BY id LIMIT {:limit}
			)`).Bind(dbx.Params{"now": now, "limit": limit}).Execute()
		if err != nil {
			return err
		}
		result, err = txApp.DB().NewQuery(`
			DELETE FROM postcard_sender_controls
			WHERE id IN (
				SELECT id FROM postcard_sender_controls
				WHERE expires_at <= {:now}
				ORDER BY id LIMIT {:limit}
			)`).Bind(dbx.Params{"now": now, "limit": limit}).Execute()
		if err != nil {
			return err
		}
		updated, err := result.RowsAffected()
		if err != nil {
			return err
		}
		counts.SenderControls = int(updated)

		result, err = txApp.DB().NewQuery(`
			UPDATE postcard_deliveries
			SET view_token_envelope = '', view_token_hash = ''
			WHERE id IN (
				SELECT id FROM postcard_deliveries
				WHERE view_expires_at != '' AND view_expires_at <= {:now}
				AND (view_token_envelope != '' OR view_token_hash != '')
				AND NOT EXISTS (
					SELECT 1 FROM postcard_delivery_attempts a
					WHERE a.delivery = postcard_deliveries.id
					AND (a.status IN ('queued', 'processing') OR (a.status = 'dead_lettered' AND a.resolved_at = ''))
				)
				ORDER BY id
				LIMIT {:limit}
			)
		`).Bind(dbx.Params{"now": now, "limit": limit}).Execute()
		if err != nil {
			return err
		}
		updated, err = result.RowsAffected()
		if err != nil {
			return err
		}
		counts.DeliveryAccess = int(updated)

		result, err = txApp.DB().NewQuery(`
			UPDATE Postcards
			SET sender_name = 'Anonymous', sender_email = 'purged@example.invalid', recipients = 'purged',
				message = '<p>Expired postcard</p>', content_purged_at = {:now}
			WHERE id IN (
				SELECT id FROM Postcards
				WHERE retention_until != '' AND retention_until <= {:now} AND content_purged_at = ''
				AND NOT EXISTS (
					SELECT 1 FROM postcard_deliveries d
					JOIN postcard_delivery_attempts a ON a.delivery = d.id
					WHERE d.postcard = Postcards.id
					AND (a.status IN ('queued', 'processing') OR (a.status = 'dead_lettered' AND a.resolved_at = ''))
				)
				ORDER BY id
				LIMIT {:limit}
			)
		`).Bind(dbx.Params{"now": now, "limit": limit}).Execute()
		if err != nil {
			return err
		}
		updated, err = result.RowsAffected()
		if err != nil {
			return err
		}
		counts.PostcardContent = int(updated)

		return nil
	})
	if err != nil {
		return PurgeCounts{}, err
	}

	return counts, nil
}
