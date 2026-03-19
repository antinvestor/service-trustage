package dsl

import "fmt"

// FindStep returns the step with the given ID, including nested steps.
func FindStep(spec *WorkflowSpec, stepID string) *StepSpec {
	if spec == nil {
		return nil
	}

	for _, step := range CollectAllSteps(spec) {
		if step.ID == stepID {
			return step
		}
	}

	return nil
}

// FindNextStep returns the next step in depth-first workflow order.
// Returns nil if the step is terminal.
// This follows the implicit ordering of the full step tree, ignoring any
// conditional transitions. Use ResolveNextStep for transition-aware navigation.
func FindNextStep(spec *WorkflowSpec, currentStepID string) *StepSpec {
	if spec == nil {
		return nil
	}

	steps := CollectAllSteps(spec)
	for i, step := range steps {
		if step.ID == currentStepID {
			if i+1 < len(steps) {
				return steps[i+1]
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
	if spec == nil {
		return nil, nil //nolint:nilnil // nil spec = no next step
	}

	next, found, err := resolveNextInSteps(spec, spec.Steps, currentStepID, vars)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil //nolint:nilnil // unknown current step behaves as terminal
	}

	return next, nil
}

// ResolveNextStepInSubtree determines the next step inside the subtree rooted at rootStepID.
// When the current step exhausts the subtree, nil is returned instead of bubbling into parent siblings.
func ResolveNextStepInSubtree(
	spec *WorkflowSpec,
	rootStepID string,
	currentStepID string,
	vars map[string]any,
) (*StepSpec, error) {
	if spec == nil {
		return nil, nil //nolint:nilnil
	}

	root := FindStep(spec, rootStepID)
	if root == nil {
		return nil, fmt.Errorf("root step %q not found", rootStepID)
	}

	next, found, err := resolveNextInSteps(spec, []*StepSpec{root}, currentStepID, vars)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil //nolint:nilnil
	}

	return next, nil
}

// ResolveNextStepInContainer determines the next step inside the named container step only.
// Supported containers are sequence, parallel, and foreach.
func ResolveNextStepInContainer(
	spec *WorkflowSpec,
	containerStepID string,
	currentStepID string,
	vars map[string]any,
) (*StepSpec, error) {
	if spec == nil {
		return nil, nil //nolint:nilnil
	}

	container := FindStep(spec, containerStepID)
	if container == nil {
		return nil, fmt.Errorf("container step %q not found", containerStepID)
	}

	var steps []*StepSpec
	switch container.Type { //nolint:exhaustive
	case StepTypeSequence:
		if container.Sequence != nil {
			steps = container.Sequence.Steps
		}
	case StepTypeParallel:
		if container.Parallel != nil {
			steps = container.Parallel.Steps
		}
	case StepTypeForeach:
		if container.Foreach != nil {
			steps = container.Foreach.Steps
		}
	default:
		return nil, fmt.Errorf("step %q is not a supported container", containerStepID)
	}

	next, found, err := resolveNextInSteps(spec, steps, currentStepID, vars)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil //nolint:nilnil
	}

	return next, nil
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
	next, err := ResolveNextStep(spec, stepID, nil)
	return err == nil && next == nil
}

// InitialStep returns the first top-level step, or nil if there are no steps.
func InitialStep(spec *WorkflowSpec) *StepSpec {
	if len(spec.Steps) == 0 {
		return nil
	}

	return spec.Steps[0]
}

func resolveNextInSteps(
	spec *WorkflowSpec,
	steps []*StepSpec,
	currentStepID string,
	vars map[string]any,
) (*StepSpec, bool, error) {
	for i, step := range steps {
		if step.ID == currentStepID {
			next, err := resolveCurrentStepNext(spec, steps, i, vars)
			return next, true, err
		}

		next, found, err := resolveNextWithinStep(spec, step, currentStepID, vars)
		if err != nil {
			return nil, true, err
		}
		if !found {
			continue
		}
		if next != nil {
			return next, true, nil
		}
		if i+1 < len(steps) {
			return steps[i+1], true, nil
		}

		return nil, true, nil
	}

	return nil, false, nil
}

func resolveNextWithinStep(
	spec *WorkflowSpec,
	step *StepSpec,
	currentStepID string,
	vars map[string]any,
) (*StepSpec, bool, error) {
	switch step.Type { //nolint:exhaustive // only composite types contain substeps
	case StepTypeSequence:
		if step.Sequence == nil {
			return nil, false, nil
		}
		return resolveNextInSteps(spec, step.Sequence.Steps, currentStepID, vars)
	case StepTypeIf:
		if step.If == nil {
			return nil, false, nil
		}

		if next, found, err := resolveNextInSteps(spec, step.If.Then, currentStepID, vars); found || err != nil {
			return next, found, err
		}

		return resolveNextInSteps(spec, step.If.Else, currentStepID, vars)
	case StepTypeParallel:
		if step.Parallel == nil {
			return nil, false, nil
		}
		return resolveNextInSteps(spec, step.Parallel.Steps, currentStepID, vars)
	case StepTypeForeach:
		if step.Foreach == nil {
			return nil, false, nil
		}
		return resolveNextInSteps(spec, step.Foreach.Steps, currentStepID, vars)
	default:
		return nil, false, nil
	}
}

func resolveCurrentStepNext(
	spec *WorkflowSpec,
	siblings []*StepSpec,
	index int,
	vars map[string]any,
) (*StepSpec, error) {
	currentStep := siblings[index]

	if !currentStep.OnSuccess.IsEmpty() {
		return resolveExplicitTransition(spec, currentStep.OnSuccess, vars)
	}

	switch currentStep.Type { //nolint:exhaustive // only special implicit semantics handled here
	case StepTypeSequence:
		if currentStep.Sequence != nil && len(currentStep.Sequence.Steps) > 0 {
			return currentStep.Sequence.Steps[0], nil
		}
	case StepTypeIf:
		branch, err := resolveIfBranch(currentStep, vars)
		if err != nil {
			return nil, err
		}
		switch branch {
		case "then":
			if currentStep.If != nil && len(currentStep.If.Then) > 0 {
				return currentStep.If.Then[0], nil
			}
		case "else":
			if currentStep.If != nil && len(currentStep.If.Else) > 0 {
				return currentStep.If.Else[0], nil
			}
		}
	}

	if index+1 < len(siblings) {
		return siblings[index+1], nil
	}

	return nil, nil //nolint:nilnil // terminal within this container
}

func resolveExplicitTransition(
	spec *WorkflowSpec,
	transition TransitionTarget,
	vars map[string]any,
) (*StepSpec, error) {
	if transition.Static != "" {
		target := FindStep(spec, transition.Static)
		if target == nil {
			return nil, fmt.Errorf("transition target step %q not found", transition.Static)
		}

		return target, nil
	}

	return resolveConditionalTransition(spec, transition.Conditional, vars)
}

func resolveIfBranch(step *StepSpec, vars map[string]any) (string, error) {
	output, _ := vars["output"].(map[string]any)
	if branch, ok := output["branch"].(string); ok && (branch == "then" || branch == "else") {
		return branch, nil
	}

	return "", fmt.Errorf("if step %q did not provide resolved branch", step.ID)
}
