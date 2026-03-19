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
