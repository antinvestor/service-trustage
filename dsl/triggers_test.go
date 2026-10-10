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
)

const routedStepsJSON = `"steps": [
    {"id": "start", "type": "call", "call": {"action": "log.entry", "input": {}}},
    {"id": "wait_cancel", "type": "signal_wait", "signal_wait": {"signal_name": "proposal.cancelled"}},
    {
      "id": "branch", "type": "if",
      "if": {
        "expr": "payload.x == 1",
        "then": [{"id": "wait_final", "type": "signal_wait", "signal_wait": {"signal_name": "finalized"}}]
      }
    }
  ]`

func parseRouted(t *testing.T, routing string) *WorkflowSpec {
	t.Helper()

	doc := `{"version": "1.0", "name": "routed", ` + routing + `, ` + routedStepsJSON + `}`

	spec, err := Parse([]byte(doc))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	return spec
}

func TestParseTriggersAndSignals(t *testing.T) {
	spec := parseRouted(t, `
  "triggers": [{"event_type": "proposal.created", "filter": "payload.amount > 0", "key": "payload.proposal_id"}],
  "signals": [
    {"event_type": "proposal.cancelled", "key": "payload.proposal_id"},
    {"event_type": "intent.finalized", "signal_name": "finalized", "instance": "payload.meta.instance_id", "filter": "true"}
  ]`)

	if len(spec.Triggers) != 1 || len(spec.Signals) != 2 {
		t.Fatalf("expected 1 trigger and 2 signals, got %d and %d", len(spec.Triggers), len(spec.Signals))
	}

	tr := spec.Triggers[0]
	if tr.EventType != "proposal.created" || tr.Filter != "payload.amount > 0" || tr.Key != "payload.proposal_id" {
		t.Fatalf("trigger fields not parsed: %+v", tr)
	}

	if got := spec.Signals[0].EffectiveSignalName(); got != "proposal.cancelled" {
		t.Fatalf("signal name should default to event_type, got %q", got)
	}

	if got := spec.Signals[1].EffectiveSignalName(); got != "finalized" {
		t.Fatalf("explicit signal name lost, got %q", got)
	}

	if spec.Signals[1].Instance != "payload.meta.instance_id" {
		t.Fatalf("instance expression not parsed: %+v", spec.Signals[1])
	}

	if res := Validate(spec); !res.Valid() {
		t.Fatalf("expected valid spec, got %v", res.Errors)
	}
}

func TestValidateWithoutRoutingStaysValid(t *testing.T) {
	spec, err := Parse([]byte(`{"version": "1.0", "name": "plain", "steps": [
    {"id": "a", "type": "call", "call": {"action": "log.entry", "input": {}}}
  ]}`))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if res := Validate(spec); !res.Valid() {
		t.Fatalf("expected valid, got %v", res.Errors)
	}
}

func TestValidateRoutingRules(t *testing.T) {
	tests := []struct {
		name    string
		routing string
		wantErr string
	}{
		{
			name:    "trigger needs event_type",
			routing: `"triggers": [{"event_type": " "}]`,
			wantErr: "triggers[0]: event_type is required",
		},
		{
			name:    "trigger filter must compile",
			routing: `"triggers": [{"event_type": "a", "filter": "payload.x >"}]`,
			wantErr: "triggers[0]: invalid filter",
		},
		{
			name:    "trigger filter must be bool",
			routing: `"triggers": [{"event_type": "a", "filter": "'yes'"}]`,
			wantErr: "triggers[0]: filter must evaluate to bool",
		},
		{
			name:    "trigger key must compile",
			routing: `"triggers": [{"event_type": "a", "key": "payload.("}]`,
			wantErr: "triggers[0]: invalid key",
		},
		{
			name: "signal needs event_type",
			routing: `"triggers": [{"event_type": "a", "key": "payload.id"}],
			  "signals": [{"event_type": "", "signal_name": "finalized", "key": "payload.id"}]`,
			wantErr: "signals[0]: event_type is required",
		},
		{
			name:    "signal needs key or instance",
			routing: `"signals": [{"event_type": "proposal.cancelled"}]`,
			wantErr: "signals[0]: exactly one of key and instance is required",
		},
		{
			name: "signal must not set both key and instance",
			routing: `"triggers": [{"event_type": "a", "key": "payload.id"}],
			  "signals": [{"event_type": "proposal.cancelled", "key": "payload.id", "instance": "payload.i"}]`,
			wantErr: "signals[0]: exactly one of key and instance is required",
		},
		{
			name: "key route requires every trigger to declare key",
			routing: `"triggers": [{"event_type": "a", "key": "payload.id"}, {"event_type": "b"}],
			  "signals": [{"event_type": "proposal.cancelled", "key": "payload.id"}]`,
			wantErr: "signals[0]: a key route requires every trigger to declare key (triggers[1] has none)",
		},
		{
			name:    "signal filter must be bool",
			routing: `"signals": [{"event_type": "proposal.cancelled", "instance": "payload.i", "filter": "1"}]`,
			wantErr: "signals[0]: filter must evaluate to bool",
		},
		{
			name:    "signal instance must compile",
			routing: `"signals": [{"event_type": "proposal.cancelled", "instance": "payload.i +"}]`,
			wantErr: "signals[0]: invalid instance",
		},
		{
			name: "signal key must compile",
			routing: `"triggers": [{"event_type": "a", "key": "payload.id"}],
			  "signals": [{"event_type": "proposal.cancelled", "key": "payload.("}]`,
			wantErr: "signals[0]: invalid key",
		},
		{
			name:    "signal name must be waited on",
			routing: `"signals": [{"event_type": "proposal.voted", "instance": "payload.i"}]`,
			wantErr: `signals[0]: no signal_wait step waits on signal "proposal.voted"`,
		},
		{
			name:    "explicit signal name must be waited on",
			routing: `"signals": [{"event_type": "x", "signal_name": "nope", "instance": "payload.i"}]`,
			wantErr: `signals[0]: no signal_wait step waits on signal "nope"`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res := Validate(parseRouted(t, tc.routing))
			if res.Valid() {
				t.Fatalf("expected validation error containing %q", tc.wantErr)
			}

			for _, e := range res.Errors {
				if strings.Contains(e.Message, tc.wantErr) {
					return
				}
			}

			t.Fatalf("expected error containing %q, got %v", tc.wantErr, res.Errors)
		})
	}
}

func TestFormatCorrelationKey(t *testing.T) {
	tests := []struct {
		in   any
		want string
		ok   bool
	}{
		{"p-1", "p-1", true},
		{"", "", false},
		{int64(42), "42", true},
		{uint64(7), "7", true},
		{float64(42), "42", true},
		{1.5, "1.5", true},
		{true, "true", true},
		{nil, "", false},
		{map[string]any{"a": 1}, "", false},
		{[]any{1}, "", false},
	}

	for _, tc := range tests {
		got, ok := FormatCorrelationKey(tc.in)
		if got != tc.want || ok != tc.ok {
			t.Fatalf("FormatCorrelationKey(%#v) = (%q, %v), want (%q, %v)", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}

func TestValidateRoutingAcceptsDynFilterAndFallbackWait(t *testing.T) {
	spec, err := Parse([]byte(`{
  "version": "1.0",
  "name": "fallback-wait",
  "triggers": [{"event_type": "a", "filter": "payload.enabled", "key": "payload.id"}],
  "signals": [{"event_type": "late", "key": "payload.id"}],
  "steps": [
    {
      "id": "work", "type": "call", "call": {"action": "log.entry", "input": {}},
      "on_error": {"strategy": "fallback", "fallback": [
        {"id": "wait_late", "type": "signal_wait", "signal_wait": {"signal_name": "late"}}
      ]}
    }
  ]
}`))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if res := Validate(spec); !res.Valid() {
		t.Fatalf("expected valid, got %v", res.Errors)
	}
}
