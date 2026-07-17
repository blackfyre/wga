package utils

import (
	"time"

	"github.com/pocketbase/pocketbase"
)

const cacheExpirySuffix = ":meta:expires_unix_nano"

func cacheExpiryKey(key string) string {
	return key + cacheExpirySuffix
}

func SetCachedValue(app *pocketbase.PocketBase, key string, value any, ttl time.Duration) {
	app.Store().Set(key, value)

	if ttl <= 0 {
		app.Store().Set(cacheExpiryKey(key), int64(0))
		return
	}

	app.Store().Set(cacheExpiryKey(key), time.Now().Add(ttl).UnixNano())
}

func GetCachedValue[T any](app *pocketbase.PocketBase, key string) (T, bool) {
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
