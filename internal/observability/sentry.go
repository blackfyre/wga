package observability

import (
	"context"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"net/http"
	"time"

	"github.com/blackfyre/wga/internal/buildinfo"
	"github.com/blackfyre/wga/internal/config"
	"github.com/blackfyre/wga/internal/logging"
	"github.com/blackfyre/wga/internal/requestfailure"
	"github.com/getsentry/sentry-go"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

const flushTimeout = 2 * time.Second

const (
	sanitisedExceptionMessage = "server failure"
	unknownRoute              = "<unmatched>"
)

// BrowserConfiguration contains the public settings required by the browser SDK.
type BrowserConfiguration struct {
	DSN         string
	Environment string
	Release     string
}

var browserConfiguration BrowserConfiguration

// Monitor reports unexpected server failures to Sentry when enabled.
type Monitor struct {
	enabled        bool
	captureFailure func(error, requestFailure)
	captureMessage func(string)
	flush          func() bool
	recoverPanic   func(any, requestFailure)
	logFailure     func(requestFailure)
}

type requestFailure struct {
	cause       error
	requestID   string
	method      string
	route       string
	status      int
	category    string
	errorType   string
	hasCause    bool
	fingerprint []string
}

// Configure initialises server Sentry and stores the public browser configuration.
func Configure(settings config.Sentry, environment config.Environment, logger *slog.Logger) Monitor {
	return configure(
		settings.DSN(),
		settings.BrowserDSN(),
		buildinfo.Version,
		string(environment),
		logger,
		sentry.Init,
	)
}

func configure(serverDSN string, browserDSN string, release string, environment string, logger *slog.Logger, initialise func(sentry.ClientOptions) error) Monitor {
	browserConfiguration = BrowserConfiguration{DSN: browserDSN, Environment: environment, Release: release}
	if serverDSN == "" {
		logger.Warn("Server Sentry monitoring disabled",
			"event", "observability.sentry.server_disabled",
			"environment", environment,
		)
		return Monitor{}
	}

	if err := initialise(sentry.ClientOptions{
		Dsn:         serverDSN,
		Environment: environment,
		Release:     release,
		BeforeSend:  sanitiseServerEvent,
	}); err != nil {
		logger.Error("Sentry monitoring initialisation failed",
			"event", "observability.sentry.initialisation_failed",
			"environment", environment,
			"error_type", logging.ErrorType(err),
			"error", logging.Redact(err),
		)
		return Monitor{}
	}

	return Monitor{
		enabled: true,
		captureFailure: func(err error, failure requestFailure) {
			sentry.WithScope(func(scope *sentry.Scope) {
				applyFailureScope(scope, failure)
				sentry.CaptureException(err)
			})
		},
		captureMessage: func(message string) { sentry.CaptureMessage(message) },
		flush:          func() bool { return sentry.Flush(flushTimeout) },
		recoverPanic: func(value any, failure requestFailure) {
			sentry.WithScope(func(scope *sentry.Scope) {
				applyFailureScope(scope, failure)
				sentry.CurrentHub().Recover(value)
			})
		},
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

	monitor := m
	monitor.logFailure = func(failure requestFailure) {
		app.Logger().Error("Unexpected request failure",
			"event", "observability.request.failed",
			"request_id", failure.requestID,
			"method", failure.method,
			"route", failure.route,
			"status", failure.status,
			"failure_category", failure.category,
			"error_type", failure.errorType,
			"has_cause", failure.hasCause,
		)
	}

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.BindFunc(func(e *core.RequestEvent) error {
			return monitor.intercept(e, e.Next, e.Status)
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
<meta name="sentry-release" content="%s">
</head>
<body>
<p>%s</p>
<script type="module" src="/assets/js/app.js"></script>
</body>
</html>`, html.EscapeString(settings.DSN), html.EscapeString(settings.Environment), html.EscapeString(settings.Release), html.EscapeString(message))
}

func (m Monitor) intercept(e *core.RequestEvent, next func() error, responseStatus func() int) (err error) {
	if !m.enabled {
		return next()
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			failure := m.failure(e, nil, responseStatus(), true)
			m.log(failure)
			m.recoverPanic(recovered, failure)
			panic(recovered)
		}
	}()

	err = next()
	if shouldCapture(err) {
		failure := m.failure(e, err, responseStatus(), false)
		m.report(failure)
	} else if err == nil && responseStatus() >= 500 {
		failure := m.failure(e, nil, responseStatus(), false)
		if !errors.Is(failure.cause, context.Canceled) && !errors.Is(failure.cause, context.DeadlineExceeded) {
			m.report(failure)
		}
	}

	return err
}

func (m Monitor) report(failure requestFailure) {
	m.log(failure)
	m.captureFailure(failure.cause, failure)
}

func (m Monitor) log(failure requestFailure) {
	if m.logFailure != nil {
		m.logFailure(failure)
	}
}

func (m Monitor) failure(e *core.RequestEvent, returned error, status int, panicked bool) requestFailure {
	if panicked || status < http.StatusInternalServerError {
		status = http.StatusInternalServerError
	}

	cause := returned
	category := "returned_error"
	if recorded, ok := requestfailure.From(e); ok {
		if recorded.Cause != nil {
			cause = recorded.Cause
		}
		if recorded.Category != "" {
			category = recorded.Category
		}
	}
	if panicked {
		category = "panic"
	}
	if cause == nil {
		category = "status_without_cause"
		cause = fmt.Errorf("request completed with HTTP status %d", status)
	}

	route := e.Request.Pattern
	if route == "" {
		route = unknownRoute
	}

	failure := requestFailure{
		cause:     cause,
		requestID: logging.RequestID(e),
		method:    e.Request.Method,
		route:     route,
		status:    status,
		category:  category,
		errorType: logging.ErrorType(cause),
		hasCause:  category != "status_without_cause",
	}
	failure.fingerprint = []string{"wga-server", failure.method, failure.route, failure.category, fmt.Sprint(failure.status)}

	return failure
}

func applyFailureScope(scope *sentry.Scope, failure requestFailure) {
	scope.SetTag("request_id", failure.requestID)
	scope.SetTag("http.method", failure.method)
	scope.SetTag("http.route", failure.route)
	scope.SetTag("http.status_code", fmt.Sprint(failure.status))
	scope.SetTag("failure.category", failure.category)
	scope.SetTag("failure.error_type", failure.errorType)
	scope.SetTag("failure.has_cause", fmt.Sprint(failure.hasCause))
	scope.SetFingerprint(failure.fingerprint)
}

func sanitiseServerEvent(event *sentry.Event, _ *sentry.EventHint) *sentry.Event {
	event.Request = nil
	event.Message = sanitisedExceptionMessage
	event.User = sentry.User{}
	for index := range event.Exception {
		event.Exception[index].Value = sanitisedExceptionMessage
	}

	return event
}

func shouldCapture(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	return router.ToApiError(err).Status >= 500
}
