package migrations

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestPostcardPublicSharingMigrationAndRollback(t *testing.T) {
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
	for _, name := range []string{"include_music", "retention_until", "sender_email_purged_at", "content_purged_at"} {
		if postcards.Fields.GetByName(name) == nil {
			t.Fatalf("missing postcards.%s", name)
		}
	}
	deliveries, err := app.FindCollectionByNameOrId("tracking_postcard_deliveries")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"view_token_envelope", "view_token_hash", "view_expires_at", "recipient_purged_at"} {
		if deliveries.Fields.GetByName(name) == nil {
			t.Fatalf("missing deliveries.%s", name)
		}
	}
	if deliveries.Fields.GetByName("view_token") != nil {
		t.Fatal("deliveries retains plaintext view_token field")
	}
	envelopeField, ok := deliveries.Fields.GetByName("view_token_envelope").(*core.TextField)
	if !ok || envelopeField.Max <= 0 {
		t.Fatal("deliveries.view_token_envelope must have a bounded text field")
	}
	for _, name := range []string{"pbx_postcard_delivery_view_token", "pbx_postcard_delivery_view_expiry"} {
		if !hasIndex(deliveries.Indexes, name) {
			t.Fatalf("missing index %s", name)
		}
	}
	attempts, err := app.FindCollectionByNameOrId("tracking_postcard_delivery_attempts")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"direction", "integration_key", "message_type", "deduplication_key", "causation_message_id", "payload_reference", "payload_format", "payload_retention_until", "payload_purged_at", "transport_metadata", "last_error_code", "result_summary"} {
		if attempts.Fields.GetByName(name) == nil {
			t.Fatalf("missing attempts.%s", name)
		}
	}
	for _, name := range []string{"pbx_postcard_attempt_dedup", "pbx_postcard_attempt_correlation", "pbx_postcard_attempt_causation", "pbx_postcard_attempt_unresolved"} {
		if !hasIndex(attempts.Indexes, name) {
			t.Fatalf("missing index %s", name)
		}
	}
	for collectionName, collection := range map[string]struct {
		list   *string
		view   *string
		create *string
		update *string
		delete *string
	}{
		"deliveries": {deliveries.ListRule, deliveries.ViewRule, deliveries.CreateRule, deliveries.UpdateRule, deliveries.DeleteRule},
		"attempts":   {attempts.ListRule, attempts.ViewRule, attempts.CreateRule, attempts.UpdateRule, attempts.DeleteRule},
	} {
		if collection.list != nil || collection.view != nil || collection.create != nil || collection.update != nil || collection.delete != nil {
			t.Fatalf("%s API rules must remain closed", collectionName)
		}
	}

	if err := removePostcardPublicSharing(app); err != nil {
		t.Fatalf("rollback: %v", err)
	}
	postcards, _ = app.FindCollectionByNameOrId("postcards")
	deliveries, _ = app.FindCollectionByNameOrId("tracking_postcard_deliveries")
	attempts, _ = app.FindCollectionByNameOrId("tracking_postcard_delivery_attempts")
	if postcards.Fields.GetByName("include_music") == nil || deliveries.Fields.GetByName("view_token_hash") == nil || attempts.Fields.GetByName("integration_key") == nil {
		t.Fatal("rollback removed authoritative postcard sharing fields")
	}
	if hasIndex(deliveries.Indexes, "pbx_postcard_delivery_view_token") || hasIndex(attempts.Indexes, "pbx_postcard_attempt_dedup") {
		t.Fatal("rollback retained postcard sharing indexes")
	}
}
