package migrations

import (
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

// TestPostcardDeliveryLifecycleMigrationCreatesAdditiveSchema verifies the delivery schema is additive.
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

// TestPostcardDeliveryLifecycleMigrationsRollBackAndReapply verifies rollback restores a reapplicable schema.
func TestPostcardDeliveryLifecycleMigrationsRollBackAndReapply(t *testing.T) {
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
	runner := core.NewMigrationsRunner(app, core.AppMigrations)
	if _, err := runner.Down(3); err != nil {
		t.Fatalf("roll back postcard delivery migrations: %v", err)
	}
	for _, collection := range []string{"postcard_deliveries", "postcard_delivery_attempts"} {
		if _, err := app.FindCollectionByNameOrId(collection); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("expected no %s collection after rollback, got %v", collection, err)
		}
	}
	postcards, err := app.FindCollectionByNameOrId("postcards")
	if err != nil {
		t.Fatalf("find postcards collection after rollback: %v", err)
	}
	for _, field := range []string{"correlation_id", "received_at"} {
		if postcards.Fields.GetByName(field) != nil {
			t.Fatalf("expected no postcard field %q after rollback", field)
		}
	}
	if _, err := runner.Up(); err != nil {
		t.Fatalf("reapply postcard delivery migrations: %v", err)
	}
	if _, err := app.FindCollectionByNameOrId("postcard_delivery_attempts"); err != nil {
		t.Fatalf("find attempts collection after reapply: %v", err)
	}
}

func TestContributorRefreshMigrationCreatesAdditiveSchema(t *testing.T) {
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

	snapshots, err := app.FindCollectionByNameOrId("contributor_snapshots")
	if err != nil {
		t.Fatalf("find contributor snapshots: %v", err)
	}
	for _, field := range []string{"key", "payload"} {
		if snapshots.Fields.GetByName(field) == nil {
			t.Fatalf("missing snapshot field %q", field)
		}
	}

	executions, err := app.FindCollectionByNameOrId("contributor_refresh_executions")
	if err != nil {
		t.Fatalf("find contributor refresh executions: %v", err)
	}
	for _, field := range []string{"run_id", "attempt", "status", "completed_at", "error_class"} {
		if executions.Fields.GetByName(field) == nil {
			t.Fatalf("missing execution field %q", field)
		}
	}
	if !hasIndex(executions.Indexes, "pbx_contributor_refresh_active") {
		t.Fatal("expected contributor refresh active index")
	}

	runner := core.NewMigrationsRunner(app, core.AppMigrations)
	if _, err := runner.Down(1); err != nil {
		t.Fatalf("roll back contributor migration: %v", err)
	}
	for _, collection := range []string{"contributor_snapshots", "contributor_refresh_executions"} {
		if _, err := app.FindCollectionByNameOrId(collection); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("expected no %s collection after rollback, got %v", collection, err)
		}
	}
	if _, err := runner.Up(); err != nil {
		t.Fatalf("reapply contributor migration: %v", err)
	}
}

// hasIndex reports whether a collection index definition contains name.
func hasIndex(indexes []string, name string) bool {
	for _, index := range indexes {
		if strings.Contains(index, name) {
			return true
		}
	}
	return false
}
