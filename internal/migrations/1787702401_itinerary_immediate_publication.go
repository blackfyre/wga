package migrations

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(backfillItineraryImmediatePublication, removeItineraryImmediatePublication)
}

// backfillItineraryImmediatePublication reconciles the legacy pending state
// with the immediate-publication contract (ADR 0014 state machine: publication
// now transitions draft -> approved). Records that predate the change were left
// in `pending` awaiting moderation, which made their public-looking token return
// 404. They are promoted to `approved` in one bounded, set-based update so they
// become readable at once while preserving each record's listed flag and expiry.
//
// The update is transactional (a single statement), idempotent (a second run
// finds no pending rows), and deliberately narrow (it matches only
// `status = 'pending'`, so drafts, already-approved, and rejected records are
// untouched — rejected records stay denied). It is performed as raw SQL rather
// than per-record saves so the migration never revalidates legacy rows or trips
// the lifecycle hook while reconciling data that predates the current rules.
func backfillItineraryImmediatePublication(app core.App) error {
	itineraries, err := app.FindCollectionByNameOrId("itineraries")
	if err != nil {
		return err
	}

	_, err = app.DB().Update(
		itineraries.Name,
		dbx.Params{"status": "approved"},
		dbx.HashExp{"status": "pending"},
	).Execute()

	return err
}

// removeItineraryImmediatePublication is a deliberate no-op. The backfill
// outcome is authoritative: a code rollback disables the public routes rather
// than re-hiding published records, matching removeItineraries and
// removeParticipationPublication. There is no structural artifact (index or
// field) to remove, and reversing the data would downgrade records that may
// have been published under the new contract, so nothing is changed.
func removeItineraryImmediatePublication(_ core.App) error {
	return nil
}
