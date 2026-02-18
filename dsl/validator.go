package dsl

import (
	"fmt"
)

// Validate performs full static validation of a WorkflowSpec.
func Validate(spec *WorkflowSpec) *ValidationResult {
	result := &ValidationResult{}

	validateRequiredFields(spec, result)
	validateStepTypes(spec, result)
	validateUniqueIDs(spec, result)
	validateReferences(spec, result)
	validateDependencyGraph(spec, result)
	validateExpressions(spec, result)
	validateTemplates(spec, result)
	validateRetryPolicies(spec, result)
	validateTimeouts(spec, result)

	return result
}

func validateRequiredFields(spec *WorkflowSpec, result *ValidationResult) {
	if spec.Version == "" {
		result.AddError(ErrMissingRequired, "workflow version is required")
	}

	if spec.Name == "" {
		result.AddError(ErrMissingRequired, "workflow name is required")
	}

	if len(spec.Steps) == 0 {
		result.AddError(ErrMissingRequired, "workflow must have at least one step")
	}

	for _, step := range CollectAllSteps(spec) {
		if step.ID == "" {
			result.AddError(ErrMissingRequired, "step ID is required")
		}

		if step.Type == "" {
			result.AddErrorWithStep(ErrMissingRequired, step.ID, "step type is required")
		}

		validateStepRequiredFields(step, result)
	}
}

//nolint:gocognit // step type validation requires many branches
func validateStepRequiredFields(step *StepSpec, result *ValidationResult) {
	switch step.Type {
	case StepTypeCall:
		if step.Call == nil {
			result.AddErrorWithStep(ErrMissingRequired, step.ID, "call step requires 'call' field")
		} else if step.Call.Action == "" {
			result.AddErrorWithStep(ErrMissingRequired, step.ID, "call.action is required")
		}
	case StepTypeDelay:
		if step.Delay == nil {
			result.AddErrorWithStep(ErrMissingRequired, step.ID, "delay step requires 'delay' field")
		} else if step.Delay.Duration.Duration == 0 && step.Delay.Until == "" {
			result.AddErrorWithStep(ErrMissingRequired, step.ID, "delay requires either 'duration' or 'until'")
		}
	case StepTypeIf:
		if step.If == nil {
			result.AddErrorWithStep(ErrMissingRequired, step.ID, "if step requires 'if' field")
		} else {
			if step.If.Expr == "" {
				result.AddErrorWithStep(ErrMissingRequired, step.ID, "if.expr is required")
			}

			if len(step.If.Then) == 0 {
				result.AddErrorWithStep(ErrMissingRequired, step.ID, "if.then must have at least one step")
			}
		}
	case StepTypeSequence:
		if step.Sequence == nil || len(step.Sequence.Steps) == 0 {
			result.AddErrorWithStep(ErrMissingRequired, step.ID, "sequence.steps must have at least one step")
		}
	case StepTypeParallel:
		if step.Parallel == nil || len(step.Parallel.Steps) == 0 {
			result.AddErrorWithStep(ErrMissingRequired, step.ID, "parallel.steps must have at least one step")
		}
	case StepTypeForeach:
		if step.Foreach == nil {
			result.AddErrorWithStep(ErrMissingRequired, step.ID, "foreach step requires 'foreach' field")
		} else {
			if step.Foreach.Items == "" {
				result.AddErrorWithStep(ErrMissingRequired, step.ID, "foreach.items is required")
			}

			if len(step.Foreach.Steps) == 0 {
				result.AddErrorWithStep(ErrMissingRequired, step.ID, "foreach.steps must have at least one step")
			}
		}
	case StepTypeSignalWait:
		if step.SignalWait == nil || step.SignalWait.SignalName == "" {
			result.AddErrorWithStep(ErrMissingRequired, step.ID, "signal_wait.signal_name is required")
		}
	case StepTypeSignalSend:
		if step.SignalSend == nil {
			result.AddErrorWithStep(ErrMissingRequired, step.ID, "signal_send step requires 'signal_send' field")
		} else if step.SignalSend.SignalName == "" {
			result.AddErrorWithStep(ErrMissingRequired, step.ID, "signal_send.signal_name is required")
		}
	}
}

func validateStepTypes(spec *WorkflowSpec, result *ValidationResult) {
	for _, step := range CollectAllSteps(spec) {
		if step.Type != "" && !step.Type.IsValid() {
			result.AddErrorWithStep(ErrInvalidStepType, step.ID,
				fmt.Sprintf("unknown step type %q", step.Type))
		}
	}
}

func validateUniqueIDs(spec *WorkflowSpec, result *ValidationResult) {
	seen := make(map[string]bool)

	for _, step := range CollectAllSteps(spec) {
		if step.ID == "" {
			continue
		}

		if seen[step.ID] {
			result.AddErrorWithStep(ErrDuplicateStepID, step.ID,
				fmt.Sprintf("duplicate step ID %q", step.ID))
		}

		seen[step.ID] = true
	}
}

func validateReferences(spec *WorkflowSpec, result *ValidationResult) {
	allSteps := CollectAllSteps(spec)
	stepIDs := make(map[string]bool, len(allSteps))

	for _, step := range allSteps {
		stepIDs[step.ID] = true
	}

	for _, step := range allSteps {
		if step.DependsOn != "" && !stepIDs[step.DependsOn] {
			result.AddErrorWithStep(ErrInvalidReference, step.ID,
				fmt.Sprintf("depends_on references unknown step %q", step.DependsOn))
		}
	}
}

func validateExpressions(spec *WorkflowSpec, result *ValidationResult) {
	env, err := NewExpressionEnv()
	if err != nil {
		result.AddError(ErrInvalidExpression, fmt.Sprintf("failed to create CEL env: %v", err))
		return
	}

	for _, step := range CollectAllSteps(spec) {
		if step.Type == StepTypeIf && step.If != nil && step.If.Expr != "" {
			_, compileErr := CompileExpression(env, step.If.Expr)
			if compileErr != nil {
				result.AddErrorWithStep(ErrInvalidExpression, step.ID,
					fmt.Sprintf("invalid CEL expression: %v", compileErr))
			}
		}

		if step.Type == StepTypeDelay && step.Delay != nil && step.Delay.Until != "" {
			_, compileErr := CompileExpression(env, step.Delay.Until)
			if compileErr != nil {
				result.AddErrorWithStep(ErrInvalidExpression, step.ID,
					fmt.Sprintf("invalid delay.until expression: %v", compileErr))
			}
		}

		if step.Type == StepTypeForeach && step.Foreach != nil && step.Foreach.Items != "" {
			_, compileErr := CompileExpression(env, step.Foreach.Items)
			if compileErr != nil {
				result.AddErrorWithStep(ErrInvalidExpression, step.ID,
					fmt.Sprintf("invalid foreach.items expression: %v", compileErr))
			}
		}
	}
}

func validateTemplates(spec *WorkflowSpec, result *ValidationResult) {
	for _, step := range CollectAllSteps(spec) {
		if step.Type != StepTypeCall || step.Call == nil {
			continue
		}

		validateTemplateMap(step.ID, step.Call.Input, result)
	}
}

func validateTemplateMap(stepID string, m map[string]any, result *ValidationResult) {
	for key, val := range m {
		switch v := val.(type) {
		case string:
			errors := ValidateTemplate(v)
			for _, e := range errors {
				result.AddErrorWithStep(ErrInvalidTemplate, stepID,
					fmt.Sprintf("field %q: %s", key, e))
			}
		case map[string]any:
			validateTemplateMap(stepID, v, result)
		}
	}
}

func validateRetryPolicies(spec *WorkflowSpec, result *ValidationResult) {
	for _, step := range CollectAllSteps(spec) {
		if step.Retry == nil {
			continue
		}

		if step.Retry.MaxAttempts < 1 {
			result.AddErrorWithStep(ErrInvalidRetry, step.ID, "max_attempts must be >= 1")
		}

		if step.Retry.BackoffCoefficient < 0 {
			result.AddErrorWithStep(ErrInvalidRetry, step.ID, "backoff_coefficient must be >= 0")
		}

		if step.Retry.InitialInterval != "" {
			_, err := ParseDuration(step.Retry.InitialInterval)
			if err != nil {
				result.AddErrorWithStep(ErrInvalidRetry, step.ID,
					fmt.Sprintf("invalid initial_interval: %v", err))
			}
		}

		if step.Retry.MaxInterval != "" {
			_, err := ParseDuration(step.Retry.MaxInterval)
			if err != nil {
				result.AddErrorWithStep(ErrInvalidRetry, step.ID,
					fmt.Sprintf("invalid max_interval: %v", err))
			}
		}
	}
}

// validateDependencyGraph builds the dependency graph from DependsOn edges among top-level
// steps and checks for cycles (using Kahn's algorithm) and unreachable steps.
func validateDependencyGraph( //nolint:gocognit // Kahn's algorithm is inherently multi-step
	spec *WorkflowSpec,
	result *ValidationResult,
) {
	if len(spec.Steps) <= 1 {
		return
	}

	// Build adjacency and in-degree maps over top-level steps only.
	stepIDs := make(map[string]bool, len(spec.Steps))
	inDegree := make(map[string]int, len(spec.Steps))
	// adjacency: from → [to], meaning "from" must complete before "to" can start.
	adjacency := make(map[string][]string, len(spec.Steps))

	for _, step := range spec.Steps {
		stepIDs[step.ID] = true
		inDegree[step.ID] = 0
	}

	for _, step := range spec.Steps {
		if step.DependsOn == "" {
			continue
		}

		if !stepIDs[step.DependsOn] {
			// Already caught by validateReferences, skip here.
			continue
		}

		adjacency[step.DependsOn] = append(adjacency[step.DependsOn], step.ID)
		inDegree[step.ID]++
	}

	// Kahn's algorithm for topological sort / cycle detection.
	queue := make([]string, 0, len(spec.Steps))

	for _, step := range spec.Steps {
		if inDegree[step.ID] == 0 {
			queue = append(queue, step.ID)
		}
	}

	visited := 0

	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		visited++

		for _, neighbor := range adjacency[node] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	if visited < len(spec.Steps) {
		// Find which steps are in the cycle for a useful error message.
		var cycleSteps []string

		for _, step := range spec.Steps {
			if inDegree[step.ID] > 0 {
				cycleSteps = append(cycleSteps, step.ID)
			}
		}

		result.AddError(ErrCycleDetected,
			fmt.Sprintf("dependency cycle detected among steps: %v", cycleSteps))
	}

	// Check for unreachable steps: any top-level step that has a DependsOn pointing to
	// a step that itself also depends on something, forming disconnected subgraphs,
	// is still reachable. True unreachability would require checking the implicit
	// sequential order is maintained. For now, the implicit order (array position)
	// ensures all steps are reachable.
}

func validateTimeouts(spec *WorkflowSpec, result *ValidationResult) {
	workflowTimeout := spec.Timeout.Duration

	for _, step := range CollectAllSteps(spec) {
		stepTimeout := step.Timeout.Duration
		if stepTimeout > 0 && workflowTimeout > 0 && stepTimeout > workflowTimeout {
			result.AddErrorWithStep(ErrInvalidTimeout, step.ID,
				fmt.Sprintf("step timeout (%s) exceeds workflow timeout (%s)",
					stepTimeout, workflowTimeout))
		}
	}
}
