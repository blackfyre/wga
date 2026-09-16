package observability

import (
	"context"
	"errors"
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
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	telemetryServiceName   = "wga"
	telemetryShutdownTTL   = 5 * time.Second
	telemetryExportTimeout = 5 * time.Second
	telemetryMetricPeriod  = 15 * time.Second
	telemetrySpanQueueSize = 2048
)

// Tracer owns the optional OpenTelemetry trace and metric pipelines.
type Tracer struct {
	enabled       bool
	tracer        trace.Tracer
	meterProvider metric.MeterProvider
	propagator    propagation.TextMapPropagator
	shutdown      func(context.Context) error
}

// ConfigureTracing initialises OTLP traces and metrics when a collector endpoint
// is configured. Collector reachability is deliberately not a startup dependency.
func ConfigureTracing(settings config.OpenTelemetry, environment config.Environment, logger *slog.Logger) (Tracer, error) {
	if !settings.Enabled() {
		logger.Info("OpenTelemetry disabled",
			"event", "observability.otel.disabled",
			"environment", environment,
		)
		return Tracer{}, nil
	}

	ctx := context.Background()
	traceExporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpointURL(settings.Endpoint()),
		otlptracegrpc.WithTimeout(telemetryExportTimeout),
	)
	if err != nil {
		return Tracer{}, fmt.Errorf("initialise OpenTelemetry trace exporter: %w", err)
	}

	metricExporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpointURL(settings.Endpoint()),
		otlpmetricgrpc.WithTimeout(telemetryExportTimeout),
	)
	if err != nil {
		_ = traceExporter.Shutdown(ctx)
		return Tracer{}, fmt.Errorf("initialise OpenTelemetry metric exporter: %w", err)
	}

	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(telemetryServiceName),
		semconv.ServiceVersion(buildinfo.Version),
		semconv.DeploymentEnvironmentNameKey.String(string(environment)),
	)
	traceProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter,
			sdktrace.WithMaxQueueSize(telemetrySpanQueueSize),
			sdktrace.WithExportTimeout(telemetryExportTimeout),
		),
		sdktrace.WithResource(res),
	)
	metricReader := sdkmetric.NewPeriodicReader(metricExporter,
		sdkmetric.WithInterval(telemetryMetricPeriod),
		sdkmetric.WithTimeout(telemetryExportTimeout),
	)
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(metricReader),
		sdkmetric.WithResource(res),
	)

	runtime := newTracer(traceProvider, meterProvider, func(ctx context.Context) error {
		return shutdownConcurrently(ctx, traceProvider.Shutdown, meterProvider.Shutdown)
	})
	otel.SetTextMapPropagator(runtime.propagator)
	otel.SetTracerProvider(traceProvider)
	otel.SetMeterProvider(meterProvider)

	logger.Info("OpenTelemetry enabled",
		"event", "observability.otel.enabled",
		"endpoint", settings.Endpoint(),
		"environment", environment,
	)
	return runtime, nil
}

func shutdownConcurrently(ctx context.Context, shutdowns ...func(context.Context) error) error {
	errorsByShutdown := make(chan error, len(shutdowns))
	for _, shutdown := range shutdowns {
		go func() {
			errorsByShutdown <- shutdown(ctx)
		}()
	}

	var shutdownErrors []error
	for range shutdowns {
		shutdownErrors = append(shutdownErrors, <-errorsByShutdown)
	}
	return errors.Join(shutdownErrors...)
}

func newTracer(traceProvider trace.TracerProvider, meterProvider metric.MeterProvider, shutdown func(context.Context) error) Tracer {
	propagator := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})
	return Tracer{
		enabled:       true,
		tracer:        traceProvider.Tracer(telemetryServiceName),
		meterProvider: meterProvider,
		propagator:    propagator,
		shutdown:      shutdown,
	}
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

// Shutdown flushes configured traces and metrics within a bounded deadline.
func (t Tracer) Shutdown() error {
	if !t.enabled || t.shutdown == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), telemetryShutdownTTL)
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
