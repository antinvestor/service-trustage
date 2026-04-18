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

package connector

import (
	"context"
	"encoding/json"
)

// Adapter is the interface all connector adapters implement.
type Adapter interface {
	// Type returns the adapter's unique type identifier (e.g., "webhook.call").
	Type() string

	// DisplayName returns a human-readable name for the adapter.
	DisplayName() string

	// InputSchema returns the JSON Schema for the adapter's input.
	InputSchema() json.RawMessage

	// ConfigSchema returns the JSON Schema for the adapter's configuration.
	ConfigSchema() json.RawMessage

	// OutputSchema returns the JSON Schema for the adapter's output.
	OutputSchema() json.RawMessage

	// Execute runs the adapter with the given request.
	Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResponse, *ExecutionError)

	// Validate checks if the input and config are valid without executing.
	Validate(req *ExecuteRequest) error
}
