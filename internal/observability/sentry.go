package observability

import (
	"log/slog"
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
	recoverPanic     func(any)
}

// Configure initialises Sentry and stores the public browser configuration.
func Configure(settings config.Sentry, environment config.Environment, logger *slog.Logger) Monitor {
	return configure(settings.DSN(), string(environment), logger, sentry.Init)
}

func configure(dsn string, environment string, logger *slog.Logger, initialise func(sentry.ClientOptions) error) Monitor {
	browserConfiguration = BrowserConfiguration{}
	if dsn == "" {
		logger.Warn("Sentry monitoring disabled",
			"event", "observability.sentry.disabled",
			"environment", environment,
		)
		return Monitor{}
	}

	if err := initialise(sentry.ClientOptions{Dsn: dsn, Environment: environment}); err != nil {
		logger.Error("Sentry monitoring initialisation failed",
			"event", "observability.sentry.initialisation_failed",
			"environment", environment,
			"error_type", logging.ErrorType(err),
			"error", logging.Redact(err),
		)
		return Monitor{}
	}

	browserConfiguration = BrowserConfiguration{DSN: dsn, Environment: environment}
	return Monitor{
		enabled:          true,
		captureException: func(err error) { sentry.CaptureException(err) },
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
			return m.intercept(e.Next)
		})

		return se.Next()
	})
}

// Flush waits for pending Sentry events before the process exits.
func (m Monitor) Flush() {
	if m.enabled {
		sentry.Flush(flushTimeout)
	}
}

func (m Monitor) intercept(next func() error) (err error) {
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
	}

	return err
}

func shouldCapture(err error) bool {
	if err == nil {
		return false
	}

	return router.ToApiError(err).Status >= 500
}
