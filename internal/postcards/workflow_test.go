package postcards

import (
	"errors"
	"net"
	"net/mail"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/testutils"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/mailer"
	"github.com/pocketbase/pocketbase/tools/types"
)

func TestQueueNormalisesRecipientsAndCreatesAttempts(t *testing.T) {
	app := testutils.NewTestApp(t)
	artworkID := installPostcardSchema(t, app)

	postcard, err := Queue(app, QueueInput{
		SenderName:  "sender",
		SenderEmail: "sender@example.test",
		Recipients:  []string{" First@Example.Test ", "first@example.test", "second@example.test"},
		Message:     "message",
		ImageID:     artworkID,
	})
	if err != nil {
		t.Fatalf("queue postcard: %v", err)
	}
	if got, want := postcard.GetString("recipients"), "first@example.test,second@example.test"; got != want {
		t.Fatalf("recipients = %q, want %q", got, want)
	}
	if postcard.GetString("correlation_id") == "" {
		t.Fatal("expected correlation id")
	}
	deliveries, err := app.FindRecordsByFilter(collectionDeliveries, "postcard = {:postcard}", "", 0, 0, map[string]any{"postcard": postcard.Id})
	if err != nil {
		t.Fatalf("find deliveries: %v", err)
	}
	if got := len(deliveries); got != 2 {
		t.Fatalf("deliveries = %d, want 2", got)
	}
	attempts, err := app.FindRecordsByFilter(collectionDeliveryAttempts, "", "", 0, 0)
	if err != nil {
		t.Fatalf("find attempts: %v", err)
	}
	if got := len(attempts); got != 2 {
		t.Fatalf("attempts = %d, want 2", got)
	}
}

func TestCompleteMarksParentSentOnlyAfterEveryRecipient(t *testing.T) {
	app := testutils.NewTestApp(t)
	artworkID := installPostcardSchema(t, app)
	postcard, err := Queue(app, QueueInput{
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

func TestMarkReceivedIsIdempotent(t *testing.T) {
	app := testutils.NewTestApp(t)
	artworkID := installPostcardSchema(t, app)
	postcard, err := Queue(app, QueueInput{
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

func TestEarlyReceiptTransitionsToReceivedAfterDeliveryCompletes(t *testing.T) {
	app := testutils.NewTestApp(t)
	artworkID := installPostcardSchema(t, app)
	postcard, err := Queue(app, QueueInput{
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

func TestDeadLetterLeavesPostcardQueued(t *testing.T) {
	app := testutils.NewTestApp(t)
	artworkID := installPostcardSchema(t, app)
	postcard, err := Queue(app, QueueInput{
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

func TestStartTransportRequiresTheClaimToken(t *testing.T) {
	app := testutils.NewTestApp(t)
	artworkID := installPostcardSchema(t, app)
	if _, err := Queue(app, QueueInput{
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

func TestRetrySchedulesAClaimedAttempt(t *testing.T) {
	app := testutils.NewTestApp(t)
	artworkID := installPostcardSchema(t, app)
	if _, err := Queue(app, QueueInput{
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

func TestExpiredPreTransportClaimDoesNotConsumeAttempt(t *testing.T) {
	app := testutils.NewTestApp(t)
	artworkID := installPostcardSchema(t, app)
	if _, err := Queue(app, QueueInput{
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

func TestClassifyDeliveryError(t *testing.T) {
	if failure := classifyDeliveryError(&net.DNSError{}); !failure.retryable || failure.class != "dns_failed" {
		t.Fatalf("dns failure = %#v", failure)
	}
	if failure := classifyDeliveryError(errors.New("unknown transport failure")); failure.retryable || failure.class != "ambiguous_transport_outcome" {
		t.Fatalf("unknown failure = %#v", failure)
	}
}

func TestRenderMessageIncludesDeliveryHeader(t *testing.T) {
	postcard := core.NewRecord(core.NewBaseCollection("Postcards"))
	postcard.Set("sender_name", "sender")
	message, err := renderMessage(postcard, "recipient@example.test", "delivery-123", postcardTestConfig(t))
	if err != nil {
		t.Fatalf("render message: %v", err)
	}
	if got := message.Headers["X-WGA-Delivery-ID"]; got != "delivery-123" {
		t.Fatalf("delivery header = %q, want %q", got, "delivery-123")
	}
}

func TestSendMail(t *testing.T) {
	app := testutils.NewTestApp(t)
	message := &mailer.Message{
		From:    mail.Address{Name: "sender", Address: "sender@example.com"},
		To:      []mail.Address{{Name: "recipient", Address: "recipient@example.com"}},
		Subject: "Test Subject",
		HTML:    "<html><body>Test Body</body></html>",
	}
	if err := app.NewMailClient().Send(message); err != nil {
		if strings.Contains(err.Error(), "failed to locate a sendmail executable path") {
			t.Skip("sendmail is unavailable in this environment")
		}
		t.Fatalf("send mail: %v", err)
	}
}

func TestLogDeliveryUsesOnlySafeExecutionIdentifiers(t *testing.T) {
	app := testutils.NewTestApp(t)
	captured := testutils.CaptureLogs(app)
	attempt := core.NewRecord(core.NewBaseCollection("Attempt"))
	attempt.Set("correlation_id", "correlation-123")
	attempt.Set("attempt_count", 2)
	delivery := core.NewRecord(core.NewBaseCollection("Delivery"))
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
	if got := entry.Data["run_id"]; got != "run-123" {
		t.Fatalf("run_id = %v, want %q", got, "run-123")
	}
}

func installPostcardSchema(t *testing.T, app core.App) string {
	t.Helper()
	artworks := core.NewBaseCollection("Artworks")
	artworks.Id = "artworks"
	artworks.MarkAsNew()
	if err := app.Save(artworks); err != nil {
		t.Fatalf("create artwork collection: %v", err)
	}
	artwork := core.NewRecord(artworks)
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
		&core.SelectField{Name: "status", Values: []string{"queued", "sent", "received"}, MaxSelect: 1, Required: true},
		&core.DateField{Name: "sent_at"},
		&core.TextField{Name: "correlation_id"},
		&core.DateField{Name: "received_at"},
	)
	if err := app.Save(postcards); err != nil {
		t.Fatalf("create postcards collection: %v", err)
	}

	deliveries := core.NewBaseCollection("postcardDeliveries")
	deliveries.Id = collectionDeliveries
	deliveries.MarkAsNew()
	deliveries.Fields.Add(
		&core.RelationField{Name: "postcard", CollectionId: postcards.Id, Required: true},
		&core.TextField{Name: "recipient", Required: true},
		&core.SelectField{Name: "status", Values: []string{"pending", "sent", "cancelled"}, MaxSelect: 1, Required: true},
		&core.DateField{Name: "sent_at"},
		&core.DateField{Name: "cancelled_at"},
	)
	if err := app.Save(deliveries); err != nil {
		t.Fatalf("create deliveries collection: %v", err)
	}

	attempts := core.NewBaseCollection("postcardDeliveryAttempts")
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
	)
	if err := app.Save(attempts); err != nil {
		t.Fatalf("create attempts collection: %v", err)
	}

	return artwork.Id
}

func postcardTestConfig(t *testing.T) config.Postcards {
	t.Helper()
	values := map[string]string{
		"WGA_ENV":            "test",
		"WGA_PROTOCOL":       "http",
		"WGA_HOSTNAME":       "example.test",
		"WGA_SENDER_NAME":    "WGA",
		"WGA_SENDER_ADDRESS": "sender@example.test",
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
