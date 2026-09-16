package observability

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
)

const operationInstrumentationName = "github.com/blackfyre/wga/internal/observability"

// Operation identifies a bounded, material application operation.
type Operation uint8

const (
	OperationCollectionHoldingsLoad Operation = iota + 1
	OperationArtistAvailabilityLoad
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

func (operation Operation) telemetryValue() (string, bool) {
	switch operation {
	case OperationCollectionHoldingsLoad:
		return "wga.repository.collection_holdings.load", true
	case OperationArtistAvailabilityLoad:
		return "wga.repository.artist_availability.load", true
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
