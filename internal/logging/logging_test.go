package logging

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/testutils"
	"github.com/google/uuid"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func TestRequestIDMiddlewareCorrelatesRequestLogs(t *testing.T) {
	var captured func() []*core.Log

	scenario := tests.ApiScenario{
		Name:                  "request ID is retained in the route log",
		Method:                http.MethodGet,
		URL:                   "/request-id",
		Headers:               map[string]string{RequestIDHeader: "client-supplied"},
		ExpectedStatus:        http.StatusNoContent,
		DisableTestAppCleanup: true,
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			app := testutils.NewTestApp(t)
			captured = testutils.CaptureLogs(app)
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
			if requestID == "client-supplied" {
				t.Fatal("response reused a client-supplied request ID")
			}
			if _, err := uuid.Parse(requestID); err != nil {
				t.Fatalf("response request ID %q is not a UUID: %v", requestID, err)
			}

			testutils.FlushLogs(t, app)
			for _, event := range []string{"test.request.started", "test.request.completed"} {
				entry := testutils.LogWithEvent(captured(), event)
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
	app := testutils.NewTestApp(t)
	captured := testutils.CaptureLogs(app)
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

	testutils.FlushLogs(t, app)
	output := fmt.Sprint(testutils.LogData(captured()))
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
	app := testutils.NewTestApp(t)
	captured := testutils.CaptureLogs(app)

	ContextLogger(app, WithRequestID(context.Background(), "request-123")).Info(
		"Context correlation test",
		"event", "test.context",
	)

	testutils.FlushLogs(t, app)
	entry := testutils.LogWithEvent(captured(), "test.context")
	if entry == nil {
		t.Fatal("expected a context correlation log")
	}
	if got := entry.Data["request_id"]; got != "request-123" {
		t.Fatalf("request_id = %v, want %q", got, "request-123")
	}
}
