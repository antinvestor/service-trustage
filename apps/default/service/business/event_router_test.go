//nolint:testpackage // tests need access to unexported helpers
package business

import "testing"

func TestEvaluateTriggerFilter_DefaultsTrue(t *testing.T) {
	ok, err := evaluateTriggerFilter("", map[string]any{"x": 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatalf("expected true for empty filter")
	}
}

func TestEvaluateTriggerFilter_WithCEL(t *testing.T) {
	ok, err := evaluateTriggerFilter("payload.amount > 10", map[string]any{"amount": 15})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatalf("expected filter to match")
	}

	ok, err = evaluateTriggerFilter("payload.amount > 10", map[string]any{"amount": 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatalf("expected filter to not match")
	}
}
