package observability

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const operationInstrumentationName = "github.com/blackfyre/wga/internal/observability"

// Operation identifies a bounded, material application operation.
type Operation uint8

const (
	OperationCollectionHoldingsLoad Operation = iota + 1
	OperationArtistAvailabilityLoad
	OperationArtistIndexCount
	OperationArtistIndexList
)

// Workflow identifies a bounded, material application workflow stage.
type Workflow uint8

const (
	WorkflowArtworkSearchResults Workflow = iota + 1
	WorkflowArtworkSearchFacets
	WorkflowDualReference
	WorkflowDualLeftPane
	WorkflowDualRightPane
	WorkflowDualRender
)

// ObserveOperation records one stable operation span and duration measurement.
// The returned error is never attached as telemetry text.
func ObserveOperation[T any](ctx context.Context, operation Operation, run func() (T, error)) (T, error) {
	name, ok := operation.telemetryValue()
	if !ok {
		return run()
	}

	ctx, span := otel.Tracer(operationInstrumentationName).Start(ctx, name)
	started := time.Now()
	value, err := run()
	outcome := "success"
	if err != nil {
		outcome = "failure"
		span.SetStatus(codes.Error, "operation failed")
	}
	span.SetAttributes(
		attribute.String("wga.operation.name", name),
		attribute.String("wga.operation.outcome", outcome),
	)
	span.End()
	recordOperationDuration(ctx, name, outcome, time.Since(started))
	return value, err
}

// StartWorkflow starts an allow-listed workflow stage and returns a bounded
// completion callback. Unknown stages are telemetry no-ops.
func StartWorkflow(ctx context.Context, workflow Workflow) (context.Context, func(error)) {
	name, ok := workflow.telemetryValue()
	if !ok || !trace.SpanFromContext(ctx).IsRecording() {
		return ctx, func(error) {}
	}

	ctx, span := otel.Tracer(operationInstrumentationName).Start(ctx, name)
	return ctx, func(err error) {
		outcome := "success"
		if err != nil {
			outcome = "failure"
			span.SetStatus(codes.Error, "workflow failed")
		}
		span.SetAttributes(
			attribute.String("wga.workflow.name", name),
			attribute.String("wga.workflow.outcome", outcome),
		)
		span.End()
	}
}

func (operation Operation) telemetryValue() (string, bool) {
	switch operation {
	case OperationCollectionHoldingsLoad:
		return "wga.repository.collection_holdings.load", true
	case OperationArtistAvailabilityLoad:
		return "wga.repository.artist_availability.load", true
	case OperationArtistIndexCount:
		return "wga.repository.artist_index.count", true
	case OperationArtistIndexList:
		return "wga.repository.artist_index.list", true
	default:
		return "", false
	}
}

func (workflow Workflow) telemetryValue() (string, bool) {
	switch workflow {
	case WorkflowArtworkSearchResults:
		return "wga.workflow.artwork_search.results", true
	case WorkflowArtworkSearchFacets:
		return "wga.workflow.artwork_search.facets", true
	case WorkflowDualReference:
		return "wga.workflow.dual.reference", true
	case WorkflowDualLeftPane:
		return "wga.workflow.dual.left_pane", true
	case WorkflowDualRightPane:
		return "wga.workflow.dual.right_pane", true
	case WorkflowDualRender:
		return "wga.workflow.dual.render", true
	default:
		return "", false
	}
}

func recordOperationDuration(ctx context.Context, name string, outcome string, duration time.Duration) {
	histogram, err := otel.Meter(operationInstrumentationName).Float64Histogram(
		"wga.repository.operation.duration",
		metric.WithDescription("Duration of allow-listed repository operations."),
		metric.WithUnit("s"),
	)
	if err != nil {
		return
	}
	histogram.Record(ctx, duration.Seconds(), metric.WithAttributes(
		attribute.String("wga.operation.name", name),
		attribute.String("wga.operation.outcome", outcome),
	))
}
