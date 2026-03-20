//nolint:testpackage // package-local DSL tests exercise unexported parser helpers intentionally.
package dsl

import "testing"

func TestParseAndCollectAllSteps(t *testing.T) {
	spec, err := Parse([]byte(`{
  "version": "1.0",
  "name": "parser-workflow",
  "steps": [
    {
      "id": "seq",
      "type": "sequence",
      "sequence": {
        "steps": [
          {"id": "child_a", "type": "call", "call": {"action": "log.entry", "input": {}}},
          {"id": "child_b", "type": "call", "call": {"action": "log.entry", "input": {}}}
        ]
      }
    },
    {"id": "after", "type": "call", "call": {"action": "log.entry", "input": {}}}
  ]
}`))
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	steps := CollectAllSteps(spec)
	if len(steps) != 4 {
		t.Fatalf("expected 4 steps, got %d", len(steps))
	}
	if steps[1].ID != "child_a" || steps[2].ID != "child_b" || steps[3].ID != "after" {
		t.Fatalf("unexpected depth-first order: %#v %#v %#v", steps[1].ID, steps[2].ID, steps[3].ID)
	}
}

func TestParseRejectsMissingRequiredFields(t *testing.T) {
	_, err := Parse([]byte(`{"version":"1.0","steps":[]}`))
	if err == nil {
		t.Fatal("expected parse error")
	}
}
