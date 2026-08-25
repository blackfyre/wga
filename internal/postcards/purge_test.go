package postcards

import (
	"testing"
	"time"

	"github.com/blackfyre/wga/internal/testutils"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

// queueTerminalPostcard queues an already-expired postcard and completes every
// delivery so both its delivery access and its postcard content are eligible
// for purge.
func queueTerminalPostcard(t *testing.T, app core.App, artworkID string, recipients ...string) *QueueResult {
	t.Helper()
	past := types.NowDateTime().Add(-31 * 24 * time.Hour)
	result, err := QueueWithAccess(app, postcardTestKeyring(t), QueueInput{
		SenderName:  "sender",
		SenderEmail: "sender@example.test",
		Recipients:  recipients,
		Message:     "hello",
		ImageID:     artworkID,
	}, past)
	if err != nil {
		t.Fatalf("queue postcard: %v", err)
	}
	for range recipients {
		claim, err := claimDue(app, types.NowDateTime())
		if err != nil {
			t.Fatalf("claim attempt: %v", err)
		}
		if claim == nil {
			t.Fatalf("expected claimable attempt for %d recipients", len(recipients))
		}
		if err := startTransport(app, claim, types.NowDateTime()); err != nil {
			t.Fatalf("start transport: %v", err)
		}
		if err := complete(app, claim, types.NowDateTime()); err != nil {
			t.Fatalf("complete attempt: %v", err)
		}
	}

	return result
}

// TestPurgeExpiredRecipientAccessBoundsBacklogAndDrainsToZero verifies a
// backlog larger than the limit is drained in bounded, deterministic batches
// until repeated calls terminate at zero.
func TestPurgeExpiredRecipientAccessBoundsBacklogAndDrainsToZero(t *testing.T) {
	app := testutils.NewTestApp(t)
	artworkID := installPostcardSchema(t, app)
	for range 5 {
		queueTerminalPostcard(t, app, artworkID, "recipient@example.test")
	}

	steps := []struct {
		access  int
		content int
	}{{2, 2}, {2, 2}, {1, 1}, {0, 0}}
	for _, step := range steps {
		counts, err := PurgeExpiredRecipientAccess(app, types.NowDateTime(), 2)
		if err != nil {
			t.Fatalf("purge: %v", err)
		}
		if counts.DeliveryAccess != step.access || counts.PostcardContent != step.content {
			t.Fatalf("purge = %+v, want delivery=%d content=%d", counts, step.access, step.content)
		}
	}
}

// TestPurgeExpiredRecipientAccessExactLimit verifies a backlog exactly matching
// the limit is fully purged in one call and a repeat call is a no-op.
func TestPurgeExpiredRecipientAccessExactLimit(t *testing.T) {
	app := testutils.NewTestApp(t)
	artworkID := installPostcardSchema(t, app)
	for range 3 {
		queueTerminalPostcard(t, app, artworkID, "recipient@example.test")
	}

	counts, err := PurgeExpiredRecipientAccess(app, types.NowDateTime(), 3)
	if err != nil {
		t.Fatalf("purge: %v", err)
	}
	if counts.DeliveryAccess != 3 || counts.PostcardContent != 3 {
		t.Fatalf("purge = %+v, want delivery=3 content=3", counts)
	}
	counts, err = PurgeExpiredRecipientAccess(app, types.NowDateTime(), 3)
	if err != nil {
		t.Fatalf("second purge: %v", err)
	}
	if counts.DeliveryAccess != 0 || counts.PostcardContent != 0 {
		t.Fatalf("second purge = %+v, want 0/0", counts)
	}
}

// TestPurgeExpiredRecipientAccessReportsPerKindCounts verifies delivery-access
// and postcard-content counts are reported independently and drained at their
// own rates.
func TestPurgeExpiredRecipientAccessReportsPerKindCounts(t *testing.T) {
	app := testutils.NewTestApp(t)
	artworkID := installPostcardSchema(t, app)
	result := queueTerminalPostcard(t, app, artworkID, "first@example.test", "second@example.test", "third@example.test")
	if len(result.Access) != 3 {
		t.Fatalf("access = %d, want 3", len(result.Access))
	}

	steps := []struct {
		access  int
		content int
	}{{1, 1}, {1, 0}, {1, 0}, {0, 0}}
	for _, step := range steps {
		counts, err := PurgeExpiredRecipientAccess(app, types.NowDateTime(), 1)
		if err != nil {
			t.Fatalf("purge: %v", err)
		}
		if counts.DeliveryAccess != step.access || counts.PostcardContent != step.content {
			t.Fatalf("purge = %+v, want delivery=%d content=%d", counts, step.access, step.content)
		}
	}

	postcard, err := app.FindRecordById(collectionPostcards, result.Postcard.Id)
	if err != nil {
		t.Fatal(err)
	}
	if postcard.GetString("sender_name") != "Anonymous" || postcard.GetString("content_purged_at") == "" {
		t.Fatalf("postcard content not anonymised: sender_name=%q content_purged_at=%q", postcard.GetString("sender_name"), postcard.GetString("content_purged_at"))
	}
}

// TestPurgeExpiredRecipientAccessExcludesProcessingAttempts verifies a claimed
// but in-flight delivery is retained.
func TestPurgeExpiredRecipientAccessExcludesProcessingAttempts(t *testing.T) {
	app := testutils.NewTestApp(t)
	artworkID := installPostcardSchema(t, app)
	past := types.NowDateTime().Add(-31 * 24 * time.Hour)
	result, err := QueueWithAccess(app, postcardTestKeyring(t), QueueInput{
		SenderName: "sender", SenderEmail: "sender@example.test", Recipients: []string{"recipient@example.test"}, Message: "hello", ImageID: artworkID,
	}, past)
	if err != nil {
		t.Fatal(err)
	}
	claim, err := claimDue(app, types.NowDateTime())
	if err != nil {
		t.Fatal(err)
	}
	if claim == nil {
		t.Fatal("expected claim")
	}

	counts, err := PurgeExpiredRecipientAccess(app, types.NowDateTime(), maxPurgeBatchSize)
	if err != nil {
		t.Fatal(err)
	}
	if counts.DeliveryAccess != 0 || counts.PostcardContent != 0 {
		t.Fatalf("purge = %+v, want 0/0 for processing attempt", counts)
	}
	delivery, err := app.FindRecordById(collectionDeliveries, result.Access[0].DeliveryID)
	if err != nil {
		t.Fatal(err)
	}
	if delivery.GetString("view_token_hash") == "" || delivery.GetString("view_token_envelope") == "" {
		t.Fatal("processing delivery access was purged")
	}
	postcard, err := app.FindRecordById(collectionPostcards, result.Postcard.Id)
	if err != nil {
		t.Fatal(err)
	}
	if postcard.GetString("content_purged_at") != "" {
		t.Fatal("processing postcard content was purged")
	}
}

// TestPurgeExpiredRecipientAccessRejectsInvalidLimit verifies non-positive and
// unreasonably large limits are rejected before any work is performed.
func TestPurgeExpiredRecipientAccessRejectsInvalidLimit(t *testing.T) {
	app := testutils.NewTestApp(t)
	installPostcardSchema(t, app)

	for _, limit := range []int{0, -1, maxPurgeBatchSize + 1} {
		if _, err := PurgeExpiredRecipientAccess(app, types.NowDateTime(), limit); err == nil {
			t.Fatalf("limit %d accepted", limit)
		}
	}
}

// TestPurgeExpiredRecipientAccessRollsBackOnContentStageFailure verifies the
// delivery-access purge is rolled back when the postcard-content stage fails.
func TestPurgeExpiredRecipientAccessRollsBackOnContentStageFailure(t *testing.T) {
	app := testutils.NewTestApp(t)
	artworkID := installPostcardSchema(t, app)
	result := queueTerminalPostcard(t, app, artworkID, "recipient@example.test")
	deliveryID := result.Access[0].DeliveryID

	delivery, err := app.FindRecordById(collectionDeliveries, deliveryID)
	if err != nil {
		t.Fatal(err)
	}
	if delivery.GetString("view_token_hash") == "" {
		t.Fatal("expected pre-purge token hash")
	}

	if _, err := app.DB().NewQuery(`CREATE TRIGGER fail_postcard_content_purge BEFORE UPDATE ON Postcards BEGIN SELECT RAISE(ABORT, 'injected purge failure'); END;`).Execute(); err != nil {
		t.Fatalf("create trigger: %v", err)
	}
	defer func() {
		if _, err := app.DB().NewQuery(`DROP TRIGGER IF EXISTS fail_postcard_content_purge`).Execute(); err != nil {
			t.Errorf("drop trigger: %v", err)
		}
	}()

	if _, err := PurgeExpiredRecipientAccess(app, types.NowDateTime(), maxPurgeBatchSize); err == nil {
		t.Fatal("expected purge to fail")
	}

	delivery, err = app.FindRecordById(collectionDeliveries, deliveryID)
	if err != nil {
		t.Fatal(err)
	}
	if delivery.GetString("view_token_hash") == "" {
		t.Fatal("delivery access purged despite content-stage rollback")
	}
}
