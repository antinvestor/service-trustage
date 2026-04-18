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

func TestEvaluateCondition(t *testing.T) {
	env, err := NewExpressionEnv()
	if err != nil {
		t.Fatalf("create env failed: %v", err)
	}

	ast, err := CompileExpression(env, `payload.amount > 10`)
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}

	matched, err := EvaluateCondition(env, ast, map[string]any{
		"payload": map[string]any{"amount": 42},
	})
	if err != nil {
		t.Fatalf("evaluate failed: %v", err)
	}
	if !matched {
		t.Fatal("expected condition to match")
	}
}

func TestEvaluateConditionRejectsNonBool(t *testing.T) {
	env, err := NewExpressionEnv()
	if err != nil {
		t.Fatalf("create env failed: %v", err)
	}

	ast, err := CompileExpression(env, `payload.amount`)
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}

	_, err = EvaluateCondition(env, ast, map[string]any{
		"payload": map[string]any{"amount": 42},
	})
	if err == nil {
		t.Fatal("expected non-bool error")
	}
}

func TestEvaluateExpressionReturnsValue(t *testing.T) {
	env, err := NewExpressionEnv()
	if err != nil {
		t.Fatalf("create env failed: %v", err)
	}

	ast, err := CompileExpression(env, `payload.amount + 8`)
	if err != nil {
		t.Fatalf("compile failed: %v", err)
	}

	value, err := EvaluateExpression(env, ast, map[string]any{
		"payload": map[string]any{"amount": 34},
	})
	if err != nil {
		t.Fatalf("evaluate failed: %v", err)
	}
	if value.(int64) != 42 {
		t.Fatalf("expected 42, got %#v", value)
	}
}
