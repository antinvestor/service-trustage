package dsl

import "fmt"

// FindStep returns the top-level step with the given ID, or nil if not found.
func FindStep(spec *WorkflowSpec, stepID string) *StepSpec {
	for _, step := range spec.Steps {
		if step.ID == stepID {
			return step
		}
	}

	return nil
}

// FindNextStep returns the top-level step that should execute after the given step ID.
// Returns nil if the step is the last one (terminal).
// This follows the implicit ordering of top-level steps, ignoring any conditional transitions.
// Use ResolveNextStep for conditional transition evaluation.
func FindNextStep(spec *WorkflowSpec, currentStepID string) *StepSpec {
	for i, step := range spec.Steps {
		if step.ID == currentStepID {
			if i+1 < len(spec.Steps) {
				return spec.Steps[i+1]
			}

			return nil
		}
	}

	return nil
}

// ResolveNextStep determines the next step by evaluating the current step's on_success transition.
// If the current step has an explicit on_success transition (static or conditional), it is used.
// Otherwise, falls back to implicit sequential ordering via FindNextStep.
// The vars map is used for CEL expression evaluation in conditional transitions.
// Returns nil (no error) when the current step is terminal.
func ResolveNextStep(spec *WorkflowSpec, currentStepID string, vars map[string]any) (*StepSpec, error) {
	currentStep := FindStep(spec, currentStepID)
	if currentStep == nil {
		return nil, nil //nolint:nilnil // nil step = terminal, not an error
	}

	// If no explicit transition, fall back to implicit sequential ordering.
	if currentStep.OnSuccess.IsEmpty() {
		return FindNextStep(spec, currentStepID), nil
	}

	// Static transition: simple step ID.
	if currentStep.OnSuccess.Static != "" {
		target := FindStep(spec, currentStep.OnSuccess.Static)
		if target == nil {
			return nil, fmt.Errorf("transition target step %q not found", currentStep.OnSuccess.Static)
		}

		return target, nil
	}

	// Conditional transitions: evaluate CEL conditions in order.
	return resolveConditionalTransition(spec, currentStep.OnSuccess.Conditional, vars)
}

// resolveConditionalTransition evaluates each conditional target in order and returns the first match.
func resolveConditionalTransition(
	spec *WorkflowSpec,
	conditions []ConditionalTarget,
	vars map[string]any,
) (*StepSpec, error) {
	env, err := NewExpressionEnv()
	if err != nil {
		return nil, fmt.Errorf("create CEL env for transition: %w", err)
	}

	for _, ct := range conditions {
		// Empty condition acts as a default/fallback (always matches).
		if ct.Condition == "" {
			target := FindStep(spec, ct.Target)
			if target == nil {
				return nil, fmt.Errorf("default transition target %q not found", ct.Target)
			}

			return target, nil
		}

		ast, compileErr := CompileExpression(env, ct.Condition)
		if compileErr != nil {
			return nil, fmt.Errorf("compile transition condition %q: %w", ct.Condition, compileErr)
		}

		matched, evalErr := EvaluateCondition(env, ast, vars)
		if evalErr != nil {
			return nil, fmt.Errorf("evaluate transition condition %q: %w", ct.Condition, evalErr)
		}

		if matched {
			target := FindStep(spec, ct.Target)
			if target == nil {
				return nil, fmt.Errorf("transition target %q not found", ct.Target)
			}

			return target, nil
		}
	}

	// No condition matched — terminal.
	return nil, nil //nolint:nilnil // no matching condition = terminal, not an error
}

// IsTerminalStep returns true if there is no next step after the given step.
func IsTerminalStep(spec *WorkflowSpec, stepID string) bool {
	return FindNextStep(spec, stepID) == nil
}

// InitialStep returns the first top-level step, or nil if there are no steps.
func InitialStep(spec *WorkflowSpec) *StepSpec {
	if len(spec.Steps) == 0 {
		return nil
	}

	return spec.Steps[0]
}
