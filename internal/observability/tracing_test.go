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
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.43.0"
)

func TestConfigureTracingDisabledOutsideDevelopment(t *testing.T) {
	tracer, err := ConfigureTracing(config.EnvironmentProduction, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("configure production tracing: %v", err)
	}
	if tracer.enabled {
		t.Fatal("production tracing must be disabled")
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
		tracer:     provider.Tracer(tracingServiceName),
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
