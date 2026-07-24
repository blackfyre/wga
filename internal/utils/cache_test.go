package utils

import (
	"testing"
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
