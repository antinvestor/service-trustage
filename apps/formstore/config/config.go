package config

import "github.com/pitabwire/frame/config"

// Config holds all configuration for the Form Store service.
type Config struct {
	config.ConfigurationDefault

	// Server.
	ServerPort string `env:"SERVER_PORT" envDefault:"8081"`

	// Valkey.
	ValkeyCacheURL string `env:"VALKEY_CACHE_URL" envDefault:"redis://localhost:6379"`

	// File service URL for uploading files.
	FileServiceURL                   string `env:"FILE_SERVICE_URL"                      envDefault:"https://files.antinvestor.com"`
	FileServiceWorkloadAPITargetPath string `env:"FILE_SERVICE_WORKLOAD_API_TARGET_PATH" envDefault:""`

	// Maximum submission body size in bytes (default 10MB).
	MaxSubmissionSize int64 `env:"MAX_SUBMISSION_SIZE" envDefault:"10485760"`

	// Rate limiting (per tenant, per minute).
	SubmissionRateLimit int `env:"SUBMISSION_RATE_LIMIT" envDefault:"100"`
}
