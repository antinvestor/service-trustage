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

package adapters

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/antinvestor/service-trustage/connector"
)

const (
	logEntryType        = "log.entry"
	logEntryDisplayName = "Log Entry"
)

// LogEntryAdapter writes a structured audit log entry into workflow output.
// Single purpose: record a workflow event for observability. Pure computation, no HTTP calls.
type LogEntryAdapter struct{}

// NewLogEntryAdapter creates a new LogEntryAdapter.
func NewLogEntryAdapter() *LogEntryAdapter {
	return &LogEntryAdapter{}
}

func (a *LogEntryAdapter) Type() string        { return logEntryType }
func (a *LogEntryAdapter) DisplayName() string { return logEntryDisplayName }

func (a *LogEntryAdapter) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"required": ["level", "message"],
		"properties": {
			"level": {"type": "string", "enum": ["info", "warn", "error", "debug"], "description": "Log severity level"},
			"message": {"type": "string", "description": "Log message"},
			"data": {"type": "object", "description": "Structured key-value data to include"}
		}
	}`)
}

func (a *LogEntryAdapter) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{"type": "object"}`)
}

func (a *LogEntryAdapter) OutputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"logged": {"type": "boolean"},
			"timestamp": {"type": "string", "format": "date-time"},
			"level": {"type": "string"},
			"message": {"type": "string"}
		}
	}`)
}

func (a *LogEntryAdapter) Validate(req *connector.ExecuteRequest) error {
	if req.Input["level"] == nil {
		return errors.New("level is required")
	}

	level, _ := req.Input["level"].(string)
	if level != "info" && level != "warn" && level != "error" && level != "debug" {
		return errors.New("level must be one of: info, warn, error, debug")
	}

	if req.Input["message"] == nil {
		return errors.New("message is required")
	}

	return nil
}

func (a *LogEntryAdapter) Execute(
	_ context.Context,
	req *connector.ExecuteRequest,
) (*connector.ExecuteResponse, *connector.ExecutionError) {
	level, _ := req.Input["level"].(string)
	message, _ := req.Input["message"].(string)

	output := map[string]any{
		"logged":    true,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"level":     level,
		"message":   message,
	}

	if data, ok := req.Input["data"].(map[string]any); ok {
		output["data"] = data
	}

	return &connector.ExecuteResponse{Output: output}, nil
}
