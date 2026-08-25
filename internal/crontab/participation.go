package crontab

import (
	"log/slog"
	"time"

	"github.com/blackfyre/wga/internal/handlers/guestbook"
	"github.com/blackfyre/wga/internal/logging"
	"github.com/blackfyre/wga/internal/postcards"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

const (
	// participationPurgeSchedule runs the shared anonymous-participation
	// retention job once a day.
	participationPurgeSchedule = "15 3 * * *"

	// guestbookPurgeBatchSize mirrors guestbook.PurgeExpiredPrivateEntries,
	// which removes at most 100 expired non-approved entries per call. A run
	// keeps draining while a full batch is removed.
	guestbookPurgeBatchSize = 100

	// guestbookPurgeMaxBatches bounds one run so a pathological backlog cannot
	// make the job unbounded.
	guestbookPurgeMaxBatches = 50

	// postcardPurgeBatchSize is the conservative per-call limit passed to
	// postcards.PurgeExpiredRecipientAccess, matching the guestbook batch so a
	// pathological postcard backlog stays bounded.
	postcardPurgeBatchSize = 100

	// postcardPurgeMaxBatches bounds one run's postcard drain consistently with
	// the guestbook drain.
	postcardPurgeMaxBatches = 50
)

// guestbookPurgeFn removes one bounded batch of expired private guestbook
// entries and reports how many were removed.
type guestbookPurgeFn func(now time.Time) (int, error)

// postcardPurgeFn removes one bounded batch of expired postcard recipient
// access and postcard content and reports the per-kind counts removed.
type postcardPurgeFn func(now types.DateTime, limit int) (postcards.PurgeCounts, error)

// participationPurgeResult reports the redacted aggregate counts removed by one
// run and whether either backlog exhausted its per-run batch budget.
type participationPurgeResult struct {
	guestbookRemoved       int
	deliveryAccessRemoved  int
	postcardContentRemoved int
	exhausted              bool
}

// drainGuestbook removes expired private guestbook entries in bounded batches
// until a batch is not full or the per-run batch budget is exhausted. exhausted
// reports that full batches were removed for the entire budget, so more work
// may remain for the next scheduled run.
func drainGuestbook(purge guestbookPurgeFn, now time.Time) (removed int, exhausted bool, err error) {
	for batch := 0; batch < guestbookPurgeMaxBatches; batch++ {
		count, err := purge(now)
		if err != nil {
			return removed, false, err
		}
		removed += count
		if count < guestbookPurgeBatchSize {
			return removed, false, nil
		}
	}

	return removed, true, nil
}

// drainPostcards removes expired postcard recipient access and content in
// bounded batches. Because each call purges the two kinds independently, a
// backlog is only considered drained once both per-kind counts are a partial
// batch; while either kind still fills a whole batch, more work may remain.
// exhausted reports that at least one kind kept filling a full batch for the
// entire budget.
func drainPostcards(purge postcardPurgeFn, now types.DateTime) (deliveryAccess, postcardContent int, exhausted bool, err error) {
	for batch := 0; batch < postcardPurgeMaxBatches; batch++ {
		counts, err := purge(now, postcardPurgeBatchSize)
		if err != nil {
			return deliveryAccess, postcardContent, false, err
		}
		deliveryAccess += counts.DeliveryAccess
		postcardContent += counts.PostcardContent
		if counts.DeliveryAccess < postcardPurgeBatchSize && counts.PostcardContent < postcardPurgeBatchSize {
			return deliveryAccess, postcardContent, false, nil
		}
	}

	return deliveryAccess, postcardContent, true, nil
}

// purgeParticipation drains the guestbook and postcard backlogs against a
// single fixed time snapshot and combines their exhaustion into one result.
func purgeParticipation(now types.DateTime, guestbook guestbookPurgeFn, postcard postcardPurgeFn) (participationPurgeResult, error) {
	result := participationPurgeResult{}

	guestbookRemoved, guestbookExhausted, err := drainGuestbook(guestbook, now.Time())
	if err != nil {
		return result, err
	}
	result.guestbookRemoved = guestbookRemoved

	deliveryAccessRemoved, postcardContentRemoved, postcardExhausted, err := drainPostcards(postcard, now)
	if err != nil {
		return result, err
	}
	result.deliveryAccessRemoved = deliveryAccessRemoved
	result.postcardContentRemoved = postcardContentRemoved
	result.exhausted = guestbookExhausted || postcardExhausted

	return result, nil
}

// runParticipationPurge drains expired private guestbook entries and expired
// postcard recipient access in bounded batches against one fixed time snapshot.
// It never removes approved guestbook archive records, active/processing
// postcards, or unresolved dead letters; those retention rules live in the
// owning feature purge APIs.
func runParticipationPurge(app core.App) (participationPurgeResult, error) {
	now := types.NowDateTime()

	return purgeParticipation(now,
		func(now time.Time) (int, error) {
			return guestbook.PurgeExpiredPrivateEntries(app, now)
		},
		func(now types.DateTime, limit int) (postcards.PurgeCounts, error) {
			return postcards.PurgeExpiredRecipientAccess(app, now, limit)
		},
	)
}

// logParticipationRun emits the redacted aggregate counts and the run outcome.
// Only aggregate guestbook, delivery-access, and postcard-content counts are
// logged; no raw identities leave the retention path.
func logParticipationRun(logger *slog.Logger, result participationPurgeResult, err error) {
	if err != nil {
		logger.Error("Participation lifecycle run failed",
			"event", "participation.purge.run",
			"outcome", "failed",
			"error_type", logging.ErrorType(err),
			"error", logging.Redact(err),
		)
		return
	}

	if result.exhausted {
		logger.Warn("Participation lifecycle run exhausted its batch budget",
			"event", "participation.purge.run",
			"outcome", "exhausted",
			"guestbook_entries_removed", result.guestbookRemoved,
			"postcard_delivery_access_removed", result.deliveryAccessRemoved,
			"postcard_content_removed", result.postcardContentRemoved,
		)
		return
	}

	logger.Info("Participation lifecycle run completed",
		"event", "participation.purge.run",
		"outcome", "completed",
		"guestbook_entries_removed", result.guestbookRemoved,
		"postcard_delivery_access_removed", result.deliveryAccessRemoved,
		"postcard_content_removed", result.postcardContentRemoved,
	)
}

// registerParticipationPurge wires the shared anonymous-participation retention
// job into the PocketBase scheduler. It is the serial integration owner's
// feature-owned lifecycle job for guestbook and postcard retention.
func registerParticipationPurge(app core.App) {
	app.Logger().Info("Participation lifecycle schedule registered", "event", "participation.purge.schedule_registered")
	app.Cron().MustAdd("participation-purge", participationPurgeSchedule, func() {
		runID := logging.NewRunID()
		logger := logging.RunLogger(app, runID)
		logger.Info("Participation lifecycle run started", "event", "participation.purge.run", "outcome", "started")

		result, err := runParticipationPurge(app)
		logParticipationRun(logger, result, err)
	})
}
