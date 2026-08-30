package observability

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/logging"
	"github.com/blackfyre/wga/internal/requestfailure"
	"github.com/blackfyre/wga/internal/testutils"
	"github.com/getsentry/sentry-go"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
)

func TestConfigure(t *testing.T) {
	t.Cleanup(func() {
		browserConfiguration = BrowserConfiguration{}
	})

	tests := []struct {
		name       string
		serverDSN  string
		browserDSN string
		initialise func(sentry.ClientOptions) error
		wantEnable bool
		wantEvent  string
	}{
		{
			name:      "disabled without DSN",
			wantEvent: "observability.sentry.server_disabled",
		},
		{
			name:      "disabled after initialisation failure",
			serverDSN: "https://public@example.ingest.sentry.io/1",
			initialise: func(sentry.ClientOptions) error {
				return errors.New("initialisation failed")
			},
			wantEvent: "observability.sentry.initialisation_failed",
		},
		{
			name:      "enabled with configured DSN",
			serverDSN: "https://public@example.ingest.sentry.io/1",
			initialise: func(options sentry.ClientOptions) error {
				if options.Environment != "production" {
					t.Fatalf("expected production environment, got %q", options.Environment)
				}
				return nil
			},
			wantEnable: true,
		},
		{
			name:       "browser configuration is independent from server monitoring",
			browserDSN: "https://browser@example.ingest.sentry.io/2",
			wantEvent:  "observability.sentry.server_disabled",
		},
		{
			name:       "server and browser use separate DSNs",
			serverDSN:  "https://server@example.ingest.sentry.io/1",
			browserDSN: "https://browser@example.ingest.sentry.io/2",
			initialise: func(options sentry.ClientOptions) error {
				if options.Dsn != "https://server@example.ingest.sentry.io/1" {
					t.Fatalf("expected server DSN, got %q", options.Dsn)
				}
				return nil
			},
			wantEnable: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var logs bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&logs, nil))
			initialise := test.initialise
			if initialise == nil {
				initialise = func(sentry.ClientOptions) error {
					t.Fatal("Sentry should not initialise without a DSN")
					return nil
				}
			}

			monitor := configure(test.serverDSN, test.browserDSN, "test-release", "production", logger, initialise)
			if monitor.enabled != test.wantEnable {
				t.Fatalf("expected enabled %t, got %t", test.wantEnable, monitor.enabled)
			}
			if got := BrowserConfig(); got.DSN != test.browserDSN || got.Environment != "production" || got.Release != "test-release" {
				t.Fatalf("expected browser configuration %+v, got %+v", BrowserConfiguration{DSN: test.browserDSN, Environment: "production", Release: "test-release"}, got)
			}
			if test.wantEvent != "" && !strings.Contains(logs.String(), test.wantEvent) {
				t.Fatalf("expected log event %q, got %q", test.wantEvent, logs.String())
			}
			for _, dsn := range []string{test.serverDSN, test.browserDSN} {
				if dsn != "" && strings.Contains(logs.String(), dsn) {
					t.Fatalf("log must not contain DSN: %q", logs.String())
				}
			}
		})
	}
}

func TestMonitorIntercept(t *testing.T) {
	t.Run("captures server errors without changing the returned error", func(t *testing.T) {
		var captured error
		monitor := Monitor{
			enabled: true,
			captureFailure: func(err error, _ requestFailure) {
				captured = err
			},
		}
		serverError := router.NewInternalServerError("unexpected", nil)
		event := monitorRequestEvent(t, "/sentry-server-error")

		if got := monitor.intercept(event, func() error { return serverError }, func() int { return 0 }); got != serverError {
			t.Fatalf("expected original error, got %v", got)
		}
		if captured != serverError {
			t.Fatalf("expected captured error %v, got %v", serverError, captured)
		}
	})

	t.Run("does not capture client errors", func(t *testing.T) {
		captured := false
		monitor := Monitor{
			enabled: true,
			captureFailure: func(error, requestFailure) {
				captured = true
			},
		}
		event := monitorRequestEvent(t, "/sentry-client-error")

		if err := monitor.intercept(event, func() error { return router.NewBadRequestError("invalid", nil) }, func() int { return 0 }); err == nil {
			t.Fatal("expected original client error")
		}
		if captured {
			t.Fatal("client error must not be captured")
		}
	})

	t.Run("captures and rethrows panics", func(t *testing.T) {
		var recovered any
		monitor := Monitor{
			enabled: true,
			recoverPanic: func(value any, _ requestFailure) {
				recovered = value
			},
		}
		event := monitorRequestEvent(t, "/sentry-panic")

		defer func() {
			if got := recover(); got != "panic value" {
				t.Fatalf("expected original panic, got %v", got)
			}
			if recovered != "panic value" {
				t.Fatalf("expected captured panic, got %v", recovered)
			}
		}()
		_ = monitor.intercept(event, func() error {
			panic("panic value")
		}, func() int { return 0 })
	})

	t.Run("captures written server responses", func(t *testing.T) {
		var captured error
		monitor := Monitor{
			enabled: true,
			captureFailure: func(err error, _ requestFailure) {
				captured = err
			},
		}
		event := monitorRequestEvent(t, "/sentry-written-error")

		if err := monitor.intercept(event, func() error { return nil }, func() int { return http.StatusInternalServerError }); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if captured == nil {
			t.Fatal("expected written server response to be captured")
		}
	})

	t.Run("does not capture rendered cancelled or deadline failures", func(t *testing.T) {
		for _, test := range []struct {
			name  string
			cause error
		}{
			{name: "cancelled", cause: context.Canceled},
			{name: "deadline", cause: context.DeadlineExceeded},
		} {
			t.Run(test.name, func(t *testing.T) {
				captured := false
				logged := false
				monitor := Monitor{
					enabled: true,
					captureFailure: func(error, requestFailure) {
						captured = true
					},
					logFailure: func(requestFailure) {
						logged = true
					},
				}
				event := monitorRequestEvent(t, "/sentry-written-"+test.name)
				requestfailure.Record(event, requestfailure.Failure{
					Category: "server_fault",
					Cause:    test.cause,
				})

				if err := monitor.intercept(event, func() error { return nil }, func() int { return http.StatusInternalServerError }); err != nil {
					t.Fatalf("expected nil error, got %v", err)
				}
				if captured {
					t.Fatal("rendered request-end failure must not be captured")
				}
				if logged {
					t.Fatal("rendered request-end failure must not emit an unexpected failure log")
				}
			})
		}
	})
}

func TestMonitorFailureDiagnosis(t *testing.T) {
	var captured requestFailure
	monitor := Monitor{
		enabled: true,
		captureFailure: func(_ error, failure requestFailure) {
			captured = failure
		},
	}
	event := monitorRequestEvent(t, "/artists/example?token=secret")
	event.Request.Pattern = "GET /artists/{artist}"
	serverError := router.NewInternalServerError("token=secret", nil)

	if err := monitor.intercept(event, func() error { return serverError }, func() int { return 0 }); err != serverError {
		t.Fatalf("returned error = %v, want %v", err, serverError)
	}

	if got, want := captured.requestID, "request-123"; got != want {
		t.Errorf("request ID = %q, want %q", got, want)
	}
	if got, want := captured.method, http.MethodGet; got != want {
		t.Errorf("method = %q, want %q", got, want)
	}
	if got, want := captured.route, "GET /artists/{artist}"; got != want {
		t.Errorf("route = %q, want %q", got, want)
	}
	if got, want := captured.status, http.StatusInternalServerError; got != want {
		t.Errorf("status = %d, want %d", got, want)
	}
	if got, want := captured.category, "returned_error"; got != want {
		t.Errorf("category = %q, want %q", got, want)
	}
	if strings.Contains(strings.Join(captured.fingerprint, " "), "secret") {
		t.Fatalf("fingerprint contains sensitive value: %v", captured.fingerprint)
	}
}

func TestSanitiseServerEvent(t *testing.T) {
	event := &sentry.Event{
		Message: "token=secret",
		Request: &sentry.Request{URL: "/artists/example?token=secret"},
		Exception: []sentry.Exception{
			{Type: "*errors.errorString", Value: "token=secret"},
			{Type: "panic", Value: "password=secret"},
		},
	}

	sanitised := sanitiseServerEvent(event, nil)
	if sanitised.Request != nil {
		t.Fatal("server request must not be sent to Sentry")
	}
	if sanitised.Message != sanitisedExceptionMessage {
		t.Errorf("message = %q, want %q", sanitised.Message, sanitisedExceptionMessage)
	}
	for _, exception := range sanitised.Exception {
		if exception.Value != sanitisedExceptionMessage {
			t.Errorf("exception value = %q, want %q", exception.Value, sanitisedExceptionMessage)
		}
	}
}

func TestShouldCapture(t *testing.T) {
	if shouldCapture(context.Canceled) {
		t.Fatal("cancelled request must not be captured")
	}
	if shouldCapture(context.DeadlineExceeded) {
		t.Fatal("timed out request must not be captured")
	}
	if !shouldCapture(router.NewInternalServerError("unexpected", nil)) {
		t.Fatal("server error must be captured")
	}
}

func monitorRequestEvent(t testing.TB, target string) *core.RequestEvent {
	t.Helper()

	request := httptest.NewRequest(http.MethodGet, target, nil)
	request.Pattern = "GET /sentry-test"
	event := &core.RequestEvent{
		Event: router.Event{
			Request:  request,
			Response: httptest.NewRecorder(),
		},
	}
	logging.SetRequestID(event, "request-123")
	return event
}

func TestMonitorCaptureMessage(t *testing.T) {
	message := ""
	monitor := Monitor{
		enabled: true,
		captureMessage: func(value string) {
			message = value
		},
	}

	if !monitor.CaptureMessage("It works!") {
		t.Fatal("expected enabled monitor to capture test message")
	}
	if message != "It works!" {
		t.Fatalf("expected test message, got %q", message)
	}
	if (Monitor{}).CaptureMessage("It works!") {
		t.Fatal("expected disabled monitor to skip test message")
	}
}

func TestMonitorRegisterTestRoute(t *testing.T) {
	message := ""
	scenario := tests.ApiScenario{
		Name:           "non-production test route queues Sentry events",
		Method:         http.MethodGet,
		URL:            "/sentry-test",
		ExpectedStatus: http.StatusOK,
		ExpectedContent: []string{
			"Sentry test event queued.",
			`<script type="module" src="/assets/js/app.js"></script>`,
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			app, err := tests.NewTestApp()
			if err != nil {
				t.Fatalf("create test app: %v", err)
			}
			monitor := Monitor{
				enabled: true,
				captureMessage: func(value string) {
					message = value
				},
				flush: func() bool { return true },
			}
			monitor.RegisterTestRoute(app, config.EnvironmentStaging)
			return app
		},
		AfterTestFunc: func(t testing.TB, _ *tests.TestApp, _ *http.Response) {
			if message != "It works!" {
				t.Fatalf("expected test message, got %q", message)
			}
		},
	}

	scenario.Test(t)
}

func TestMonitorRegisterTestRouteWhenFlushTimesOut(t *testing.T) {
	scenario := tests.ApiScenario{
		Name:           "non-production test route reports a Sentry flush timeout",
		Method:         http.MethodGet,
		URL:            "/sentry-test",
		ExpectedStatus: http.StatusServiceUnavailable,
		ExpectedContent: []string{
			"Server Sentry test event did not flush before timeout.",
			`<meta name="sentry-dsn" content="https://browser@example.ingest.sentry.io/2">`,
			`<meta name="sentry-release" content="test-release">`,
			`<script type="module" src="/assets/js/app.js"></script>`,
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			app, err := tests.NewTestApp()
			if err != nil {
				t.Fatalf("create test app: %v", err)
			}
			monitor := Monitor{
				enabled:        true,
				captureMessage: func(string) {},
				flush:          func() bool { return false },
			}
			browserConfiguration = BrowserConfiguration{
				DSN:         "https://browser@example.ingest.sentry.io/2",
				Environment: "staging",
				Release:     "test-release",
			}
			t.Cleanup(func() {
				browserConfiguration = BrowserConfiguration{}
			})
			monitor.RegisterTestRoute(app, config.EnvironmentStaging)
			return app
		},
	}

	scenario.Test(t)
}

func TestMonitorRegisterTestRouteWhenDisabled(t *testing.T) {
	scenario := tests.ApiScenario{
		Name:            "non-production test route reports disabled monitoring",
		Method:          http.MethodGet,
		URL:             "/sentry-test",
		ExpectedStatus:  http.StatusServiceUnavailable,
		ExpectedContent: []string{"Sentry monitoring is disabled"},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			app, err := tests.NewTestApp()
			if err != nil {
				t.Fatalf("create test app: %v", err)
			}
			Monitor{}.RegisterTestRoute(app, config.EnvironmentStaging)
			return app
		},
	}

	scenario.Test(t)
}

func TestMonitorRegisterTestRouteWhenOnlyBrowserMonitoringIsEnabled(t *testing.T) {
	scenario := tests.ApiScenario{
		Name:           "non-production test route loads browser monitoring without server monitoring",
		Method:         http.MethodGet,
		URL:            "/sentry-test",
		ExpectedStatus: http.StatusOK,
		ExpectedContent: []string{
			`<meta name="sentry-dsn" content="https://browser@example.ingest.sentry.io/2">`,
			`<script type="module" src="/assets/js/app.js"></script>`,
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			app, err := tests.NewTestApp()
			if err != nil {
				t.Fatalf("create test app: %v", err)
			}
			browserConfiguration = BrowserConfiguration{
				DSN:         "https://browser@example.ingest.sentry.io/2",
				Environment: "staging",
			}
			t.Cleanup(func() {
				browserConfiguration = BrowserConfiguration{}
			})
			Monitor{}.RegisterTestRoute(app, config.EnvironmentStaging)
			return app
		},
	}

	scenario.Test(t)
}

func TestMonitorRegisterTestRouteInProduction(t *testing.T) {
	scenario := tests.ApiScenario{
		Name:            "production does not register the Sentry test route",
		Method:          http.MethodGet,
		URL:             "/sentry-test",
		ExpectedStatus:  http.StatusNotFound,
		ExpectedContent: []string{"The requested resource wasn't found."},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			app, err := tests.NewTestApp()
			if err != nil {
				t.Fatalf("create test app: %v", err)
			}
			Monitor{enabled: true}.RegisterTestRoute(app, config.EnvironmentProduction)
			return app
		},
	}

	scenario.Test(t)
}

func TestMonitorRegisterCapturesServerErrors(t *testing.T) {
	var captured error
	var diagnosis requestFailure
	var capturedLogs func() []*core.Log
	scenario := tests.ApiScenario{
		Name:           "server error is captured without changing the response",
		Method:         http.MethodGet,
		URL:            "/sentry-server-error",
		ExpectedStatus: http.StatusInternalServerError,
		ExpectedContent: []string{
			"unexpected",
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			app := testutils.NewTestApp(t)
			capturedLogs = testutils.CaptureLogs(app)
			logging.RegisterRequestIDMiddleware(app)
			monitor := Monitor{
				enabled: true,
				captureFailure: func(err error, failure requestFailure) {
					captured = err
					diagnosis = failure
				},
			}
			monitor.Register(app)
			app.OnServe().BindFunc(func(se *core.ServeEvent) error {
				se.Router.GET("/sentry-server-error", func(e *core.RequestEvent) error {
					return e.HTML(http.StatusInternalServerError, "unexpected")
				})

				return se.Next()
			})
			return app
		},
		AfterTestFunc: func(t testing.TB, app *tests.TestApp, _ *http.Response) {
			if captured == nil {
				t.Fatal("expected server error to be captured")
			}
			testutils.FlushLogs(t, app)
			logs := testutils.LogsWithEvent(capturedLogs(), "observability.request.failed")
			if len(logs) != 1 {
				t.Fatalf("failure logs = %d, want 1", len(logs))
			}
			if got, want := logs[0].Data["request_id"], diagnosis.requestID; got != want {
				t.Errorf("logged request ID = %v, want %q", got, want)
			}
			if got, want := logs[0].Data["route"], diagnosis.route; got != want {
				t.Errorf("logged route = %v, want %q", got, want)
			}
			if got, want := logs[0].Data["failure_category"], diagnosis.category; got != want {
				t.Errorf("logged category = %v, want %q", got, want)
			}
		},
	}

	scenario.Test(t)
}
