package crontab

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/blackfyre/wga/internal/postcards"
	"github.com/pocketbase/pocketbase/tools/types"
)

func fixedDateTime(t *testing.T) types.DateTime {
	t.Helper()
	now, err := types.ParseDateTime("2026-01-02 03:04:05.000Z")
	if err != nil {
		t.Fatalf("parse datetime: %v", err)
	}
	return now
}

func TestDrainGuestbookStopsOnPartialBatch(t *testing.T) {
	now := time.Unix(1000, 0)
	batch := 0
	results := []int{guestbookPurgeBatchSize, guestbookPurgeBatchSize, 7}
	purge := func(time.Time) (int, error) {
		count := results[batch]
		batch++
		return count, nil
	}

	removed, exhausted, err := drainGuestbook(purge, now)
	if err != nil {
		t.Fatalf("drain: %v", err)
	}
	if removed != guestbookPurgeBatchSize*2+7 {
		t.Errorf("removed = %d, want %d", removed, guestbookPurgeBatchSize*2+7)
	}
	if exhausted {
		t.Error("partial batch must not report exhausted")
	}
}

func TestDrainGuestbookExhaustsBatchBudget(t *testing.T) {
	now := time.Unix(1000, 0)
	purge := func(time.Time) (int, error) {
		return guestbookPurgeBatchSize, nil
	}

	removed, exhausted, err := drainGuestbook(purge, now)
	if err != nil {
		t.Fatalf("drain: %v", err)
	}
	if removed != guestbookPurgeBatchSize*guestbookPurgeMaxBatches {
		t.Errorf("removed = %d, want %d", removed, guestbookPurgeBatchSize*guestbookPurgeMaxBatches)
	}
	if !exhausted {
		t.Error("full batches across the whole budget must report exhausted")
	}
}

func TestDrainGuestbookPropagatesError(t *testing.T) {
	now := time.Unix(1000, 0)
	boom := errors.New("purge failed")
	purge := func(time.Time) (int, error) {
		return 0, boom
	}

	if _, _, err := drainGuestbook(purge, now); !errors.Is(err, boom) {
		t.Fatalf("error = %v, want %v", err, boom)
	}
}

func TestDrainGuestbookStopsImmediatelyOnEmptyResult(t *testing.T) {
	now := time.Unix(1000, 0)
	calls := 0
	purge := func(time.Time) (int, error) {
		calls++
		return 0, nil
	}

	removed, exhausted, err := drainGuestbook(purge, now)
	if err != nil {
		t.Fatalf("drain: %v", err)
	}
	if removed != 0 || exhausted {
		t.Errorf("empty result = removed %d exhausted %t", removed, exhausted)
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1", calls)
	}
}

func TestDrainPostcardsStopsWhenBothKindsPartial(t *testing.T) {
	now := fixedDateTime(t)
	calls := 0
	steps := []postcards.PurgeCounts{
		{DeliveryAccess: postcardPurgeBatchSize, PostcardContent: postcardPurgeBatchSize},
		{DeliveryAccess: postcardPurgeBatchSize, PostcardContent: postcardPurgeBatchSize},
		{DeliveryAccess: 7, PostcardContent: 3},
	}
	purge := func(types.DateTime, int) (postcards.PurgeCounts, error) {
		counts := steps[calls]
		calls++
		return counts, nil
	}

	deliveryAccess, postcardContent, exhausted, err := drainPostcards(purge, now)
	if err != nil {
		t.Fatalf("drain: %v", err)
	}
	if deliveryAccess != postcardPurgeBatchSize*2+7 {
		t.Errorf("delivery access = %d, want %d", deliveryAccess, postcardPurgeBatchSize*2+7)
	}
	if postcardContent != postcardPurgeBatchSize*2+3 {
		t.Errorf("postcard content = %d, want %d", postcardContent, postcardPurgeBatchSize*2+3)
	}
	if exhausted {
		t.Error("partial batch must not report exhausted")
	}
	if calls != 3 {
		t.Errorf("calls = %d, want 3", calls)
	}
}

func TestDrainPostcardsDrainsIndependentFullBacklogs(t *testing.T) {
	now := fixedDateTime(t)
	tests := []struct {
		name          string
		steps         []postcards.PurgeCounts
		wantAccess    int
		wantContent   int
		wantExhausted bool
		wantCalls     int
	}{
		{
			name: "delivery access full while content partial",
			steps: []postcards.PurgeCounts{
				{DeliveryAccess: postcardPurgeBatchSize, PostcardContent: 0},
				{DeliveryAccess: postcardPurgeBatchSize, PostcardContent: 0},
				{DeliveryAccess: 40, PostcardContent: 0},
			},
			wantAccess:    postcardPurgeBatchSize*2 + 40,
			wantContent:   0,
			wantExhausted: false,
			wantCalls:     3,
		},
		{
			name: "postcard content full while delivery access partial",
			steps: []postcards.PurgeCounts{
				{DeliveryAccess: 0, PostcardContent: postcardPurgeBatchSize},
				{DeliveryAccess: 0, PostcardContent: 12},
			},
			wantAccess:    0,
			wantContent:   postcardPurgeBatchSize + 12,
			wantExhausted: false,
			wantCalls:     2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			purge := func(types.DateTime, int) (postcards.PurgeCounts, error) {
				counts := tt.steps[calls]
				calls++
				return counts, nil
			}

			deliveryAccess, postcardContent, exhausted, err := drainPostcards(purge, now)
			if err != nil {
				t.Fatalf("drain: %v", err)
			}
			if deliveryAccess != tt.wantAccess {
				t.Errorf("delivery access = %d, want %d", deliveryAccess, tt.wantAccess)
			}
			if postcardContent != tt.wantContent {
				t.Errorf("postcard content = %d, want %d", postcardContent, tt.wantContent)
			}
			if exhausted != tt.wantExhausted {
				t.Errorf("exhausted = %t, want %t", exhausted, tt.wantExhausted)
			}
			if calls != tt.wantCalls {
				t.Errorf("calls = %d, want %d", calls, tt.wantCalls)
			}
		})
	}
}

func TestDrainPostcardsStopsImmediatelyOnEmptyResult(t *testing.T) {
	now := fixedDateTime(t)
	calls := 0
	purge := func(types.DateTime, int) (postcards.PurgeCounts, error) {
		calls++
		return postcards.PurgeCounts{}, nil
	}

	deliveryAccess, postcardContent, exhausted, err := drainPostcards(purge, now)
	if err != nil {
		t.Fatalf("drain: %v", err)
	}
	if deliveryAccess != 0 || postcardContent != 0 || exhausted {
		t.Errorf("empty result = access %d content %d exhausted %t", deliveryAccess, postcardContent, exhausted)
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1", calls)
	}
}

func TestDrainPostcardsExhaustsBatchBudget(t *testing.T) {
	now := fixedDateTime(t)
	purge := func(types.DateTime, int) (postcards.PurgeCounts, error) {
		return postcards.PurgeCounts{
			DeliveryAccess:  postcardPurgeBatchSize,
			PostcardContent: postcardPurgeBatchSize,
		}, nil
	}

	deliveryAccess, postcardContent, exhausted, err := drainPostcards(purge, now)
	if err != nil {
		t.Fatalf("drain: %v", err)
	}
	if deliveryAccess != postcardPurgeBatchSize*postcardPurgeMaxBatches {
		t.Errorf("delivery access = %d, want %d", deliveryAccess, postcardPurgeBatchSize*postcardPurgeMaxBatches)
	}
	if postcardContent != postcardPurgeBatchSize*postcardPurgeMaxBatches {
		t.Errorf("postcard content = %d, want %d", postcardContent, postcardPurgeBatchSize*postcardPurgeMaxBatches)
	}
	if !exhausted {
		t.Error("full batches across the whole budget must report exhausted")
	}
}

func TestDrainPostcardsExhaustsWhenOneKindStaysFull(t *testing.T) {
	now := fixedDateTime(t)
	purge := func(types.DateTime, int) (postcards.PurgeCounts, error) {
		return postcards.PurgeCounts{DeliveryAccess: postcardPurgeBatchSize, PostcardContent: 0}, nil
	}

	deliveryAccess, postcardContent, exhausted, err := drainPostcards(purge, now)
	if err != nil {
		t.Fatalf("drain: %v", err)
	}
	if deliveryAccess != postcardPurgeBatchSize*postcardPurgeMaxBatches {
		t.Errorf("delivery access = %d, want %d", deliveryAccess, postcardPurgeBatchSize*postcardPurgeMaxBatches)
	}
	if postcardContent != 0 {
		t.Errorf("postcard content = %d, want 0", postcardContent)
	}
	if !exhausted {
		t.Error("a kind that stays full for the whole budget must report exhausted")
	}
}

func TestDrainPostcardsPropagatesError(t *testing.T) {
	now := fixedDateTime(t)
	boom := errors.New("purge failed")
	purge := func(types.DateTime, int) (postcards.PurgeCounts, error) {
		return postcards.PurgeCounts{}, boom
	}

	if _, _, _, err := drainPostcards(purge, now); !errors.Is(err, boom) {
		t.Fatalf("error = %v, want %v", err, boom)
	}
}

func TestPurgeParticipationCombinesResults(t *testing.T) {
	now := fixedDateTime(t)

	t.Run("neither backlog exhausted", func(t *testing.T) {
		guestbook := func(time.Time) (int, error) { return guestbookPurgeBatchSize - 1, nil }
		postcard := func(types.DateTime, int) (postcards.PurgeCounts, error) {
			return postcards.PurgeCounts{DeliveryAccess: 5, PostcardContent: 3}, nil
		}

		result, err := purgeParticipation(now, guestbook, postcard)
		if err != nil {
			t.Fatalf("purge: %v", err)
		}
		if result.exhausted {
			t.Error("combined result must not report exhausted")
		}
		if result.guestbookRemoved != guestbookPurgeBatchSize-1 {
			t.Errorf("guestbook removed = %d, want %d", result.guestbookRemoved, guestbookPurgeBatchSize-1)
		}
		if result.deliveryAccessRemoved != 5 || result.postcardContentRemoved != 3 {
			t.Errorf("postcard counts = %d/%d, want 5/3", result.deliveryAccessRemoved, result.postcardContentRemoved)
		}
	})

	t.Run("guestbook exhausted", func(t *testing.T) {
		guestbook := func(time.Time) (int, error) { return guestbookPurgeBatchSize, nil }
		postcard := func(types.DateTime, int) (postcards.PurgeCounts, error) {
			return postcards.PurgeCounts{}, nil
		}

		result, err := purgeParticipation(now, guestbook, postcard)
		if err != nil {
			t.Fatalf("purge: %v", err)
		}
		if !result.exhausted {
			t.Error("combined result must report exhausted when guestbook exhausts")
		}
		if result.guestbookRemoved != guestbookPurgeBatchSize*guestbookPurgeMaxBatches {
			t.Errorf("guestbook removed = %d, want %d", result.guestbookRemoved, guestbookPurgeBatchSize*guestbookPurgeMaxBatches)
		}
	})

	t.Run("postcard exhausted", func(t *testing.T) {
		guestbook := func(time.Time) (int, error) { return 0, nil }
		postcard := func(types.DateTime, int) (postcards.PurgeCounts, error) {
			return postcards.PurgeCounts{DeliveryAccess: postcardPurgeBatchSize, PostcardContent: postcardPurgeBatchSize}, nil
		}

		result, err := purgeParticipation(now, guestbook, postcard)
		if err != nil {
			t.Fatalf("purge: %v", err)
		}
		if !result.exhausted {
			t.Error("combined result must report exhausted when postcard exhausts")
		}
		if result.deliveryAccessRemoved != postcardPurgeBatchSize*postcardPurgeMaxBatches {
			t.Errorf("delivery access removed = %d, want %d", result.deliveryAccessRemoved, postcardPurgeBatchSize*postcardPurgeMaxBatches)
		}
	})
}

func TestPurgeParticipationPropagatesErrors(t *testing.T) {
	now := fixedDateTime(t)

	t.Run("guestbook error", func(t *testing.T) {
		boom := errors.New("guestbook purge failed")
		if _, err := purgeParticipation(now, func(time.Time) (int, error) {
			return 0, boom
		}, func(types.DateTime, int) (postcards.PurgeCounts, error) {
			return postcards.PurgeCounts{}, nil
		}); !errors.Is(err, boom) {
			t.Fatalf("error = %v, want %v", err, boom)
		}
	})

	t.Run("postcard error", func(t *testing.T) {
		boom := errors.New("postcard purge failed")
		if _, err := purgeParticipation(now, func(time.Time) (int, error) {
			return 0, nil
		}, func(types.DateTime, int) (postcards.PurgeCounts, error) {
			return postcards.PurgeCounts{}, boom
		}); !errors.Is(err, boom) {
			t.Fatalf("error = %v, want %v", err, boom)
		}
	})
}

type recordHandler struct {
	records []slog.Record
}

func (h *recordHandler) Enabled(context.Context, slog.Level) bool { return true }
func (h *recordHandler) Handle(_ context.Context, r slog.Record) error {
	h.records = append(h.records, r)
	return nil
}
func (h *recordHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *recordHandler) WithGroup(string) slog.Handler      { return h }

func recordWithMessage(records []slog.Record, msg string) *slog.Record {
	for i := range records {
		if records[i].Message == msg {
			return &records[i]
		}
	}
	return nil
}

func recordAttr(t *testing.T, record *slog.Record, key string) (slog.Value, bool) {
	t.Helper()
	var value slog.Value
	found := false
	record.Attrs(func(a slog.Attr) bool {
		if a.Key == key {
			value = a.Value
			found = true
			return false
		}
		return true
	})
	return value, found
}

func assertIntAttr(t *testing.T, record *slog.Record, key string, want int) {
	t.Helper()
	value, ok := recordAttr(t, record, key)
	if !ok {
		t.Fatalf("log missing attribute %q", key)
	}
	if value.Kind() != slog.KindInt64 {
		t.Fatalf("attribute %q kind = %s, want int64", key, value.Kind())
	}
	if got := value.Int64(); got != int64(want) {
		t.Errorf("attribute %q = %d, want %d", key, got, want)
	}
}

func TestLogParticipationRunOutcomesAndFields(t *testing.T) {
	t.Run("completed", func(t *testing.T) {
		handler := &recordHandler{}
		logger := slog.New(handler)
		logParticipationRun(logger, participationPurgeResult{
			guestbookRemoved:       3,
			deliveryAccessRemoved:  4,
			postcardContentRemoved: 5,
		}, nil)

		record := recordWithMessage(handler.records, "Participation lifecycle run completed")
		if record == nil {
			t.Fatal("expected a completed log")
		}
		if record.Level != slog.LevelInfo {
			t.Errorf("level = %s, want INFO", record.Level)
		}
		assertOutcome(t, record, "completed")
		assertIntAttr(t, record, "guestbook_entries_removed", 3)
		assertIntAttr(t, record, "postcard_delivery_access_removed", 4)
		assertIntAttr(t, record, "postcard_content_removed", 5)
	})

	t.Run("exhausted", func(t *testing.T) {
		handler := &recordHandler{}
		logger := slog.New(handler)
		logParticipationRun(logger, participationPurgeResult{
			guestbookRemoved:       12,
			deliveryAccessRemoved:  34,
			postcardContentRemoved: 56,
			exhausted:              true,
		}, nil)

		record := recordWithMessage(handler.records, "Participation lifecycle run exhausted its batch budget")
		if record == nil {
			t.Fatal("expected an exhausted log")
		}
		if record.Level != slog.LevelWarn {
			t.Errorf("level = %s, want WARN", record.Level)
		}
		assertOutcome(t, record, "exhausted")
		assertIntAttr(t, record, "guestbook_entries_removed", 12)
		assertIntAttr(t, record, "postcard_delivery_access_removed", 34)
		assertIntAttr(t, record, "postcard_content_removed", 56)
	})

	t.Run("failed", func(t *testing.T) {
		handler := &recordHandler{}
		logger := slog.New(handler)
		logParticipationRun(logger, participationPurgeResult{}, errors.New("boom"))

		record := recordWithMessage(handler.records, "Participation lifecycle run failed")
		if record == nil {
			t.Fatal("expected a failed log")
		}
		if record.Level != slog.LevelError {
			t.Errorf("level = %s, want ERROR", record.Level)
		}
		assertOutcome(t, record, "failed")
		errorType, ok := recordAttr(t, record, "error_type")
		if !ok || errorType.String() == "" {
			t.Error("failed log missing a non-empty error_type")
		}
		redacted, ok := recordAttr(t, record, "error")
		if !ok || redacted.String() != "[REDACTED]" {
			t.Errorf("failed log error = %v, want [REDACTED]", redacted)
		}
	})
}

func assertOutcome(t *testing.T, record *slog.Record, want string) {
	t.Helper()
	value, ok := recordAttr(t, record, "outcome")
	if !ok {
		t.Fatalf("log missing attribute %q", "outcome")
	}
	if got := value.String(); got != want {
		t.Errorf("outcome = %q, want %q", got, want)
	}
}
