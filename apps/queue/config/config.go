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
