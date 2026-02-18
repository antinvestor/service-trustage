package dsl

import (
	"fmt"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker"
)

const celCostBudget = 10000

// defaultCostEstimator provides default cost estimates for CEL expressions.
type defaultCostEstimator struct{}

func (defaultCostEstimator) EstimateSize(_ checker.AstNode) *checker.SizeEstimate {
	return nil
}

func (defaultCostEstimator) EstimateCallCost(
	_, _ string,
	_ *checker.AstNode,
	_ []checker.AstNode,
) *checker.CallEstimate {
	return nil
}

// NewExpressionEnv creates a new CEL environment with standard variables and cost budget.
func NewExpressionEnv() (*cel.Env, error) {
	env, err := cel.NewEnv(
		cel.Variable("payload", cel.DynType),
		cel.Variable("metadata", cel.DynType),
		cel.Variable("vars", cel.DynType),
		cel.Variable("env", cel.DynType),
		cel.Variable("now", cel.TimestampType),
		cel.Variable("item", cel.DynType),
		cel.Variable("index", cel.IntType),
		cel.Variable("output", cel.DynType),
		cel.Variable("signal", cel.DynType),
	)
	if err != nil {
		return nil, fmt.Errorf("create CEL env: %w", err)
	}

	return env, nil
}

// CompileExpression compiles a CEL expression and validates its cost.
func CompileExpression(env *cel.Env, expr string) (*cel.Ast, error) {
	ast, issues := env.Compile(expr)
	if issues != nil && issues.Err() != nil {
		return nil, fmt.Errorf("compile expression %q: %w", expr, issues.Err())
	}

	costEst, err := env.EstimateCost(ast, defaultCostEstimator{})
	if err != nil {
		return nil, fmt.Errorf("estimate cost for %q: %w", expr, err)
	}

	if costEst.Max > celCostBudget {
		return nil, fmt.Errorf(
			"expression %q exceeds cost budget (max=%d, budget=%d)",
			expr,
			costEst.Max,
			celCostBudget,
		)
	}

	return ast, nil
}

// EvaluateExpression evaluates a compiled CEL expression and returns its result as any.
func EvaluateExpression(env *cel.Env, ast *cel.Ast, vars map[string]any) (any, error) {
	prg, err := env.Program(ast)
	if err != nil {
		return nil, fmt.Errorf("program creation: %w", err)
	}

	out, _, err := prg.Eval(vars)
	if err != nil {
		return nil, fmt.Errorf("expression evaluation: %w", err)
	}

	return out.Value(), nil
}

// EvaluateCondition evaluates a compiled CEL expression against a set of variables.
func EvaluateCondition(env *cel.Env, ast *cel.Ast, vars map[string]any) (bool, error) {
	prg, err := env.Program(ast)
	if err != nil {
		return false, fmt.Errorf("program creation: %w", err)
	}

	out, _, err := prg.Eval(vars)
	if err != nil {
		return false, fmt.Errorf("expression evaluation: %w", err)
	}

	result, ok := out.Value().(bool)
	if !ok {
		return false, fmt.Errorf("expression did not evaluate to bool, got %T", out.Value())
	}

	return result, nil
}
