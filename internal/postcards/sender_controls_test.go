package postcards

import (
	"strings"
	"testing"
	"time"

	"github.com/blackfyre/wga/internal/testutils"
	"github.com/pocketbase/pocketbase/tools/types"
)

func TestIssueAndRecoverSenderControl(t *testing.T) {
	app := testutils.NewTestApp(t)
	artworkID := installPostcardSchema(t, app)
	now := types.NowDateTime()
	queued, err := QueueWithAccess(app, postcardTestKeyring(t), QueueInput{
		SenderName: "sender", SenderEmail: "sender@example.test", Recipients: []string{"recipient@example.test"}, Message: "hello", ImageID: artworkID,
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	access, err := IssueSenderControl(app, postcardTestKeyring(t), queued.Postcard.Id, now)
	if err != nil {
		t.Fatal(err)
	}
	control, err := app.FindRecordById(collectionSenderControls, access.ControlID)
	if err != nil {
		t.Fatal(err)
	}
	if control.GetString("token_hash") != HashRecipientToken(access.Token) || control.GetString("token_envelope") == "" || strings.Contains(control.GetString("token_envelope"), access.Token) {
		t.Fatal("sender control does not protect its token")
	}
	recovered, err := RecoverSenderControl(app, postcardTestKeyring(t), queued.Postcard.Id, now)
	if err != nil {
		t.Fatal(err)
	}
	if recovered.Token != access.Token || recovered.ControlID != access.ControlID {
		t.Fatal("sender control recovery returned different access")
	}
	found, err := FindSenderControl(app, access.Token, now)
	if err != nil || found.ID != access.ControlID || found.Postcard.Id != queued.Postcard.Id {
		t.Fatalf("find sender control: %#v, %v", found, err)
	}
}

func TestRecoverSenderControlRejectsExpiredControl(t *testing.T) {
	app := testutils.NewTestApp(t)
	artworkID := installPostcardSchema(t, app)
	past := types.NowDateTime().Add(-31 * 24 * time.Hour)
	submissionKey, err := NewSubmissionKey()
	if err != nil {
		t.Fatal(err)
	}
	queued, err := QueueWithAccess(app, postcardTestKeyring(t), QueueInput{
		SenderName: "sender", SenderEmail: "sender@example.test", Recipients: []string{"recipient@example.test"}, Message: "hello", ImageID: artworkID, SubmissionKey: submissionKey,
	}, past)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := IssueSenderControl(app, postcardTestKeyring(t), queued.Postcard.Id, past); err != nil {
		t.Fatal(err)
	}
	if _, err := RecoverSenderControl(app, postcardTestKeyring(t), queued.Postcard.Id, types.NowDateTime()); err != ErrSenderControlUnavailable {
		t.Fatalf("recover expired control = %v, want unavailable", err)
	}
}

func TestPurgeExpiredSenderControlKeepsUnresolvedDeliveryWork(t *testing.T) {
	app := testutils.NewTestApp(t)
	artworkID := installPostcardSchema(t, app)
	past := types.NowDateTime().Add(-31 * 24 * time.Hour)
	queued, err := QueueWithAccess(app, postcardTestKeyring(t), QueueInput{
		SenderName: "sender", SenderEmail: "sender@example.test", Recipients: []string{"recipient@example.test"}, Message: "hello", ImageID: artworkID,
	}, past)
	if err != nil {
		t.Fatal(err)
	}
	access, err := IssueSenderControl(app, postcardTestKeyring(t), queued.Postcard.Id, past)
	if err != nil {
		t.Fatal(err)
	}
	counts, err := PurgeExpiredRecipientAccess(app, types.NowDateTime(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if counts.SenderControls != 1 {
		t.Fatalf("sender controls purged = %d, want 1", counts.SenderControls)
	}
	if _, err := app.FindRecordById(collectionSenderControls, access.ControlID); err == nil {
		t.Fatal("expired sender control was retained")
	}
	postcard, err := app.FindRecordById(collectionPostcards, queued.Postcard.Id)
	if err != nil || postcard.GetString("submission_key_hash") != "" {
		t.Fatal("expired sender submission key was retained")
	}
	delivery, err := app.FindRecordById(collectionDeliveries, queued.Access[0].DeliveryID)
	if err != nil || delivery.GetString("view_token_hash") == "" {
		t.Fatal("unresolved delivery access was purged")
	}
}

func TestIssueSenderControlDoesNotOutlivePostcardRetention(t *testing.T) {
	app := testutils.NewTestApp(t)
	artworkID := installPostcardSchema(t, app)
	now := types.NowDateTime()
	queued, err := QueueWithAccess(app, postcardTestKeyring(t), QueueInput{
		SenderName: "sender", SenderEmail: "sender@example.test", Recipients: []string{"recipient@example.test"}, Message: "hello", ImageID: artworkID,
	}, now.Add(-29*24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	postcard, err := app.FindRecordById(collectionPostcards, queued.Postcard.Id)
	if err != nil {
		t.Fatal(err)
	}
	access, err := IssueSenderControl(app, postcardTestKeyring(t), queued.Postcard.Id, now)
	if err != nil {
		t.Fatal(err)
	}
	if !access.ExpiresAt.Equal(postcard.GetDateTime("retention_until")) {
		t.Fatal("sender control expiry outlives postcard retention")
	}
}

func TestRecoverSenderControlReturnsUniformUnavailableError(t *testing.T) {
	app := testutils.NewTestApp(t)
	artworkID := installPostcardSchema(t, app)
	now := types.NowDateTime()
	queued, err := QueueWithAccess(app, postcardTestKeyring(t), QueueInput{
		SenderName: "sender", SenderEmail: "sender@example.test", Recipients: []string{"recipient@example.test"}, Message: "hello", ImageID: artworkID,
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	access, err := IssueSenderControl(app, postcardTestKeyring(t), queued.Postcard.Id, now)
	if err != nil {
		t.Fatal(err)
	}
	control, err := app.FindRecordById(collectionSenderControls, access.ControlID)
	if err != nil {
		t.Fatal(err)
	}
	control.Set("revoked_at", now)
	if err := app.Save(control); err != nil {
		t.Fatal(err)
	}
	if _, err := RecoverSenderControl(app, postcardTestKeyring(t), queued.Postcard.Id, now); err != ErrSenderControlUnavailable {
		t.Fatalf("recover revoked control = %v", err)
	}
	if _, err := RecoverSenderControl(app, postcardTestKeyring(t), "missing", now); err != ErrSenderControlUnavailable {
		t.Fatalf("recover missing control = %v", err)
	}
	second, err := QueueWithAccess(app, postcardTestKeyring(t), QueueInput{
		SenderName: "sender", SenderEmail: "second@example.test", Recipients: []string{"second-recipient@example.test"}, Message: "hello", ImageID: artworkID,
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	secondAccess, err := IssueSenderControl(app, postcardTestKeyring(t), second.Postcard.Id, now)
	if err != nil {
		t.Fatal(err)
	}
	secondControl, err := app.FindRecordById(collectionSenderControls, secondAccess.ControlID)
	if err != nil {
		t.Fatal(err)
	}
	secondControl.Set("token_envelope", "corrupt")
	if err := app.Save(secondControl); err != nil {
		t.Fatal(err)
	}
	if _, err := RecoverSenderControl(app, postcardTestKeyring(t), second.Postcard.Id, now); err != ErrSenderControlUnavailable {
		t.Fatalf("recover corrupt control = %v", err)
	}
}
