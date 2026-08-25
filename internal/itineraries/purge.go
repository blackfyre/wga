package itineraries

import (
	"github.com/blackfyre/wga/internal/logging"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

// PurgeResult reports the counts removed by a purge run for observability.
type PurgeResult struct {
	ExpiredPublished int
	AbandonedDrafts  int
}

// Purge removes expired published itineraries and abandoned drafts. Deleting an
// itinerary cascades to its stops through the stop relation. Counts are
// returned for redacted, operational logging by the owning job.
func Purge(app core.App) (PurgeResult, error) {
	var result PurgeResult

	if err := app.RunInTransaction(func(txApp core.App) error {
		if err := purgeExpired(txApp, &result); err != nil {
			return err
		}
		return purgeAbandoned(txApp, &result)
	}); err != nil {
		return PurgeResult{}, err
	}

	return result, nil
}

func purgeExpired(app core.App, result *PurgeResult) error {
	now := types.NowDateTime()
	expired, err := app.FindRecordsByFilter(
		CollectionItineraries,
		"status != {:draft} && expires_at != '' && expires_at < {:now}",
		"",
		0,
		0,
		dbx.Params{"draft": string(StatusDraft), "now": now},
	)
	if err != nil {
		return err
	}

	for _, record := range expired {
		if err := app.Delete(record); err != nil {
			return err
		}
		result.ExpiredPublished++
	}

	return nil
}

func purgeAbandoned(app core.App, result *PurgeResult) error {
	cutoff := types.NowDateTime().Add(-PublicationLifetime)
	abandoned, err := app.FindRecordsByFilter(
		CollectionItineraries,
		"status = {:draft} && updated < {:cutoff}",
		"",
		0,
		0,
		dbx.Params{"draft": string(StatusDraft), "cutoff": cutoff},
	)
	if err != nil {
		return err
	}

	for _, record := range abandoned {
		if err := app.Delete(record); err != nil {
			return err
		}
		result.AbandonedDrafts++
	}

	return nil
}

// RegisterPurgeJob wires the daily purge into the PocketBase scheduler. It is
// feature-owned lifecycle code invoked by the serial integration owner from the
// shared crontab registration.
func RegisterPurgeJob(app core.App) {
	const schedule = "0 3 * * *"

	app.Logger().Info("Itinerary purge schedule registered", "event", "itineraries.purge.schedule_registered")
	app.Cron().MustAdd("itineraries-purge", schedule, func() {
		runID := logging.NewRunID()
		logger := logging.RunLogger(app, runID)
		logger.Info("Itinerary purge run started", "event", "itineraries.purge.run", "outcome", "started")

		result, err := Purge(app)
		if err != nil {
			logger.Error("Itinerary purge run failed",
				"event", "itineraries.purge.run",
				"outcome", "failed",
				"error_type", logging.ErrorType(err),
				"error", logging.Redact(err),
			)
			return
		}

		logger.Info("Itinerary purge run completed",
			"event", "itineraries.purge.run",
			"outcome", "completed",
			"expired_published", result.ExpiredPublished,
			"abandoned_drafts", result.AbandonedDrafts,
		)
	})
}
