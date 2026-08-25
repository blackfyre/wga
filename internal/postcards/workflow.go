package postcards

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/blackfyre/wga/internal/config"
	"github.com/google/uuid"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

const (
	collectionPostcards        = "postcards"
	collectionDeliveries       = "tracking_postcard_deliveries"
	collectionDeliveryAttempts = "tracking_postcard_delivery_attempts"
	defaultMaxAttempts         = 5
	maxRecipients              = 5
	maxSenderNameRunes         = 120
	maxMessageBytes            = 5000
	recipientTokenBytes        = 32

	integrationDirection   = "outbound"
	integrationKey         = "postcard_email"
	integrationMessageType = "recipient_link_delivery"
)

// RecipientTokenValidity is the public pickup-link lifetime and payload-retention deadline.
const RecipientTokenValidity = 30 * 24 * time.Hour

var (
	// ErrNoRecipients indicates that a postcard has no deliverable recipients.
	ErrNoRecipients = errors.New("postcard requires at least one recipient")
	// ErrInvalidAttemptTransition indicates that the requested lifecycle transition is not allowed.
	ErrInvalidAttemptTransition = errors.New("postcard delivery attempt cannot transition from its current status")
	// ErrInvalidPostcard indicates that submitted postcard content violates the application contract.
	ErrInvalidPostcard = errors.New("invalid postcard")
	// ErrArtworkUnavailable indicates that the selected artwork is not publicly eligible.
	ErrArtworkUnavailable = errors.New("postcard artwork is not published")
	// ErrRecipientAccessDenied prevents token-shape, lookup, status, and expiry details leaking.
	ErrRecipientAccessDenied = errors.New("postcard recipient access denied")
)

// QueueInput contains the content and recipients required to queue a postcard.
type QueueInput struct {
	SenderName    string
	SenderEmail   string
	Recipients    []string
	Message       string
	ImageID       string
	NotifySender  bool
	IncludeMusic  bool
	CorrelationID string
}

// RecipientAccess contains the one-time delivery material and bounded public expiry.
type RecipientAccess struct {
	DeliveryID string
	Recipient  string
	Token      string
	ExpiresAt  types.DateTime
}

// QueueResult is the atomically persisted postcard and recipient access material.
type QueueResult struct {
	Postcard *core.Record
	Access   []RecipientAccess
}

// Queue atomically persists a postcard and its encrypted recipient access material.
func Queue(app core.App, keyring config.PostcardTokenKeyring, input QueueInput) (*core.Record, error) {
	result, err := QueueWithAccess(app, keyring, input, types.NowDateTime())
	if err != nil {
		return nil, err
	}
	return result.Postcard, nil
}

// QueueWithAccess persists a published-work postcard and all delivery work atomically.
func QueueWithAccess(app core.App, keyring config.PostcardTokenKeyring, input QueueInput, now types.DateTime) (*QueueResult, error) {
	if _, _, err := activeRecipientTokenKey(keyring); err != nil {
		return nil, err
	}
	if err := validateQueueInput(&input); err != nil {
		return nil, err
	}
	recipients, err := normaliseRecipients(input.Recipients)
	if err != nil {
		return nil, err
	}
	if len(recipients) == 0 {
		return nil, ErrNoRecipients
	}
	if len(recipients) > maxRecipients {
		return nil, fmtInvalid("too many recipients")
	}
	if input.CorrelationID == "" {
		input.CorrelationID = uuid.NewString()
	}

	result := &QueueResult{Access: make([]RecipientAccess, 0, len(recipients))}
	err = app.RunInTransaction(func(txApp core.App) error {
		if _, err := txApp.FindFirstRecordByFilter("artworks", "id = {:id} && published = true", map[string]any{"id": input.ImageID}); err != nil {
			return ErrArtworkUnavailable
		}

		postcards, err := txApp.FindCollectionByNameOrId(collectionPostcards)
		if err != nil {
			return err
		}
		postcard := core.NewRecord(postcards)
		postcard.Set("status", "queued")
		postcard.Set("correlation_id", input.CorrelationID)
		postcard.Set("sender_name", input.SenderName)
		postcard.Set("sender_email", input.SenderEmail)
		postcard.Set("recipients", strings.Join(recipients, ","))
		postcard.Set("message", input.Message)
		postcard.Set("image_id", input.ImageID)
		postcard.Set("notify_sender", input.NotifySender)
		postcard.Set("include_music", input.IncludeMusic)
		postcard.Set("retention_until", now.Add(RecipientTokenValidity))
		if err := txApp.Save(postcard); err != nil {
			return err
		}
		result.Postcard = postcard

		deliveries, err := txApp.FindCollectionByNameOrId(collectionDeliveries)
		if err != nil {
			return err
		}
		attempts, err := txApp.FindCollectionByNameOrId(collectionDeliveryAttempts)
		if err != nil {
			return err
		}
		expiresAt := now.Add(RecipientTokenValidity)
		for _, recipient := range recipients {
			token, err := newRecipientToken()
			if err != nil {
				return err
			}
			delivery := core.NewRecord(deliveries)
			delivery.Set("postcard", postcard.Id)
			delivery.Set("recipient", recipient)
			delivery.Set("status", "pending")
			delivery.Set("view_expires_at", expiresAt)
			if err := txApp.Save(delivery); err != nil {
				return err
			}
			envelope, err := sealRecipientToken(keyring, delivery.Id, token)
			if err != nil {
				return err
			}
			delivery.Set("view_token_envelope", envelope)
			delivery.Set("view_token_hash", HashRecipientToken(token))
			if err := txApp.Save(delivery); err != nil {
				return err
			}

			messageID := uuid.NewString()
			attempt := core.NewRecord(attempts)
			attempt.Set("delivery", delivery.Id)
			attempt.Set("sequence", 1)
			attempt.Set("status", "queued")
			attempt.Set("correlation_id", input.CorrelationID)
			attempt.Set("message_id", messageID)
			attempt.Set("attempt_count", 0)
			attempt.Set("max_attempts", defaultMaxAttempts)
			attempt.Set("available_at", now)
			setIntegrationMessageProfile(attempt, delivery.Id, messageID, expiresAt)
			if err := txApp.Save(attempt); err != nil {
				return err
			}
			result.Access = append(result.Access, RecipientAccess{DeliveryID: delivery.Id, Recipient: recipient, Token: token, ExpiresAt: expiresAt})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func validateQueueInput(input *QueueInput) error {
	input.SenderName = strings.TrimSpace(input.SenderName)
	input.SenderEmail = strings.TrimSpace(input.SenderEmail)
	input.ImageID = strings.TrimSpace(input.ImageID)
	if input.SenderName == "" || strings.ContainsAny(input.SenderName, "\r\n") || utf8.RuneCountInString(input.SenderName) > maxSenderNameRunes {
		return fmtInvalid("invalid sender name")
	}
	parsed, err := mail.ParseAddress(input.SenderEmail)
	if err != nil || parsed.Address != input.SenderEmail || len(input.SenderEmail) > 254 {
		return fmtInvalid("invalid sender email")
	}
	if input.ImageID == "" || strings.TrimSpace(input.Message) == "" || len(input.Message) > maxMessageBytes {
		return fmtInvalid("invalid postcard content")
	}
	return nil
}

func fmtInvalid(detail string) error {
	return errors.Join(ErrInvalidPostcard, errors.New(detail))
}

func setIntegrationMessageProfile(attempt *core.Record, deliveryID string, messageID string, retentionUntil types.DateTime) {
	attempt.Set("direction", integrationDirection)
	attempt.Set("integration_key", integrationKey)
	attempt.Set("message_type", integrationMessageType)
	attempt.Set("deduplication_key", messageID)
	attempt.Set("payload_reference", deliveryID)
	attempt.Set("payload_format", "normalised_record")
	attempt.Set("payload_retention_until", retentionUntil)
	attempt.Set("transport_metadata", map[string]any{"channel": "email", "template": "postcard:notification"})
}

func newRecipientToken() (string, error) {
	material := make([]byte, recipientTokenBytes)
	if _, err := rand.Read(material); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(material), nil
}

// ValidRecipientToken rejects arbitrary IDs and non-canonical token encodings.
func ValidRecipientToken(token string) bool {
	if len(token) != base64.RawURLEncoding.EncodedLen(recipientTokenBytes) {
		return false
	}
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	return err == nil && len(decoded) == recipientTokenBytes && base64.RawURLEncoding.EncodeToString(decoded) == token
}

// HashRecipientToken returns the indexed lookup form without exposing bearer material.
func HashRecipientToken(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}

// RecipientView is the authorised recipient delivery and postcard record pair.
type RecipientView struct {
	Delivery *core.Record
	Postcard *core.Record
}

// FindRecipientView resolves only canonical, unexpired, non-cancelled bearer tokens.
func FindRecipientView(app core.App, token string, now types.DateTime) (*RecipientView, error) {
	if !ValidRecipientToken(token) {
		return nil, ErrRecipientAccessDenied
	}
	delivery, err := app.FindFirstRecordByFilter(collectionDeliveries,
		"view_token_hash = {:hash} && view_expires_at > {:now} && status != 'cancelled'",
		map[string]any{"hash": HashRecipientToken(token), "now": now})
	if err != nil {
		return nil, ErrRecipientAccessDenied
	}
	postcard, err := app.FindRecordById(collectionPostcards, delivery.GetString("postcard"))
	if err != nil || postcard.GetString("status") == "cancelled" {
		return nil, ErrRecipientAccessDenied
	}
	return &RecipientView{Delivery: delivery, Postcard: postcard}, nil
}

// expandLegacyQueuedPostcards moves pre-lifecycle queued postcards into review without sending them.
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
		if err != nil || len(existing) != 0 {
			return err
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
			messageID := uuid.NewString()
			attempt := core.NewRecord(attempts)
			attempt.Set("delivery", delivery.Id)
			attempt.Set("sequence", 1)
			attempt.Set("status", "dead_lettered")
			attempt.Set("correlation_id", correlationID)
			attempt.Set("message_id", messageID)
			attempt.Set("attempt_count", 0)
			attempt.Set("max_attempts", defaultMaxAttempts)
			attempt.Set("available_at", now)
			attempt.Set("dead_lettered_at", now)
			attempt.Set("last_error_class", "legacy_unknown")
			attempt.Set("last_error_retryable", false)
			setIntegrationMessageProfile(attempt, delivery.Id, messageID, now)
			if err := txApp.Save(attempt); err != nil {
				return err
			}
		}
		return nil
	})
}

// MarkReceived records the first successful postcard pickup rendering.
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

// ResolveAttempt applies an operator resolution to a dead-lettered delivery attempt.
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
		attempt.Set("payload_purged_at", now)
		if err := txApp.Save(attempt); err != nil {
			return err
		}
		delivery, err := txApp.FindRecordById(collectionDeliveries, attempt.GetString("delivery"))
		if err != nil {
			return err
		}
		delivery.Set("recipient", "purged:"+delivery.Id)
		delivery.Set("recipient_purged_at", now)
		delivery.Set("view_token_envelope", "")
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
		return finalizePostcard(txApp, delivery.GetString("postcard"), now)
	})
}

// ReplayAttempt creates a linked queued message after a dead-lettered message is reconciled.
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
		if err := txApp.DB().NewQuery(`SELECT COALESCE(MAX(sequence), 0) AS sequence FROM postcard_delivery_attempts WHERE delivery = {:delivery}`).Bind(map[string]any{"delivery": attempt.GetString("delivery")}).One(&maxSequence); err != nil {
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
		messageID := uuid.NewString()
		replay = core.NewRecord(collection)
		replay.Set("delivery", attempt.GetString("delivery"))
		replay.Set("sequence", maxSequence.Sequence+1)
		replay.Set("replay_of", attempt.Id)
		replay.Set("causation_message_id", attempt.Id)
		replay.Set("status", "queued")
		replay.Set("correlation_id", attempt.GetString("correlation_id"))
		replay.Set("message_id", messageID)
		replay.Set("attempt_count", 0)
		replay.Set("max_attempts", attempt.GetInt("max_attempts"))
		replay.Set("available_at", now)
		setIntegrationMessageProfile(replay, attempt.GetString("delivery"), messageID, attempt.GetDateTime("payload_retention_until"))
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
		trimmed := strings.TrimSpace(recipient)
		parsed, err := mail.ParseAddress(trimmed)
		if err != nil || parsed.Address == "" || parsed.Address != trimmed || len(trimmed) > 254 {
			return nil, errors.New("postcard recipient must be a valid email address")
		}
		at := strings.LastIndex(parsed.Address, "@")
		if at <= 0 || at == len(parsed.Address)-1 {
			return nil, errors.New("postcard recipient must be a valid email address")
		}
		address := parsed.Address[:at+1] + strings.ToLower(parsed.Address[at+1:])
		if _, exists := unique[address]; exists {
			continue
		}
		unique[address] = struct{}{}
		normalised = append(normalised, address)
	}
	return normalised, nil
}
