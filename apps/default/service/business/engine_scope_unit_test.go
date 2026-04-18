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

//nolint:testpackage // package-local tests exercise unexported scope helpers intentionally.
package business

import (
	"encoding/json"
	"testing"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
	"github.com/antinvestor/service-trustage/dsl"
	"github.com/antinvestor/service-trustage/pkg/cryptoutil"
)

func TestBranchChildrenFromStepParallelAndForeach(t *testing.T) {
	t.Parallel()

	parallelStep := &dsl.StepSpec{
		ID:   "fanout",
		Type: dsl.StepTypeParallel,
		Parallel: &dsl.ParallelSpec{
			Steps: []*dsl.StepSpec{
				{ID: "child-a", Type: dsl.StepTypeCall},
				{ID: "child-b", Type: dsl.StepTypeCall},
			},
		},
	}

	children, err := branchChildrenFromStep(parallelStep, json.RawMessage(`{"value":1}`), "inst-1", "exec-1")
	if err != nil {
		t.Fatalf("branchChildrenFromStep(parallel) error = %v", err)
	}
	if len(children) != 2 || children[0].EntryState != "child-a" || children[1].EntryState != "child-b" {
		t.Fatalf("parallel children = %+v", children)
	}

	foreachStep := &dsl.StepSpec{
		ID:   "loop",
		Type: dsl.StepTypeForeach,
		Foreach: &dsl.ForeachSpec{
			Items:          "payload.items",
			ItemVar:        "item",
			IndexVar:       "idx",
			MaxConcurrency: 2,
			Steps: []*dsl.StepSpec{
				{ID: "process", Type: dsl.StepTypeCall},
			},
		},
	}

	children, err = branchChildrenFromStep(foreachStep, json.RawMessage(`{"items":["a","b"]}`), "inst-1", "exec-1")
	if err != nil {
		t.Fatalf("branchChildrenFromStep(foreach) error = %v", err)
	}
	if len(children) != 2 || children[0].EntryState != "process" || children[1].Index != 1 {
		t.Fatalf("foreach children = %+v", children)
	}

	var payload map[string]any
	if unmarshalErr := json.Unmarshal(children[1].Input, &payload); unmarshalErr != nil {
		t.Fatalf("unmarshal child input: %v", unmarshalErr)
	}
	if payload["item"] != "b" || payload["idx"].(float64) != 1 || payload["scope_index"].(float64) != 1 {
		t.Fatalf("foreach payload = %+v", payload)
	}
}

func TestBranchChildrenHelpersAndScopeSuccess(t *testing.T) {
	t.Parallel()

	if _, err := branchChildrenFromStep(&dsl.StepSpec{Type: dsl.StepTypeParallel}, nil, "inst", "exec"); err == nil {
		t.Fatal("expected missing parallel config error")
	}
	if _, err := branchChildrenFromStep(&dsl.StepSpec{
		Type: dsl.StepTypeForeach,
		Foreach: &dsl.ForeachSpec{
			Items: "payload.value",
			Steps: []*dsl.StepSpec{{ID: "child", Type: dsl.StepTypeCall}},
		},
	}, json.RawMessage(`{"value":"x"}`), "inst", "exec"); err == nil {
		t.Fatal("expected foreach list error")
	}

	output, shouldResume := evaluateScopeSuccess(models.WorkflowScopeRun{
		ScopeType:     string(dsl.StepTypeParallel),
		WaitAll:       false,
		TotalChildren: 2,
	}, []byte(`[null,{"ok":true}]`), 1, 0)
	if !shouldResume || len(output) == 0 {
		t.Fatalf("parallel winner output = %s resume=%v", output, shouldResume)
	}

	output, shouldResume = evaluateScopeSuccess(models.WorkflowScopeRun{
		ScopeType:     string(dsl.StepTypeParallel),
		WaitAll:       true,
		TotalChildren: 2,
	}, []byte(`[{"a":1},{"b":2}]`), 2, 0)
	if !shouldResume || string(output) != `{"branches":[{"a":1},{"b":2}]}` {
		t.Fatalf("parallel waitAll output = %s resume=%v", output, shouldResume)
	}

	output, shouldResume = evaluateScopeSuccess(models.WorkflowScopeRun{
		ScopeType:     string(dsl.StepTypeForeach),
		TotalChildren: 2,
	}, []byte(`[{"a":1},{"b":2}]`), 2, 0)
	if !shouldResume || string(output) != `{"items":[{"a":1},{"b":2}]}` {
		t.Fatalf("foreach output = %s resume=%v", output, shouldResume)
	}

	if got := emptyScopeOutput(dsl.StepTypeParallel); string(got) != `{"branches":[]}` {
		t.Fatalf("empty parallel output = %s", got)
	}
	if got := emptyScopeOutput(dsl.StepTypeForeach); string(got) != `{"items":[]}` {
		t.Fatalf("empty foreach output = %s", got)
	}
}

func TestScopeHelpers_DeterministicIDsAndSignalPayload(t *testing.T) {
	t.Parallel()

	first := scopedChildInstanceID("exec-1", "fanout", 0, "child")
	second := scopedChildInstanceID("exec-1", "fanout", 0, "child")
	third := scopedChildInstanceID("exec-1", "fanout", 1, "child")
	if first != second || first == third {
		t.Fatalf("scopedChildInstanceID() results = %q %q %q", first, second, third)
	}

	output, err := buildSignalOutputPayload("approval", json.RawMessage(`{"status":"ok"}`))
	if err != nil {
		t.Fatalf("buildSignalOutputPayload() error = %v", err)
	}
	if string(output) != `{"approval":{"status":"ok"}}` {
		t.Fatalf("wrapped signal payload = %s", output)
	}
	if output, err = buildSignalOutputPayload(
		"",
		json.RawMessage(`{"status":"ok"}`),
	); err != nil ||
		string(output) != `{"status":"ok"}` {
		t.Fatalf("passthrough signal payload = %s err=%v", output, err)
	}
	if _, err = buildSignalOutputPayload("approval", json.RawMessage(`{bad`)); err == nil {
		t.Fatal("expected invalid signal payload error")
	}

	token := "token-123"
	if got := cryptoutilHash(token); got != cryptoutil.HashToken(token) {
		t.Fatalf("cryptoutilHash() = %q", got)
	}
}

func TestExtractForeachItems(t *testing.T) {
	t.Parallel()

	children := []scopedChildDefinition{
		{Index: 0, Item: "first"},
		{Index: 1, Item: map[string]any{"id": 2}},
	}

	items := extractForeachItems(children)
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0] != "first" {
		t.Fatalf("expected first item, got %#v", items[0])
	}

	payload, ok := items[1].(map[string]any)
	if !ok || payload["id"] != 2 {
		t.Fatalf("expected second item payload, got %#v", items[1])
	}
}
