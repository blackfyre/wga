package observability

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/blackfyre/wga/internal/config"
	"github.com/getsentry/sentry-go"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
)

func TestConfigure(t *testing.T) {
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
			wantEvent: "observability.sentry.disabled",
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
			wantEvent:  "observability.sentry.disabled",
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

			monitor := configure(test.serverDSN, test.browserDSN, "production", logger, initialise)
			if monitor.enabled != test.wantEnable {
				t.Fatalf("expected enabled %t, got %t", test.wantEnable, monitor.enabled)
			}
			if got := BrowserConfig(); got.DSN != test.browserDSN || got.Environment != "production" {
				t.Fatalf("expected browser configuration %+v, got %+v", BrowserConfiguration{DSN: test.browserDSN, Environment: "production"}, got)
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
			captureException: func(err error) {
				captured = err
			},
		}
		serverError := router.NewInternalServerError("unexpected", nil)

		if got := monitor.intercept(func() error { return serverError }, func() int { return 0 }); got != serverError {
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

		if err := monitor.intercept(func() error { return router.NewBadRequestError("invalid", nil) }, func() int { return 0 }); err == nil {
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
		}, func() int { return 0 })
	})

	t.Run("captures written server responses", func(t *testing.T) {
		var captured error
		monitor := Monitor{
			enabled: true,
			captureException: func(err error) {
				captured = err
			},
		}

		if err := monitor.intercept(func() error { return nil }, func() int { return http.StatusInternalServerError }); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if captured == nil {
			t.Fatal("expected written server response to be captured")
		}
	})
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
		Name:            "non-production test route reports a Sentry flush timeout",
		Method:          http.MethodGet,
		URL:             "/sentry-test",
		ExpectedStatus:  http.StatusServiceUnavailable,
		ExpectedContent: []string{"Sentry test event did not flush before timeout"},
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
	scenario := tests.ApiScenario{
		Name:           "server error is captured without changing the response",
		Method:         http.MethodGet,
		URL:            "/sentry-server-error",
		ExpectedStatus: http.StatusInternalServerError,
		ExpectedContent: []string{
			"unexpected",
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
					return e.HTML(http.StatusInternalServerError, "unexpected")
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
