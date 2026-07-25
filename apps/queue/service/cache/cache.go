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
	"errors"
	"fmt"

	"github.com/pitabwire/frame/v2/cache"
	"github.com/pitabwire/frame/v2/cache/valkey"
	"github.com/pitabwire/frame/v2/data"
)

// SetupCache creates a Valkey-backed cache when DSN is redis://.
// When requireValkey is true, failures are fatal (no silent in-memory fallback).
func SetupCache(cacheURI string, requireValkey bool) (cache.RawCache, error) {
	cacheDSN := data.DSN(cacheURI)
	opts := []cache.Option{cache.WithDSN(cacheDSN)}

	if cacheDSN.IsRedis() {
		c, err := valkey.New(opts...)
		if err == nil {
			return c, nil
		}
		if requireValkey {
			return nil, fmt.Errorf("valkey required but connection failed: %w", err)
		}
		return cache.NewInMemoryCache(), nil
	}

	if requireValkey {
		return nil, errors.New("valkey required but cache URL is not redis")
	}

	return cache.NewInMemoryCache(), nil
}
