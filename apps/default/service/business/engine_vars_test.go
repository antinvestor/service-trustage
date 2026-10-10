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

//nolint:testpackage // tests exercise unexported engine helpers.
package business

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
	"github.com/antinvestor/service-trustage/dsl"
)

func TestPlanAdvance(t *testing.T) {
	t.Parallel()

	spec, err := dsl.Parse([]byte(`{
  "version": "1.2",
  "name": "plan",
  "steps": [
    {"id": "call_step", "type": "call", "call": {"action": "log.entry", "input": {}, "output_var": "result"},
     "on_success": [
       {"condition": "has(vars.result) && vars.result.value == 'ok' && has(vars.earlier)", "target": "good"},
       {"condition": "", "target": "bad"}
     ]},
    {"id": "wait_signal", "type": "signal_wait", "signal_wait": {"signal_name": "approved", "output_var": "approval"}},
    {"id": "plain", "type": "call", "call": {"action": "log.entry", "input": {}}},
    {"id": "good", "type": "call", "call": {"action": "log.entry", "input": {}}},
    {"id": "bad", "type": "call", "call": {"action": "log.entry", "input": {}}}
  ]
}`))
	require.NoError(t, err)

	tests := []struct {
		name      string
		state     string
		vars      string
		output    string
		carried   map[string]any
		wantNext  string
		wantVars  string
		wantInput string
	}{
		{
			name:      "call output_var is stored and seen by its own transition",
			state:     "call_step",
			vars:      `{"earlier":{"n":1}}`,
			output:    `{"value":"ok"}`,
			wantNext:  "good",
			wantVars:  `{"earlier":{"n":1},"result":{"value":"ok"}}`,
			wantInput: `{"room":"r-1"}`,
		},
		{
			name:      "transition without the earlier var takes the default",
			state:     "call_step",
			vars:      `{}`,
			output:    `{"value":"ok"}`,
			wantNext:  "bad",
			wantVars:  `{"result":{"value":"ok"}}`,
			wantInput: `{"room":"r-1"}`,
		},
		{
			name:      "signal_wait output_var stores the signal payload",
			state:     "wait_signal",
			vars:      `{}`,
			output:    `{"approval":{"approved":true}}`,
			wantNext:  "plain",
			wantVars:  `{"approval":{"approved":true}}`,
			wantInput: `{"room":"r-1"}`,
		},
		{
			name:      "step without output_var leaves vars untouched and carries the payload",
			state:     "plain",
			vars:      `{"result":{"value":"ok"}}`,
			output:    `{"logged":true}`,
			wantNext:  "good",
			wantVars:  "",
			wantInput: `{"room":"r-1"}`,
		},
		{
			name:      "branch vars carried back to the parent",
			state:     "plain",
			vars:      `{}`,
			output:    `{}`,
			carried:   map[string]any{"got": map[string]any{"v": 2.0}},
			wantNext:  "good",
			wantVars:  `{"got":{"v":2}}`,
			wantInput: `{"room":"r-1"}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			instance := &models.WorkflowInstance{WorkflowName: "plan", WorkflowVersion: 1, Vars: tc.vars}
			instance.ID = "inst-1"
			exec := &models.WorkflowStateExecution{State: tc.state, InputPayload: `{"room":"r-1"}`}
			exec.ID = "exec-1"

			plan, planErr := planAdvance(spec, instance, exec, json.RawMessage(tc.output), tc.carried)
			require.NoError(t, planErr)
			require.NotNil(t, plan.nextStep)
			require.Equal(t, tc.wantNext, plan.nextStep.ID)
			require.JSONEq(t, tc.wantInput, string(plan.nextInput))

			if tc.wantVars == "" {
				require.Empty(t, plan.varsJSON)
				return
			}
			require.JSONEq(t, tc.wantVars, plan.varsJSON)
		})
	}
}

func TestEvaluateForeachItems_ReadsInstanceVars(t *testing.T) {
	t.Parallel()

	items, err := evaluateForeachItems(
		&dsl.ForeachSpec{Items: "vars.rows.items"},
		json.RawMessage(`{"room":"r-1"}`),
		`{"rows":{"items":["a","b"]}}`,
	)
	require.NoError(t, err)
	require.Equal(t, []any{"a", "b"}, items)
}
