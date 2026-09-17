package utils

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

var errCachedValueLoaderPanicked = errors.New("cached value loader panicked")

const (
	cacheExpirySuffix = ":meta:expires_unix_nano"
	cacheStateSuffix  = ":meta:state"
)

const cacheInstrumentationName = "github.com/blackfyre/wga/internal/utils"

// CacheName identifies one of the application caches approved for telemetry.
// Unknown and zero values deliberately disable instrumentation so storage keys
// cannot become metric dimensions.
type CacheName uint8

const (
	CacheCollectionHoldings CacheName = iota + 1
	CacheArtistAvailability
)

type cacheOutcome string

const (
	cacheOutcomeHit         cacheOutcome = "hit"
	cacheOutcomeMiss        cacheOutcome = "miss"
	cacheOutcomeShared      cacheOutcome = "shared"
	cacheOutcomeLoadFailure cacheOutcome = "load_failure"
)

type cachedValueState struct {
	mu         sync.Mutex
	generation uint64
	loading    *cachedValueLoad
}

type cachedValueLoad struct {
	generation uint64
	done       chan struct{}
	value      any
	err        error
}

func cacheExpiryKey(key string) string {
	return key + cacheExpirySuffix
}

func cacheStateKey(key string) string {
	return key + cacheStateSuffix
}

func cachedValueStateFor(app core.App, key string) *cachedValueState {
	return app.Store().GetOrSet(cacheStateKey(key), func() any {
		return &cachedValueState{}
	}).(*cachedValueState)
}

func SetCachedValue(app core.App, key string, value any, ttl time.Duration) {
	app.Store().Set(key, value)

	if ttl <= 0 {
		app.Store().Set(cacheExpiryKey(key), int64(0))
		return
	}

	app.Store().Set(cacheExpiryKey(key), time.Now().Add(ttl).UnixNano())
}

func GetCachedValue[T any](app core.App, key string) (T, bool) {
	var zero T

	if !app.Store().Has(key) {
		return zero, false
	}

	expiryRaw := app.Store().Get(cacheExpiryKey(key))
	if expiryUnixNano, ok := expiryRaw.(int64); ok && expiryUnixNano > 0 {
		if time.Now().UnixNano() > expiryUnixNano {
			return zero, false
		}
	}

	raw := app.Store().Get(key)
	typedValue, ok := raw.(T)
	if !ok {
		return zero, false
	}

	return typedValue, true
}

func GetOrLoadCachedValue[T any](app core.App, key string, ttl time.Duration, load func() (T, error)) (T, error) {
	return GetOrLoadInstrumentedCachedValue(context.Background(), app, key, ttl, 0, load)
}

// GetOrLoadInstrumentedCachedValue loads and records bounded telemetry for an
// allow-listed cache without exposing its internal storage key.
func GetOrLoadInstrumentedCachedValue[T any](ctx context.Context, app core.App, key string, ttl time.Duration, cacheName CacheName, load func() (T, error)) (T, error) {
	var zero T
	state := cachedValueStateFor(app, key)

	state.mu.Lock()
	if cached, ok := GetCachedValue[T](app, key); ok {
		state.mu.Unlock()
		recordCacheRequest(ctx, cacheName, cacheOutcomeHit)
		return cached, nil
	}
	generation := state.generation
	if loading := state.loading; loading != nil && loading.generation == generation {
		state.mu.Unlock()
		recordCacheRequest(ctx, cacheName, cacheOutcomeShared)
		<-loading.done
		if loading.err != nil {
			return zero, loading.err
		}
		if value, ok := loading.value.(T); ok {
			return value, nil
		}
		return GetOrLoadInstrumentedCachedValue(ctx, app, key, ttl, cacheName, load)
	}

	loading := &cachedValueLoad{
		generation: generation,
		done:       make(chan struct{}),
	}
	state.loading = loading
	state.mu.Unlock()

	var value T
	var err error
	loadStarted := time.Now()
	defer func() {
		panicValue := recover()
		loadOutcome := "success"
		if panicValue != nil || err != nil {
			loadOutcome = "failure"
			recordCacheRequest(ctx, cacheName, cacheOutcomeLoadFailure)
		} else {
			recordCacheRequest(ctx, cacheName, cacheOutcomeMiss)
		}
		recordCacheLoadDuration(ctx, cacheName, loadOutcome, time.Since(loadStarted))

		state.mu.Lock()
		loading.value = value
		loading.err = err
		if panicValue != nil {
			loading.err = errCachedValueLoaderPanicked
		} else if err == nil && generation == state.generation {
			SetCachedValue(app, key, value, ttl)
		}
		if state.loading == loading {
			state.loading = nil
		}
		close(loading.done)
		state.mu.Unlock()

		if panicValue != nil {
			panic(panicValue)
		}
	}()

	value, err = load()

	if err != nil {
		return zero, err
	}
	return value, nil
}

func DeleteCachedValue(app core.App, key string) {
	DeleteInstrumentedCachedValue(context.Background(), app, key, 0)
}

// DeleteInstrumentedCachedValue invalidates a cache generation and records the
// invalidation only when cacheName is allow-listed.
func DeleteInstrumentedCachedValue(ctx context.Context, app core.App, key string, cacheName CacheName) {
	state := cachedValueStateFor(app, key)
	state.mu.Lock()
	defer state.mu.Unlock()

	state.generation++
	app.Store().Remove(key)
	app.Store().Remove(cacheExpiryKey(key))
	recordCacheInvalidation(ctx, cacheName)
}

func (name CacheName) telemetryValue() (string, bool) {
	switch name {
	case CacheCollectionHoldings:
		return "collection_holdings", true
	case CacheArtistAvailability:
		return "artist_availability", true
	default:
		return "", false
	}
}

func recordCacheRequest(ctx context.Context, cacheName CacheName, outcome cacheOutcome) {
	name, ok := cacheName.telemetryValue()
	if !ok {
		return
	}
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.AddEvent("wga.cache.request", trace.WithAttributes(
			attribute.String("wga.cache.name", name),
			attribute.String("wga.cache.outcome", outcome.eventValue()),
		))
	}
	counter, err := otel.Meter(cacheInstrumentationName).Int64Counter(
		"wga.cache.requests",
		metric.WithDescription("Cache requests by bounded outcome."),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return
	}
	counter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("wga.cache.name", name),
		attribute.String("wga.cache.outcome", string(outcome)),
	))
}

func (outcome cacheOutcome) eventValue() string {
	if outcome == cacheOutcomeLoadFailure {
		return "failure"
	}
	return string(outcome)
}

func recordCacheLoadDuration(ctx context.Context, cacheName CacheName, outcome string, duration time.Duration) {
	name, ok := cacheName.telemetryValue()
	if !ok {
		return
	}
	histogram, err := otel.Meter(cacheInstrumentationName).Float64Histogram(
		"wga.cache.load.duration",
		metric.WithDescription("Cache loader duration."),
		metric.WithUnit("s"),
	)
	if err != nil {
		return
	}
	histogram.Record(ctx, duration.Seconds(), metric.WithAttributes(
		attribute.String("wga.cache.name", name),
		attribute.String("wga.cache.outcome", outcome),
	))
}

func recordCacheInvalidation(ctx context.Context, cacheName CacheName) {
	name, ok := cacheName.telemetryValue()
	if !ok {
		return
	}
	counter, err := otel.Meter(cacheInstrumentationName).Int64Counter(
		"wga.cache.invalidations",
		metric.WithDescription("Cache invalidations."),
		metric.WithUnit("{invalidation}"),
	)
	if err != nil {
		return
	}
	counter.Add(ctx, 1, metric.WithAttributes(attribute.String("wga.cache.name", name)))
}
