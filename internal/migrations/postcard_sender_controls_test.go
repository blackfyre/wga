package migrations

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestPostcardSenderControlsMigrationCreatesAdditiveSchema(t *testing.T) {
	configureMigrations(t)
	app := newMigrationTestApp(t, t.TempDir())
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})
	if err := app.RunAllMigrations(); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	postcards, err := app.FindCollectionByNameOrId("postcards")
	if err != nil {
		t.Fatal(err)
	}
	if postcards.Fields.GetByName("submission_key_hash") == nil || postcards.Fields.GetByName("failed_at") == nil {
		t.Fatal("missing postcard submission or failure fields")
	}
	if !hasIndex(postcards.Indexes, "pbx_postcard_submission_key") {
		t.Fatal("missing postcard submission key index")
	}
	status, ok := postcards.Fields.GetByName("status").(*core.SelectField)
	if !ok || !contains(status.Values, "failed") {
		t.Fatal("postcards.status must support failed")
	}

	deliveries, err := app.FindCollectionByNameOrId("tracking_postcard_deliveries")
	if err != nil {
		t.Fatal(err)
	}
	if deliveries.Fields.GetByName("failed_at") == nil {
		t.Fatal("missing delivery failed_at field")
	}
	deliveryStatus, ok := deliveries.Fields.GetByName("status").(*core.SelectField)
	if !ok || !contains(deliveryStatus.Values, "failed") {
		t.Fatal("deliveries.status must support failed")
	}

	controls, err := app.FindCollectionByNameOrId("tracking_postcard_sender_controls")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"postcard", "token_hash", "token_envelope", "expires_at", "revoked_at", "purged_at"} {
		if controls.Fields.GetByName(name) == nil {
			t.Fatalf("missing sender control field %q", name)
		}
	}
	if controls.ListRule != nil || controls.ViewRule != nil || controls.CreateRule != nil || controls.UpdateRule != nil || controls.DeleteRule != nil {
		t.Fatal("sender controls API rules must remain closed")
	}
	for _, name := range []string{"pbx_postcard_sender_control_postcard", "pbx_postcard_sender_control_token", "pbx_postcard_sender_control_expiry"} {
		if !hasIndex(controls.Indexes, name) {
			t.Fatalf("missing sender controls index %q", name)
		}
	}
}

func TestPostcardSenderControlsMigrationPreservesExistingPostcards(t *testing.T) {
	app := newMigrationTestApp(t, t.TempDir())
	t.Cleanup(func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	})
	if err := createCurrentSchema(app); err != nil {
		t.Fatalf("create pre-upgrade schema: %v", err)
	}
	if err := addPostcardPublicSharing(app); err != nil {
		t.Fatalf("apply pre-upgrade postcard schema: %v", err)
	}

	postcards, err := app.FindCollectionByNameOrId("postcards")
	if err != nil {
		t.Fatal(err)
	}
	artworks, err := app.FindCollectionByNameOrId("artworks")
	if err != nil {
		t.Fatal(err)
	}
	artwork := core.NewRecord(artworks)
	artwork.Set("title", "Existing artwork")
	if err := app.Save(artwork); err != nil {
		t.Fatalf("save existing artwork: %v", err)
	}
	record := core.NewRecord(postcards)
	record.Set("sender_name", "Existing sender")
	record.Set("sender_email", "sender@example.test")
	record.Set("recipients", "recipient@example.test")
	record.Set("message", "Existing postcard")
	record.Set("image_id", artwork.Id)
	record.Set("status", "queued")
	if err := app.Save(record); err != nil {
		t.Fatalf("save existing postcard: %v", err)
	}
	deliveries, err := app.FindCollectionByNameOrId("tracking_postcard_deliveries")
	if err != nil {
		t.Fatal(err)
	}
	delivery := core.NewRecord(deliveries)
	delivery.Set("postcard", record.Id)
	delivery.Set("recipient", "recipient@example.test")
	delivery.Set("status", "pending")
	delivery.Set("view_token_hash", "existing-token-hash")
	delivery.Set("view_token_envelope", "existing-token-envelope")
	if err := app.Save(delivery); err != nil {
		t.Fatalf("save existing delivery: %v", err)
	}

	if err := addPostcardSenderControls(app); err != nil {
		t.Fatalf("reapply additive migration: %v", err)
	}
	preserved, err := app.FindRecordById("postcards", record.Id)
	if err != nil {
		t.Fatalf("find existing postcard: %v", err)
	}
	if preserved.GetString("sender_name") != "Existing sender" || preserved.GetString("recipients") != "recipient@example.test" || preserved.GetString("submission_key_hash") != "" {
		t.Fatal("additive migration changed existing postcard data")
	}
	preservedDelivery, err := app.FindRecordById("tracking_postcard_deliveries", delivery.Id)
	if err != nil {
		t.Fatalf("find existing delivery: %v", err)
	}
	if preservedDelivery.GetString("view_token_hash") != "existing-token-hash" || preservedDelivery.GetString("view_token_envelope") != "existing-token-envelope" {
		t.Fatal("additive migration changed existing recipient access data")
	}
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
