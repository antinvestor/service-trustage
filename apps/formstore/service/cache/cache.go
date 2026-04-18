// Copyright 2023-2026 Ant Investor Ltd
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
