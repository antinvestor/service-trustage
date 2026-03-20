//nolint:testpackage // package-local DSL tests exercise unexported helpers intentionally.
package dsl

import "testing"

func TestValidationErrorFormattingAndResultHelpers(t *testing.T) {
	stepErr := (&ValidationError{
		Code:    ErrMissingRequired,
		Message: "missing",
		StepID:  "step_a",
	}).Error()
	if stepErr == "" {
		t.Fatal("expected formatted step error")
	}

	pathErr := (&ValidationError{
		Code:    ErrInvalidTemplate,
		Message: "bad",
		Path:    "$.steps[0]",
	}).Error()
	if pathErr == "" {
		t.Fatal("expected formatted path error")
	}

	result := &ValidationResult{}
	result.AddError(ErrMissingRequired, "workflow name is required")
	result.AddErrorWithStep(ErrInvalidStepType, "step_a", "invalid")
	result.AddErrorWithPath(ErrInvalidTemplate, "$.steps[0]", "bad template")

	if result.Valid() {
		t.Fatal("expected invalid result")
	}
	if result.Error() == nil {
		t.Fatal("expected combined error")
	}
}
