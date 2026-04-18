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
	"encoding/json"
	"fmt"
)

// ErrorClass is the exhaustive set of error classifications.
// Workers must return exactly one of these for every failure.
type ErrorClass string

const (
	ErrorRetryable          ErrorClass = "retryable"
	ErrorFatal              ErrorClass = "fatal"
	ErrorCompensatable      ErrorClass = "compensatable"
	ErrorExternalDependency ErrorClass = "external_dependency"
)

// IsValid returns true if the error class is known.
func (ec ErrorClass) IsValid() bool {
	switch ec {
	case ErrorRetryable, ErrorFatal, ErrorCompensatable, ErrorExternalDependency:
		return true
	default:
		return false
	}
}

// ExecuteRequest is the input to an adapter's Execute method.
type ExecuteRequest struct {
	Input          map[string]any    `json:"input"`
	Config         map[string]any    `json:"config,omitempty"`
	Credentials    map[string]string `json:"credentials,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	IdempotencyKey string            `json:"idempotency_key,omitempty"`
}

// ExecuteResponse is the output from a successful adapter execution.
type ExecuteResponse struct {
	Output   map[string]any  `json:"output"`
	Metadata map[string]any  `json:"metadata,omitempty"`
	RawBody  json.RawMessage `json:"raw_body,omitempty"`
}

// ExecutionError is the structured error from a failed adapter execution.
type ExecutionError struct {
	Class   ErrorClass     `json:"class"`
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

// Error implements the error interface.
func (e *ExecutionError) Error() string {
	return fmt.Sprintf("[%s] %s: %s", e.Class, e.Code, e.Message)
}
