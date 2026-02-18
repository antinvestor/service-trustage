package adapters

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/antinvestor/service-trustage/connector"
	"github.com/antinvestor/service-trustage/dsl"
)

const (
	dataTransformType        = "data.transform"
	dataTransformDisplayName = "Transform Data"
)

// DataTransformAdapter transforms data using CEL expressions.
// Single purpose: reshape data between workflow steps. Pure computation, no HTTP calls.
type DataTransformAdapter struct{}

// NewDataTransformAdapter creates a new DataTransformAdapter.
func NewDataTransformAdapter() *DataTransformAdapter {
	return &DataTransformAdapter{}
}

func (a *DataTransformAdapter) Type() string        { return dataTransformType }
func (a *DataTransformAdapter) DisplayName() string { return dataTransformDisplayName }

func (a *DataTransformAdapter) InputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"required": ["source"],
		"properties": {
			"source": {"type": "object", "description": "Source data to transform"},
			"expression": {"type": "string", "description": "Single CEL expression to evaluate against source"},
			"mappings": {
				"type": "object",
				"additionalProperties": {"type": "string"},
				"description": "Map of output_key -> CEL expression pairs evaluated against source"
			}
		}
	}`)
}

func (a *DataTransformAdapter) ConfigSchema() json.RawMessage {
	return json.RawMessage(`{"type": "object"}`)
}

func (a *DataTransformAdapter) OutputSchema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"result": {"description": "Result of single expression evaluation"},
			"data": {"type": "object", "description": "Results of mapping evaluations"}
		}
	}`)
}

func (a *DataTransformAdapter) Validate(req *connector.ExecuteRequest) error {
	if req.Input["source"] == nil {
		return errors.New("source is required")
	}

	hasExpr := req.Input["expression"] != nil
	hasMappings := req.Input["mappings"] != nil

	if !hasExpr && !hasMappings {
		return errors.New("at least one of expression or mappings is required")
	}

	return nil
}

func (a *DataTransformAdapter) Execute(
	_ context.Context,
	req *connector.ExecuteRequest,
) (*connector.ExecuteResponse, *connector.ExecutionError) {
	source, _ := req.Input["source"].(map[string]any)
	if source == nil {
		source = map[string]any{}
	}

	celEnv, err := dsl.NewExpressionEnv()
	if err != nil {
		return nil, &connector.ExecutionError{
			Class:   connector.ErrorFatal,
			Code:    "CEL_ENV_ERROR",
			Message: fmt.Sprintf("failed to create CEL environment: %v", err),
		}
	}

	vars := map[string]any{
		"payload":  source,
		"vars":     source,
		"metadata": map[string]any{},
		"env":      map[string]any{},
	}

	output := map[string]any{}

	// Evaluate single expression.
	if exprStr, ok := req.Input["expression"].(string); ok && exprStr != "" {
		ast, compileErr := dsl.CompileExpression(celEnv, exprStr)
		if compileErr != nil {
			return nil, &connector.ExecutionError{
				Class:   connector.ErrorFatal,
				Code:    "EXPRESSION_ERROR",
				Message: fmt.Sprintf("expression compile failed: %v", compileErr),
			}
		}

		result, evalErr := dsl.EvaluateExpression(celEnv, ast, vars)
		if evalErr != nil {
			return nil, &connector.ExecutionError{
				Class:   connector.ErrorFatal,
				Code:    "EXPRESSION_ERROR",
				Message: fmt.Sprintf("expression evaluation failed: %v", evalErr),
			}
		}

		output["result"] = result
	}

	// Evaluate mappings.
	if mappings, ok := req.Input["mappings"].(map[string]any); ok {
		data := make(map[string]any, len(mappings))

		for key, exprRaw := range mappings {
			exprStr, isStr := exprRaw.(string)
			if !isStr {
				return nil, &connector.ExecutionError{
					Class:   connector.ErrorFatal,
					Code:    "MAPPING_ERROR",
					Message: fmt.Sprintf("mapping %q: expression must be a string", key),
				}
			}

			ast, compileErr := dsl.CompileExpression(celEnv, exprStr)
			if compileErr != nil {
				return nil, &connector.ExecutionError{
					Class:   connector.ErrorFatal,
					Code:    "MAPPING_ERROR",
					Message: fmt.Sprintf("mapping %q compile failed: %v", key, compileErr),
				}
			}

			result, evalErr := dsl.EvaluateExpression(celEnv, ast, vars)
			if evalErr != nil {
				return nil, &connector.ExecutionError{
					Class:   connector.ErrorFatal,
					Code:    "MAPPING_ERROR",
					Message: fmt.Sprintf("mapping %q evaluation failed: %v", key, evalErr),
				}
			}

			data[key] = result
		}

		output["data"] = data
	}

	return &connector.ExecuteResponse{Output: output}, nil
}
