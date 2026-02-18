# ADR-002: JSON DSL as Workflow Definition Format

## Status

Accepted

## Context

Workflow definitions must be:

1. **Authorable by users** via a visual workflow builder UI
2. **Generatable by AI** via LLM structured output (JSON mode)
3. **Versionable** with non-breaking schema evolution
4. **Storable** in PostgreSQL as JSONB for querying and indexing
5. **Validatable** before execution to catch errors early
6. **Inspectable** for debugging and audit purposes
7. **Interpretable** by the state engine for contract-driven execution
8. **Extensible** for new step types without breaking existing definitions

The format must serve three producers (visual builder, AI, API) and one consumer (the state engine's DSL interpreter). It is not executable code -- it is an intermediate representation that the engine reads and executes step by step.

## Decision

Define a **custom JSON DSL** with the following structure:

### Envelope

```json
{
  "version": "1.0",
  "steps": { ... }
}
```

The `version` field is a semver string that identifies the DSL schema version. The interpreter uses this to select the correct parsing and validation logic. New versions add capabilities without breaking existing definitions.

### Step Structure

Every step is an object with a common shape:

```json
{
  "id": "unique-step-id",
  "type": "call | delay | if | sequence | parallel | foreach | signal_wait | signal_send",
  "name": "Human-readable step name",
  "description": "Optional description for documentation",
  "retry": { "max_attempts": 3, "initial_interval": "1s", "backoff_coefficient": 2.0, "max_interval": "60s" },
  "timeout": "5m",
  "on_error": { "action": "continue | fail | goto", "target": "step-id" },
  "...type-specific fields"
}
```

### Step Types

#### Phase 1 (v1.0)

| Type | Purpose | Key Fields |
|------|---------|------------|
| `call` | Execute a connector adapter | `action`, `input`, `output_var` |
| `delay` | Wait for a duration or until a timestamp | `duration`, `until` |
| `if` | Conditional branching | `condition`, `then`, `else` |
| `sequence` | Execute steps in order | `steps` (array of steps) |

#### Phase 2 (v1.1)

| Type | Purpose | Key Fields |
|------|---------|------------|
| `parallel` | Execute steps concurrently | `branches` (array of steps), `max_concurrency` |
| `foreach` | Iterate over a collection | `items`, `item_var`, `index_var`, `body`, `max_concurrency` |

#### Phase 3 (v1.2)

| Type | Purpose | Key Fields |
|------|---------|------------|
| `signal_wait` | Wait for an external signal (approval, callback) | `signal_name`, `timeout`, `output_var` |
| `signal_send` | Send a signal to another workflow | `target_workflow_id`, `signal_name`, `payload` |

#### Future Step Types

| Type | Purpose |
|------|---------|
| `sub_workflow` | Execute another workflow definition as a child |
| `switch` | Multi-branch conditional (pattern matching) |
| `loop` | Repeat until condition is met |
| `map` | Transform a collection with an expression |
| `approve` | Human approval with timeout and escalation |
| `try_catch` | Explicit error handling block |
| `gate` | Wait for multiple conditions before proceeding |
| `transform` | Apply data transformation expressions |
| `log` | Emit structured log entry |
| `notify` | Send notification via configured channel |

### Expressions and Templates

- **CEL expressions** for boolean conditions: `payload.amount > 100 && payload.status == "active"`
- **Template syntax** (`{{ }}`) for variable interpolation in string fields: `"Hello, {{ payload.name }}"`
- Templates may reference: `payload`, `metadata`, `vars`, `env`, `item`, `index`

### Example Definition

```json
{
  "version": "1.0",
  "steps": {
    "id": "root",
    "type": "sequence",
    "steps": [
      {
        "id": "send-welcome",
        "type": "call",
        "name": "Send welcome email",
        "action": "email.send",
        "input": {
          "to": "{{ payload.email }}",
          "subject": "Welcome, {{ payload.name }}!",
          "template": "welcome-v1"
        },
        "output_var": "email_result",
        "retry": { "max_attempts": 3, "initial_interval": "5s", "backoff_coefficient": 2.0 }
      },
      {
        "id": "wait-24h",
        "type": "delay",
        "name": "Wait 24 hours",
        "duration": "24h"
      },
      {
        "id": "check-engagement",
        "type": "if",
        "name": "Check if user engaged",
        "condition": "vars.email_result.opened == true",
        "then": {
          "id": "send-followup",
          "type": "call",
          "name": "Send follow-up offer",
          "action": "email.send",
          "input": {
            "to": "{{ payload.email }}",
            "subject": "A special offer for you",
            "template": "followup-offer-v1"
          },
          "output_var": "followup_result"
        },
        "else": {
          "id": "send-reminder",
          "type": "call",
          "name": "Send reminder",
          "action": "email.send",
          "input": {
            "to": "{{ payload.email }}",
            "subject": "Don't miss out, {{ payload.name }}",
            "template": "reminder-v1"
          },
          "output_var": "reminder_result"
        }
      }
    ]
  }
}
```

## Alternatives Considered

| Option | Pros | Cons | Verdict |
|--------|------|------|---------|
| **Custom JSON DSL** | Universal interchange format. PostgreSQL JSONB enables SQL queries. AI structured output produces validated JSON reliably. Version envelope for evolution. Declarative enables static analysis. | Must build validator, template engine, expression evaluator. Custom format requires documentation. No existing IDE support. | **Chosen** |
| **YAML** | More readable for humans. Widespread in DevOps tooling. | Type coercion ambiguity (`yes`/`no`, `1.0` vs `"1.0"`). AI produces less reliable YAML (indentation errors). No JSONB equivalent in PostgreSQL. Must convert to JSON for storage. | Rejected |
| **Protocol Buffers** | Strict schema. Fast serialization. Code generation. | Not human-readable or editable. AI cannot produce binary protobuf. Poor fit for dynamic step structures. No visual builder round-trip. | Rejected |
| **CUE** | Powerful type system. Validation built in. Composable. | Steep learning curve. Small ecosystem. AI cannot reliably produce CUE syntax. No PostgreSQL native support. | Rejected |
| **Starlark** | Python-like syntax. Sandboxed execution. Used by Bazel. | Too powerful (Turing-complete subset). Security risk from user code execution. Not declarative. AI output unreliable. Cannot statically analyze. | Rejected |
| **HCL** | Used by Terraform. Block syntax is readable. | Niche outside HashiCorp ecosystem. Poor AI output quality. No JSONB storage. Complex parser. | Rejected |
| **Existing DSL (n8n/Zapier)** | Proven in production. Existing visual builders. | Proprietary formats. Not AI-friendly structured output. Tight coupling to specific UI. Cannot extend step types. | Rejected |

## Rationale

1. **JSON is the universal interchange format.** Every language, tool, database, and AI model speaks JSON. There is no serialization barrier between the visual builder, AI, API, storage layer, and interpreter.

2. **PostgreSQL JSONB enables powerful queries.** Workflow definitions stored as JSONB support indexing, path queries, and containment checks. Find all workflows that use `email.send`, query step counts, filter by version -- all via SQL.

3. **AI structured output produces validated JSON.** LLMs with JSON mode (OpenAI, Anthropic, etc.) produce well-formed JSON reliably. The JSON Schema for the DSL can be provided as a system prompt constraint, and the output can be validated immediately.

4. **Version envelope enables non-breaking evolution.** Adding new step types or fields in v1.1 does not break v1.0 definitions. The interpreter checks the version and applies the correct parsing logic. Migration functions can upgrade older versions.

5. **Declarative DSL enables static analysis.** Because the DSL is data, not code, we can perform cycle detection (DFS on step references), reachability analysis (are all steps reachable from root?), type checking (do connector inputs match schema?), and CEL expression compilation -- all before execution.

## Consequences

**Positive:**

- AI generates validated workflow definitions via structured output
- Visual builder reads and writes the same format without transformation
- JSONB storage enables SQL-based analytics and querying of workflow structure
- Version envelope provides a clear migration path for schema evolution
- Static analysis catches errors (cycles, unreachable steps, type mismatches, invalid expressions) before execution
- Step tree is composable -- any step can contain other steps

**Negative:**

- Must build and maintain a custom JSON Schema validator
- Must build a template engine for `{{ }}` interpolation
- Must build a CEL expression evaluator (see ADR-003)
- Custom format requires documentation and examples for users
- No existing IDE support (autocomplete, syntax highlighting) without building a JSON Schema and Language Server

## Implementation Notes

### Go Types

Core types live in `dsl/types.go`:

```go
// WorkflowSpec is the top-level envelope.
type WorkflowSpec struct {
    Version string `json:"version"`
    Steps   *Step  `json:"steps"`
}

// Step is a node in the workflow tree.
type Step struct {
    ID          string       `json:"id"`
    Type        StepType     `json:"type"`
    Name        string       `json:"name,omitempty"`
    Description string       `json:"description,omitempty"`
    Retry       *RetryPolicy `json:"retry,omitempty"`
    Timeout     string       `json:"timeout,omitempty"`
    OnError     *ErrorHandler `json:"on_error,omitempty"`

    // call
    Action    string         `json:"action,omitempty"`
    Input     map[string]any `json:"input,omitempty"`
    OutputVar string         `json:"output_var,omitempty"`

    // delay
    Duration string `json:"duration,omitempty"`
    Until    string `json:"until,omitempty"`

    // if
    Condition string `json:"condition,omitempty"`
    Then      *Step  `json:"then,omitempty"`
    Else      *Step  `json:"else,omitempty"`

    // sequence, parallel
    Steps    []*Step `json:"steps,omitempty"`
    Branches []*Step `json:"branches,omitempty"`

    // foreach
    Items          string `json:"items,omitempty"`
    ItemVar        string `json:"item_var,omitempty"`
    IndexVar       string `json:"index_var,omitempty"`
    Body           *Step  `json:"body,omitempty"`
    MaxConcurrency int    `json:"max_concurrency,omitempty"`

    // signal_wait, signal_send
    SignalName       string         `json:"signal_name,omitempty"`
    TargetWorkflowID string         `json:"target_workflow_id,omitempty"`
    Payload          map[string]any `json:"payload,omitempty"`
}

// RetryPolicy configures step-level retry behavior.
type RetryPolicy struct {
    MaxAttempts        int     `json:"max_attempts"`
    InitialInterval    string  `json:"initial_interval"`
    BackoffCoefficient float64 `json:"backoff_coefficient"`
    MaxInterval        string  `json:"max_interval,omitempty"`
}

// ErrorHandler configures step-level error behavior.
type ErrorHandler struct {
    Action string `json:"action"` // continue, fail, goto
    Target string `json:"target,omitempty"`
}
```

### Validation

Validation lives in `dsl/validator.go` and runs in this order:

1. **Schema validation**: JSON structure matches expected shape for the declared version
2. **ID uniqueness**: No duplicate step IDs within a definition
3. **Reference validation**: `on_error.target` and `goto` references point to existing step IDs
4. **Cycle detection**: DFS traversal ensures no circular step references
5. **Reachability**: All steps are reachable from the root step
6. **CEL expression compilation**: All `condition` fields parse as valid CEL expressions
7. **Template resolution**: All `{{ }}` templates reference known variable paths
8. **Timeout validation**: All `timeout` and `duration` fields parse as valid Go durations
9. **Connector validation**: All `call` step `action` fields reference registered connector types
