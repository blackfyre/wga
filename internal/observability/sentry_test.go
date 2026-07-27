package observability

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/getsentry/sentry-go"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
)

func TestConfigure(t *testing.T) {
	tests := []struct {
		name       string
		dsn        string
		initialise func(sentry.ClientOptions) error
		wantEnable bool
		wantEvent  string
	}{
		{
			name:      "disabled without DSN",
			wantEvent: "observability.sentry.disabled",
		},
		{
			name: "disabled after initialisation failure",
			dsn:  "https://public@example.ingest.sentry.io/1",
			initialise: func(sentry.ClientOptions) error {
				return errors.New("initialisation failed")
			},
			wantEvent: "observability.sentry.initialisation_failed",
		},
		{
			name: "enabled with configured DSN",
			dsn:  "https://public@example.ingest.sentry.io/1",
			initialise: func(options sentry.ClientOptions) error {
				if options.Environment != "production" {
					t.Fatalf("expected production environment, got %q", options.Environment)
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

			monitor := configure(test.dsn, "production", logger, initialise)
			if monitor.enabled != test.wantEnable {
				t.Fatalf("expected enabled %t, got %t", test.wantEnable, monitor.enabled)
			}
			if BrowserConfig().DSN != test.dsn && test.wantEnable {
				t.Fatalf("expected browser DSN %q, got %q", test.dsn, BrowserConfig().DSN)
			}
			if !test.wantEnable && BrowserConfig().DSN != "" {
				t.Fatalf("expected empty browser DSN, got %q", BrowserConfig().DSN)
			}
			if test.wantEvent != "" && !strings.Contains(logs.String(), test.wantEvent) {
				t.Fatalf("expected log event %q, got %q", test.wantEvent, logs.String())
			}
			if test.dsn != "" && strings.Contains(logs.String(), test.dsn) {
				t.Fatalf("log must not contain DSN: %q", logs.String())
			}
		})
	}
}

func TestMonitorIntercept(t *testing.T) {
	t.Run("captures server errors without changing the returned error", func(t *testing.T) {
		var captured error
		monitor := Monitor{
			enabled: true,
			captureException: func(err error) {
				captured = err
			},
		}
		serverError := router.NewInternalServerError("unexpected", nil)

		if got := monitor.intercept(func() error { return serverError }); got != serverError {
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
			captureException: func(error) {
				captured = true
			},
		}

		if err := monitor.intercept(func() error { return router.NewBadRequestError("invalid", nil) }); err == nil {
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
			recoverPanic: func(value any) {
				recovered = value
			},
		}

		defer func() {
			if got := recover(); got != "panic value" {
				t.Fatalf("expected original panic, got %v", got)
			}
			if recovered != "panic value" {
				t.Fatalf("expected captured panic, got %v", recovered)
			}
		}()
		monitor.intercept(func() error {
			panic("panic value")
		})
	})
}

func TestMonitorRegisterCapturesServerErrors(t *testing.T) {
	var captured error
	scenario := tests.ApiScenario{
		Name:           "server error is captured without changing the response",
		Method:         http.MethodGet,
		URL:            "/sentry-server-error",
		ExpectedStatus: http.StatusInternalServerError,
		ExpectedContent: []string{
			"Unexpected.",
		},
		TestAppFactory: func(t testing.TB) *tests.TestApp {
			app, err := tests.NewTestApp()
			if err != nil {
				t.Fatalf("create test app: %v", err)
			}
			monitor := Monitor{
				enabled: true,
				captureException: func(err error) {
					captured = err
				},
			}
			monitor.Register(app)
			app.OnServe().BindFunc(func(se *core.ServeEvent) error {
				se.Router.GET("/sentry-server-error", func(e *core.RequestEvent) error {
					return e.InternalServerError("unexpected", nil)
				})

				return se.Next()
			})
			return app
		},
		AfterTestFunc: func(t testing.TB, _ *tests.TestApp, _ *http.Response) {
			if captured == nil {
				t.Fatal("expected server error to be captured")
			}
		},
	}

	scenario.Test(t)
}
