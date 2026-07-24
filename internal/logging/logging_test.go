package logging

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/logger"
)

func TestRequestIDMiddlewareCorrelatesRequestLogs(t *testing.T) {
	var captured func() []*core.Log

	scenario := tests.ApiScenario{
		Name:                  "request ID is retained in the route log",
		Method:                http.MethodGet,
		URL:                   "/request-id",
		ExpectedStatus:        http.StatusNoContent,
		DisableTestAppCleanup: true,
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			app := newTestApp(t)
			captured = captureLogs(app)
			RegisterRequestIDMiddleware(app)
			app.OnServe().BindFunc(func(se *core.ServeEvent) error {
				se.Router.GET("/request-id", func(e *core.RequestEvent) error {
					if RequestID(e) != RequestIDFromContext(e.Request.Context()) {
						return e.InternalServerError("request ID was not propagated", nil)
					}

					RequestLogger(app, e).Info("Request correlation test", "event", "test.request.started")
					RequestLogger(app, e).Info("Request correlation test", "event", "test.request.completed")

					return e.NoContent(http.StatusNoContent)
				})

				return se.Next()
			})

			return app
		},
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, response *http.Response) {
			requestID := response.Header.Get(RequestIDHeader)
			if _, err := uuid.Parse(requestID); err != nil {
				t.Fatalf("response request ID %q is not a UUID: %v", requestID, err)
			}

			flushLogs(t, app)
			for _, event := range []string{"test.request.started", "test.request.completed"} {
				entry := logWithEvent(captured(), event)
				if entry == nil {
					t.Fatalf("expected a %q log", event)
				}
				if got := entry.Data["request_id"]; got != requestID {
					t.Fatalf("%s request_id = %v, want %q", event, got, requestID)
				}
			}
		},
	}

	scenario.Test(t)
}

func TestRedactKeepsSensitiveValuesOutOfCapturedLogs(t *testing.T) {
	app := newTestApp(t)
	captured := captureLogs(app)
	request := httptest.NewRequest(http.MethodPost, "/postcard", strings.NewReader("message-body-value"))
	request.Header.Set("Authorization", "credential-value")
	sensitiveValues := []string{
		"captcha-token-value",
		"credential-value",
		"person@example.test",
		"message-body-value",
	}

	for _, value := range sensitiveValues {
		app.Logger().Info("Redaction test", "event", "test.redaction", "sensitive", Redact(value))
	}
	app.Logger().Info("Redaction test", "event", "test.redaction", "request", Redact(request))

	flushLogs(t, app)
	output := fmt.Sprint(logData(captured()))
	for _, value := range sensitiveValues {
		if strings.Contains(output, value) {
			t.Fatalf("captured log contains %q: %s", value, output)
		}
	}
	if !strings.Contains(output, Redacted) {
		t.Fatalf("captured log does not contain %q: %s", Redacted, output)
	}
}

func TestContextLoggerRetainsRequestID(t *testing.T) {
	app := newTestApp(t)
	captured := captureLogs(app)

	ContextLogger(app, WithRequestID(context.Background(), "request-123")).Info(
		"Context correlation test",
		"event", "test.context",
	)

	flushLogs(t, app)
	entry := logWithEvent(captured(), "test.context")
	if entry == nil {
		t.Fatal("expected a context correlation log")
	}
	if got := entry.Data["request_id"]; got != "request-123" {
		t.Fatalf("request_id = %v, want %q", got, "request-123")
	}
}

func newTestApp(t testing.TB) *tests.TestApp {
	t.Helper()

	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("create test app: %v", err)
	}
	t.Cleanup(app.Cleanup)
	app.Settings().Logs.MaxDays = 1

	return app
}

func captureLogs(app *tests.TestApp) func() []*core.Log {
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

func flushLogs(t testing.TB, app *tests.TestApp) {
	t.Helper()

	handler, ok := app.Logger().Handler().(*logger.BatchHandler)
	if !ok {
		t.Fatalf("expected BatchHandler, got %T", app.Logger().Handler())
	}
	if err := handler.WriteAll(context.Background()); err != nil {
		t.Fatalf("write logs: %v", err)
	}
}

func logWithEvent(logs []*core.Log, event string) *core.Log {
	for _, entry := range logs {
		if entry.Data["event"] == event {
			return entry
		}
	}

	return nil
}

func logData(logs []*core.Log) []any {
	data := make([]any, len(logs))
	for index, entry := range logs {
		data[index] = entry.Data
	}

	return data
}
