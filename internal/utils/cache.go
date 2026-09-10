package utils

import (
	"sync"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

const (
	cacheExpirySuffix = ":meta:expires_unix_nano"
	cacheStateSuffix  = ":meta:state"
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
	var zero T
	state := cachedValueStateFor(app, key)

	state.mu.Lock()
	if cached, ok := GetCachedValue[T](app, key); ok {
		state.mu.Unlock()
		return cached, nil
	}
	generation := state.generation
	if loading := state.loading; loading != nil && loading.generation == generation {
		state.mu.Unlock()
		<-loading.done
		if loading.err != nil {
			return zero, loading.err
		}
		if value, ok := loading.value.(T); ok {
			return value, nil
		}
		return GetOrLoadCachedValue(app, key, ttl, load)
	}

	loading := &cachedValueLoad{
		generation: generation,
		done:       make(chan struct{}),
	}
	state.loading = loading
	state.mu.Unlock()

	value, err := load()

	state.mu.Lock()
	loading.value = value
	loading.err = err
	if err == nil && generation == state.generation {
		SetCachedValue(app, key, value, ttl)
	}
	if state.loading == loading {
		state.loading = nil
	}
	close(loading.done)
	state.mu.Unlock()

	if err != nil {
		return zero, err
	}
	return value, nil
}

func DeleteCachedValue(app core.App, key string) {
	state := cachedValueStateFor(app, key)
	state.mu.Lock()
	defer state.mu.Unlock()

	state.generation++
	app.Store().Remove(key)
	app.Store().Remove(cacheExpiryKey(key))
}
