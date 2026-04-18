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

//nolint:testpackage // package-local DSL tests exercise unexported navigation helpers intentionally.
package dsl

import "testing"

func TestFindStepAndNextStepUseNestedWorkflowOrder(t *testing.T) {
	spec := &WorkflowSpec{
		Version: "1.0",
		Name:    "nested",
		Steps: []*StepSpec{
			{
				ID:   "sequence_root",
				Type: StepTypeSequence,
				Sequence: &SequenceSpec{
					Steps: []*StepSpec{
						{ID: "child_a", Type: StepTypeCall, Call: &CallSpec{Action: "log.entry"}},
						{ID: "child_b", Type: StepTypeCall, Call: &CallSpec{Action: "log.entry"}},
					},
				},
			},
			{ID: "after", Type: StepTypeCall, Call: &CallSpec{Action: "log.entry"}},
		},
	}

	if got := FindStep(spec, "child_b"); got == nil || got.ID != "child_b" {
		t.Fatalf("expected nested child_b step, got %#v", got)
	}

	if got := FindNextStep(spec, "sequence_root"); got == nil || got.ID != "child_a" {
		t.Fatalf("expected first nested child after sequence root, got %#v", got)
	}

	if got := FindNextStep(spec, "child_b"); got == nil || got.ID != "after" {
		t.Fatalf("expected top-level after step after nested child, got %#v", got)
	}
}

func TestResolveNextStepFollowsNestedTransitionTargets(t *testing.T) {
	spec := &WorkflowSpec{
		Version: "1.0",
		Name:    "transitioned",
		Steps: []*StepSpec{
			{
				ID:   "sequence_root",
				Type: StepTypeSequence,
				Sequence: &SequenceSpec{
					Steps: []*StepSpec{
						{
							ID:        "child_a",
							Type:      StepTypeCall,
							Call:      &CallSpec{Action: "log.entry"},
							OnSuccess: TransitionTarget{Static: "nested_target"},
						},
						{ID: "child_b", Type: StepTypeCall, Call: &CallSpec{Action: "log.entry"}},
					},
				},
			},
			{ID: "nested_target", Type: StepTypeCall, Call: &CallSpec{Action: "log.entry"}},
		},
	}

	next, err := ResolveNextStep(spec, "child_a", map[string]any{"output": map[string]any{"ok": true}})
	if err != nil {
		t.Fatalf("resolve next step: %v", err)
	}

	if next == nil || next.ID != "nested_target" {
		t.Fatalf("expected explicit nested target, got %#v", next)
	}
}

func TestResolveNextStepConditionalTransitionAndTerminalChecks(t *testing.T) {
	spec := &WorkflowSpec{
		Version: "1.0",
		Name:    "conditional",
		Steps: []*StepSpec{
			{
				ID:   "decision",
				Type: StepTypeCall,
				Call: &CallSpec{Action: "log.entry"},
				OnSuccess: TransitionTarget{
					Conditional: []ConditionalTarget{
						{Condition: "output.ok == true", Target: "approved"},
						{Condition: "", Target: "rejected"},
					},
				},
			},
			{ID: "approved", Type: StepTypeCall, Call: &CallSpec{Action: "log.entry"}},
			{ID: "rejected", Type: StepTypeCall, Call: &CallSpec{Action: "log.entry"}},
		},
	}

	next, err := ResolveNextStep(spec, "decision", map[string]any{
		"output": map[string]any{"ok": true},
	})
	if err != nil {
		t.Fatalf("resolve next step: %v", err)
	}
	if next == nil || next.ID != "approved" {
		t.Fatalf("expected approved branch, got %#v", next)
	}

	if IsTerminalStep(spec, "approved") {
		t.Fatal("approved should not be terminal because rejected follows in depth-first order")
	}
}

func TestResolveNextStepIfBranchesAndMissingBranchError(t *testing.T) {
	spec := &WorkflowSpec{
		Version: "1.0",
		Name:    "if-branches",
		Steps: []*StepSpec{
			{
				ID:   "check",
				Type: StepTypeIf,
				If: &IfSpec{
					Expr: "payload.ok",
					Then: []*StepSpec{{ID: "then_step", Type: StepTypeCall, Call: &CallSpec{Action: "log.entry"}}},
					Else: []*StepSpec{{ID: "else_step", Type: StepTypeCall, Call: &CallSpec{Action: "log.entry"}}},
				},
			},
			{ID: "after", Type: StepTypeCall, Call: &CallSpec{Action: "log.entry"}},
		},
	}

	next, err := ResolveNextStep(spec, "check", map[string]any{"output": map[string]any{"branch": "else"}})
	if err != nil {
		t.Fatalf("resolve if branch: %v", err)
	}
	if next == nil || next.ID != "else_step" {
		t.Fatalf("expected else_step, got %#v", next)
	}

	if _, resolveErr := ResolveNextStep(spec, "check", map[string]any{"output": map[string]any{}}); resolveErr == nil {
		t.Fatal("expected missing branch error")
	}
}

func TestInitialStepHandlesEmptyWorkflow(t *testing.T) {
	if InitialStep(&WorkflowSpec{}) != nil {
		t.Fatal("expected nil initial step for empty workflow")
	}
}

func TestResolveNextStepInSubtreeAndContainer(t *testing.T) {
	spec := &WorkflowSpec{
		Version: "1.0",
		Name:    "scoped",
		Steps: []*StepSpec{
			{
				ID:   "outer",
				Type: StepTypeSequence,
				Sequence: &SequenceSpec{
					Steps: []*StepSpec{
						{
							ID:   "branch",
							Type: StepTypeParallel,
							Parallel: &ParallelSpec{
								Steps: []*StepSpec{
									{ID: "left", Type: StepTypeCall, Call: &CallSpec{Action: "log.entry"}},
									{ID: "right", Type: StepTypeCall, Call: &CallSpec{Action: "log.entry"}},
								},
							},
						},
						{ID: "after_branch", Type: StepTypeCall, Call: &CallSpec{Action: "log.entry"}},
					},
				},
			},
			{ID: "tail", Type: StepTypeCall, Call: &CallSpec{Action: "log.entry"}},
		},
	}

	next, err := ResolveNextStepInSubtree(spec, "outer", "right", nil)
	if err != nil {
		t.Fatalf("resolve in subtree: %v", err)
	}
	if next == nil || next.ID != "after_branch" {
		t.Fatalf("expected after_branch, got %#v", next)
	}

	next, err = ResolveNextStepInContainer(spec, "branch", "left", nil)
	if err != nil {
		t.Fatalf("resolve in container: %v", err)
	}
	if next == nil || next.ID != "right" {
		t.Fatalf("expected right sibling, got %#v", next)
	}

	next, err = ResolveNextStepInContainer(spec, "branch", "right", nil)
	if err != nil {
		t.Fatalf("resolve terminal in container: %v", err)
	}
	if next != nil {
		t.Fatalf("expected subtree terminal, got %#v", next)
	}
}

func TestResolveNextStepInContainerErrors(t *testing.T) {
	spec := &WorkflowSpec{
		Version: "1.0",
		Name:    "invalid-container",
		Steps: []*StepSpec{
			{ID: "plain", Type: StepTypeCall, Call: &CallSpec{Action: "log.entry"}},
		},
	}

	if _, err := ResolveNextStepInSubtree(spec, "missing", "plain", nil); err == nil {
		t.Fatal("expected missing root error")
	}

	if _, err := ResolveNextStepInContainer(spec, "missing", "plain", nil); err == nil {
		t.Fatal("expected missing container error")
	}

	if _, err := ResolveNextStepInContainer(spec, "plain", "plain", nil); err == nil {
		t.Fatal("expected unsupported container error")
	}
}
