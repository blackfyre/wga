package utils

import (
	"errors"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/pocketbase/pocketbase"
)

func TestGetCachedValueReturnsTypedValue(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: "./wga_data"})

	SetCachedValue(app, "cache:test:string", "hello", 0)

	value, ok := GetCachedValue[string](app, "cache:test:string")
	if !ok {
		t.Fatalf("expected cached value to exist")
	}

	if value != "hello" {
		t.Fatalf("expected cached value 'hello', got %q", value)
	}
}

func TestGetCachedValueReturnsFalseForTypeMismatch(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: "./wga_data"})

	SetCachedValue(app, "cache:test:mismatch", "hello", 0)

	_, ok := GetCachedValue[int](app, "cache:test:mismatch")
	if ok {
		t.Fatalf("expected type mismatch to return false")
	}
}

func TestGetCachedValueRespectsExpiry(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: "./wga_data"})

	SetCachedValue(app, "cache:test:ttl", "hello", 20*time.Millisecond)
	time.Sleep(40 * time.Millisecond)

	_, ok := GetCachedValue[string](app, "cache:test:ttl")
	if ok {
		t.Fatalf("expected cached value to be expired")
	}
}

func TestDeleteCachedValueRemovesValueAndExpiry(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: "./wga_data"})
	key := "cache:test:delete"

	SetCachedValue(app, key, "hello", time.Hour)
	DeleteCachedValue(app, key)

	if app.Store().Has(key) {
		t.Fatalf("expected cached value to be removed")
	}

	if app.Store().Has(cacheExpiryKey(key)) {
		t.Fatalf("expected cache expiry to be removed")
	}
}

func TestGetOrLoadCachedValueCachesLoadedValue(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: "./wga_data"})
	key := "cache:test:load"

	value, err := GetOrLoadCachedValue(app, key, time.Hour, func() (string, error) {
		return "hello", nil
	})
	if err != nil {
		t.Fatalf("load cached value: %v", err)
	}

	if value != "hello" {
		t.Fatalf("expected loaded value 'hello', got %q", value)
	}

	cached, ok := GetCachedValue[string](app, key)
	if !ok || cached != "hello" {
		t.Fatalf("expected loaded value to be cached, got %q", cached)
	}
}

func TestGetOrLoadCachedValueDoesNotRestoreInvalidatedValue(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: "./wga_data"})
		key := "cache:test:invalidation-race"
		staleStarted := make(chan struct{})
		freshStarted := make(chan struct{})
		releaseStale := make(chan struct{})
		releaseFresh := make(chan struct{})
		type result struct {
			value string
			err   error
		}
		staleResult := make(chan result, 1)
		freshResult := make(chan result, 1)

		go func() {
			value, err := GetOrLoadCachedValue(app, key, time.Hour, func() (string, error) {
				close(staleStarted)
				<-releaseStale
				return "stale", nil
			})
			staleResult <- result{value: value, err: err}
		}()

		<-staleStarted
		DeleteCachedValue(app, key)
		go func() {
			value, err := GetOrLoadCachedValue(app, key, time.Hour, func() (string, error) {
				close(freshStarted)
				<-releaseFresh
				return "fresh", nil
			})
			freshResult <- result{value: value, err: err}
		}()

		<-freshStarted
		close(releaseFresh)
		fresh := <-freshResult
		if fresh.err != nil || fresh.value != "fresh" {
			t.Fatalf("fresh load = (%q, %v), want (fresh, nil)", fresh.value, fresh.err)
		}

		close(releaseStale)
		stale := <-staleResult
		if stale.err != nil || stale.value != "stale" {
			t.Fatalf("stale in-flight load = (%q, %v), want (stale, nil)", stale.value, stale.err)
		}

		cached, ok := GetCachedValue[string](app, key)
		if !ok || cached != "fresh" {
			t.Fatalf("cached value = (%q, %t), want (fresh, true)", cached, ok)
		}
	})
}

func TestGetOrLoadCachedValueCoalescesConcurrentMisses(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: "./wga_data"})
		const callers = 12
		started := make(chan struct{})
		release := make(chan struct{})
		results := make(chan string, callers)
		errors := make(chan error, callers)
		var loads atomic.Int32

		for range callers {
			go func() {
				value, err := GetOrLoadCachedValue(app, "cache:test:coalesced", time.Hour, func() (string, error) {
					if loads.Add(1) == 1 {
						close(started)
					}
					<-release
					return "shared", nil
				})
				results <- value
				errors <- err
			}()
		}

		<-started
		synctest.Wait()
		if got := loads.Load(); got != 1 {
			t.Fatalf("loader calls before release = %d, want 1", got)
		}
		close(release)

		for range callers {
			if err := <-errors; err != nil {
				t.Fatalf("coalesced load: %v", err)
			}
			if value := <-results; value != "shared" {
				t.Fatalf("coalesced value = %q, want shared", value)
			}
		}
		if got := loads.Load(); got != 1 {
			t.Fatalf("loader calls = %d, want 1", got)
		}
	})
}

func TestGetOrLoadCachedValueSharesButDoesNotCacheFailure(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: "./wga_data"})
		const callers = 8
		wantErr := errors.New("load failed")
		started := make(chan struct{})
		release := make(chan struct{})
		results := make(chan error, callers)
		var loads atomic.Int32

		for range callers {
			go func() {
				_, err := GetOrLoadCachedValue(app, "cache:test:error", time.Hour, func() (string, error) {
					if loads.Add(1) == 1 {
						close(started)
					}
					<-release
					return "", wantErr
				})
				results <- err
			}()
		}

		<-started
		synctest.Wait()
		if got := loads.Load(); got != 1 {
			t.Fatalf("loader calls before release = %d, want 1", got)
		}
		close(release)
		for range callers {
			if err := <-results; !errors.Is(err, wantErr) {
				t.Fatalf("coalesced error = %v, want %v", err, wantErr)
			}
		}

		value, err := GetOrLoadCachedValue(app, "cache:test:error", time.Hour, func() (string, error) {
			loads.Add(1)
			return "recovered", nil
		})
		if err != nil || value != "recovered" {
			t.Fatalf("retry = (%q, %v), want (recovered, nil)", value, err)
		}
		if got := loads.Load(); got != 2 {
			t.Fatalf("loader calls after retry = %d, want 2", got)
		}
	})
}

func TestGetOrLoadCachedValueLoadsUnrelatedKeysIndependently(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: "./wga_data"})
		startedA := make(chan struct{})
		startedB := make(chan struct{})
		release := make(chan struct{})
		results := make(chan string, 2)

		for key, started := range map[string]chan struct{}{
			"cache:test:key-a": startedA,
			"cache:test:key-b": startedB,
		} {
			go func() {
				value, err := GetOrLoadCachedValue(app, key, time.Hour, func() (string, error) {
					close(started)
					<-release
					return key, nil
				})
				if err != nil {
					t.Errorf("load %s: %v", key, err)
				}
				results <- value
			}()
		}

		<-startedA
		<-startedB
		close(release)

		seen := map[string]bool{}
		for range 2 {
			seen[<-results] = true
		}
		if !seen["cache:test:key-a"] || !seen["cache:test:key-b"] {
			t.Fatalf("loaded keys = %v, want both unrelated keys", seen)
		}
	})
}
