package cache

import (
	"github.com/pitabwire/frame/cache"
	"github.com/pitabwire/frame/cache/valkey"
	"github.com/pitabwire/frame/data"
)

// SetupCache creates a RawCache backed by Valkey when the DSN is a Redis URI,
// otherwise falls back to an in-memory cache.
func SetupCache(cacheURI string) (cache.RawCache, error) {
	cacheDSN := data.DSN(cacheURI)
	opts := []cache.Option{cache.WithDSN(cacheDSN)}

	switch {
	case cacheDSN.IsRedis():
		return valkey.New(opts...)
	default:
		return cache.NewInMemoryCache(), nil
	}
}
