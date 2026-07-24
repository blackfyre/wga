package migrations

import (
	"strings"
	"testing"
)

func TestPostcardDeliveryLifecycleMigrationCreatesAdditiveSchema(t *testing.T) {
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
		t.Fatalf("find postcards collection: %v", err)
	}
	for _, field := range []string{"correlation_id", "received_at"} {
		if postcards.Fields.GetByName(field) == nil {
			t.Fatalf("missing postcard field %q", field)
		}
	}

	deliveries, err := app.FindCollectionByNameOrId("postcard_deliveries")
	if err != nil {
		t.Fatalf("find deliveries collection: %v", err)
	}
	if deliveries.Fields.GetByName("postcard") == nil || deliveries.Fields.GetByName("recipient") == nil {
		t.Fatal("expected postcard delivery relation and recipient fields")
	}
	if !hasIndex(deliveries.Indexes, "pbx_postcard_delivery_recipient") {
		t.Fatal("expected unique postcard recipient index")
	}

	attempts, err := app.FindCollectionByNameOrId("postcard_delivery_attempts")
	if err != nil {
		t.Fatalf("find attempts collection: %v", err)
	}
	for _, field := range []string{"delivery", "status", "claim_token", "claim_expires_at", "transport_started_at", "resolution_code"} {
		if attempts.Fields.GetByName(field) == nil {
			t.Fatalf("missing attempt field %q", field)
		}
	}
}

func hasIndex(indexes []string, name string) bool {
	for _, index := range indexes {
		if strings.Contains(index, name) {
			return true
		}
	}
	return false
}
