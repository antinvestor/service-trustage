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
	"fmt"

	"github.com/antinvestor/service-trustage/connector"
)

const (
	formValidateType        = "form.validate"
	formValidateDisplayName = "Validate Form Data"
)

// FormValidateAdapter validates form submission fields against expected rules.
// Single purpose: check that form data has all required fields with correct types.
// Pure computation, no HTTP calls.
type FormValidateAdapter struct{}

// NewFormValidateAdapter creates a new FormValidateAdapter.
func NewFormValidateAdapter() *FormValidateAdapter {
	return &FormValidateAdapter{}
}

func (a *FormValidateAdapter) Type() string        { return formValidateType }
func (a *FormValidateAdapter) DisplayName() string { return formValidateDisplayName }

func (a *FormValidateAdapter) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"required": ["fields", "required_fields"],
		"properties": {
			"fields": {"type": "object", "description": "Form fields to validate"},
			"required_fields": {
				"type": "array",
				"items": {"type": "string"},
				"description": "List of field names that must be present and non-empty"
			},
			"field_types": {
				"type": "object",
				"additionalProperties": {"type": "string", "enum": ["string", "number", "boolean", "array", "object"]},
				"description": "Expected type for each field"
			}
		}
	}`)
}

func (a *FormValidateAdapter) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{"type": "object"}`)
}

func (a *FormValidateAdapter) OutputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"valid": {"type": "boolean"},
			"errors": {
				"type": "array",
				"items": {"type": "string"},
				"description": "List of validation error messages"
			},
			"fields": {"type": "object", "description": "The validated fields (pass-through)"}
		}
	}`)
}

func (a *FormValidateAdapter) Validate(req *connector.ExecuteRequest) error {
	if req.Input["fields"] == nil {
		return errors.New("fields is required")
	}

	if req.Input["required_fields"] == nil {
		return errors.New("required_fields is required")
	}

	return nil
}

func (a *FormValidateAdapter) Execute(
	_ context.Context,
	req *connector.ExecuteRequest,
) (*connector.ExecuteResponse, *connector.ExecutionError) {
	fields, _ := req.Input["fields"].(map[string]any)
	if fields == nil {
		fields = map[string]any{}
	}

	var validationErrors []string

	// Check required fields.
	validationErrors = append(validationErrors, checkRequiredFields(fields, req.Input)...)

	// Check field types.
	validationErrors = append(validationErrors, checkFieldTypes(fields, req.Input)...)

	isValid := len(validationErrors) == 0

	output := map[string]any{
		"valid":  isValid,
		"errors": validationErrors,
		"fields": fields,
	}

	if !isValid {
		return nil, &connector.ExecutionError{
			Class:   connector.ErrorFatal,
			Code:    "VALIDATION_FAILED",
			Message: fmt.Sprintf("form validation failed: %d errors", len(validationErrors)),
			Details: output,
		}
	}

	return &connector.ExecuteResponse{Output: output}, nil
}

// checkRequiredFields verifies all required fields are present and non-empty.
func checkRequiredFields(fields map[string]any, input map[string]any) []string {
	requiredRaw, ok := input["required_fields"].([]any)
	if !ok {
		return nil
	}

	var errs []string

	for _, reqFieldRaw := range requiredRaw {
		reqField, isStr := reqFieldRaw.(string)
		if !isStr {
			continue
		}

		val, exists := fields[reqField]
		if !exists {
			errs = append(errs, fmt.Sprintf("missing required field %q", reqField))
			continue
		}

		if isEmptyString(val) {
			errs = append(errs, fmt.Sprintf("field %q must not be empty", reqField))
		}
	}

	return errs
}

// checkFieldTypes verifies that present fields match their declared types.
func checkFieldTypes(fields map[string]any, input map[string]any) []string {
	fieldTypes, ok := input["field_types"].(map[string]any)
	if !ok {
		return nil
	}

	var errs []string

	for fieldName, expectedTypeRaw := range fieldTypes {
		expectedType, isStr := expectedTypeRaw.(string)
		if !isStr {
			continue
		}

		val, exists := fields[fieldName]
		if !exists {
			continue // Type check only applies to present fields.
		}

		if !matchesType(val, expectedType) {
			errs = append(errs, fmt.Sprintf("field %q: expected type %s", fieldName, expectedType))
		}
	}

	return errs
}

// isEmptyString returns true if the value is a string and its content is empty.
func isEmptyString(val any) bool {
	str, ok := val.(string)
	return ok && str == ""
}

// matchesType checks if a value matches the expected JSON type.
func matchesType(val any, expectedType string) bool {
	switch expectedType {
	case "string":
		_, ok := val.(string)
		return ok
	case "number":
		_, ok := val.(float64)
		return ok
	case "boolean":
		_, ok := val.(bool)
		return ok
	case "array":
		_, ok := val.([]any)
		return ok
	case "object":
		_, ok := val.(map[string]any)
		return ok
	default:
		return true
	}
}
