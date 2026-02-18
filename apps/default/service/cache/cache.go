package cache

import (
	"github.com/pitabwire/frame/cache"
	"github.com/pitabwire/frame/cache/valkey"
	"github.com/pitabwire/frame/data"
)

// SetupCache creates a RawCache backed by Valkey when the DSN is a Redis URI,
// otherwise falls back to an in-memory cache. If Valkey connection fails,
// it falls back to in-memory with a warning.
func SetupCache(cacheURI string) (cache.RawCache, error) {
	cacheDSN := data.DSN(cacheURI)
	opts := []cache.Option{cache.WithDSN(cacheDSN)}

	if cacheDSN.IsRedis() {
		c, err := valkey.New(opts...)
		if err == nil {
			return c, nil
		}
		// Valkey unavailable — fall back to in-memory cache.
		return cache.NewInMemoryCache(), nil
	}

	return cache.NewInMemoryCache(), nil
}
