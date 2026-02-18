# ADR-003: CEL for Safe Expression Evaluation

## Status

Accepted

## Context

The JSON DSL (ADR-002) requires a safe expression language for:

1. **Condition evaluation**: `if` steps need boolean expressions (`payload.amount > 100`)
2. **Trigger filtering**: Event router filters events before starting workflows (`payload.status == "active"`)
3. **List generation**: `foreach` step `items` fields need to evaluate to lists (`payload.line_items`)
4. **Dynamic timestamps**: `delay` step `until` fields need timestamp expressions (`now + duration("24h")`)
5. **Data transformation**: Extracting and reshaping data between steps

The expression language must be:

- **Safe**: No infinite loops, no unbounded memory allocation, no side effects, no I/O
- **Deterministic**: Same input must produce same output (required for consistent state transitions)
- **Fast**: Expressions are evaluated in the hot path of every workflow step
- **Expressive**: Boolean logic, arithmetic, string operations, collection operations, temporal operations
- **Sandboxed**: No access to filesystem, network, environment variables, or Go runtime
- **Compilable**: Parse once at definition time, evaluate many times at runtime

## Decision

Use **Google CEL** (Common Expression Language) via the `google/cel-go` library.

### Standard Environment

Every CEL expression is evaluated with these variables in scope:

| Variable | Type | Description |
|----------|------|-------------|
| `payload` | `map(string, dyn)` | The triggering event's payload |
| `metadata` | `map(string, dyn)` | Event metadata (source, timestamp, headers) |
| `vars` | `map(string, dyn)` | Variables set by previous steps via `output_var` |
| `env` | `map(string, dyn)` | Tenant-scoped environment variables (non-secret) |
| `now` | `timestamp` | Current timestamp (injected by the engine at evaluation time) |
| `item` | `dyn` | Current item in `foreach` iteration |
| `index` | `int` | Current index in `foreach` iteration |

### Custom Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `has` | `has(map, string) -> bool` | Check if key exists in map |
| `contains` | `contains(string, string) -> bool` | Check if string contains substring |
| `matches` | `matches(string, string) -> bool` | Check if string matches regex pattern |
| `len` | `len(list\|string\|map) -> int` | Return length of collection or string |
| `default` | `default(dyn, dyn) -> dyn` | Return first argument if non-null, else second |
| `duration` | `duration(string) -> duration` | Parse duration string ("5m", "24h") |
| `format_time` | `format_time(timestamp, string) -> string` | Format timestamp with layout string |

### Cost Budget

Every CEL evaluation is constrained to a maximum cost of **10,000 units**. This prevents pathological expressions from consuming unbounded resources. The cost model is built into CEL and accounts for iteration, string operations, and nested access.

### Restrictions

- No user-defined macros or functions
- No mutable state (all expressions are pure)
- No access to Go runtime or reflection
- Regex patterns are compiled with RE2 (guaranteed linear time)

## Alternatives Considered

| Option | Pros | Cons | Verdict |
|--------|------|------|---------|
| **CEL (google/cel-go)** | Provably non-Turing-complete. Built-in cost budgeting. Deterministic. Compile-once evaluate-many. Battle-tested at Google (Firebase, Envoy, Kubernetes). Strong type system. | Users must learn CEL syntax. Stricter type system than JavaScript. Error messages can be cryptic. Smaller community than general-purpose languages. | **Chosen** |
| **expr-lang/expr** | Go-native. Simple syntax. Fast. Popular in Go ecosystem. | Turing-complete (unbounded loops). Weaker sandboxing guarantees. No built-in cost budgeting. Harder to prove termination. | Rejected |
| **Starlark** | Python-like syntax familiar to many users. Used by Bazel/Buck. Sandboxed. | Too powerful for expression evaluation. Slow interpreter. Not deterministic (dict ordering). Overkill for boolean conditions. | Rejected |
| **JavaScript (goja)** | Familiar syntax. Huge ecosystem. | Turing-complete. Fragile sandboxing (prototype pollution, eval). Not deterministic. Heavy runtime. Security risk surface. | Rejected |
| **JSONPath / JMESPath** | Simple path-based access. Widely known. | Read-only, no boolean logic or arithmetic. Cannot express conditions. Cannot transform data. | Rejected |
| **Custom parser** | Full control over syntax and semantics. | Must build and maintain lexer, parser, type checker, evaluator. Years of edge cases. No ecosystem. | Rejected |
| **Go text/template** | Built into standard library. Template syntax familiar. | Not designed for boolean evaluation. Side effects possible via function maps. No cost budgeting. Poor error messages. Not deterministic. | Rejected |

## Rationale

1. **Provably non-Turing-complete.** CEL guarantees termination for every expression. There are no loops, no recursion, no unbounded iteration. This is a hard requirement for a multi-tenant platform where user-authored expressions run on shared infrastructure.

2. **Built-in cost budgeting.** CEL's cost model assigns a numeric cost to every operation. By setting a maximum cost budget (10,000), we prevent pathological expressions (deeply nested access, large regex patterns, quadratic string operations) from consuming excessive resources.

3. **Deterministic evaluation.** CEL expressions are pure functions: same input always produces same output. This ensures consistent behavior across retries and state transitions.

4. **Compile-once, evaluate-many.** CEL expressions are compiled into an AST at workflow definition time (during validation). At runtime, evaluation uses the pre-compiled program with only variable binding. This makes per-evaluation cost sub-microsecond.

5. **Battle-tested at Google scale.** CEL is used in Firebase Security Rules, Envoy RBAC, Kubernetes admission policies, and Google Cloud IAM conditions. The implementation is mature, well-tested, and actively maintained.

6. **Type system catches errors at compile time.** CEL's type checker validates expressions against the declared variable types. A typo like `payload.naem` (instead of `payload.name`) is caught at definition time, not at runtime in the middle of a workflow execution.

## Consequences

**Positive:**

- Guaranteed termination for every expression, regardless of input
- Deterministic evaluation ensures consistent state transitions
- Type errors caught at workflow definition time, not execution time
- Sub-microsecond evaluation after compilation
- Cost budgeting prevents resource exhaustion from pathological expressions
- Well-documented syntax with existing tutorials and reference material

**Negative:**

- Users must learn CEL syntax (different from JavaScript, Python, or SQL)
- Stricter type system means some expressions require explicit type handling (e.g., `int(payload.count) > 5` instead of `payload.count > 5`)
- Custom functions must be carefully implemented to maintain determinism and bounded cost
- Error messages from the CEL compiler can be cryptic for non-technical users
- Limited string manipulation compared to general-purpose languages

## Implementation Notes

### Core API

Expression evaluation lives in `dsl/expression.go`:

```go
// NewCELEnvironment creates a CEL environment with the standard
// Orchestrator variables and custom functions.
func NewCELEnvironment() (*cel.Env, error) {
    return cel.NewEnv(
        cel.Variable("payload", cel.MapType(cel.StringType, cel.DynType)),
        cel.Variable("metadata", cel.MapType(cel.StringType, cel.DynType)),
        cel.Variable("vars", cel.MapType(cel.StringType, cel.DynType)),
        cel.Variable("env", cel.MapType(cel.StringType, cel.DynType)),
        cel.Variable("now", cel.TimestampType),
        cel.Variable("item", cel.DynType),
        cel.Variable("index", cel.IntType),
        ext.Strings(),
        customFunctions(),
    )
}

// CompileExpression parses and type-checks a CEL expression,
// returning a compiled program for repeated evaluation.
func CompileExpression(env *cel.Env, expr string) (cel.Program, error) {
    ast, iss := env.Compile(expr)
    if iss.Err() != nil {
        return nil, fmt.Errorf("compile expression: %w", iss.Err())
    }
    return env.Program(ast, cel.CostLimit(10_000))
}

// EvaluateBool evaluates a compiled CEL program with the given
// variables and returns a boolean result.
func EvaluateBool(prg cel.Program, vars map[string]any) (bool, error) {
    out, _, err := prg.Eval(vars)
    if err != nil {
        return false, fmt.Errorf("evaluate expression: %w", err)
    }
    b, ok := out.Value().(bool)
    if !ok {
        return false, fmt.Errorf("expression result is %T, not bool", out.Value())
    }
    return b, nil
}
```

### Custom Function Extension Plan

| Phase | Functions | Purpose |
|-------|-----------|---------|
| Phase 1 | `has`, `contains`, `matches`, `len`, `default` | Core utilities for conditions and null handling |
| Phase 2 | `duration`, `format_time`, `parse_time`, `hash` | Temporal operations and deterministic hashing |
| Phase 3 | `lookup`, `geo_distance` | Data enrichment and geographic calculations |

All custom functions must satisfy three invariants:

1. **Pure**: No side effects, no I/O, no mutable state
2. **Deterministic**: Same arguments always produce same result
3. **Bounded cost**: Execution time proportional to input size with a known upper bound

### Expression Examples

```cel
// Simple condition
payload.amount > 100

// Compound condition with null safety
has(payload, "email") && payload.email != "" && payload.status == "active"

// String matching
matches(payload.domain, "^.*\\.edu$")

// Collection check
len(payload.items) > 0 && payload.items[0].type == "premium"

// Temporal comparison
now - metadata.created_at > duration("1h")

// Default values
default(payload.priority, "normal") == "high"

// Foreach item access
item.quantity * item.unit_price > 50.0
```

### Extensibility

| Use Case | Expression Pattern |
|----------|--------------------|
| SLA enforcement | `now - metadata.created_at > duration("4h")` |
| Data transformation | `payload.items.filter(i, i.active == true)` |
| Dynamic routing | `payload.region == "eu" \|\| payload.country in ["DE", "FR", "IT"]` |
| Aggregate conditions | `payload.items.map(i, i.amount).reduce(acc, v, acc + v) > 1000` |
| Template guards | `has(payload, "name") && len(payload.name) > 0` |
| A/B testing | `hash(payload.user_id) % 100 < 50` |
| Geo-routing | `geo_distance(payload.lat, payload.lon, 51.5074, -0.1278) < 100.0` |
| Business hours | `now.getHours() >= 9 && now.getHours() < 17 && now.getDayOfWeek() > 0 && now.getDayOfWeek() < 6` |
