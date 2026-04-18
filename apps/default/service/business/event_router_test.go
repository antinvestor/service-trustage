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
