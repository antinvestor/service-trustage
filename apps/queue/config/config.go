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

package config

import "github.com/pitabwire/frame/config"

// Config holds all configuration for the Generic Queue service.
type Config struct {
	config.ConfigurationDefault

	// Server.
	ServerPort string `env:"SERVER_PORT" envDefault:"8082"`

	// Valkey.
	ValkeyCacheURL string `env:"VALKEY_CACHE_URL" envDefault:"redis://localhost:6379"`

	// Stats cache TTL in seconds.
	StatsCacheTTLSeconds int `env:"STATS_CACHE_TTL_SECONDS" envDefault:"30"`

	// Rate limiting (per tenant, per minute).
	EnqueueRateLimit int `env:"ENQUEUE_RATE_LIMIT" envDefault:"200"`
}
