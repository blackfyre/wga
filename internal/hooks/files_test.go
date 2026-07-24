package hooks

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/logging"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/logger"
	"github.com/pocketbase/pocketbase/tools/router"
)

func TestLogFileDownloadExcludesRequestEvent(t *testing.T) {
	app := newHookTestApp(t)
	captured := captureHookLogs(app)
	collection := core.NewBaseCollection("artworks")
	field := &core.FileField{Name: "image"}
	collection.Fields.Add(field)
	record := core.NewRecord(collection)
	record.Id = "artwork-record"
	request := httptest.NewRequest(http.MethodGet, "/api/files/artworks/artwork-record/secret-file-name", strings.NewReader("secret-message-body"))
	request.Header.Set("Authorization", "secret-token")
	requestEvent := &core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  request,
			Response: httptest.NewRecorder(),
		},
	}
	logging.SetRequestID(requestEvent, "request-123")

	logFileDownload(app, &core.FileDownloadRequestEvent{
		RequestEvent: requestEvent,
		Record:       record,
		FileField:    field,
		ServedPath:   "secret-served-path",
		ServedName:   "secret-file-name",
	})

	flushHookLogs(t, app)
	entry := hookLogWithEvent(captured(), "file.download.served")
	if entry == nil {
		t.Fatal("expected a file download log")
	}
	if got := entry.Data["request_id"]; got != "request-123" {
		t.Fatalf("request_id = %v, want %q", got, "request-123")
	}
	if got := entry.Data["record_id"]; got != record.Id {
		t.Fatalf("record_id = %v, want %q", got, record.Id)
	}
	if got := entry.Data["file_field"]; got != field.Name {
		t.Fatalf("file_field = %v, want %q", got, field.Name)
	}

	output := fmt.Sprint(hookLogData(captured()))
	for _, sensitive := range []string{"secret-token", "secret-message-body", "secret-served-path", "secret-file-name"} {
		if strings.Contains(output, sensitive) {
			t.Fatalf("captured log contains %q: %s", sensitive, output)
		}
	}
}

func newHookTestApp(t *testing.T) *tests.TestApp {
	t.Helper()

	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("create test app: %v", err)
	}
	t.Cleanup(app.Cleanup)
	app.Settings().Logs.MaxDays = 1
	handler, ok := app.Logger().Handler().(*logger.BatchHandler)
	if !ok {
		t.Fatalf("expected BatchHandler, got %T", app.Logger().Handler())
	}
	handler.SetLevel(slog.LevelDebug)

	return app
}

func captureHookLogs(app *tests.TestApp) func() []*core.Log {
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

func flushHookLogs(t *testing.T, app *tests.TestApp) {
	t.Helper()

	handler, ok := app.Logger().Handler().(*logger.BatchHandler)
	if !ok {
		t.Fatalf("expected BatchHandler, got %T", app.Logger().Handler())
	}
	if err := handler.WriteAll(context.Background()); err != nil {
		t.Fatalf("write logs: %v", err)
	}
}

func hookLogWithEvent(logs []*core.Log, event string) *core.Log {
	for _, entry := range logs {
		if entry.Data["event"] == event {
			return entry
		}
	}

	return nil
}

func hookLogData(logs []*core.Log) []any {
	data := make([]any, len(logs))
	for index, entry := range logs {
		data[index] = entry.Data
	}

	return data
}
