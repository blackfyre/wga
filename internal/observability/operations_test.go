package observability

import (
	"context"
	"errors"
	"strings"
	"testing"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestObserveOperationUsesAllowListedTelemetry(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(provider)
	t.Cleanup(func() {
		otel.SetTracerProvider(previous)
		_ = provider.Shutdown(context.Background())
	})

	secret := errors.New("private record artist-123 failed")
	_, err := ObserveOperation(t.Context(), OperationArtistIndexCount, func() (int, error) {
		return 0, secret
	})
	if !errors.Is(err, secret) {
		t.Fatalf("ObserveOperation() error = %v, want original error", err)
	}
	_, _ = ObserveOperation(t.Context(), Operation(255), func() (int, error) {
		return 1, nil
	})

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("span count = %d, want 1", len(spans))
	}
	span := spans[0]
	if span.Name != "wga.repository.artist_index.count" {
		t.Fatalf("span name = %q", span.Name)
	}
	if span.Status.Description != "operation failed" {
		t.Fatalf("status description = %q", span.Status.Description)
	}
	assertSpanExcludesText(t, span, "artist-123")
}

func TestStartWorkflowUsesAllowListedTelemetry(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	previous := otel.GetTracerProvider()
	otel.SetTracerProvider(provider)
	t.Cleanup(func() {
		otel.SetTracerProvider(previous)
		_ = provider.Shutdown(context.Background())
	})

	ctx, parent := provider.Tracer("workflow-test").Start(t.Context(), "request")
	_, finish := StartWorkflow(ctx, WorkflowDualReference)
	finish(errors.New("private pane /artists/secret failed"))
	_, finishUnknown := StartWorkflow(ctx, Workflow(255))
	finishUnknown(nil)
	parent.End()

	spans := exporter.GetSpans()
	if len(spans) != 2 {
		t.Fatalf("span count = %d, want workflow and parent", len(spans))
	}
	span := spans[0]
	if span.Name != "wga.workflow.dual.reference" {
		t.Fatalf("span name = %q", span.Name)
	}
	if span.Status.Description != "workflow failed" {
		t.Fatalf("status description = %q", span.Status.Description)
	}
	assertSpanExcludesText(t, span, "/artists/secret")
}

func assertSpanExcludesText(t *testing.T, span tracetest.SpanStub, forbidden string) {
	t.Helper()
	values := []string{span.Name, span.Status.Description}
	for _, attr := range span.Attributes {
		values = append(values, string(attr.Key), attr.Value.AsString())
	}
	if strings.Contains(strings.Join(values, " "), forbidden) {
		t.Fatalf("span contains forbidden text %q: %#v", forbidden, values)
	}
}
