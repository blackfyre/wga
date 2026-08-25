package postcards

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/testutils"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/mailer"
	"github.com/pocketbase/pocketbase/tools/types"
)

// TestQueueNormalisesRecipientsAndCreatesAttempts verifies durable recipient work is created once.
func TestQueueNormalisesRecipientsAndCreatesAttempts(t *testing.T) {
	app := testutils.NewTestApp(t)
	artworkID := installPostcardSchema(t, app)

	postcard, err := Queue(app, postcardTestKeyring(t), QueueInput{
		SenderName:  "sender",
		SenderEmail: "sender@example.test",
		Recipients:  []string{" First@Example.Test ", "First@example.test", "first@example.test", "second@example.test"},
		Message:     "message",
		ImageID:     artworkID,
	})
	if err != nil {
		t.Fatalf("queue postcard: %v", err)
	}
	if got, want := postcard.GetString("recipients"), "First@example.test,first@example.test,second@example.test"; got != want {
		t.Fatalf("recipients = %q, want %q", got, want)
	}
	if postcard.GetString("correlation_id") == "" {
		t.Fatal("expected correlation id")
	}
	deliveries, err := app.FindRecordsByFilter(collectionDeliveries, "postcard = {:postcard}", "", 0, 0, map[string]any{"postcard": postcard.Id})
	if err != nil {
		t.Fatalf("find deliveries: %v", err)
	}
	if got := len(deliveries); got != 3 {
		t.Fatalf("deliveries = %d, want 3", got)
	}
	attempts, err := app.FindRecordsByFilter(collectionDeliveryAttempts, "", "", 0, 0)
	if err != nil {
		t.Fatalf("find attempts: %v", err)
	}
	if got := len(attempts); got != 3 {
		t.Fatalf("attempts = %d, want 3", got)
	}
}

// TestCompleteMarksParentSentOnlyAfterEveryRecipient verifies parent completion waits for all recipients.
func TestCompleteMarksParentSentOnlyAfterEveryRecipient(t *testing.T) {
	app := testutils.NewTestApp(t)
	artworkID := installPostcardSchema(t, app)
	postcard, err := Queue(app, postcardTestKeyring(t), QueueInput{
		SenderName: "sender", SenderEmail: "sender@example.test", Recipients: []string{"first@example.test", "second@example.test"}, Message: "message", ImageID: artworkID,
	})
	if err != nil {
		t.Fatalf("queue postcard: %v", err)
	}

	first, err := claimDue(app, types.NowDateTime())
	if err != nil {
		t.Fatalf("claim first attempt: %v", err)
	}
	if first == nil {
		t.Fatal("expected first claim")
	}
	if err := startTransport(app, first, types.NowDateTime()); err != nil {
		t.Fatalf("start first transport: %v", err)
	}
	if err := complete(app, first, types.NowDateTime()); err != nil {
		t.Fatalf("complete first attempt: %v", err)
	}
	stored, err := app.FindRecordById(collectionPostcards, postcard.Id)
	if err != nil {
		t.Fatalf("reload postcard: %v", err)
	}
	if got := stored.GetString("status"); got != "queued" {
		t.Fatalf("status after partial delivery = %q, want queued", got)
	}

	second, err := claimDue(app, types.NowDateTime())
	if err != nil {
		t.Fatalf("claim second attempt: %v", err)
	}
	if second == nil {
		t.Fatal("expected second claim")
	}
	if err := startTransport(app, second, types.NowDateTime()); err != nil {
		t.Fatalf("start second transport: %v", err)
	}
	if err := complete(app, second, types.NowDateTime()); err != nil {
		t.Fatalf("complete second attempt: %v", err)
	}
	stored, err = app.FindRecordById(collectionPostcards, postcard.Id)
	if err != nil {
		t.Fatalf("reload completed postcard: %v", err)
	}
	if got := stored.GetString("status"); got != "sent" {
		t.Fatalf("status after complete delivery = %q, want sent", got)
	}
}

// TestMarkReceivedIsIdempotent verifies repeated pickup renders retain the first receipt timestamp.
func TestMarkReceivedIsIdempotent(t *testing.T) {
	app := testutils.NewTestApp(t)
	artworkID := installPostcardSchema(t, app)
	postcard, err := Queue(app, postcardTestKeyring(t), QueueInput{
		SenderName: "sender", SenderEmail: "sender@example.test", Recipients: []string{"recipient@example.test"}, Message: "message", ImageID: artworkID,
	})
	if err != nil {
		t.Fatalf("queue postcard: %v", err)
	}
	postcard.Set("status", "sent")
	if err := app.Save(postcard); err != nil {
		t.Fatalf("save sent postcard: %v", err)
	}
	if err := MarkReceived(app, postcard.Id); err != nil {
		t.Fatalf("mark received: %v", err)
	}
	stored, err := app.FindRecordById(collectionPostcards, postcard.Id)
	if err != nil {
		t.Fatalf("reload received postcard: %v", err)
	}
	firstReceivedAt := stored.GetString("received_at")
	if stored.GetString("status") != "received" || firstReceivedAt == "" {
		t.Fatalf("expected received postcard, got status=%q received_at=%q", stored.GetString("status"), firstReceivedAt)
	}
	if err := MarkReceived(app, postcard.Id); err != nil {
		t.Fatalf("mark received again: %v", err)
	}
	stored, err = app.FindRecordById(collectionPostcards, postcard.Id)
	if err != nil {
		t.Fatalf("reload postcard: %v", err)
	}
	if got := stored.GetString("received_at"); got != firstReceivedAt {
		t.Fatalf("received_at changed from %q to %q", firstReceivedAt, got)
	}
}

// TestEarlyReceiptTransitionsToReceivedAfterDeliveryCompletes verifies an early pickup remains recorded.
func TestEarlyReceiptTransitionsToReceivedAfterDeliveryCompletes(t *testing.T) {
	app := testutils.NewTestApp(t)
	artworkID := installPostcardSchema(t, app)
	postcard, err := Queue(app, postcardTestKeyring(t), QueueInput{
		SenderName: "sender", SenderEmail: "sender@example.test", Recipients: []string{"recipient@example.test"}, Message: "message", ImageID: artworkID,
	})
	if err != nil {
		t.Fatalf("queue postcard: %v", err)
	}
	if err := MarkReceived(app, postcard.Id); err != nil {
		t.Fatalf("record early receipt: %v", err)
	}
	stored, err := app.FindRecordById(collectionPostcards, postcard.Id)
	if err != nil {
		t.Fatalf("reload early receipt: %v", err)
	}
	if got := stored.GetString("status"); got != "queued" {
		t.Fatalf("early receipt status = %q, want queued", got)
	}
	if stored.GetString("received_at") == "" {
		t.Fatal("expected early receipt timestamp")
	}
	claim, err := claimDue(app, types.NowDateTime())
	if err != nil {
		t.Fatalf("claim attempt: %v", err)
	}
	if err := startTransport(app, claim, types.NowDateTime()); err != nil {
		t.Fatalf("start transport: %v", err)
	}
	if err := complete(app, claim, types.NowDateTime()); err != nil {
		t.Fatalf("complete attempt: %v", err)
	}
	stored, err = app.FindRecordById(collectionPostcards, postcard.Id)
	if err != nil {
		t.Fatalf("reload completed postcard: %v", err)
	}
	if got := stored.GetString("status"); got != "received" {
		t.Fatalf("completed early receipt status = %q, want received", got)
	}
}

// TestDeadLetterLeavesPostcardQueued verifies a failed attempt cannot mark its parent sent.
func TestDeadLetterLeavesPostcardQueued(t *testing.T) {
	app := testutils.NewTestApp(t)
	artworkID := installPostcardSchema(t, app)
	postcard, err := Queue(app, postcardTestKeyring(t), QueueInput{
		SenderName: "sender", SenderEmail: "sender@example.test", Recipients: []string{"recipient@example.test"}, Message: "message", ImageID: artworkID,
	})
	if err != nil {
		t.Fatalf("queue postcard: %v", err)
	}
	claim, err := claimDue(app, types.NowDateTime())
	if err != nil {
		t.Fatalf("claim attempt: %v", err)
	}
	if err := deadLetter(app, claim, deliveryFailure{class: "ambiguous_transport_outcome"}, types.NowDateTime()); err != nil {
		t.Fatalf("dead letter attempt: %v", err)
	}
	stored, err := app.FindRecordById(collectionPostcards, postcard.Id)
	if err != nil {
		t.Fatalf("reload postcard: %v", err)
	}
	if got := stored.GetString("status"); got != "queued" {
		t.Fatalf("status after failed delivery = %q, want queued", got)
	}
	if got := stored.GetString("sent_at"); got != "" {
		t.Fatalf("sent_at after failed delivery = %q, want empty", got)
	}
}

// TestClosedDeliveryResolutionCancelsParentPostcard verifies closed work terminally cancels the parent.
func TestClosedDeliveryResolutionCancelsParentPostcard(t *testing.T) {
	app := testutils.NewTestApp(t)
	artworkID := installPostcardSchema(t, app)
	postcard, err := Queue(app, postcardTestKeyring(t), QueueInput{
		SenderName: "sender", SenderEmail: "sender@example.test", Recipients: []string{"recipient@example.test"}, Message: "message", ImageID: artworkID,
	})
	if err != nil {
		t.Fatalf("queue postcard: %v", err)
	}
	claim, err := claimDue(app, types.NowDateTime())
	if err != nil {
		t.Fatalf("claim attempt: %v", err)
	}
	if err := deadLetter(app, claim, deliveryFailure{class: "smtp_permanent_failure"}, types.NowDateTime()); err != nil {
		t.Fatalf("dead letter attempt: %v", err)
	}
	if err := ResolveAttempt(app, claim.Attempt.Id, "closed_without_replay", "closed by operator"); err != nil {
		t.Fatalf("close delivery: %v", err)
	}
	stored, err := app.FindRecordById(collectionPostcards, postcard.Id)
	if err != nil {
		t.Fatalf("reload postcard: %v", err)
	}
	if got := stored.GetString("status"); got != "cancelled" {
		t.Fatalf("status after closed delivery = %q, want cancelled", got)
	}
}

// TestStartTransportRequiresTheClaimToken verifies a stale worker cannot start transport.
func TestStartTransportRequiresTheClaimToken(t *testing.T) {
	app := testutils.NewTestApp(t)
	artworkID := installPostcardSchema(t, app)
	if _, err := Queue(app, postcardTestKeyring(t), QueueInput{
		SenderName: "sender", SenderEmail: "sender@example.test", Recipients: []string{"recipient@example.test"}, Message: "message", ImageID: artworkID,
	}); err != nil {
		t.Fatalf("queue postcard: %v", err)
	}
	claim, err := claimDue(app, types.NowDateTime())
	if err != nil {
		t.Fatalf("claim attempt: %v", err)
	}
	if err := startTransport(app, claim, types.NowDateTime()); err != nil {
		t.Fatalf("start transport: %v", err)
	}
	attempt, err := app.FindRecordById(collectionDeliveryAttempts, claim.Attempt.Id)
	if err != nil {
		t.Fatalf("reload attempt: %v", err)
	}
	if got := attempt.GetString("transport_started_at"); got == "" {
		t.Fatal("expected transport start timestamp")
	}
	if got := attempt.GetInt("attempt_count"); got != 1 {
		t.Fatalf("attempt_count = %d, want 1", got)
	}
	claim.Token = "different-token"
	if err := startTransport(app, claim, types.NowDateTime()); err == nil {
		t.Fatal("expected a stale claim token to be rejected")
	}
}

// TestRetrySchedulesAClaimedAttempt verifies retryable failures receive a future availability time.
func TestRetrySchedulesAClaimedAttempt(t *testing.T) {
	app := testutils.NewTestApp(t)
	artworkID := installPostcardSchema(t, app)
	if _, err := Queue(app, postcardTestKeyring(t), QueueInput{
		SenderName: "sender", SenderEmail: "sender@example.test", Recipients: []string{"recipient@example.test"}, Message: "message", ImageID: artworkID,
	}); err != nil {
		t.Fatalf("queue postcard: %v", err)
	}
	now := types.NowDateTime()
	claim, err := claimDue(app, now)
	if err != nil {
		t.Fatalf("claim attempt: %v", err)
	}
	if err := startTransport(app, claim, now); err != nil {
		t.Fatalf("start transport: %v", err)
	}
	if err := retry(app, claim, deliveryFailure{class: "dial_failed", retryable: true}, now); err != nil {
		t.Fatalf("retry attempt: %v", err)
	}
	attempt, err := app.FindRecordById(collectionDeliveryAttempts, claim.Attempt.Id)
	if err != nil {
		t.Fatalf("reload attempt: %v", err)
	}
	if got := attempt.GetString("status"); got != "queued" {
		t.Fatalf("attempt status = %q, want queued", got)
	}
	if !attempt.GetDateTime("available_at").After(now) {
		t.Fatalf("retry availability = %s, want after %s", attempt.GetDateTime("available_at"), now)
	}
}

// TestExpiredPreTransportClaimDoesNotConsumeAttempt verifies safe lease expiry preserves retry budget.
func TestExpiredPreTransportClaimDoesNotConsumeAttempt(t *testing.T) {
	app := testutils.NewTestApp(t)
	artworkID := installPostcardSchema(t, app)
	if _, err := Queue(app, postcardTestKeyring(t), QueueInput{
		SenderName: "sender", SenderEmail: "sender@example.test", Recipients: []string{"recipient@example.test"}, Message: "message", ImageID: artworkID,
	}); err != nil {
		t.Fatalf("queue postcard: %v", err)
	}
	now := types.NowDateTime()
	claim, err := claimDue(app, now)
	if err != nil {
		t.Fatalf("claim attempt: %v", err)
	}
	if err := recoverExpiredClaims(app, now.Add(deliveryLease)); err != nil {
		t.Fatalf("recover expired claim: %v", err)
	}
	attempt, err := app.FindRecordById(collectionDeliveryAttempts, claim.Attempt.Id)
	if err != nil {
		t.Fatalf("reload attempt: %v", err)
	}
	if got := attempt.GetString("status"); got != "queued" {
		t.Fatalf("attempt status = %q, want queued", got)
	}
	if got := attempt.GetInt("attempt_count"); got != 0 {
		t.Fatalf("attempt_count = %d, want 0", got)
	}
}

// TestExpandLegacyQueuedPostcardsMovesAttemptsToReview verifies legacy work is held for review.
func TestExpandLegacyQueuedPostcardsMovesAttemptsToReview(t *testing.T) {
	app := testutils.NewTestApp(t)
	artworkID := installPostcardSchema(t, app)
	collection, err := app.FindCollectionByNameOrId(collectionPostcards)
	if err != nil {
		t.Fatalf("find postcards collection: %v", err)
	}
	legacy := core.NewRecord(collection)
	legacy.Set("status", "queued")
	legacy.Set("sender_name", "sender")
	legacy.Set("sender_email", "sender@example.test")
	legacy.Set("recipients", "recipient@example.test")
	legacy.Set("message", "message")
	legacy.Set("image_id", artworkID)
	if err := app.Save(legacy); err != nil {
		t.Fatalf("create legacy postcard: %v", err)
	}
	if err := expandLegacyQueuedPostcards(app); err != nil {
		t.Fatalf("expand legacy queue: %v", err)
	}
	attempts, err := app.FindRecordsByFilter(collectionDeliveryAttempts, "", "", 0, 0)
	if err != nil {
		t.Fatalf("find attempts: %v", err)
	}
	if got := len(attempts); got != 1 {
		t.Fatalf("legacy attempts = %d, want 1", got)
	}
	if got := attempts[0].GetString("status"); got != "dead_lettered" {
		t.Fatalf("legacy attempt status = %q, want dead_lettered", got)
	}
	if got := attempts[0].GetString("last_error_class"); got != "legacy_unknown" {
		t.Fatalf("legacy error class = %q, want legacy_unknown", got)
	}
}

// TestClassifyDeliveryError verifies only known pre-send failures are retryable.
func TestClassifyDeliveryError(t *testing.T) {
	if failure := classifyDeliveryError(&net.DNSError{IsTemporary: true}); !failure.retryable || failure.class != "transport_failure" {
		t.Fatalf("dns failure = %#v", failure)
	}
	if failure := classifyDeliveryError(&net.DNSError{}); failure.retryable || failure.class != "transport_failure" {
		t.Fatalf("permanent dns failure = %#v", failure)
	}
	if failure := classifyDeliveryError(errors.New("unknown transport failure")); failure.retryable || failure.class != "ambiguous_transport_outcome" {
		t.Fatalf("unknown failure = %#v", failure)
	}
}

// TestRenderMessageIncludesDeliveryHeader verifies provider reconciliation can use the durable ID.
func TestRenderMessageIncludesDeliveryHeader(t *testing.T) {
	postcard := core.NewRecord(core.NewBaseCollection("Postcards"))
	postcard.Set("sender_name", "sender")
	delivery := core.NewRecord(core.NewBaseCollection("Deliveries"))
	delivery.Id = "delivery-record-123"
	delivery.Set("recipient", "recipient@example.test")
	token, tokenErr := newRecipientToken()
	if tokenErr != nil {
		t.Fatal(tokenErr)
	}
	keyring := postcardTestKeyring(t)
	envelope, envelopeErr := sealRecipientToken(keyring, delivery.Id, token)
	if envelopeErr != nil {
		t.Fatal(envelopeErr)
	}
	delivery.Set("view_token_envelope", envelope)
	delivery.Set("view_token_hash", HashRecipientToken(token))
	delivery.Set("view_expires_at", types.NowDateTime().Add(time.Hour))
	message, err := renderMessage(postcard, delivery, "delivery-123", postcardTestConfig(t), keyring)
	if err != nil {
		t.Fatalf("render message: %v", err)
	}
	if got := message.Headers["X-WGA-Delivery-ID"]; got != "delivery-123" {
		t.Fatalf("delivery header = %q, want %q", got, "delivery-123")
	}
	if got := message.Headers["Message-ID"]; got != "<postcard-delivery-123@wga.invalid>" {
		t.Fatalf("message id = %q, want %q", got, "<postcard-delivery-123@wga.invalid>")
	}
}

// TestRenderMessageUsesConfiguredLogoAndStatesThirtyDayExpiry verifies the email
// embeds the configured public logo URL and states the bounded pickup window.
func TestRenderMessageUsesConfiguredLogoAndStatesThirtyDayExpiry(t *testing.T) {
	postcard := core.NewRecord(core.NewBaseCollection("Postcards"))
	postcard.Set("sender_name", "sender")
	delivery := core.NewRecord(core.NewBaseCollection("Deliveries"))
	delivery.Id = "delivery-record-123"
	delivery.Set("recipient", "recipient@example.test")
	token, tokenErr := newRecipientToken()
	if tokenErr != nil {
		t.Fatal(tokenErr)
	}
	keyring := postcardTestKeyring(t)
	envelope, envelopeErr := sealRecipientToken(keyring, delivery.Id, token)
	if envelopeErr != nil {
		t.Fatal(envelopeErr)
	}
	delivery.Set("view_token_envelope", envelope)
	delivery.Set("view_token_hash", HashRecipientToken(token))
	delivery.Set("view_expires_at", types.NowDateTime().Add(time.Hour))
	message, err := renderMessage(postcard, delivery, "delivery-123", postcardTestConfig(t), keyring)
	if err != nil {
		t.Fatalf("render message: %v", err)
	}
	if strings.Contains(message.HTML, "127.0.0.1") {
		t.Fatal("email contains loopback logo URL")
	}
	if !strings.Contains(message.HTML, `src="http://example.test/assets/images/logo.png"`) {
		t.Fatalf("email does not use configured logo URL: %s", message.HTML)
	}
	if !strings.Contains(message.HTML, "30 days") || strings.Contains(message.HTML, "indefinitely") {
		t.Fatalf("email expiry copy is not bounded: %s", message.HTML)
	}
}

// TestLogDeliveryUsesOnlySafeExecutionIdentifiers verifies delivery logs omit personal data.
func TestLogDeliveryUsesOnlySafeExecutionIdentifiers(t *testing.T) {
	app := testutils.NewTestApp(t)
	captured := testutils.CaptureLogs(app)
	attempt := core.NewRecord(core.NewBaseCollection("Attempt"))
	attempt.Set("correlation_id", "correlation-123")
	attempt.Set("attempt_count", 2)
	delivery := core.NewRecord(core.NewBaseCollection("Delivery"))
	delivery.Set("recipient", "secret-recipient@example.test")
	delivery.Set("view_token_envelope", "secret-view-token-envelope")
	logDelivery(app, "run-123", &ClaimedAttempt{Attempt: attempt, Delivery: delivery}, "sent")
	testutils.FlushLogs(t, app)

	entry := testutils.LogWithEvent(captured(), "postcard.delivery.attempt")
	if entry == nil {
		t.Fatal("expected delivery log")
	}
	for _, forbidden := range []string{"postcard_id", "recipient", "sender", "message", "error"} {
		if _, exists := entry.Data[forbidden]; exists {
			t.Fatalf("delivery log contains forbidden %q field", forbidden)
		}
	}
	output := fmt.Sprint(testutils.LogData(captured()))
	for _, secret := range []string{"secret-recipient@example.test", "secret-view-token-envelope"} {
		if strings.Contains(output, secret) {
			t.Fatalf("delivery log leaked %q", secret)
		}
	}
	if got := entry.Data["run_id"]; got != "run-123" {
		t.Fatalf("run_id = %v, want %q", got, "run-123")
	}
}

type recordingMailer struct{ messages []*mailer.Message }

func (m *recordingMailer) Send(message *mailer.Message) error {
	m.messages = append(m.messages, message)
	return nil
}

type retryingMailer struct {
	messages []*mailer.Message
	failures []error
}

func (m *retryingMailer) Send(message *mailer.Message) error {
	m.messages = append(m.messages, message)
	if len(m.failures) == 0 {
		return nil
	}
	failure := m.failures[0]
	m.failures = m.failures[1:]
	return failure
}

func TestQueueWithAccessCreatesBoundedOpaqueRecipientMessage(t *testing.T) {
	app := testutils.NewTestApp(t)
	artworkID := installPostcardSchema(t, app)
	now := types.NowDateTime()
	result, err := QueueWithAccess(app, postcardTestKeyring(t), QueueInput{
		SenderName: "Sender", SenderEmail: "sender@example.test", Recipients: []string{"recipient@example.test"}, Message: "Hello", ImageID: artworkID,
	}, now)
	if err != nil {
		t.Fatalf("queue postcard: %v", err)
	}
	if len(result.Access) != 1 || !ValidRecipientToken(result.Access[0].Token) {
		t.Fatalf("invalid recipient access: %#v", result.Access)
	}
	if result.Access[0].Token == result.Postcard.Id || strings.Contains(result.Access[0].Token, result.Postcard.Id) {
		t.Fatal("token exposes postcard id")
	}
	delivery, err := app.FindRecordById(collectionDeliveries, result.Access[0].DeliveryID)
	if err != nil {
		t.Fatalf("find delivery: %v", err)
	}
	if got := delivery.GetString("view_token_hash"); got != HashRecipientToken(result.Access[0].Token) {
		t.Fatalf("token hash = %q", got)
	}
	envelope := delivery.GetString("view_token_envelope")
	if envelope == "" || strings.Contains(envelope, result.Access[0].Token) {
		t.Fatal("durable envelope is empty or contains the plaintext token")
	}
	recovered, err := recoverRecipientToken(postcardTestKeyring(t), delivery.Id, envelope, delivery.GetString("view_token_hash"))
	if err != nil || recovered != result.Access[0].Token {
		t.Fatalf("recover queued token: token_match=%t err=%v", recovered == result.Access[0].Token, err)
	}
	if got := delivery.GetDateTime("view_expires_at").Time(); got.Sub(now.Add(RecipientTokenValidity).Time()).Abs() > time.Second {
		t.Fatalf("expiry = %s", got)
	}
	attempts, err := app.FindRecordsByFilter(collectionDeliveryAttempts, "", "", 0, 0)
	if err != nil || len(attempts) != 1 {
		t.Fatalf("attempts = %d, err=%v", len(attempts), err)
	}
	attempt := attempts[0]
	for field, want := range map[string]string{"direction": "outbound", "integration_key": "postcard_email", "message_type": "recipient_link_delivery", "payload_reference": delivery.Id, "payload_format": "normalised_record"} {
		if got := attempt.GetString(field); got != want {
			t.Fatalf("%s = %q, want %q", field, got, want)
		}
	}
	if attempt.GetString("deduplication_key") != attempt.GetString("message_id") {
		t.Fatal("deduplication key is not the stable message id")
	}
}

func TestQueueRejectsUnpublishedArtworkWithoutPartialRecords(t *testing.T) {
	app := testutils.NewTestApp(t)
	artworkID := installPostcardSchema(t, app)
	artwork, _ := app.FindRecordById("artworks", artworkID)
	artwork.Set("published", false)
	if err := app.Save(artwork); err != nil {
		t.Fatal(err)
	}
	_, err := QueueWithAccess(app, postcardTestKeyring(t), QueueInput{SenderName: "Sender", SenderEmail: "sender@example.test", Recipients: []string{"recipient@example.test"}, Message: "Hello", ImageID: artworkID}, types.NowDateTime())
	if !errors.Is(err, ErrArtworkUnavailable) {
		t.Fatalf("error = %v", err)
	}
	postcards, _ := app.FindRecordsByFilter(collectionPostcards, "", "", 0, 0)
	attempts, _ := app.FindRecordsByFilter(collectionDeliveryAttempts, "", "", 0, 0)
	if len(postcards) != 0 || len(attempts) != 0 {
		t.Fatalf("partial records: postcards=%d attempts=%d", len(postcards), len(attempts))
	}
}

func TestQueueRejectsHeaderBreakingSenderNameWithoutPartialRecords(t *testing.T) {
	app := testutils.NewTestApp(t)
	artworkID := installPostcardSchema(t, app)
	_, err := QueueWithAccess(app, postcardTestKeyring(t), QueueInput{
		SenderName: "Sender\r\nBcc: hidden@example.test", SenderEmail: "sender@example.test",
		Recipients: []string{"recipient@example.test"}, Message: "Hello", ImageID: artworkID,
	}, types.NowDateTime())
	if !errors.Is(err, ErrInvalidPostcard) {
		t.Fatalf("error = %v", err)
	}
	postcards, _ := app.FindRecordsByFilter(collectionPostcards, "", "", 0, 0)
	if len(postcards) != 0 {
		t.Fatalf("postcards = %d, want 0", len(postcards))
	}
}

func TestQueueRejectsInvalidKeyringWithoutPartialRecords(t *testing.T) {
	app := testutils.NewTestApp(t)
	artworkID := installPostcardSchema(t, app)

	_, err := QueueWithAccess(app, config.PostcardTokenKeyring{}, QueueInput{
		SenderName: "Sender", SenderEmail: "sender@example.test", Recipients: []string{"recipient@example.test"}, Message: "Hello", ImageID: artworkID,
	}, types.NowDateTime())
	if !errors.Is(err, errInvalidRecipientTokenKeyring) {
		t.Fatalf("error = %v, want invalid token keyring", err)
	}
	for _, collection := range []string{collectionPostcards, collectionDeliveries, collectionDeliveryAttempts} {
		records, findErr := app.FindRecordsByFilter(collection, "", "", 0, 0)
		if findErr != nil {
			t.Fatalf("find %s records: %v", collection, findErr)
		}
		if len(records) != 0 {
			t.Fatalf("%s partial records = %d, want 0", collection, len(records))
		}
	}
}

func TestQueueRollsBackPostcardAndEncryptedDeliveryOnPersistenceFailure(t *testing.T) {
	app := testutils.NewTestApp(t)
	artworkID := installPostcardSchema(t, app)
	attempts, err := app.FindCollectionByNameOrId(collectionDeliveryAttempts)
	if err != nil {
		t.Fatal(err)
	}
	attempts.Fields.Add(&core.TextField{Name: "required_test_marker", Required: true})
	if err := app.Save(attempts); err != nil {
		t.Fatal(err)
	}

	_, err = QueueWithAccess(app, postcardTestKeyring(t), QueueInput{
		SenderName: "Sender", SenderEmail: "sender@example.test", Recipients: []string{"recipient@example.test"}, Message: "Hello", ImageID: artworkID,
	}, types.NowDateTime())
	if err == nil {
		t.Fatal("expected delivery-attempt persistence failure")
	}
	for _, collection := range []string{collectionPostcards, collectionDeliveries, collectionDeliveryAttempts} {
		records, findErr := app.FindRecordsByFilter(collection, "", "", 0, 0)
		if findErr != nil {
			t.Fatalf("find %s records: %v", collection, findErr)
		}
		if len(records) != 0 {
			t.Fatalf("%s partial records = %d, want 0", collection, len(records))
		}
	}
}

func TestProcessDueSendsOnceAndPurgesDirectIdentifiers(t *testing.T) {
	app := testutils.NewTestApp(t)
	artworkID := installPostcardSchema(t, app)
	result, err := QueueWithAccess(app, postcardTestKeyring(t), QueueInput{SenderName: "Sender", SenderEmail: "sender@example.test", Recipients: []string{"recipient@example.test"}, Message: "Hello", ImageID: artworkID}, types.NowDateTime())
	if err != nil {
		t.Fatal(err)
	}
	transport := &recordingMailer{}
	if err := ProcessDue(app, transport, postcardTestConfig(t), "run-1"); err != nil {
		t.Fatalf("process due: %v", err)
	}
	if err := ProcessDue(app, transport, postcardTestConfig(t), "run-2"); err != nil {
		t.Fatalf("process due again: %v", err)
	}
	if len(transport.messages) != 1 {
		t.Fatalf("provider calls = %d, want 1", len(transport.messages))
	}
	if !strings.Contains(transport.messages[0].HTML, "token="+result.Access[0].Token) || strings.Contains(transport.messages[0].HTML, "?p="+result.Postcard.Id) {
		t.Fatal("delivery did not use opaque token URL")
	}
	delivery, _ := app.FindRecordById(collectionDeliveries, result.Access[0].DeliveryID)
	if delivery.GetString("view_token_envelope") != "" || !strings.HasPrefix(delivery.GetString("recipient"), "purged:") || delivery.GetString("view_token_hash") == "" {
		t.Fatalf("delivery privacy state: recipient=%q envelope_present=%t hash_present=%t", delivery.GetString("recipient"), delivery.GetString("view_token_envelope") != "", delivery.GetString("view_token_hash") != "")
	}
	postcard, _ := app.FindRecordById(collectionPostcards, result.Postcard.Id)
	if !strings.HasSuffix(postcard.GetString("sender_email"), "@invalid.test") || postcard.GetString("sender_email_purged_at") == "" || postcard.GetString("recipients") != "purged" {
		t.Fatal("sender email was not purged after terminal delivery")
	}
}

func TestProcessDueDeadLettersInvalidAccessBeforeTransport(t *testing.T) {
	for _, test := range []struct {
		name          string
		postcards     func(*testing.T) config.Postcards
		mutateAccess  bool
	}{
		{name: "tampered envelope", postcards: postcardTestConfig, mutateAccess: true},
		{name: "missing configured key", postcards: postcardConfigWithoutTokenKeys},
	} {
		t.Run(test.name, func(t *testing.T) {
			app := testutils.NewTestApp(t)
			artworkID := installPostcardSchema(t, app)
			result, err := QueueWithAccess(app, postcardTestKeyring(t), QueueInput{
				SenderName: "Sender", SenderEmail: "sender@example.test", Recipients: []string{"recipient@example.test"}, Message: "Hello", ImageID: artworkID,
			}, types.NowDateTime())
			if err != nil {
				t.Fatal(err)
			}
			if test.mutateAccess {
				delivery, findErr := app.FindRecordById(collectionDeliveries, result.Access[0].DeliveryID)
				if findErr != nil {
					t.Fatal(findErr)
				}
				delivery.Set("view_token_envelope", tamperEnvelope(delivery.GetString("view_token_envelope")))
				if saveErr := app.Save(delivery); saveErr != nil {
					t.Fatal(saveErr)
				}
			}

			transport := &recordingMailer{}
			if err := ProcessDue(app, transport, test.postcards(t), "run-invalid"); err != nil {
				t.Fatalf("process invalid delivery: %v", err)
			}
			if len(transport.messages) != 0 {
				t.Fatalf("provider calls = %d, want 0", len(transport.messages))
			}
			attempts, findErr := app.FindRecordsByFilter(collectionDeliveryAttempts, "", "", 0, 0)
			if findErr != nil || len(attempts) != 1 {
				t.Fatalf("attempts = %d, err=%v", len(attempts), findErr)
			}
			attempt := attempts[0]
			if attempt.GetString("status") != "dead_lettered" || attempt.GetString("last_error_class") != "invalid_message" || attempt.GetBool("last_error_retryable") {
				t.Fatalf("invalid delivery state: status=%q class=%q retryable=%t", attempt.GetString("status"), attempt.GetString("last_error_class"), attempt.GetBool("last_error_retryable"))
			}
			if attempt.GetInt("attempt_count") != 0 || attempt.GetString("transport_started_at") != "" {
				t.Fatal("invalid access crossed the transport boundary")
			}
		})
	}
}

func TestProcessDueRetryUsesStableTokenURLAndMessageID(t *testing.T) {
	app := testutils.NewTestApp(t)
	artworkID := installPostcardSchema(t, app)
	result, err := QueueWithAccess(app, postcardTestKeyring(t), QueueInput{
		SenderName: "Sender", SenderEmail: "sender@example.test", Recipients: []string{"recipient@example.test"}, Message: "Hello", ImageID: artworkID,
	}, types.NowDateTime())
	if err != nil {
		t.Fatal(err)
	}

	transport := &retryingMailer{failures: []error{&net.OpError{Op: "dial", Err: errors.New("transport unavailable")}}}
	if err := ProcessDue(app, transport, postcardTestConfig(t), "run-retry-1"); err != nil {
		t.Fatalf("first delivery run: %v", err)
	}
	attempts, err := app.FindRecordsByFilter(collectionDeliveryAttempts, "", "", 0, 0)
	if err != nil || len(attempts) != 1 {
		t.Fatalf("attempts = %d, err=%v", len(attempts), err)
	}
	attempts[0].Set("available_at", types.NowDateTime().Add(-time.Minute))
	if err := app.Save(attempts[0]); err != nil {
		t.Fatal(err)
	}
	delivery, err := app.FindRecordById(collectionDeliveries, result.Access[0].DeliveryID)
	if err != nil || delivery.GetString("view_token_envelope") == "" {
		t.Fatalf("retry envelope missing: err=%v", err)
	}

	if err := ProcessDue(app, transport, postcardTestConfig(t), "run-retry-2"); err != nil {
		t.Fatalf("second delivery run: %v", err)
	}
	if len(transport.messages) != 2 {
		t.Fatalf("provider calls = %d, want 2", len(transport.messages))
	}
	first := transport.messages[0]
	second := transport.messages[1]
	if first.HTML != second.HTML || first.Headers["Message-ID"] != second.Headers["Message-ID"] || first.Headers["X-WGA-Delivery-ID"] != second.Headers["X-WGA-Delivery-ID"] {
		t.Fatal("retry changed the recipient URL or durable message identifiers")
	}
	if !strings.Contains(second.HTML, "token="+result.Access[0].Token) {
		t.Fatal("retry did not recover the original token")
	}
}

func TestReplayRetainsUnresolvedRecipientEnvelope(t *testing.T) {
	app := testutils.NewTestApp(t)
	artworkID := installPostcardSchema(t, app)
	result, err := QueueWithAccess(app, postcardTestKeyring(t), QueueInput{
		SenderName: "Sender", SenderEmail: "sender@example.test", Recipients: []string{"recipient@example.test"}, Message: "Hello", ImageID: artworkID,
	}, types.NowDateTime())
	if err != nil {
		t.Fatal(err)
	}
	delivery, err := app.FindRecordById(collectionDeliveries, result.Access[0].DeliveryID)
	if err != nil {
		t.Fatal(err)
	}
	envelope := delivery.GetString("view_token_envelope")
	claim, err := claimDue(app, types.NowDateTime())
	if err != nil {
		t.Fatal(err)
	}
	if err := deadLetter(app, claim, deliveryFailure{class: "invalid_request"}, types.NowDateTime()); err != nil {
		t.Fatal(err)
	}
	if _, err := ReplayAttempt(app, claim.Attempt.Id); err != nil {
		t.Fatal(err)
	}

	delivery, err = app.FindRecordById(collectionDeliveries, result.Access[0].DeliveryID)
	if err != nil || delivery.GetString("view_token_envelope") != envelope {
		t.Fatalf("replay changed recipient envelope: err=%v", err)
	}
	original, err := app.FindRecordById(collectionDeliveryAttempts, claim.Attempt.Id)
	if err != nil {
		t.Fatal(err)
	}
	if original.GetString("payload_purged_at") != "" {
		t.Fatal("replay marked the still-required payload purged")
	}
}

func TestExpiredAccessCleanupRetainsActiveAndUnresolvedWork(t *testing.T) {
	app := testutils.NewTestApp(t)
	artworkID := installPostcardSchema(t, app)
	past := types.NowDateTime().Add(-31 * 24 * time.Hour)
	result, err := QueueWithAccess(app, postcardTestKeyring(t), QueueInput{SenderName: "Sender", SenderEmail: "sender@example.test", Recipients: []string{"recipient@example.test"}, Message: "Hello", ImageID: artworkID}, past)
	if err != nil {
		t.Fatal(err)
	}
	now := types.NowDateTime()
	if _, err := PurgeExpiredRecipientAccess(app, now, maxPurgeBatchSize); err != nil {
		t.Fatal(err)
	}
	delivery, _ := app.FindRecordById(collectionDeliveries, result.Access[0].DeliveryID)
	if delivery.GetString("view_token_hash") == "" || delivery.GetString("view_token_envelope") == "" {
		t.Fatal("active queued payload was purged")
	}
	claim, err := claimDue(app, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := deadLetter(app, claim, deliveryFailure{class: "invalid_request"}, now); err != nil {
		t.Fatal(err)
	}
	if _, err := PurgeExpiredRecipientAccess(app, now, maxPurgeBatchSize); err != nil {
		t.Fatal(err)
	}
	delivery, _ = app.FindRecordById(collectionDeliveries, result.Access[0].DeliveryID)
	if delivery.GetString("view_token_hash") == "" || delivery.GetString("view_token_envelope") == "" {
		t.Fatal("unresolved dead letter was purged")
	}
	if err := ResolveAttempt(app, claim.Attempt.Id, "closed_without_replay", "test closure"); err != nil {
		t.Fatal(err)
	}
	if _, err := PurgeExpiredRecipientAccess(app, now, maxPurgeBatchSize); err != nil {
		t.Fatal(err)
	}
	delivery, _ = app.FindRecordById(collectionDeliveries, result.Access[0].DeliveryID)
	if delivery.GetString("view_token_hash") != "" || delivery.GetString("view_token_envelope") != "" {
		t.Fatal("resolved expired recipient access was retained")
	}
	attempt, _ := app.FindRecordById(collectionDeliveryAttempts, claim.Attempt.Id)
	if attempt.GetString("payload_purged_at") == "" {
		t.Fatal("resolved delivery attempt did not record payload purge")
	}
	postcard, _ := app.FindRecordById(collectionPostcards, result.Postcard.Id)
	if postcard.GetString("sender_name") != "Anonymous" || postcard.GetString("message") != "<p>Expired postcard</p>" || postcard.GetString("content_purged_at") == "" {
		t.Fatal("expired postcard content was not anonymised")
	}
}

func TestFindRecipientViewEnforcesTokenExpiryAndDenial(t *testing.T) {
	app := testutils.NewTestApp(t)
	artworkID := installPostcardSchema(t, app)
	now := types.NowDateTime()
	result, err := QueueWithAccess(app, postcardTestKeyring(t), QueueInput{SenderName: "Sender", SenderEmail: "sender@example.test", Recipients: []string{"recipient@example.test"}, Message: "Hello", ImageID: artworkID}, now)
	if err != nil {
		t.Fatal(err)
	}
	access := result.Access[0]
	view, err := FindRecipientView(app, access.Token, now)
	if err != nil || view.Postcard.Id != result.Postcard.Id {
		t.Fatalf("valid token denied: view=%v err=%v", view, err)
	}
	for name, token := range map[string]struct {
		token string
		at    types.DateTime
	}{
		"arbitrary id":  {result.Postcard.Id, now},
		"unknown token": {strings.Repeat("A", len(access.Token)), now},
		"expired token": {access.Token, access.ExpiresAt},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := FindRecipientView(app, token.token, token.at); !errors.Is(err, ErrRecipientAccessDenied) {
				t.Fatalf("error = %v", err)
			}
		})
	}
	delivery, _ := app.FindRecordById(collectionDeliveries, access.DeliveryID)
	delivery.Set("status", "cancelled")
	if err := app.Save(delivery); err != nil {
		t.Fatal(err)
	}
	if _, err := FindRecipientView(app, access.Token, now); !errors.Is(err, ErrRecipientAccessDenied) {
		t.Fatalf("cancelled access error = %v", err)
	}
}

func TestRecipientTokenValidationRejectsArbitraryIdentifiers(t *testing.T) {
	token, err := newRecipientToken()
	if err != nil {
		t.Fatal(err)
	}
	if !ValidRecipientToken(token) {
		t.Fatal("generated token rejected")
	}
	for _, invalid := range []string{"", "abc", "0123456789abcde", token + "=", strings.Repeat("a", len(token)) + "/"} {
		if ValidRecipientToken(invalid) {
			t.Fatalf("accepted invalid token %q", invalid)
		}
	}
}

// installPostcardSchema creates the minimal schema required by postcard workflow tests.
func installPostcardSchema(t *testing.T, app core.App) string {
	t.Helper()
	artworks := core.NewBaseCollection("Artworks")
	artworks.Fields.Add(&core.BoolField{Name: "published"})
	artworks.Id = "artworks"
	artworks.MarkAsNew()
	if err := app.Save(artworks); err != nil {
		t.Fatalf("create artwork collection: %v", err)
	}
	artwork := core.NewRecord(artworks)
	artwork.Set("published", true)
	if err := app.Save(artwork); err != nil {
		t.Fatalf("create artwork: %v", err)
	}

	postcards := core.NewBaseCollection("Postcards")
	postcards.Id = collectionPostcards
	postcards.MarkAsNew()
	postcards.Fields.Add(
		&core.TextField{Name: "sender_name", Required: true},
		&core.EmailField{Name: "sender_email", Required: true},
		&core.TextField{Name: "recipients", Required: true},
		&core.EditorField{Name: "message", Required: true},
		&core.RelationField{Name: "image_id", CollectionId: artworks.Id, Required: true},
		&core.BoolField{Name: "notify_sender"},
		&core.SelectField{Name: "status", Values: []string{"queued", "sent", "received", "cancelled"}, MaxSelect: 1, Required: true},
		&core.DateField{Name: "sent_at"},
		&core.TextField{Name: "correlation_id"},
		&core.DateField{Name: "received_at"},
		&core.BoolField{Name: "include_music"},
		&core.DateField{Name: "retention_until"},
		&core.DateField{Name: "sender_email_purged_at"},
		&core.DateField{Name: "content_purged_at"},
	)
	if err := app.Save(postcards); err != nil {
		t.Fatalf("create postcards collection: %v", err)
	}

	deliveries := core.NewBaseCollection("postcard_deliveries")
	deliveries.Id = collectionDeliveries
	deliveries.MarkAsNew()
	deliveries.Fields.Add(
		&core.RelationField{Name: "postcard", CollectionId: postcards.Id, Required: true},
		&core.TextField{Name: "recipient", Required: true},
		&core.SelectField{Name: "status", Values: []string{"pending", "sent", "cancelled"}, MaxSelect: 1, Required: true},
		&core.DateField{Name: "sent_at"},
		&core.DateField{Name: "cancelled_at"},
		&core.TextField{Name: "view_token_envelope"},
		&core.TextField{Name: "view_token_hash"},
		&core.DateField{Name: "view_expires_at"},
		&core.DateField{Name: "recipient_purged_at"},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	if err := app.Save(deliveries); err != nil {
		t.Fatalf("create deliveries collection: %v", err)
	}

	attempts := core.NewBaseCollection("postcard_delivery_attempts")
	attempts.Id = collectionDeliveryAttempts
	attempts.MarkAsNew()
	attempts.Fields.Add(
		&core.RelationField{Name: "delivery", CollectionId: deliveries.Id, Required: true},
		&core.NumberField{Name: "sequence", Required: true},
		&core.SelectField{Name: "status", Values: []string{"queued", "processing", "processed", "dead_lettered", "cancelled"}, MaxSelect: 1, Required: true},
		&core.TextField{Name: "correlation_id", Required: true},
		&core.TextField{Name: "message_id", Required: true},
		&core.NumberField{Name: "attempt_count"},
		&core.NumberField{Name: "max_attempts", Required: true},
		&core.DateField{Name: "available_at", Required: true},
		&core.TextField{Name: "claim_token"},
		&core.DateField{Name: "claim_expires_at"},
		&core.DateField{Name: "transport_started_at"},
		&core.DateField{Name: "last_attempt_at"},
		&core.DateField{Name: "processed_at"},
		&core.DateField{Name: "dead_lettered_at"},
		&core.TextField{Name: "result_code"},
		&core.TextField{Name: "last_error_class"},
		&core.BoolField{Name: "last_error_retryable"},
		&core.TextField{Name: "last_error_summary"},
		&core.SelectField{Name: "resolution_code", Values: []string{"replayed_unmodified", "resolved_manually", "closed_without_replay", "ignored_duplicate"}, MaxSelect: 1},
		&core.TextField{Name: "resolution_summary"},
		&core.DateField{Name: "resolved_at"},
		&core.SelectField{Name: "direction", Values: []string{"outbound", "inbound"}, MaxSelect: 1},
		&core.TextField{Name: "integration_key"},
		&core.TextField{Name: "message_type"},
		&core.TextField{Name: "external_message_id"},
		&core.TextField{Name: "deduplication_key"},
		&core.TextField{Name: "payload_reference"},
		&core.TextField{Name: "payload_format"},
		&core.DateField{Name: "payload_retention_until"},
		&core.DateField{Name: "payload_purged_at"},
		&core.JSONField{Name: "transport_metadata"},
		&core.TextField{Name: "last_error_code"},
		&core.TextField{Name: "result_summary"},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	if err := app.Save(attempts); err != nil {
		t.Fatalf("create attempts collection: %v", err)
	}
	attempts.Fields.Add(
		&core.RelationField{Name: "causation_message_id", CollectionId: attempts.Id, MaxSelect: 1},
		&core.RelationField{Name: "replay_of", CollectionId: attempts.Id, MaxSelect: 1},
	)
	if err := app.Save(attempts); err != nil {
		t.Fatalf("add attempt relations: %v", err)
	}

	return artwork.Id
}

// postcardTestConfig returns deterministic postcard settings for message rendering tests.
func postcardTestConfig(t *testing.T) config.Postcards {
	t.Helper()
	keys := map[string][]byte{"primary": bytes.Repeat([]byte{0x42}, 32)}
	values := map[string]string{
		"WGA_ENV":                          "test",
		"WGA_PROTOCOL":                     "http",
		"WGA_HOSTNAME":                     "example.test",
		"WGA_SENDER_NAME":                  "WGA",
		"WGA_SENDER_ADDRESS":               "sender@example.test",
		"WGA_POSTCARD_TOKEN_KEYS":          encodedPostcardTokenKeys(t, keys),
		"WGA_POSTCARD_TOKEN_ACTIVE_KEY_ID": "primary",
	}
	runtimeConfig := config.LoadFrom(func(key string) string {
		return values[key]
	})
	server, err := runtimeConfig.Server()
	if err != nil {
		t.Fatalf("load postcard config: %v", err)
	}
	return server.Postcards
}

func postcardConfigWithoutTokenKeys(t *testing.T) config.Postcards {
	t.Helper()
	values := map[string]string{
		"WGA_ENV":            "test",
		"WGA_PROTOCOL":       "http",
		"WGA_HOSTNAME":       "example.test",
		"WGA_SENDER_NAME":    "WGA",
		"WGA_SENDER_ADDRESS": "sender@example.test",
	}
	server, err := config.LoadFrom(func(key string) string { return values[key] }).Server()
	if err != nil {
		t.Fatalf("load postcard config without token keys: %v", err)
	}

	return server.Postcards
}

func postcardTestKeyring(t *testing.T) config.PostcardTokenKeyring {
	t.Helper()

	return testPostcardKeyring(t, "primary", map[string][]byte{"primary": bytes.Repeat([]byte{0x42}, 32)})
}

func testPostcardKeyring(t *testing.T, activeKeyID string, keys map[string][]byte) config.PostcardTokenKeyring {
	t.Helper()
	values := map[string]string{
		"WGA_POSTCARD_TOKEN_KEYS":          encodedPostcardTokenKeys(t, keys),
		"WGA_POSTCARD_TOKEN_ACTIVE_KEY_ID": activeKeyID,
	}
	keyring, err := config.LoadFrom(func(key string) string { return values[key] }).PostcardTokenKeyring()
	if err != nil {
		t.Fatalf("load postcard token keyring: %v", err)
	}

	return keyring
}

func encodedPostcardTokenKeys(t *testing.T, keys map[string][]byte) string {
	t.Helper()
	encoded := make(map[string]string, len(keys))
	for keyID, material := range keys {
		encoded[keyID] = base64.RawURLEncoding.EncodeToString(material)
	}
	payload, err := json.Marshal(encoded)
	if err != nil {
		t.Fatalf("marshal postcard token keys: %v", err)
	}

	return string(payload)
}

func tamperEnvelope(envelope string) string {
	if envelope == "" {
		return "A"
	}
	replacement := byte('A')
	if envelope[len(envelope)-1] == replacement {
		replacement = 'B'
	}

	return envelope[:len(envelope)-1] + string(replacement)
}
