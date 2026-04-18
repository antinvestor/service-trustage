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
