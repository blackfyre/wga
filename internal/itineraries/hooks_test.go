package itineraries

import (
	"errors"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestLifecycleHookAllowsDraftContentMutation(t *testing.T) {
	app := installItinerarySchema(t)
	RegisterHooks(app)

	draft, err := EnsureDraft(app, "owner-hook-draft")
	if err != nil {
		t.Fatalf("EnsureDraft: %v", err)
	}
	draft.Set("title", "New Title")
	if err := app.Save(draft); err != nil {
		t.Errorf("draft content mutation must be allowed: %v", err)
	}
}

func TestLifecycleHookAllowsPublicationAndModeration(t *testing.T) {
	app := installItinerarySchema(t)
	RegisterHooks(app)

	// Full visitor publication flow must pass with hooks registered; the record
	// is immediately approved so its token is readable at once.
	published := newPublishedRecord(t, app, "owner-hook-pub")
	if published.GetString("status") != string(StatusApproved) {
		t.Fatalf("published status = %q, want approved", published.GetString("status"))
	}

	// approved -> rejected moderation.
	rec := reloadItinerary(t, app, published.Id)
	rec.Set("status", string(StatusRejected))
	if err := app.Save(rec); err != nil {
		t.Errorf("approved -> rejected must be allowed: %v", err)
	}

	// Legacy pending ingress: pending -> approved (backfill) and pending ->
	// rejected (moderation) remain allowed for records that predate the change.
	pending := newPendingRecord(t, app, "owner-hook-legacy")
	rec = reloadItinerary(t, app, pending.Id)
	rec.Set("status", string(StatusApproved))
	if err := app.Save(rec); err != nil {
		t.Errorf("pending -> approved must be allowed: %v", err)
	}
	rec = reloadItinerary(t, app, pending.Id)
	rec.Set("status", string(StatusRejected))
	if err := app.Save(rec); err != nil {
		t.Errorf("pending -> rejected must be allowed: %v", err)
	}
}

func TestLifecycleHookRejectsInvalidTransitions(t *testing.T) {
	app := installItinerarySchema(t)
	RegisterHooks(app)

	// draft -> rejected (skipping publication entirely) is forbidden.
	addStops(t, app, "owner-hook-t1", 1)
	draft := reloadItinerary(t, app, mustDraftID(t, app, "owner-hook-t1"))
	draft.Set("status", string(StatusRejected))
	if err := app.Save(draft); !errors.Is(err, ErrInvalidTransition) {
		t.Errorf("draft -> rejected = %v, want ErrInvalidTransition", err)
	}

	// draft -> pending (legacy path) is forbidden for new code.
	draft = reloadItinerary(t, app, mustDraftID(t, app, "owner-hook-t1"))
	draft.Set("status", string(StatusPending))
	if err := app.Save(draft); !errors.Is(err, ErrInvalidTransition) {
		t.Errorf("draft -> pending = %v, want ErrInvalidTransition", err)
	}

	// approved -> draft (un-publishing) is forbidden.
	published := newPublishedRecord(t, app, "owner-hook-t2")
	rec := reloadItinerary(t, app, published.Id)
	rec.Set("status", string(StatusDraft))
	if err := app.Save(rec); !errors.Is(err, ErrInvalidTransition) {
		t.Errorf("approved -> draft = %v, want ErrInvalidTransition", err)
	}

	// approved -> pending (backwards to legacy) is forbidden.
	rec = reloadItinerary(t, app, published.Id)
	rec.Set("status", string(StatusPending))
	if err := app.Save(rec); !errors.Is(err, ErrInvalidTransition) {
		t.Errorf("approved -> pending = %v, want ErrInvalidTransition", err)
	}

	// rejected -> approved and rejected -> draft are forbidden (rejected is
	// terminal).
	rejectedRecord := newPublishedRecord(t, app, "owner-hook-t3")
	rec = reloadItinerary(t, app, rejectedRecord.Id)
	rec.Set("status", string(StatusRejected))
	if err := app.Save(rec); err != nil {
		t.Fatalf("approved -> rejected: %v", err)
	}
	rec = reloadItinerary(t, app, rejectedRecord.Id)
	rec.Set("status", string(StatusApproved))
	if err := app.Save(rec); !errors.Is(err, ErrInvalidTransition) {
		t.Errorf("rejected -> approved = %v, want ErrInvalidTransition", err)
	}
	rec = reloadItinerary(t, app, rejectedRecord.Id)
	rec.Set("status", string(StatusDraft))
	if err := app.Save(rec); !errors.Is(err, ErrInvalidTransition) {
		t.Errorf("rejected -> draft = %v, want ErrInvalidTransition", err)
	}
}

func TestLifecycleHookRejectsImmutablePublicationContent(t *testing.T) {
	app := installItinerarySchema(t)
	RegisterHooks(app)

	published := newPublishedRecord(t, app, "owner-hook-imm")
	rec := reloadItinerary(t, app, published.Id)
	rec.Set("status", string(StatusApproved))
	if err := app.Save(rec); err != nil {
		t.Fatalf("pending -> approved: %v", err)
	}

	fields := []struct {
		field string
		value any
	}{
		{"title", "Hijacked title"},
		{"intro", "Hijacked intro"},
		{"creator", "Hijacked creator"},
		{"owner", "hijacked-owner-digest"},
		{"token", "hijacked-token"},
		{"listed", true},
	}
	for _, tc := range fields {
		rec := reloadItinerary(t, app, published.Id)
		rec.Set(tc.field, tc.value)
		if err := app.Save(rec); !errors.Is(err, ErrImmutablePublication) {
			t.Errorf("changing %s = %v, want ErrImmutablePublication", tc.field, err)
		}
	}
}

func TestLifecycleHookAllowsNoopSaveOfPublishedRecord(t *testing.T) {
	app := installItinerarySchema(t)
	RegisterHooks(app)

	published := newPublishedRecord(t, app, "owner-hook-noop")
	rec := reloadItinerary(t, app, published.Id)
	rec.Set("status", string(StatusApproved))
	if err := app.Save(rec); err != nil {
		t.Fatalf("pending -> approved: %v", err)
	}

	// A no-op save (only autodate `updated` changes) must be permitted.
	rec = reloadItinerary(t, app, published.Id)
	if err := app.Save(rec); err != nil {
		t.Errorf("no-op save of approved record must be allowed: %v", err)
	}
}

func newPublishedRecord(t *testing.T, app core.App, owner string) *core.Record {
	t.Helper()
	addStops(t, app, owner, 1)
	if err := SetMeta(app, owner, Meta{Title: "Journey"}); err != nil {
		t.Fatalf("SetMeta: %v", err)
	}
	published, err := Publish(app, owner)
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	return published
}

// newPendingRecord creates a legacy pending record by publishing and then
// demoting the record through raw SQL. The lifecycle hook forbids entering
// pending from the new code paths, so raw SQL is the only faithful way to
// simulate a pre-immediate-publication row.
func newPendingRecord(t *testing.T, app core.App, owner string) *core.Record {
	t.Helper()
	published := newPublishedRecord(t, app, owner)
	if _, err := app.DB().NewQuery(
		"UPDATE itineraries SET status = 'pending' WHERE id = {:id}",
	).Bind(map[string]any{"id": published.Id}).Execute(); err != nil {
		t.Fatalf("demote to pending: %v", err)
	}
	return reloadItinerary(t, app, published.Id)
}

func reloadItinerary(t *testing.T, app core.App, id string) *core.Record {
	t.Helper()
	record, err := app.FindRecordById(CollectionItineraries, id)
	if err != nil {
		t.Fatalf("reload itinerary: %v", err)
	}
	return record
}
