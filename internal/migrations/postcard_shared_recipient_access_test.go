package migrations

import "testing"

func TestPostcardSharedRecipientAccessMigrationCreatesAdditiveSchema(t *testing.T) {
	app := newMigrationTestApp(t, t.TempDir())
	t.Cleanup(func() { _ = app.ResetBootstrapState() })
	configureMigrations(t)
	if err := createCurrentSchema(app); err != nil {
		t.Fatal(err)
	}
	if err := addPostcardPublicSharing(app); err != nil {
		t.Fatal(err)
	}
	if err := addPostcardSharedRecipientAccess(app); err != nil {
		t.Fatal(err)
	}

	postcards, err := app.FindCollectionByNameOrId("postcards")
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"view_token_hash", "view_expires_at"} {
		if postcards.Fields.GetByName(field) == nil {
			t.Fatalf("missing postcards.%s", field)
		}
	}
	if !hasIndex(postcards.Indexes, "pbx_postcard_shared_view_token") || !hasIndex(postcards.Indexes, "pbx_postcard_shared_view_expiry") {
		t.Fatal("missing shared postcard access indexes")
	}
	deliveries, err := app.FindCollectionByNameOrId("tracking_postcard_deliveries")
	if err != nil {
		t.Fatal(err)
	}
	if deliveries.Fields.GetByName("recipient_mask") == nil {
		t.Fatal("missing postcard_deliveries.recipient_mask")
	}
}
