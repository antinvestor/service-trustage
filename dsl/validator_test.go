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

//nolint:testpackage // package-local DSL tests exercise unexported validator helpers intentionally.
package dsl

import (
	"strings"
	"testing"
	"time"
)

func TestValidateAcceptsExecutableDelayAndIfWorkflow(t *testing.T) {
	spec, err := Parse([]byte(`{
  "version": "1.0",
  "name": "validator-workflow",
  "steps": [
    {
      "id": "wait",
      "type": "delay",
      "delay": {"duration": "1h"}
    },
    {
      "id": "check",
      "type": "if",
      "if": {
        "expr": "payload.amount > 10",
        "then": [
          {"id": "high", "type": "call", "call": {"action": "log.entry", "input": {}}}
        ]
      }
    }
  ]
}`))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	result := Validate(spec)
	if !result.Valid() {
		t.Fatalf("expected valid workflow, got %v", result.Errors)
	}
}

func TestValidateDetectsDuplicateIDsAndInvalidExpressions(t *testing.T) {
	spec, err := Parse([]byte(`{
  "version": "1.0",
  "name": "invalid-workflow",
  "steps": [
    {"id": "dup", "type": "call", "call": {"action": "log.entry", "input": {}}},
    {"id": "dup", "type": "delay", "delay": {"until": "payload.amount >"}}
  ]
}`))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	result := Validate(spec)
	if result.Valid() {
		t.Fatal("expected validation failure")
	}
}

func TestValidateDetectsCyclesRetryAndTimeoutErrors(t *testing.T) {
	spec, err := Parse([]byte(`{
  "version": "1.0",
  "name": "bad-graph",
  "timeout": "1h",
  "steps": [
    {
      "id": "a",
      "type": "call",
      "depends_on": "b",
      "timeout": "2h",
      "retry": {
        "max_attempts": 0,
        "initial_interval": "bad"
      },
      "call": {"action": "log.entry", "input": {}}
    },
    {
      "id": "b",
      "type": "call",
      "depends_on": "a",
      "call": {"action": "log.entry", "input": {}}
    }
  ]
}`))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	result := Validate(spec)
	if result.Valid() {
		t.Fatal("expected validation failure")
	}
}

func TestValidateDetectsMissingFieldsAcrossStepTypes(t *testing.T) {
	spec := &WorkflowSpec{
		Version: "1.0",
		Name:    "missing-fields",
		Steps: []*StepSpec{
			{ID: "call_missing", Type: StepTypeCall},
			{ID: "delay_missing", Type: StepTypeDelay, Delay: &DelaySpec{}},
			{ID: "if_missing", Type: StepTypeIf, If: &IfSpec{}},
			{ID: "sequence_missing", Type: StepTypeSequence, Sequence: &SequenceSpec{}},
			{ID: "parallel_missing", Type: StepTypeParallel, Parallel: &ParallelSpec{}},
			{ID: "foreach_missing", Type: StepTypeForeach, Foreach: &ForeachSpec{}},
			{ID: "signal_wait_missing", Type: StepTypeSignalWait, SignalWait: &SignalWaitSpec{}},
			{ID: "signal_send_missing", Type: StepTypeSignalSend, SignalSend: &SignalSendSpec{}},
		},
	}

	result := Validate(spec)
	if result.Valid() {
		t.Fatal("expected validation failure")
	}
}

func TestValidateDetectsNestedTemplateErrors(t *testing.T) {
	spec, err := Parse([]byte(`{
  "version": "1.0",
  "name": "bad-template",
  "steps": [
    {
      "id": "step",
      "type": "call",
      "call": {
        "action": "log.entry",
        "input": {
          "nested": {
            "message": "{{    }}"
          }
        }
      }
    }
  ]
}`))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	result := Validate(spec)
	if result.Valid() {
		t.Fatal("expected validation failure")
	}
}

func TestValidateSchedules(t *testing.T) {
	cases := []struct {
		name      string
		schedules []*ScheduleSpec
		wantValid bool
		wantMsg   string
	}{
		{name: "nil slice is valid", schedules: nil, wantValid: true},
		{name: "empty slice is valid", schedules: []*ScheduleSpec{}, wantValid: true},
		{
			name: "valid single schedule",
			schedules: []*ScheduleSpec{
				{Name: "nightly", CronExpr: "0 2 * * *"},
			},
			wantValid: true,
		},
		{
			name: "empty name",
			schedules: []*ScheduleSpec{
				{Name: "", CronExpr: "*/5 * * * *"},
			},
			wantValid: false,
			wantMsg:   "name",
		},
		{
			name: "duplicate names",
			schedules: []*ScheduleSpec{
				{Name: "same", CronExpr: "*/5 * * * *"},
				{Name: "same", CronExpr: "0 2 * * *"},
			},
			wantValid: false,
			wantMsg:   "duplicate",
		},
		{
			name: "invalid cron",
			schedules: []*ScheduleSpec{
				{Name: "bad", CronExpr: "every 5 minutes"},
			},
			wantValid: false,
			wantMsg:   "cron",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			spec := &WorkflowSpec{
				Version: "v1",
				Name:    "w",
				Steps: []*StepSpec{{
					ID:    "s",
					Type:  StepTypeDelay,
					Delay: &DelaySpec{Duration: Duration{Duration: time.Second}},
				}},
				Schedules: tc.schedules,
			}
			result := Validate(spec)
			if tc.wantValid && !result.Valid() {
				t.Fatalf("expected valid, got errors: %v", result.Error())
			}
			if !tc.wantValid {
				if result.Valid() {
					t.Fatalf("expected invalid")
				}
				if tc.wantMsg != "" && !strings.Contains(result.Error().Error(), tc.wantMsg) {
					t.Fatalf("expected error containing %q, got %v", tc.wantMsg, result.Error())
				}
			}
		})
	}
}
