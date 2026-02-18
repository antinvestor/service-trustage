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
