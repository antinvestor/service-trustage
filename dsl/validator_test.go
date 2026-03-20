//nolint:testpackage // package-local DSL tests exercise unexported validator helpers intentionally.
package dsl

import "testing"

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
