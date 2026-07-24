package postcards

import (
	"errors"
	"net/mail"
	"strings"

	"github.com/google/uuid"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

const (
	collectionPostcards        = "postcards"
	collectionDeliveries       = "postcard_deliveries"
	collectionDeliveryAttempts = "postcard_delivery_attempts"
	defaultMaxAttempts         = 5
)

var ErrNoRecipients = errors.New("postcard requires at least one recipient")
var ErrInvalidAttemptTransition = errors.New("postcard delivery attempt cannot transition from its current status")

type QueueInput struct {
	SenderName    string
	SenderEmail   string
	Recipients    []string
	Message       string
	ImageID       string
	NotifySender  bool
	CorrelationID string
}

func Queue(app core.App, input QueueInput) (*core.Record, error) {
	recipients, err := normaliseRecipients(input.Recipients)
	if err != nil {
		return nil, err
	}
	if len(recipients) == 0 {
		return nil, ErrNoRecipients
	}
	if input.CorrelationID == "" {
		input.CorrelationID = uuid.NewString()
	}

	var postcard *core.Record
	err = app.RunInTransaction(func(txApp core.App) error {
		postcards, err := txApp.FindCollectionByNameOrId(collectionPostcards)
		if err != nil {
			return err
		}
		postcard = core.NewRecord(postcards)
		postcard.Set("status", "queued")
		postcard.Set("correlation_id", input.CorrelationID)
		postcard.Set("sender_name", input.SenderName)
		postcard.Set("sender_email", input.SenderEmail)
		postcard.Set("recipients", strings.Join(recipients, ","))
		postcard.Set("message", input.Message)
		postcard.Set("image_id", input.ImageID)
		postcard.Set("notify_sender", input.NotifySender)
		if err := txApp.Save(postcard); err != nil {
			return err
		}

		deliveries, err := txApp.FindCollectionByNameOrId(collectionDeliveries)
		if err != nil {
			return err
		}
		attempts, err := txApp.FindCollectionByNameOrId(collectionDeliveryAttempts)
		if err != nil {
			return err
		}
		now := types.NowDateTime()
		for _, recipient := range recipients {
			delivery := core.NewRecord(deliveries)
			delivery.Set("postcard", postcard.Id)
			delivery.Set("recipient", recipient)
			delivery.Set("status", "pending")
			if err := txApp.Save(delivery); err != nil {
				return err
			}

			attempt := core.NewRecord(attempts)
			attempt.Set("delivery", delivery.Id)
			attempt.Set("sequence", 1)
			attempt.Set("status", "queued")
			attempt.Set("correlation_id", input.CorrelationID)
			attempt.Set("message_id", uuid.NewString())
			attempt.Set("attempt_count", 0)
			attempt.Set("max_attempts", defaultMaxAttempts)
			attempt.Set("available_at", now)
			if err := txApp.Save(attempt); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return postcard, nil
}

func expandLegacyQueuedPostcards(app core.App) error {
	queued, err := app.FindRecordsByFilter(collectionPostcards, `status = 'queued'`, "", 0, 0)
	if err != nil {
		return err
	}
	for _, postcard := range queued {
		existing, err := app.FindRecordsByFilter(collectionDeliveries, `postcard = {:postcard}`, "", 1, 0, map[string]any{"postcard": postcard.Id})
		if err != nil {
			return err
		}
		if len(existing) != 0 {
			continue
		}
		if err := createLegacyReviewAttempts(app, postcard.Id); err != nil {
			return err
		}
	}

	return nil
}

func createLegacyReviewAttempts(app core.App, postcardID string) error {
	return app.RunInTransaction(func(txApp core.App) error {
		postcard, err := txApp.FindRecordById(collectionPostcards, postcardID)
		if err != nil {
			return err
		}
		existing, err := txApp.FindRecordsByFilter(collectionDeliveries, `postcard = {:postcard}`, "", 1, 0, map[string]any{"postcard": postcardID})
		if err != nil {
			return err
		}
		if len(existing) != 0 {
			return nil
		}
		correlationID := postcard.GetString("correlation_id")
		if correlationID == "" {
			correlationID = uuid.NewString()
			postcard.Set("correlation_id", correlationID)
			if err := txApp.Save(postcard); err != nil {
				return err
			}
		}
		recipients, err := normaliseRecipients(strings.Split(postcard.GetString("recipients"), ","))
		if err != nil || len(recipients) == 0 {
			recipients = []string{"legacy-" + postcard.Id + "@invalid.test"}
		}
		deliveries, err := txApp.FindCollectionByNameOrId(collectionDeliveries)
		if err != nil {
			return err
		}
		attempts, err := txApp.FindCollectionByNameOrId(collectionDeliveryAttempts)
		if err != nil {
			return err
		}
		now := types.NowDateTime()
		for _, recipient := range recipients {
			delivery := core.NewRecord(deliveries)
			delivery.Set("postcard", postcard.Id)
			delivery.Set("recipient", recipient)
			delivery.Set("status", "pending")
			if err := txApp.Save(delivery); err != nil {
				return err
			}
			attempt := core.NewRecord(attempts)
			attempt.Set("delivery", delivery.Id)
			attempt.Set("sequence", 1)
			attempt.Set("status", "dead_lettered")
			attempt.Set("correlation_id", correlationID)
			attempt.Set("message_id", uuid.NewString())
			attempt.Set("attempt_count", 0)
			attempt.Set("max_attempts", defaultMaxAttempts)
			attempt.Set("available_at", now)
			attempt.Set("dead_lettered_at", now)
			attempt.Set("last_error_class", "legacy_unknown")
			if err := txApp.Save(attempt); err != nil {
				return err
			}
		}

		return nil
	})
}

func MarkReceived(app core.App, postcardID string) error {
	return app.RunInTransaction(func(txApp core.App) error {
		postcard, err := txApp.FindRecordById(collectionPostcards, postcardID)
		if err != nil {
			return err
		}
		if postcard.GetString("received_at") != "" {
			return nil
		}
		postcard.Set("received_at", types.NowDateTime())
		if postcard.GetString("status") == "sent" {
			postcard.Set("status", "received")
		}
		return txApp.Save(postcard)
	})
}

func ResolveAttempt(app core.App, attemptID string, code string, summary string) error {
	if code != "resolved_manually" && code != "closed_without_replay" && code != "ignored_duplicate" {
		return errors.New("invalid postcard delivery resolution code")
	}
	return app.RunInTransaction(func(txApp core.App) error {
		attempt, err := txApp.FindRecordById(collectionDeliveryAttempts, attemptID)
		if err != nil {
			return err
		}
		if attempt.GetString("status") != "dead_lettered" || attempt.GetString("resolution_code") != "" {
			return ErrInvalidAttemptTransition
		}
		now := types.NowDateTime()
		attempt.Set("resolution_code", code)
		attempt.Set("resolution_summary", summary)
		attempt.Set("resolved_at", now)
		if err := txApp.Save(attempt); err != nil {
			return err
		}
		delivery, err := txApp.FindRecordById(collectionDeliveries, attempt.GetString("delivery"))
		if err != nil {
			return err
		}
		if code == "closed_without_replay" {
			delivery.Set("status", "cancelled")
			delivery.Set("cancelled_at", now)
		} else {
			delivery.Set("status", "sent")
			delivery.Set("sent_at", now)
		}
		if err := txApp.Save(delivery); err != nil {
			return err
		}
		if code == "closed_without_replay" {
			return nil
		}
		return markPostcardSent(txApp, delivery.GetString("postcard"), now)
	})
}

func ReplayAttempt(app core.App, attemptID string) (*core.Record, error) {
	var replay *core.Record
	err := app.RunInTransaction(func(txApp core.App) error {
		attempt, err := txApp.FindRecordById(collectionDeliveryAttempts, attemptID)
		if err != nil {
			return err
		}
		if attempt.GetString("status") != "dead_lettered" || attempt.GetString("resolution_code") != "" {
			return ErrInvalidAttemptTransition
		}
		var maxSequence struct {
			Sequence int `db:"sequence"`
		}
		if err := txApp.DB().NewQuery(`SELECT COALESCE(MAX(sequence), 0) AS sequence FROM postcardDeliveryAttempts WHERE delivery = {:delivery}`).Bind(map[string]any{"delivery": attempt.GetString("delivery")}).One(&maxSequence); err != nil {
			return err
		}
		now := types.NowDateTime()
		attempt.Set("resolution_code", "replayed_unmodified")
		attempt.Set("resolved_at", now)
		if err := txApp.Save(attempt); err != nil {
			return err
		}
		collection, err := txApp.FindCollectionByNameOrId(collectionDeliveryAttempts)
		if err != nil {
			return err
		}
		replay = core.NewRecord(collection)
		replay.Set("delivery", attempt.GetString("delivery"))
		replay.Set("sequence", maxSequence.Sequence+1)
		replay.Set("replay_of", attempt.Id)
		replay.Set("status", "queued")
		replay.Set("correlation_id", attempt.GetString("correlation_id"))
		replay.Set("message_id", uuid.NewString())
		replay.Set("attempt_count", 0)
		replay.Set("max_attempts", attempt.GetInt("max_attempts"))
		replay.Set("available_at", now)
		return txApp.Save(replay)
	})
	if err != nil {
		return nil, err
	}
	return replay, nil
}

func normaliseRecipients(recipients []string) ([]string, error) {
	unique := make(map[string]struct{}, len(recipients))
	normalised := make([]string, 0, len(recipients))
	for _, recipient := range recipients {
		parsed, err := mail.ParseAddress(strings.TrimSpace(recipient))
		if err != nil || parsed.Address == "" {
			return nil, errors.New("postcard recipient must be a valid email address")
		}
		address := strings.ToLower(parsed.Address)
		if _, exists := unique[address]; exists {
			continue
		}
		unique[address] = struct{}{}
		normalised = append(normalised, address)
	}

	return normalised, nil
}
