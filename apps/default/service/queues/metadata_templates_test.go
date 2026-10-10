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

//nolint:testpackage // tests need access to unexported types
package queues

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/antinvestor/service-trustage/apps/default/service/business"
	"github.com/antinvestor/service-trustage/connector"
	"github.com/antinvestor/service-trustage/dsl"
)

type capturingAdapter struct {
	inputs *[]map[string]any
}

func (a capturingAdapter) Type() string                  { return "capture.adapter" }
func (a capturingAdapter) DisplayName() string           { return "Capture" }
func (a capturingAdapter) InputSchema() json.RawMessage  { return json.RawMessage(`{"type":"object"}`) }
func (a capturingAdapter) ConfigSchema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }
func (a capturingAdapter) OutputSchema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }
func (a capturingAdapter) Validate(_ *connector.ExecuteRequest) error {
	return nil
}

func (a capturingAdapter) Execute(
	_ context.Context,
	req *connector.ExecuteRequest,
) (*connector.ExecuteResponse, *connector.ExecutionError) {
	*a.inputs = append(*a.inputs, req.Input)
	return &connector.ExecuteResponse{Output: map[string]any{}}, nil
}

func metadataCmd() business.ExecutionCommand {
	return business.ExecutionCommand{
		ExecutionID:     "exec-9",
		InstanceID:      "inst-9",
		CorrelationKey:  "p-1",
		Workflow:        "vote",
		WorkflowVersion: 3,
		State:           "call",
		Attempt:         1,
		ExecutionToken:  "token",
		InputPayload:    json.RawMessage(`{"proposal_id":"p-1","metadata":{"source":"web"}}`),
	}
}

func TestExecutionWorker_CallInputResolvesExecutionMetadata(t *testing.T) {
	spec := &dsl.WorkflowSpec{
		Version: "1.0",
		Name:    "vote",
		Steps: []*dsl.StepSpec{
			{ID: "call", Type: dsl.StepTypeCall, Call: &dsl.CallSpec{
				Action: "capture.adapter",
				Input: map[string]any{
					"request": map[string]any{
						"context": map[string]any{
							"instance_id":      "{{ metadata.instance_id }}",
							"execution_id":     "{{ metadata.execution_id }}",
							"workflow_name":    "{{ metadata.workflow_name }}",
							"workflow_version": "{{ metadata.workflow_version }}",
							"correlation_key":  "{{ metadata.correlation_key }}",
						},
						"proposal_id": "{{ payload.proposal_id }}",
						"source":      "{{ metadata.source }}",
					},
				},
			}},
		},
	}
	blob, err := json.Marshal(spec)
	require.NoError(t, err)

	var inputs []map[string]any
	registry := connector.NewRegistry()
	require.NoError(t, registry.Register(capturingAdapter{inputs: &inputs}))

	engine := &stubEngine{}
	worker := NewExecutionWorker(engine, &stubDefRepo{dsl: string(blob)}, registry).(*ExecutionWorker)

	message, err := json.Marshal(metadataCmd())
	require.NoError(t, err)
	require.NoError(t, worker.Handle(context.Background(), nil, message))

	require.Len(t, engine.commits, 1)
	require.Nil(t, engine.commits[0].Error)
	require.Len(t, inputs, 1)
	require.Equal(t, map[string]any{
		"request": map[string]any{
			"context": map[string]any{
				"instance_id":      "inst-9",
				"execution_id":     "exec-9",
				"workflow_name":    "vote",
				"workflow_version": "3",
				"correlation_key":  "p-1",
			},
			"proposal_id": "p-1",
			"source":      "web",
		},
	}, inputs[0])
}

func TestResolveSignalSend_ResolvesExecutionMetadata(t *testing.T) {
	cmd := metadataCmd()

	target, payload, err := resolveSignalSend(&dsl.SignalSendSpec{
		TargetWorkflowID: "{{ payload.proposal_id }}",
		SignalName:       "done",
		Payload:          map[string]any{"from": "{{ metadata.instance_id }}", "key": "{{ metadata.correlation_key }}"},
	}, cmd.InputPayload, executionMetadata(&cmd))
	require.NoError(t, err)
	require.Equal(t, "p-1", target)
	require.JSONEq(t, `{"from":"inst-9","key":"p-1"}`, string(payload))
}

func TestExecutionMetadata_EmptyCorrelationKeyResolvesEmpty(t *testing.T) {
	cmd := metadataCmd()
	cmd.CorrelationKey = ""
	cmd.InputPayload = json.RawMessage(`{}`)

	resolved, err := resolveStepInput(
		map[string]any{"key": "{{ metadata.correlation_key }}"},
		cmd.InputPayload,
		executionMetadata(&cmd),
	)
	require.NoError(t, err)
	require.Equal(t, map[string]any{"key": ""}, resolved)
}
