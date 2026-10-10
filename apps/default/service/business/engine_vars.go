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

package business

import (
	"encoding/json"
	"fmt"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
	"github.com/antinvestor/service-trustage/dsl"
)

// Instance vars: every step that declares output_var stores its output on the
// instance (workflow_instances.vars) as vars.<output_var>. Vars persist for
// the rest of the instance, are written in the same transaction as the
// commit that produced them, and are what later templates and expressions
// read as `vars`. A step's `payload` is the instance input carried
// unchanged from step to step.

// decodeInstanceVars parses an instance's stored vars. Empty means none.
func decodeInstanceVars(raw string) (map[string]any, error) {
	vars := map[string]any{}
	if raw == "" {
		return vars, nil
	}

	if err := json.Unmarshal([]byte(raw), &vars); err != nil {
		return nil, fmt.Errorf("decode instance vars: %w", err)
	}

	if vars == nil {
		vars = map[string]any{}
	}

	return vars, nil
}

// outputVarValue returns the output_var a completed step sets and its value.
// A call stores its whole output; a signal_wait's output is already wrapped
// as {output_var: signal payload}, so it stores the signal payload.
func outputVarValue(step *dsl.StepSpec, output json.RawMessage) (string, any, bool, error) {
	if step == nil {
		return "", nil, false, nil
	}

	var name string
	switch step.Type { //nolint:exhaustive // only call and signal_wait declare output_var
	case dsl.StepTypeCall:
		if step.Call != nil {
			name = step.Call.OutputVar
		}
	case dsl.StepTypeSignalWait:
		if step.SignalWait != nil {
			name = step.SignalWait.OutputVar
		}
	}

	if name == "" {
		return "", nil, false, nil
	}

	var value any
	if len(output) > 0 {
		if err := json.Unmarshal(output, &value); err != nil {
			return "", nil, false, fmt.Errorf("decode output of %s: %w", step.ID, err)
		}
	}

	if step.Type == dsl.StepTypeSignalWait {
		if wrapped, ok := value.(map[string]any); ok {
			value = wrapped[name]
		}
	}

	return name, value, true, nil
}

// nextInstanceVars applies a completed step's output_var (and any vars a
// finished branch scope carries back) to the instance's vars. It returns the
// resulting vars and, when they changed, their encoding to persist.
func nextInstanceVars(
	instance *models.WorkflowInstance,
	step *dsl.StepSpec,
	output json.RawMessage,
	carried map[string]any,
) (map[string]any, string, error) {
	vars, err := decodeInstanceVars(instance.Vars)
	if err != nil {
		return nil, "", err
	}

	changed := false
	for key, value := range carried {
		vars[key] = value
		changed = true
	}

	name, value, ok, err := outputVarValue(step, output)
	if err != nil {
		return nil, "", err
	}
	if ok {
		vars[name] = value
		changed = true
	}

	if !changed {
		return vars, "", nil
	}

	encoded, err := json.Marshal(vars)
	if err != nil {
		return nil, "", fmt.Errorf("encode instance vars: %w", err)
	}

	return vars, string(encoded), nil
}

// expressionScope is what transition conditions see: the step's own
// `output`, the instance `vars` (including this step's output_var), the
// step's `payload` and the execution `metadata`.
func expressionScope(
	instance *models.WorkflowInstance,
	exec *models.WorkflowStateExecution,
	input json.RawMessage,
	output json.RawMessage,
	vars map[string]any,
) (map[string]any, error) {
	var outputData any
	if len(output) > 0 {
		if err := json.Unmarshal(output, &outputData); err != nil {
			return nil, fmt.Errorf("unmarshal output for transition: %w", err)
		}
	}

	payload := map[string]any{}
	if len(input) > 0 {
		if err := json.Unmarshal(input, &payload); err != nil {
			return nil, fmt.Errorf("unmarshal input for transition: %w", err)
		}
	}

	return map[string]any{
		"output":   outputData,
		"vars":     vars,
		"payload":  payload,
		"metadata": instanceMetadata(instance, exec.ID),
	}, nil
}

func instanceMetadata(instance *models.WorkflowInstance, executionID string) map[string]any {
	return map[string]any{
		"instance_id":      instance.ID,
		"execution_id":     executionID,
		"workflow_name":    instance.WorkflowName,
		"workflow_version": instance.WorkflowVersion,
		"correlation_key":  instance.CorrelationKey,
	}
}

// branchCarriedVars collects the vars the completed children of a branch
// scope set, in branch order, for the parent to keep.
func branchCarriedVars(children []*models.WorkflowInstance) (map[string]any, error) {
	carried := map[string]any{}

	for _, child := range children {
		if child.Status != models.InstanceStatusCompleted {
			continue
		}

		vars, err := decodeInstanceVars(child.Vars)
		if err != nil {
			return nil, err
		}

		for key, value := range vars {
			carried[key] = value
		}
	}

	return carried, nil
}

// advancePlan is how a completed execution moves its instance on.
type advancePlan struct {
	nextStep  *dsl.StepSpec
	nextInput json.RawMessage
	varsJSON  string
}

// planAdvance applies the step's output_var, resolves the next step against
// the expression scope and carries the step's input forward unchanged as the
// next step's payload. Templates are not resolved here: the worker resolves
// each step's templates exactly once, when it runs the step.
func planAdvance(
	spec *dsl.WorkflowSpec,
	instance *models.WorkflowInstance,
	exec *models.WorkflowStateExecution,
	output json.RawMessage,
	carried map[string]any,
) (*advancePlan, error) {
	step := dsl.FindStep(spec, exec.State)

	vars, varsJSON, err := nextInstanceVars(instance, step, output, carried)
	if err != nil {
		return nil, err
	}

	input := executionInputPayload(exec)

	scope, err := expressionScope(instance, exec, input, output, vars)
	if err != nil {
		return nil, err
	}

	nextStep, err := resolveNextStepForInstance(spec, instance, exec.State, scope)
	if err != nil {
		return nil, fmt.Errorf("resolve next step: %w", err)
	}

	plan := &advancePlan{nextStep: nextStep, varsJSON: varsJSON}
	if nextStep != nil {
		plan.nextInput = input
	}

	return plan, nil
}

// instanceVarsJSON is the vars an execution command carries to the worker.
func instanceVarsJSON(instance *models.WorkflowInstance) json.RawMessage {
	if instance == nil || instance.Vars == "" {
		return json.RawMessage(`{}`)
	}

	return json.RawMessage(instance.Vars)
}
