package itineraries

import (
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/tools/types"
)

func TestPurgeRemovesExpiredAndAbandoned(t *testing.T) {
	app := installItinerarySchema(t)

	// A published, expired itinerary.
	owner := "owner-expired"
	addStops(t, app, owner, 1)
	if err := SetMeta(app, owner, Meta{Title: "Expired"}); err != nil {
		t.Fatalf("SetMeta: %v", err)
	}
	expired, err := Publish(app, owner)
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	expired.Set("expires_at", types.NowDateTime().Add(-time.Hour))
	if err := app.Save(expired); err != nil {
		t.Fatalf("expire itinerary: %v", err)
	}

	// A still-live published itinerary.
	liveOwner := "owner-live"
	addStops(t, app, liveOwner, 1)
	if err := SetMeta(app, liveOwner, Meta{Title: "Live"}); err != nil {
		t.Fatalf("SetMeta: %v", err)
	}
	live, err := Publish(app, liveOwner)
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}

	// An abandoned draft older than one year. The autodate field resets
	// `updated` on save, so age the record directly via SQL.
	abandonedOwner := "owner-abandoned"
	addStops(t, app, abandonedOwner, 1)
	abandoned, err := FindDraft(app, abandonedOwner)
	if err != nil {
		t.Fatalf("FindDraft: %v", err)
	}
	if _, err := app.DB().NewQuery(
		"UPDATE itineraries SET updated = {:updated} WHERE id = {:id}",
	).Bind(map[string]any{
		"updated": types.NowDateTime().Add(-PublicationLifetime - time.Hour),
		"id":      abandoned.Id,
	}).Execute(); err != nil {
		t.Fatalf("age draft: %v", err)
	}

	result, err := Purge(app)
	if err != nil {
		t.Fatalf("Purge: %v", err)
	}
	if result.ExpiredPublished != 1 {
		t.Errorf("ExpiredPublished = %d, want 1", result.ExpiredPublished)
	}
	if result.AbandonedDrafts != 1 {
		t.Errorf("AbandonedDrafts = %d, want 1", result.AbandonedDrafts)
	}

	if _, err := app.FindRecordById(CollectionItineraries, expired.Id); err == nil {
		t.Error("expired itinerary must be removed")
	}
	if _, err := app.FindRecordById(CollectionItineraries, abandoned.Id); err == nil {
		t.Error("abandoned draft must be removed")
	}
	if _, err := app.FindRecordById(CollectionItineraries, live.Id); err != nil {
		t.Error("live itinerary must be retained")
	}
}

func TestPurgeCascadesToStops(t *testing.T) {
	app := installItinerarySchema(t)

	owner := "owner-cascade"
	addStops(t, app, owner, 2)
	if err := SetMeta(app, owner, Meta{Title: "Cascade"}); err != nil {
		t.Fatalf("SetMeta: %v", err)
	}
	published, err := Publish(app, owner)
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	published.Set("expires_at", types.NowDateTime().Add(-time.Hour))
	if err := app.Save(published); err != nil {
		t.Fatalf("expire itinerary: %v", err)
	}

	if _, err := Purge(app); err != nil {
		t.Fatalf("Purge: %v", err)
	}

	stops, err := app.FindRecordsByFilter(CollectionItineraryStops, "itinerary = {:itinerary}", "", 0, 0, map[string]any{"itinerary": published.Id})
	if err != nil {
		t.Fatalf("find stops: %v", err)
	}
	if len(stops) != 0 {
		t.Errorf("stops = %d, want 0 after cascade", len(stops))
	}
}

func TestPurgeIgnoresRecentDraft(t *testing.T) {
	app := installItinerarySchema(t)

	owner := "owner-recent"
	addStops(t, app, owner, 1)

	result, err := Purge(app)
	if err != nil {
		t.Fatalf("Purge: %v", err)
	}
	if result.AbandonedDrafts != 0 {
		t.Errorf("AbandonedDrafts = %d, want 0 for a recent draft", result.AbandonedDrafts)
	}
	if _, err := FindDraft(app, owner); err != nil {
		t.Error("recent draft must be retained")
	}
}

func TestPurgeIsIdempotent(t *testing.T) {
	app := installItinerarySchema(t)

	// A published, expired itinerary.
	owner := "owner-idempotent"
	addStops(t, app, owner, 1)
	if err := SetMeta(app, owner, Meta{Title: "Expired"}); err != nil {
		t.Fatalf("SetMeta: %v", err)
	}
	expired, err := Publish(app, owner)
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	expired.Set("expires_at", types.NowDateTime().Add(-time.Hour))
	if err := app.Save(expired); err != nil {
		t.Fatalf("expire itinerary: %v", err)
	}

	first, err := Purge(app)
	if err != nil {
		t.Fatalf("first Purge: %v", err)
	}
	if first.ExpiredPublished != 1 {
		t.Fatalf("first purge ExpiredPublished = %d, want 1", first.ExpiredPublished)
	}

	// A second run finds nothing more to remove.
	second, err := Purge(app)
	if err != nil {
		t.Fatalf("second Purge: %v", err)
	}
	if second.ExpiredPublished != 0 || second.AbandonedDrafts != 0 {
		t.Errorf("second purge = %+v, want zero removals (idempotent)", second)
	}
}
