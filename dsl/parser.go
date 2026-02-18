package dsl

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Parse parses a JSON DSL document into a WorkflowSpec.
func Parse(data []byte) (*WorkflowSpec, error) {
	var spec WorkflowSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("parse DSL: %w", err)
	}

	if spec.Version == "" {
		return nil, errors.New("parse DSL: missing required field 'version'")
	}

	if spec.Name == "" {
		return nil, errors.New("parse DSL: missing required field 'name'")
	}

	if len(spec.Steps) == 0 {
		return nil, errors.New("parse DSL: workflow must have at least one step")
	}

	return &spec, nil
}

// CollectAllSteps returns all steps in the workflow, including nested sub-steps,
// in depth-first order.
func CollectAllSteps(spec *WorkflowSpec) []*StepSpec {
	var result []*StepSpec

	var collect func(steps []*StepSpec)
	collect = func(steps []*StepSpec) {
		for _, step := range steps {
			result = append(result, step)
			collect(step.AllSubSteps())
		}
	}

	collect(spec.Steps)

	return result
}
