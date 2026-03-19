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
	if err := json.Unmarshal(blob, &decodedStatic); err != nil {
		t.Fatalf("unmarshal static target: %v", err)
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
	if err := json.Unmarshal(blob, &decodedConditional); err != nil {
		t.Fatalf("unmarshal conditional target: %v", err)
	}
	if len(decodedConditional.Conditional) != 1 || decodedConditional.Conditional[0].Target != "then" {
		t.Fatalf("expected conditional target, got %#v", decodedConditional)
	}

	var invalid TransitionTarget
	if err := json.Unmarshal([]byte(`123`), &invalid); err == nil {
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
	if err := json.Unmarshal([]byte(`"2d"`), &duration); err != nil {
		t.Fatalf("unmarshal string duration: %v", err)
	}
	if duration.Duration != 48*time.Hour {
		t.Fatalf("expected 48h, got %s", duration.Duration)
	}

	if err := json.Unmarshal([]byte(`3000000000`), &duration); err != nil {
		t.Fatalf("unmarshal numeric duration: %v", err)
	}

	blob, err := json.Marshal(Duration{Duration: 90 * time.Second})
	if err != nil {
		t.Fatalf("marshal duration: %v", err)
	}
	if string(blob) == "" {
		t.Fatal("expected marshaled duration")
	}

	if _, err := ParseDuration("bad"); err == nil {
		t.Fatal("expected parse duration error")
	}
}
