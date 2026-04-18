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

//nolint:testpackage // package-local DSL tests exercise unexported type helpers intentionally.
package dsl

import (
	"encoding/json"
	"testing"
	"time"
)

func TestTransitionTargetMarshalAndUnmarshal(t *testing.T) {
	static := &TransitionTarget{Static: "next"}
	blob, err := json.Marshal(static)
	if err != nil {
		t.Fatalf("marshal static target: %v", err)
	}
	if string(blob) != `"next"` {
		t.Fatalf("unexpected static encoding: %s", blob)
	}

	var decodedStatic TransitionTarget
	if unmarshalErr := json.Unmarshal(blob, &decodedStatic); unmarshalErr != nil {
		t.Fatalf("unmarshal static target: %v", unmarshalErr)
	}
	if decodedStatic.Static != "next" {
		t.Fatalf("expected static target, got %#v", decodedStatic)
	}

	conditional := &TransitionTarget{Conditional: []ConditionalTarget{{Condition: "true", Target: "then"}}}
	blob, err = json.Marshal(conditional)
	if err != nil {
		t.Fatalf("marshal conditional target: %v", err)
	}

	var decodedConditional TransitionTarget
	if unmarshalErr := json.Unmarshal(blob, &decodedConditional); unmarshalErr != nil {
		t.Fatalf("unmarshal conditional target: %v", unmarshalErr)
	}
	if len(decodedConditional.Conditional) != 1 || decodedConditional.Conditional[0].Target != "then" {
		t.Fatalf("expected conditional target, got %#v", decodedConditional)
	}

	var invalid TransitionTarget
	if unmarshalErr := json.Unmarshal([]byte(`123`), &invalid); unmarshalErr == nil {
		t.Fatal("expected invalid transition target error")
	}
}

func TestStepTypeAndSubsteps(t *testing.T) {
	if !StepTypeCall.IsValid() || StepType("unknown").IsValid() {
		t.Fatal("unexpected step type validation result")
	}

	step := &StepSpec{
		Type: StepTypeForeach,
		Foreach: &ForeachSpec{
			Steps: []*StepSpec{
				{ID: "child", Type: StepTypeCall, Call: &CallSpec{Action: "log.entry", Input: map[string]any{}}},
			},
		},
	}
	if len(step.AllSubSteps()) != 1 {
		t.Fatalf("expected one foreach sub-step, got %d", len(step.AllSubSteps()))
	}
}

func TestDurationJSONAndParsing(t *testing.T) {
	var duration Duration
	if unmarshalErr := json.Unmarshal([]byte(`"2d"`), &duration); unmarshalErr != nil {
		t.Fatalf("unmarshal string duration: %v", unmarshalErr)
	}
	if duration.Duration != 48*time.Hour {
		t.Fatalf("expected 48h, got %s", duration.Duration)
	}

	if unmarshalErr := json.Unmarshal([]byte(`3000000000`), &duration); unmarshalErr != nil {
		t.Fatalf("unmarshal numeric duration: %v", unmarshalErr)
	}

	blob, err := json.Marshal(Duration{Duration: 90 * time.Second})
	if err != nil {
		t.Fatalf("marshal duration: %v", err)
	}
	if string(blob) == "" {
		t.Fatal("expected marshaled duration")
	}

	if _, parseErr := ParseDuration("bad"); parseErr == nil {
		t.Fatal("expected parse duration error")
	}
}
