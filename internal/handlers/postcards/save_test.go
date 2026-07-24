package postcards

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/logging"
	"github.com/microcosm-cc/bluemonday"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/logger"
	"github.com/pocketbase/pocketbase/tools/router"
)

func TestSavePostcardDoesNotLogSubmittedForm(t *testing.T) {
	app := newPostcardLoggingTestApp(t)
	captured := capturePostcardLogs(app)
	form := url.Values{
		"sender_name":          {"sender-name-value"},
		"sender_email":         {"sender@example.test"},
		"recipients[]":         {"recipient@example.test"},
		"message":              {"message-body-value"},
		"image_id":             {"image-id"},
		"g-recaptcha-response": {"captcha-token-value"},
		"name":                 {"honeypot-name-value"},
	}
	request := httptest.NewRequest(http.MethodPost, "/postcard", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	event := &core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  request,
			Response: httptest.NewRecorder(),
		},
	}
	logging.SetRequestID(event, "request-123")

	_ = savePostcard(app, event, bluemonday.NewPolicy(), config.Captcha{})

	flushPostcardLogs(t, app)
	entry := postcardLogWithEvent(captured(), "postcard.submission.rejected")
	if entry == nil {
		t.Fatal("expected a postcard rejection log")
	}
	if got := entry.Data["request_id"]; got != "request-123" {
		t.Fatalf("request_id = %v, want %q", got, "request-123")
	}
	if got := entry.Data["outcome"]; got != "honeypot" {
		t.Fatalf("outcome = %v, want %q", got, "honeypot")
	}

	output := fmt.Sprint(postcardLogData(captured()))
	for _, sensitive := range []string{
		"sender-name-value",
		"sender@example.test",
		"recipient@example.test",
		"message-body-value",
		"captcha-token-value",
		"honeypot-name-value",
	} {
		if strings.Contains(output, sensitive) {
			t.Fatalf("captured log contains %q: %s", sensitive, output)
		}
	}
}

func newPostcardLoggingTestApp(t *testing.T) *tests.TestApp {
	t.Helper()

	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("create test app: %v", err)
	}
	t.Cleanup(app.Cleanup)
	app.Settings().Logs.MaxDays = 1

	return app
}

func capturePostcardLogs(app *tests.TestApp) func() []*core.Log {
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

func flushPostcardLogs(t *testing.T, app *tests.TestApp) {
	t.Helper()

	handler, ok := app.Logger().Handler().(*logger.BatchHandler)
	if !ok {
		t.Fatalf("expected BatchHandler, got %T", app.Logger().Handler())
	}
	if err := handler.WriteAll(context.Background()); err != nil {
		t.Fatalf("write logs: %v", err)
	}
}

func postcardLogWithEvent(logs []*core.Log, event string) *core.Log {
	for _, entry := range logs {
		if entry.Data["event"] == event {
			return entry
		}
	}

	return nil
}

func postcardLogData(logs []*core.Log) []any {
	data := make([]any, len(logs))
	for index, entry := range logs {
		data[index] = entry.Data
	}

	return data
}
