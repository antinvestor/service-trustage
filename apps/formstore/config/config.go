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
