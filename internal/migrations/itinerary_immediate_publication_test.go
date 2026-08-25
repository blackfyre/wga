package migrations

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestItineraryImmediatePublicationBackfill(t *testing.T) {
	app := newMigrationTestApp(t, t.TempDir())
	defer func() {
		if err := app.ResetBootstrapState(); err != nil {
			t.Error(err)
		}
	}()
	installMigrationArtworks(t, app)
	if err := addItineraries(app); err != nil {
		t.Fatalf("addItineraries: %v", err)
	}

	itineraries, err := app.FindCollectionByNameOrId("itineraries")
	if err != nil {
		t.Fatalf("find itineraries: %v", err)
	}

	newRecord := func(status string) *core.Record {
		record := core.NewRecord(itineraries)
		record.Set("owner", "owner-"+status)
		record.Set("status", status)
		record.Set("token", "token-"+status)
		record.Set("title", "Title "+status)
		if err := app.Save(record); err != nil {
			t.Fatalf("save %s record: %v", status, err)
		}
		return record
	}

	pending := newRecord("pending")
	rejected := newRecord("rejected")
	approved := newRecord("approved")
	draft := newRecord("draft")

	// First run reconciles legacy pending records to approved.
	if err := backfillItineraryImmediatePublication(app); err != nil {
		t.Fatalf("backfill: %v", err)
	}

	if got := statusOf(t, app, pending.Id); got != "approved" {
		t.Errorf("pending record status = %q, want approved", got)
	}
	if got := statusOf(t, app, rejected.Id); got != "rejected" {
		t.Errorf("rejected record status = %q, want rejected (preserved)", got)
	}
	if got := statusOf(t, app, approved.Id); got != "approved" {
		t.Errorf("approved record status = %q, want approved (untouched)", got)
	}
	if got := statusOf(t, app, draft.Id); got != "draft" {
		t.Errorf("draft record status = %q, want draft (untouched)", got)
	}

	// Re-run is a no-op (idempotent): the reconciled record is not double-walked.
	if err := backfillItineraryImmediatePublication(app); err != nil {
		t.Fatalf("backfill (again): %v", err)
	}
	if got := statusOf(t, app, pending.Id); got != "approved" {
		t.Errorf("re-run must not change the reconciled record, got %q", got)
	}

	// The down migration is a deliberate no-op that preserves the authoritative
	// outcome, matching the repository's backfill rollback pattern.
	if err := removeItineraryImmediatePublication(app); err != nil {
		t.Fatalf("remove backfill: %v", err)
	}
	if got := statusOf(t, app, pending.Id); got != "approved" {
		t.Errorf("rollback must preserve the authoritative outcome, got %q", got)
	}
	if got := statusOf(t, app, rejected.Id); got != "rejected" {
		t.Errorf("rollback must keep rejected records denied, got %q", got)
	}
}

func statusOf(t *testing.T, app *core.BaseApp, id string) string {
	t.Helper()
	record, err := app.FindRecordById("itineraries", id)
	if err != nil {
		t.Fatalf("find record %s: %v", id, err)
	}
	return record.GetString("status")
}
