package observability

import (
	"fmt"
	"html"
	"log/slog"
	"net/http"
	"time"

	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/logging"
	"github.com/getsentry/sentry-go"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

const flushTimeout = 2 * time.Second

// BrowserConfiguration contains the public settings required by the browser SDK.
type BrowserConfiguration struct {
	DSN         string
	Environment string
}

var browserConfiguration BrowserConfiguration

// Monitor reports unexpected server failures to Sentry when enabled.
type Monitor struct {
	enabled          bool
	captureException func(error)
	captureMessage   func(string)
	flush            func() bool
	recoverPanic     func(any)
}

// Configure initialises server Sentry and stores the public browser configuration.
func Configure(settings config.Sentry, environment config.Environment, logger *slog.Logger) Monitor {
	return configure(
		settings.DSN(),
		settings.BrowserDSN(),
		string(environment),
		logger,
		sentry.Init,
	)
}

func configure(serverDSN string, browserDSN string, environment string, logger *slog.Logger, initialise func(sentry.ClientOptions) error) Monitor {
	browserConfiguration = BrowserConfiguration{DSN: browserDSN, Environment: environment}
	if serverDSN == "" {
		logger.Warn("Server Sentry monitoring disabled",
			"event", "observability.sentry.server_disabled",
			"environment", environment,
		)
		return Monitor{}
	}

	if err := initialise(sentry.ClientOptions{Dsn: serverDSN, Environment: environment}); err != nil {
		logger.Error("Sentry monitoring initialisation failed",
			"event", "observability.sentry.initialisation_failed",
			"environment", environment,
			"error_type", logging.ErrorType(err),
			"error", logging.Redact(err),
		)
		return Monitor{}
	}

	return Monitor{
		enabled:          true,
		captureException: func(err error) { sentry.CaptureException(err) },
		captureMessage:   func(message string) { sentry.CaptureMessage(message) },
		flush:            func() bool { return sentry.Flush(flushTimeout) },
		recoverPanic:     func(value any) { sentry.CurrentHub().Recover(value) },
	}
}

// BrowserConfig returns the public browser monitoring configuration.
func BrowserConfig() BrowserConfiguration {
	return browserConfiguration
}

// Register captures unexpected request failures without changing router behaviour.
func (m Monitor) Register(app core.App) {
	if !m.enabled {
		return
	}

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.BindFunc(func(e *core.RequestEvent) error {
			return m.intercept(e.Next, e.Status)
		})

		return se.Next()
	})
}

// RegisterTestRoute adds a non-production endpoint that sends intentional test events.
func (m Monitor) RegisterTestRoute(app core.App, environment config.Environment) {
	if environment == config.EnvironmentProduction {
		return
	}

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.GET("/sentry-test", func(e *core.RequestEvent) error {
			serverEnabled := m.CaptureMessage("It works!")
			browserEnabled := BrowserConfig().DSN != ""
			if !serverEnabled && !browserEnabled {
				return e.Error(http.StatusServiceUnavailable, "Sentry monitoring is disabled", nil)
			}

			if serverEnabled && !m.Flush() {
				return e.HTML(http.StatusServiceUnavailable, sentryTestPage("Server Sentry test event did not flush before timeout."))
			}

			return e.HTML(http.StatusOK, sentryTestPage("Sentry test event queued."))
		})

		return se.Next()
	})
}

// Flush waits for pending Sentry events before the process exits.
func (m Monitor) Flush() bool {
	if !m.enabled || m.flush == nil {
		return false
	}

	return m.flush()
}

// CaptureMessage sends an intentional test message when monitoring is enabled.
func (m Monitor) CaptureMessage(message string) bool {
	if !m.enabled {
		return false
	}

	m.captureMessage(message)
	return true
}

func sentryTestPage(message string) string {
	settings := BrowserConfig()
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
<meta name="sentry-dsn" content="%s">
<meta name="sentry-environment" content="%s">
</head>
<body>
<p>%s</p>
<script type="module" src="/assets/js/app.js"></script>
</body>
</html>`, html.EscapeString(settings.DSN), html.EscapeString(settings.Environment), html.EscapeString(message))
}

func (m Monitor) intercept(next func() error, responseStatus func() int) (err error) {
	if !m.enabled {
		return next()
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			m.recoverPanic(recovered)
			panic(recovered)
		}
	}()

	err = next()
	if shouldCapture(err) {
		m.captureException(err)
	} else if err == nil && responseStatus() >= 500 {
		m.captureException(fmt.Errorf("request completed with HTTP status %d", responseStatus()))
	}

	return err
}

func shouldCapture(err error) bool {
	if err == nil {
		return false
	}

	return router.ToApiError(err).Status >= 500
}
