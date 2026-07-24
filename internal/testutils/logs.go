package testutils

import (
	"context"
	"maps"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/logger"
)

// NewTestApp creates a test application that persists captured logs.
func NewTestApp(t testing.TB) *tests.TestApp {
	t.Helper()

	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("create test app: %v", err)
	}
	t.Cleanup(app.Cleanup)
	app.Settings().Logs.MaxDays = 1

	return app
}

// CaptureLogs returns the logs persisted after the helper is registered.
func CaptureLogs(app *tests.TestApp) func() []*core.Log {
	var captured []*core.Log
	app.OnModelCreate(core.LogsTableName).BindFunc(func(e *core.ModelEvent) error {
		log, ok := e.Model.(*core.Log)
		if ok {
			entry := *log
			entry.Data = maps.Clone(log.Data)
			captured = append(captured, &entry)
		}

		return e.Next()
	})

	return func() []*core.Log {
		return captured
	}
}

// FlushLogs persists logs queued by PocketBase's batch handler.
func FlushLogs(t testing.TB, app *tests.TestApp) {
	t.Helper()

	handler, ok := app.Logger().Handler().(*logger.BatchHandler)
	if !ok {
		t.Fatalf("expected BatchHandler, got %T", app.Logger().Handler())
	}
	if err := handler.WriteAll(context.Background()); err != nil {
		t.Fatalf("write logs: %v", err)
	}
}

// LogWithEvent returns the first captured log with the supplied event field.
func LogWithEvent(logs []*core.Log, event string) *core.Log {
	for _, entry := range logs {
		if entry.Data["event"] == event {
			return entry
		}
	}

	return nil
}

// LogsWithEvent returns all captured logs with the supplied event field.
func LogsWithEvent(logs []*core.Log, event string) []*core.Log {
	entries := []*core.Log{}
	for _, entry := range logs {
		if entry.Data["event"] == event {
			entries = append(entries, entry)
		}
	}

	return entries
}

// LogData returns all captured log data maps for redaction assertions.
func LogData(logs []*core.Log) []any {
	data := make([]any, len(logs))
	for index, entry := range logs {
		data[index] = entry.Data
	}

	return data
}
