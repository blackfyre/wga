package repositories

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"

	"github.com/pocketbase/pocketbase/tests"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestCollectionHoldingsInvalidationDuringLoadDoesNotRestoreStaleProjection(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("create test app: %v", err)
	}
	t.Cleanup(app.Cleanup)

	started := make(chan struct{})
	release := make(chan struct{})
	staleResult := make(chan []CollectionHolding, 1)
	go func() {
		value, loadErr := collectionHoldingsWithLoader(app, func() ([]CollectionHolding, error) {
			close(started)
			<-release
			return []CollectionHolding{{Value: "stale", Label: "Stale", Count: 1}}, nil
		})
		if loadErr != nil {
			staleResult <- nil
			return
		}
		staleResult <- value
	}()

	<-started
	InvalidateCollectionHoldings(app)
	fresh := []CollectionHolding{{Value: "fresh", Label: "Fresh", Count: 2}}
	loaded, err := collectionHoldingsWithLoader(app, func() ([]CollectionHolding, error) {
		return fresh, nil
	})
	if err != nil {
		t.Fatalf("load fresh projection: %v", err)
	}
	if len(loaded) != 1 || loaded[0] != fresh[0] {
		t.Fatalf("fresh projection = %#v", loaded)
	}

	close(release)
	if stale := <-staleResult; len(stale) != 1 || stale[0].Value != "stale" {
		t.Fatalf("original caller result = %#v, want its completed stale load", stale)
	}

	cached, err := collectionHoldingsWithLoader(app, func() ([]CollectionHolding, error) {
		t.Fatal("obsolete load replaced the fresh cached projection")
		return nil, nil
	})
	if err != nil {
		t.Fatalf("read cached projection: %v", err)
	}
	if len(cached) != 1 || cached[0] != fresh[0] {
		t.Fatalf("cached projection = %#v, want fresh projection", cached)
	}
}

func TestCollectionHoldingsTelemetryDistinguishesCacheAndLoadOutcomes(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	spanRecorder := tracetest.NewSpanRecorder()
	traceProvider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	previousMeterProvider := otel.GetMeterProvider()
	previousTraceProvider := otel.GetTracerProvider()
	otel.SetMeterProvider(meterProvider)
	otel.SetTracerProvider(traceProvider)
	t.Cleanup(func() {
		otel.SetMeterProvider(previousMeterProvider)
		otel.SetTracerProvider(previousTraceProvider)
		if err := meterProvider.Shutdown(context.Background()); err != nil {
			t.Errorf("shutdown metric provider: %v", err)
		}
		if err := traceProvider.Shutdown(context.Background()); err != nil {
			t.Errorf("shutdown trace provider: %v", err)
		}
	})

	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("create test app: %v", err)
	}
	t.Cleanup(app.Cleanup)
	ctx := context.Background()

	synctest.Test(t, func(t *testing.T) {
		started := make(chan struct{})
		release := make(chan struct{})
		results := make(chan error, 2)
		loads := 0
		for range 2 {
			go func() {
				_, loadErr := collectionHoldingsWithLoaderContext(ctx, app, func() ([]CollectionHolding, error) {
					loads++
					if loads == 1 {
						close(started)
					}
					<-release
					return []CollectionHolding{{Value: "private-record-id", Label: "Private", Count: 1}}, nil
				})
				results <- loadErr
			}()
		}
		<-started
		synctest.Wait()
		close(release)
		for range 2 {
			if loadErr := <-results; loadErr != nil {
				t.Fatalf("shared collection holdings load: %v", loadErr)
			}
		}
		if loads != 1 {
			t.Fatalf("shared collection holdings loads = %d, want 1", loads)
		}
	})

	if _, err := collectionHoldingsWithLoaderContext(ctx, app, func() ([]CollectionHolding, error) {
		t.Fatal("cache hit executed collection holdings loader")
		return nil, nil
	}); err != nil {
		t.Fatalf("collection holdings cache hit: %v", err)
	}

	InvalidateCollectionHoldings(app)
	wantErr := errors.New("secret record id and SELECT statement")
	if _, err := collectionHoldingsWithLoaderContext(ctx, app, func() ([]CollectionHolding, error) {
		return nil, wantErr
	}); !errors.Is(err, wantErr) {
		t.Fatalf("collection holdings failure = %v, want %v", err, wantErr)
	}

	spans := spanRecorder.Ended()
	if len(spans) != 2 {
		t.Fatalf("repository operation spans = %d, want one success and one failure", len(spans))
	}
	for i, span := range spans {
		if got := span.Name(); got != "wga.repository.collection_holdings.load" {
			t.Errorf("operation span %d name = %q", i, got)
		}
		for _, attr := range span.Attributes() {
			if attr.Key != "wga.operation.name" && attr.Key != "wga.operation.outcome" {
				t.Errorf("operation span %d has unexpected attribute %q", i, attr.Key)
			}
		}
	}
	if spans[0].Status().Code == codes.Error {
		t.Error("successful operation span has error status")
	}
	if spans[1].Status().Code != codes.Error || spans[1].Status().Description != "operation failed" {
		t.Errorf("failed operation status = (%s, %q), want stable error status", spans[1].Status().Code, spans[1].Status().Description)
	}

	var metrics metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &metrics); err != nil {
		t.Fatalf("collect repository metrics: %v", err)
	}
	counts := collectionCacheRequestCounts(t, metrics)
	for outcome, want := range map[string]int64{"miss": 1, "shared": 1, "hit": 1, "load_failure": 1} {
		if got := counts[outcome]; got != want {
			t.Errorf("collection cache outcome %q = %d, want %d", outcome, got, want)
		}
	}
	if got := collectionMetricPointCount(t, metrics, "wga.cache.invalidations"); got != 1 {
		t.Errorf("collection invalidation points = %d, want 1", got)
	}
	if got := collectionMetricPointCount(t, metrics, "wga.repository.operation.duration"); got != 2 {
		t.Errorf("repository operation duration count = %d, want 2", got)
	}
}

func collectionCacheRequestCounts(t *testing.T, metrics metricdata.ResourceMetrics) map[string]int64 {
	t.Helper()
	counts := make(map[string]int64)
	for _, scope := range metrics.ScopeMetrics {
		for _, candidate := range scope.Metrics {
			if candidate.Name != "wga.cache.requests" {
				continue
			}
			sum, ok := candidate.Data.(metricdata.Sum[int64])
			if !ok {
				t.Fatalf("cache request metric data type = %T", candidate.Data)
			}
			for _, point := range sum.DataPoints {
				outcome := ""
				for _, attr := range point.Attributes.ToSlice() {
					if attr.Key == "wga.cache.outcome" {
						outcome = attr.Value.AsString()
					}
					if attr.Key == "wga.cache.name" && attr.Value.AsString() != "collection_holdings" {
						t.Errorf("cache name = %q, want collection_holdings", attr.Value.AsString())
					}
				}
				counts[outcome] += point.Value
			}
		}
	}
	return counts
}

func collectionMetricPointCount(t *testing.T, metrics metricdata.ResourceMetrics, name string) int {
	t.Helper()
	for _, scope := range metrics.ScopeMetrics {
		for _, candidate := range scope.Metrics {
			if candidate.Name != name {
				continue
			}
			switch data := candidate.Data.(type) {
			case metricdata.Sum[int64]:
				return len(data.DataPoints)
			case metricdata.Histogram[float64]:
				count := 0
				for _, point := range data.DataPoints {
					count += int(point.Count)
				}
				return count
			default:
				t.Fatalf("metric %q data type = %T", name, candidate.Data)
			}
		}
	}
	t.Errorf("metric %q is absent", name)
	return 0
}
