package hooks

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/logging"
	"github.com/blackfyre/wga/internal/testutils"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/logger"
	"github.com/pocketbase/pocketbase/tools/router"
)

func TestLogFileDownloadExcludesRequestEvent(t *testing.T) {
	app := testutils.NewTestApp(t)
	handler, ok := app.Logger().Handler().(*logger.BatchHandler)
	if !ok {
		t.Fatalf("expected BatchHandler, got %T", app.Logger().Handler())
	}
	handler.SetLevel(slog.LevelDebug)
	captured := testutils.CaptureLogs(app)
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

	testutils.FlushLogs(t, app)
	entry := testutils.LogWithEvent(captured(), "file.download.served")
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

	output := fmt.Sprint(testutils.LogData(captured()))
	for _, sensitive := range []string{"secret-token", "secret-message-body", "secret-served-path", "secret-file-name"} {
		if strings.Contains(output, sensitive) {
			t.Fatalf("captured log contains %q: %s", sensitive, output)
		}
	}
}
