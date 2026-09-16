package observability

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"testing"

	"github.com/blackfyre/wga/internal/config"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

func TestConfigureTracingDisabledWithoutEndpoint(t *testing.T) {
	tracer, err := ConfigureTracing(config.OpenTelemetry{}, config.EnvironmentProduction, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("configure production tracing: %v", err)
	}
	if tracer.enabled {
		t.Fatal("telemetry must be disabled without a collector endpoint")
	}
}

func TestNewTracerEnabledInProduction(t *testing.T) {
	traceProvider := sdktrace.NewTracerProvider()
	meterProvider := sdkmetric.NewMeterProvider()
	shutdownCalled := false
	runtime := newTracer(traceProvider, meterProvider, func(context.Context) error {
		shutdownCalled = true
		return nil
	})
	if !runtime.enabled {
		t.Fatal("configured runtime must be enabled independently of environment")
	}
	if runtime.meterProvider != meterProvider {
		t.Fatal("configured runtime did not retain its meter provider")
	}
	if err := runtime.Shutdown(); err != nil {
		t.Fatalf("shutdown runtime: %v", err)
	}
	if !shutdownCalled {
		t.Fatal("runtime did not flush providers")
	}
}

func TestShutdownConcurrentlyAttemptsEveryProvider(t *testing.T) {
	started := make(chan string, 2)
	release := make(chan struct{})
	wantTraceErr := errors.New("trace shutdown")
	wantMetricErr := errors.New("metric shutdown")

	done := make(chan error, 1)
	go func() {
		done <- shutdownConcurrently(context.Background(),
			func(context.Context) error {
				started <- "trace"
				<-release
				return wantTraceErr
			},
			func(context.Context) error {
				started <- "metric"
				<-release
				return wantMetricErr
			},
		)
	}()

	seen := map[string]bool{<-started: true, <-started: true}
	if !seen["trace"] || !seen["metric"] {
		t.Fatalf("started shutdowns = %v, want trace and metric", seen)
	}
	close(release)
	if err := <-done; !errors.Is(err, wantTraceErr) || !errors.Is(err, wantMetricErr) {
		t.Fatalf("shutdown error = %v, want both provider errors", err)
	}
}

func TestTracerIntercept(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	t.Cleanup(func() {
		if err := provider.Shutdown(context.Background()); err != nil {
			t.Errorf("shutdown trace provider: %v", err)
		}
	})

	tracer := Tracer{
		enabled:    true,
		tracer:     provider.Tracer(telemetryServiceName),
		propagator: propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}),
	}
	event := monitorRequestEvent(t, "/artists/example?token=secret")
	event.Request.Method = http.MethodGet
	event.Request.Pattern = "GET /artists/{artist}"
	event.Request.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	requestError := errors.New("unexpected request failure")

	if err := tracer.intercept(event, func() error { return requestError }, func() int { return http.StatusInternalServerError }); err != requestError {
		t.Fatalf("expected original error, got %v", err)
	}

	spans := recorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("ended spans = %d, want 1", len(spans))
	}
	span := spans[0]
	if got, want := span.Name(), "GET /artists/{artist}"; got != want {
		t.Errorf("span name = %q, want %q", got, want)
	}
	if got, want := span.Parent().TraceID().String(), "4bf92f3577b34da6a3ce929d0e0e4736"; got != want {
		t.Errorf("parent trace ID = %q, want %q", got, want)
	}
	if got, want := span.Status().Code, codes.Error; got != want {
		t.Errorf("span status = %s, want %s", got, want)
	}
	assertSpanAttribute(t, span.Attributes(), semconv.HTTPRequestMethodKey, http.MethodGet)
	assertSpanAttribute(t, span.Attributes(), semconv.HTTPRouteKey, "/artists/{artist}")
	assertSpanAttribute(t, span.Attributes(), semconv.HTTPResponseStatusCodeKey, http.StatusInternalServerError)
}

func assertSpanAttribute(t *testing.T, attributes []attribute.KeyValue, key attribute.Key, want any) {
	t.Helper()
	for _, candidate := range attributes {
		if candidate.Key == key {
			switch expected := want.(type) {
			case string:
				if candidate.Value.AsString() != expected {
					t.Errorf("attribute %q = %q, want %q", key, candidate.Value.AsString(), expected)
				}
			case int:
				if candidate.Value.AsInt64() != int64(expected) {
					t.Errorf("attribute %q = %d, want %d", key, candidate.Value.AsInt64(), expected)
				}
			}
			return
		}
	}
	t.Errorf("attribute %q is absent", key)
}
