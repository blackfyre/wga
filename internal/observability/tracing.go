package observability

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/blackfyre/wga/internal/buildinfo"
	"github.com/blackfyre/wga/internal/config"
	"github.com/pocketbase/pocketbase/core"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	tracingServiceName = "wga"
	tracingEndpoint    = "localhost:4317"
	tracingShutdownTTL = 5 * time.Second
)

// Tracer instruments development HTTP requests and flushes their queued spans.
type Tracer struct {
	enabled    bool
	tracer     trace.Tracer
	propagator propagation.TextMapPropagator
	shutdown   func(context.Context) error
}

// ConfigureTracing initialises local OTLP tracing for development only.
func ConfigureTracing(environment config.Environment, logger *slog.Logger) (Tracer, error) {
	if !environment.IsDevelopment() {
		logger.Info("OpenTelemetry tracing disabled",
			"event", "observability.otel.disabled",
			"environment", environment,
		)
		return Tracer{}, nil
	}

	exporter, err := otlptracegrpc.New(
		context.Background(),
		otlptracegrpc.WithEndpoint(tracingEndpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return Tracer{}, fmt.Errorf("initialise OpenTelemetry trace exporter: %w", err)
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(tracingServiceName),
			semconv.ServiceVersion(buildinfo.Version),
			semconv.DeploymentEnvironmentNameKey.String(string(environment)),
		)),
	)
	propagator := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})
	otel.SetTextMapPropagator(propagator)
	otel.SetTracerProvider(provider)

	logger.Info("OpenTelemetry tracing enabled",
		"event", "observability.otel.enabled",
		"endpoint", tracingEndpoint,
		"environment", environment,
	)
	return Tracer{
		enabled:    true,
		tracer:     provider.Tracer(tracingServiceName),
		propagator: propagator,
		shutdown:   provider.Shutdown,
	}, nil
}

// Register adds request tracing without changing router responses or errors.
func (t Tracer) Register(app core.App) {
	if !t.enabled {
		return
	}

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.BindFunc(func(e *core.RequestEvent) error {
			return t.intercept(e, e.Next, e.Status)
		})

		return se.Next()
	})
}

// Shutdown flushes development spans within a bounded deadline.
func (t Tracer) Shutdown() error {
	if !t.enabled || t.shutdown == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), tracingShutdownTTL)
	defer cancel()

	return t.shutdown(ctx)
}

func (t Tracer) intercept(e *core.RequestEvent, next func() error, responseStatus func() int) (err error) {
	if !t.enabled {
		return next()
	}

	ctx := t.propagator.Extract(e.Request.Context(), propagation.HeaderCarrier(e.Request.Header))
	route := requestRoute(e.Request)
	ctx, span := t.tracer.Start(ctx, e.Request.Method+" "+route, trace.WithSpanKind(trace.SpanKindServer))
	e.Request = e.Request.WithContext(ctx)

	defer func() {
		status := responseStatus()
		span.SetAttributes(
			semconv.HTTPRequestMethodKey.String(e.Request.Method),
			semconv.HTTPRouteKey.String(route),
			semconv.HTTPResponseStatusCode(status),
		)
		if err != nil || status >= http.StatusInternalServerError {
			span.SetStatus(codes.Error, http.StatusText(status))
		}
		span.End()
	}()

	return next()
}

func requestRoute(request *http.Request) string {
	_, route, found := strings.Cut(request.Pattern, " ")
	if found {
		return route
	}

	return request.Pattern
}
