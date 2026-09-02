package postcards

import (
	"bytes"
	"errors"
	"testing"

	"github.com/blackfyre/wga/internal/testutils"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

func TestRewrapTokenKeyBoundsSelectionAndPreservesDeliveryState(t *testing.T) {
	app := testutils.NewTestApp(t)
	artworkID := installPostcardSchema(t, app)
	oldKey := bytes.Repeat([]byte{0x11}, 32)
	newKey := bytes.Repeat([]byte{0x22}, 32)
	oldOnly := testPostcardKeyring(t, "old", map[string][]byte{"old": oldKey})
	rotation := testPostcardKeyring(t, "new", map[string][]byte{"old": oldKey, "new": newKey})
	newOnly := testPostcardKeyring(t, "new", map[string][]byte{"new": newKey})

	if _, err := QueueWithAccess(app, oldOnly, QueueInput{
		SenderName: "Sender", SenderEmail: "sender@example.test",
		Recipients: []string{"first@example.test", "second@example.test", "third@example.test"},
		Message:    "Hello", ImageID: artworkID,
	}, types.NowDateTime()); err != nil {
		t.Fatal(err)
	}
	deliveries := sortedDeliveries(t, app)
	before := make(map[string]map[string]string, len(deliveries))
	for _, delivery := range deliveries {
		before[delivery.Id] = deliveryState(delivery)
	}
	messageIDs := deliveryMessageIDs(t, app)

	count, err := RewrapTokenKey(app, rotation, "old", 2)
	if err != nil {
		t.Fatalf("rewrap: %v", err)
	}
	if count != 2 {
		t.Fatalf("rewrapped = %d, want 2", count)
	}

	deliveries = sortedDeliveries(t, app)
	for index, delivery := range deliveries {
		for field, value := range before[delivery.Id] {
			if got := delivery.GetString(field); got != value {
				t.Fatalf("delivery %d field %s changed from %q to %q", index, field, value, got)
			}
		}
		keyID, _, ok := recipientTokenEnvelopeParts(delivery.GetString("view_token_envelope"))
		if !ok {
			t.Fatalf("delivery %d has malformed envelope", index)
		}
		if index < 2 {
			if keyID != "new" {
				t.Fatalf("delivery %d key = %q, want new", index, keyID)
			}
			if _, err := recoverRecipientToken(newOnly, delivery.Id, delivery.GetString("view_token_envelope"), delivery.GetString("view_token_hash")); err != nil {
				t.Fatalf("new key could not open delivery %d: %v", index, err)
			}
			if _, err := openRecipientToken(oldOnly, delivery.Id, delivery.GetString("view_token_envelope")); err == nil {
				t.Fatalf("old key opened rewrapped delivery %d", index)
			}
		} else {
			if keyID != "old" {
				t.Fatalf("bounded delivery key = %q, want old", keyID)
			}
			if _, err := recoverRecipientToken(oldOnly, delivery.Id, delivery.GetString("view_token_envelope"), delivery.GetString("view_token_hash")); err != nil {
				t.Fatalf("old key could not open unselected delivery: %v", err)
			}
		}
	}

	for deliveryID, messageID := range messageIDs {
		if got := deliveryMessageIDs(t, app)[deliveryID]; got != messageID {
			t.Fatalf("delivery message ID changed")
		}
	}
}

func TestRewrapTokenKeyRollsBackWholeBatchOnCorruption(t *testing.T) {
	app := testutils.NewTestApp(t)
	artworkID := installPostcardSchema(t, app)
	oldKey := bytes.Repeat([]byte{0x31}, 32)
	newKey := bytes.Repeat([]byte{0x32}, 32)
	oldOnly := testPostcardKeyring(t, "old", map[string][]byte{"old": oldKey})
	rotation := testPostcardKeyring(t, "new", map[string][]byte{"old": oldKey, "new": newKey})

	if _, err := QueueWithAccess(app, oldOnly, QueueInput{
		SenderName: "Sender", SenderEmail: "sender@example.test",
		Recipients: []string{"first@example.test", "second@example.test"},
		Message:    "Hello", ImageID: artworkID,
	}, types.NowDateTime()); err != nil {
		t.Fatal(err)
	}
	deliveries := sortedDeliveries(t, app)
	firstEnvelope := deliveries[0].GetString("view_token_envelope")
	corruptEnvelope := tamperEnvelope(deliveries[1].GetString("view_token_envelope"))
	deliveries[1].Set("view_token_envelope", corruptEnvelope)
	if err := app.Save(deliveries[1]); err != nil {
		t.Fatal(err)
	}

	count, err := RewrapTokenKey(app, rotation, "old", 2)
	if !errors.Is(err, errTokenRewrapFailed) || count != 0 {
		t.Fatalf("rewrap count=%d error=%v", count, err)
	}
	deliveries = sortedDeliveries(t, app)
	if deliveries[0].GetString("view_token_envelope") != firstEnvelope {
		t.Fatal("first row remained committed after later corruption")
	}
	if deliveries[1].GetString("view_token_envelope") != corruptEnvelope {
		t.Fatal("corrupt row changed despite transaction rollback")
	}
}

func TestRewrapTokenKeyRewrapsLiveSenderControls(t *testing.T) {
	app := testutils.NewTestApp(t)
	artworkID := installPostcardSchema(t, app)
	oldKey := bytes.Repeat([]byte{0x51}, 32)
	newKey := bytes.Repeat([]byte{0x52}, 32)
	oldOnly := testPostcardKeyring(t, "old", map[string][]byte{"old": oldKey})
	rotation := testPostcardKeyring(t, "new", map[string][]byte{"old": oldKey, "new": newKey})
	newOnly := testPostcardKeyring(t, "new", map[string][]byte{"new": newKey})
	queued, err := QueueWithAccess(app, oldOnly, QueueInput{SenderName: "Sender", SenderEmail: "sender@example.test", Recipients: []string{"recipient@example.test"}, Message: "Hello", ImageID: artworkID}, types.NowDateTime())
	if err != nil {
		t.Fatal(err)
	}
	access, err := IssueSenderControl(app, oldOnly, queued.Postcard.Id, types.NowDateTime())
	if err != nil {
		t.Fatal(err)
	}
	count, err := RewrapTokenKey(app, rotation, "old", 2)
	if err != nil || count != 2 {
		t.Fatalf("rewrap sender control count=%d error=%v", count, err)
	}
	control, err := app.FindRecordById(collectionSenderControls, access.ControlID)
	if err != nil {
		t.Fatal(err)
	}
	if token, err := recoverSenderControlToken(newOnly, control.Id, control.GetString("token_envelope"), control.GetString("token_hash")); err != nil || token != access.Token {
		t.Fatalf("new key did not recover sender control: %v", err)
	}
}

func TestRewrapTokenKeyRejectsUnsafeInputs(t *testing.T) {
	app := testutils.NewTestApp(t)
	oldKey := bytes.Repeat([]byte{0x41}, 32)
	newKey := bytes.Repeat([]byte{0x42}, 32)
	oldOnly := testPostcardKeyring(t, "old", map[string][]byte{"old": oldKey})
	rotation := testPostcardKeyring(t, "new", map[string][]byte{"old": oldKey, "new": newKey})

	for name, run := range map[string]func() error{
		"empty source": func() error {
			_, err := RewrapTokenKey(app, rotation, "", 1)
			return err
		},
		"invalid source": func() error {
			_, err := RewrapTokenKey(app, rotation, "bad.id", 1)
			return err
		},
		"missing source": func() error {
			_, err := RewrapTokenKey(app, rotation, "missing", 1)
			return err
		},
		"active source": func() error {
			_, err := RewrapTokenKey(app, oldOnly, "old", 1)
			return err
		},
		"zero limit": func() error {
			_, err := RewrapTokenKey(app, rotation, "old", 0)
			return err
		},
		"excessive limit": func() error {
			_, err := RewrapTokenKey(app, rotation, "old", maxTokenRewrapLimit+1)
			return err
		},
	} {
		t.Run(name, func(t *testing.T) {
			if err := run(); err == nil {
				t.Fatal("expected unsafe input to be rejected")
			}
		})
	}
}

func sortedDeliveries(t *testing.T, app core.App) []*core.Record {
	t.Helper()
	records, err := app.FindRecordsByFilter(collectionDeliveries, "", "+id", 0, 0)
	if err != nil {
		t.Fatalf("find deliveries: %v", err)
	}

	return records
}

func deliveryState(delivery *core.Record) map[string]string {
	state := make(map[string]string)
	for _, field := range []string{"postcard", "recipient", "status", "sent_at", "cancelled_at", "view_token_hash", "view_expires_at", "recipient_purged_at", "created", "updated"} {
		state[field] = delivery.GetString(field)
	}

	return state
}

func deliveryMessageIDs(t *testing.T, app core.App) map[string]string {
	t.Helper()
	attempts, err := app.FindRecordsByFilter(collectionDeliveryAttempts, "", "+id", 0, 0)
	if err != nil {
		t.Fatalf("find attempts: %v", err)
	}
	result := make(map[string]string, len(attempts))
	for _, attempt := range attempts {
		result[attempt.GetString("delivery")] = attempt.GetString("message_id")
	}

	return result
}
