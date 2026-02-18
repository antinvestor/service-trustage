# Orchestrator Architecture Specification

**Contract-Driven State Transition Engine for Stawi.dev**

**Version:** 2.0
**Status:** Accepted

---

## 0. The Problem This System Solves

This is not a "workflow engine". It is a **distributed, contract-driven state transition system with externally implemented actions**.

Every piece of data flowing between states is formally specified, validated at runtime, and auditable. The engine owns all state transitions. No external system (NATS or worker processes) can unilaterally advance workflow state.

### 0.1 Design Trade-offs

This design optimizes for **contract-driven, multi-tenant, integration-heavy automation** — not arbitrary distributed computation. Key trade-offs:

- **No deterministic replay**: Interrupted executions do not replay from history. Failed states retry from scratch. Target workloads (API calls, emails, webhooks) are fast (<5s) and idempotent, making this acceptable.
- **Declarative DSL only**: Workflows are not imperative code. The DSL is the authoring model.
- **~1 second timer precision**: Sufficient for delays measured in hours/days/weeks.

---

## 1. Design Goals

| Goal | Meaning | Enforcement Mechanism |
|------|---------|----------------------|
| **Fast integrations** | New partners ship working integrations in days, not weeks | Generated Worker SDK + contract test harness |
| **Strict data requirements** | Every state declares exactly what data it needs and produces | Schema registry + runtime validation |
| **Machine-verifiable correctness** | Workflow definitions pass static analysis before deployment | Transition validator (DAG + schema + simulation) |
| **Always-on observability** | Every execution is traceable, every transition auditable | OpenTelemetry traces + append-only audit log |
| **Scalable execution** | Engine scales horizontally without coordination | Stateless engine nodes + `FOR UPDATE SKIP LOCKED` |
| **No hidden state** | All intermediate data is stored explicitly | `workflow_state_output` table, no implicit passthrough |

---

## 2. Technology Stack

| Layer | Technology | Access Via | Purpose |
|-------|-----------|-----------|---------|
| Framework | github.com/pitabwire/frame | Direct | Service lifecycle, abstractions |
| API | ConnectRPC | Generated from proto/ | All client and service APIs |
| Database | PostgreSQL | Frame `datastore.Manager` | **Single source of truth** for all state |
| Message Queue | NATS JetStream | Frame `queue.Manager` | Transport only (delivery + redelivery) |
| Cache | Valkey | Frame `cache.Manager` | Schema cache, rate limits, quotas |
| Expressions | CEL | `github.com/google/cel-go` | Conditions, filters, mappings |
| Auth | OIDC | Frame `security.Manager` | Authentication |
| Observability | OpenTelemetry | Frame + custom | Traces, metrics, audit log |

**Critical constraint**: PostgreSQL is the only system that decides state. NATS is a notification mechanism. It never decides retries, ordering, validity, or ownership.

---

## 3. Architecture Layers

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Interaction Plane                             │
│   ConnectRPC Handlers (Form, Workflow, Connector, Trigger, Event)   │
│   Worker SDK Façade (state execution callback API)                  │
└────────────────────────────┬────────────────────────────────────────┘
                             │
┌────────────────────────────▼────────────────────────────────────────┐
│                         Control Plane                                │
│   Schema Registry    Transition Validator    Quota Manager           │
│   Mapping Engine     DSL Validator           Credential Manager      │
└────────────────────────────┬────────────────────────────────────────┘
                             │
┌────────────────────────────▼────────────────────────────────────────┐
│                        Execution Plane                               │
│   State Engine          Dispatch Scheduler     Retry Scheduler       │
│   (CAS transitions)    (FOR UPDATE SKIP       (backoff policies)    │
│                          LOCKED)                                     │
└────────────────────────────┬────────────────────────────────────────┘
                             │
┌────────────────────────────▼────────────────────────────────────────┐
│                          Data Plane                                  │
│   PostgreSQL: workflow instances, executions, outputs, schemas,      │
│               audit events, mappings, retry policies                 │
│   NATS: execution_id delivery (transport only)                      │
│   Valkey: schema cache, rate limits, credential cache               │
└────────────────────────────┬────────────────────────────────────────┘
                             │
┌────────────────────────────▼────────────────────────────────────────┐
│                       Integration Plane                              │
│   Connector Adapters (webhook, email, HTTP, CRM, etc.)              │
│   Generated Worker SDK    Contract Test Harness                     │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 4. Strongly-Typed, Versioned Workflow Contracts

### 4.1 The Core Invariant

> **No state executes unless its input has been validated against a registered schema and produced by a registered mapping from the previous state.**

This single rule prevents large automation platforms from silently decaying into unmaintainable, non-verifiable systems.

### 4.2 State Contracts

Every state transition has three formally defined contracts:

```
InputSchema   — what data this state requires to execute
OutputSchema  — what data this state produces on success
ErrorSchema   — what data this state produces on failure
```

Schemas use **JSON Schema (draft 2020-12)** for external-facing states and **Protobuf** for internal engine states. JSON Schema is chosen for the primary format because:

- Partners author schemas in their own toolchains
- Visual builder renders forms from JSON Schema
- AI generates schemas as structured output
- Protobuf can be derived from JSON Schema for internal speed

### 4.3 Schema Registry

The schema registry is an internal component, not a separate service. It stores immutable schema versions and provides compile-time-like validation for workflow definitions.

```sql
CREATE TABLE workflow_state_schemas (
    id              VARCHAR(50) PRIMARY KEY,
    tenant_id       VARCHAR(50) NOT NULL,
    partition_id    VARCHAR(50) NOT NULL,
    workflow_name   VARCHAR(255) NOT NULL,
    workflow_version INT NOT NULL,
    state           VARCHAR(255) NOT NULL,
    schema_type     VARCHAR(10) NOT NULL CHECK (schema_type IN ('input', 'output', 'error')),
    schema_hash     VARCHAR(64) NOT NULL,       -- SHA-256 of schema_blob
    schema_blob     JSONB NOT NULL,             -- JSON Schema document (immutable)
    created_at      TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE (tenant_id, workflow_name, workflow_version, state, schema_type)
);

CREATE INDEX idx_wss_tenant ON workflow_state_schemas(tenant_id, partition_id);
CREATE INDEX idx_wss_lookup ON workflow_state_schemas(tenant_id, workflow_name, workflow_version, state);
CREATE INDEX idx_wss_hash ON workflow_state_schemas(schema_hash);
```

**Immutability guarantee**: Schemas are write-once. To change a schema, create a new workflow version. This prevents running instances from having their contracts changed underneath them.

### 4.4 Runtime Validation

**Before dispatch** (engine validates input):

```
1. Look up state input schema by (workflow_name, workflow_version, state)
2. Validate input payload against schema
3. If invalid → mark execution as 'invalid_input_contract', do not dispatch
```

**Before transition commit** (engine validates output):

```
1. Look up state output schema by (workflow_name, workflow_version, state)
2. Validate worker output against schema
3. If invalid → mark execution as 'invalid_output_contract', do not transition
```

This makes correctness machine-checkable at every boundary.

---

## 5. Strict Data Channeling Between States

### 5.1 No Implicit Passthrough

The biggest long-term failure in automation systems is "states passing arbitrary blobs". This design prohibits it.

**State output is the only legal input for the next state.** There is no ambient context, no global variables, no implicit data inheritance.

```sql
-- Stores the validated output of each state execution
CREATE TABLE workflow_state_outputs (
    id              VARCHAR(50) PRIMARY KEY,
    tenant_id       VARCHAR(50) NOT NULL,
    partition_id    VARCHAR(50) NOT NULL,
    execution_id    VARCHAR(50) NOT NULL REFERENCES workflow_state_executions(execution_id),
    instance_id     VARCHAR(50) NOT NULL REFERENCES workflow_instances(id),
    state           VARCHAR(255) NOT NULL,
    schema_hash     VARCHAR(64) NOT NULL,       -- Hash of output schema used for validation
    payload         JSONB NOT NULL,             -- Validated output data
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_wso_tenant ON workflow_state_outputs(tenant_id, partition_id);
CREATE INDEX idx_wso_instance ON workflow_state_outputs(instance_id, state);
CREATE INDEX idx_wso_execution ON workflow_state_outputs(execution_id);
```

### 5.2 Explicit State Mappings

Data flows between states through declared, validated mappings:

```sql
CREATE TABLE workflow_state_mappings (
    id              VARCHAR(50) PRIMARY KEY,
    tenant_id       VARCHAR(50) NOT NULL,
    partition_id    VARCHAR(50) NOT NULL,
    workflow_name   VARCHAR(255) NOT NULL,
    workflow_version INT NOT NULL,
    from_state      VARCHAR(255) NOT NULL,
    to_state        VARCHAR(255) NOT NULL,
    mapping_expr    JSONB NOT NULL,             -- Field mapping expressions
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    modified_at     TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE (tenant_id, workflow_name, workflow_version, from_state, to_state)
);

CREATE INDEX idx_wsm_tenant ON workflow_state_mappings(tenant_id, partition_id);
CREATE INDEX idx_wsm_lookup ON workflow_state_mappings(
    tenant_id, workflow_name, workflow_version, from_state, to_state
);
```

**Mapping expression format** (JSONPath-like with CEL for transformations):

```json
{
  "user_id": "$.identity.user_id",
  "risk_score": "$.risk.score",
  "full_name": "$.first_name + ' ' + $.last_name",
  "is_high_risk": "$.risk.score > 80"
}
```

### 5.3 Mapping Evaluation and Validation

When transitioning from state A to state B:

```
1. Load output of state A from workflow_state_outputs
2. Load mapping (A → B) from workflow_state_mappings
3. Evaluate each mapping expression against A's output
4. Validate mapped result against B's input schema
5. If any required field is missing → reject transition before execution
6. If validation passes → use mapped result as B's input
```

The mapping engine is implemented in the `dsl/` package (pure Go, no infrastructure dependencies) using CEL for expression evaluation.

---

## 6. State Execution as a Deterministic Command

Workers never receive ad-hoc payloads. They receive a structured, verifiable command envelope:

```go
// ExecutionCommand is the immutable instruction sent to a worker.
// The worker SDK deserializes this; integrators never construct it manually.
type ExecutionCommand struct {
    ExecutionID     string          `json:"execution_id"`
    InstanceID      string          `json:"instance_id"`
    TenantID        string          `json:"tenant_id"`
    Workflow        string          `json:"workflow"`
    WorkflowVersion int             `json:"workflow_version"`
    State           string          `json:"state"`
    StateVersion    int             `json:"state_version"`
    Attempt         int             `json:"attempt"`
    InputPayload    json.RawMessage `json:"input_payload"`
    InputSchemaHash string          `json:"input_schema_hash"`
    ExecutionToken  string          `json:"execution_token"`
    TraceID         string          `json:"trace_id"`
}
```

The `ExecutionToken` is generated and persisted by the engine at dispatch time. Workers must present it when committing results. This prevents:

- Stale workers from committing results for re-dispatched executions
- Workers from committing results for executions they were not assigned
- Replay attacks against the commit endpoint

---

## 7. PostgreSQL as the Correctness Boundary

### 7.1 Core Tables

PostgreSQL is the **single source of truth** for all workflow state. No state transitions occur outside of PostgreSQL transactions.

```sql
-- Workflow instance: the running "copy" of a workflow definition
CREATE TABLE workflow_instances (
    id              VARCHAR(50) PRIMARY KEY,
    tenant_id       VARCHAR(50) NOT NULL,
    partition_id    VARCHAR(50) NOT NULL,
    workflow_name   VARCHAR(255) NOT NULL,
    workflow_version INT NOT NULL,
    current_state   VARCHAR(255) NOT NULL,
    status          VARCHAR(30) NOT NULL DEFAULT 'running'
                    CHECK (status IN ('running', 'completed', 'failed', 'cancelled', 'suspended')),
    revision        BIGINT NOT NULL DEFAULT 1,       -- CAS revision counter
    trigger_event_id VARCHAR(50),                     -- Event that started this instance
    metadata        JSONB DEFAULT '{}',               -- Workflow-level metadata
    started_at      TIMESTAMPTZ DEFAULT NOW(),
    finished_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    modified_at     TIMESTAMPTZ DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX idx_wi_tenant ON workflow_instances(tenant_id, partition_id);
CREATE INDEX idx_wi_status ON workflow_instances(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_wi_workflow ON workflow_instances(tenant_id, workflow_name, workflow_version);
CREATE INDEX idx_wi_trigger ON workflow_instances(trigger_event_id) WHERE trigger_event_id IS NOT NULL;


-- State execution: each attempt to execute a state
CREATE TABLE workflow_state_executions (
    execution_id    VARCHAR(50) PRIMARY KEY,
    tenant_id       VARCHAR(50) NOT NULL,
    partition_id    VARCHAR(50) NOT NULL,
    instance_id     VARCHAR(50) NOT NULL REFERENCES workflow_instances(id),
    state           VARCHAR(255) NOT NULL,
    state_version   INT NOT NULL DEFAULT 1,
    attempt         INT NOT NULL DEFAULT 1,
    status          VARCHAR(30) NOT NULL DEFAULT 'pending'
                    CHECK (status IN (
                        'pending',                  -- Awaiting dispatch
                        'dispatched',               -- Sent to worker via NATS
                        'running',                  -- Worker has acknowledged
                        'completed',                -- Worker returned valid output
                        'failed',                   -- Worker returned error (retryable)
                        'fatal',                    -- Worker returned fatal error (no retry)
                        'timed_out',                -- Execution exceeded deadline
                        'invalid_input_contract',   -- Input failed schema validation
                        'invalid_output_contract',  -- Output failed schema validation
                        'stale',                    -- Superseded by newer execution
                        'retry_scheduled'           -- Awaiting retry
                    )),
    execution_token VARCHAR(64) NOT NULL,           -- One-time token for commit auth
    input_schema_hash VARCHAR(64) NOT NULL,
    output_schema_hash VARCHAR(64),                 -- Set after commit
    error_class     VARCHAR(30),                    -- retryable, fatal, compensatable, external_dependency
    error_message   TEXT,
    next_retry_at   TIMESTAMPTZ,                    -- When retry is due (if retry_scheduled)
    trace_id        VARCHAR(64),                    -- OpenTelemetry trace ID
    started_at      TIMESTAMPTZ,
    finished_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_wse_tenant ON workflow_state_executions(tenant_id, partition_id);
CREATE INDEX idx_wse_instance ON workflow_state_executions(instance_id, state);
CREATE INDEX idx_wse_pending ON workflow_state_executions(status, created_at)
    WHERE status = 'pending';
CREATE INDEX idx_wse_retry ON workflow_state_executions(next_retry_at)
    WHERE status = 'retry_scheduled';
CREATE INDEX idx_wse_dispatched ON workflow_state_executions(status, created_at)
    WHERE status = 'dispatched';
CREATE INDEX idx_wse_trace ON workflow_state_executions(trace_id);
```

### 7.2 Strict Transition CAS (Compare-And-Swap)

Every state transition uses optimistic concurrency control:

```sql
UPDATE workflow_instances
SET current_state = $new_state,
    revision = revision + 1,
    modified_at = NOW()
WHERE id = $instance_id
  AND tenant_id = $tenant_id
  AND current_state = $expected_state
  AND revision = $expected_revision
  AND status = 'running';
```

If zero rows affected → **stale execution**. The engine marks the execution as `stale` and does not advance the workflow.

This eliminates race conditions between concurrent workers, retry schedulers, and cancellation requests. No distributed locks are needed.

### 7.3 Two Transaction Types

All workflow state mutations happen in exactly one of two transaction patterns:

**Dispatch Transaction** (engine → worker):

```
BEGIN;
  -- Create execution record
  INSERT INTO workflow_state_executions (...) VALUES (...);
  -- Validate input against schema (application-level, within same tx)
  -- If invalid: set status = 'invalid_input_contract', COMMIT, done.
  -- If valid: set status = 'pending'
COMMIT;
-- Then publish execution_id to NATS (outside transaction)
```

**Commit Transaction** (worker → engine):

```
BEGIN;
  -- Verify execution token
  SELECT ... FROM workflow_state_executions
    WHERE execution_id = $id AND execution_token = $token AND status = 'dispatched'
    FOR UPDATE;
  -- Validate output against schema (application-level)
  -- If invalid: set status = 'invalid_output_contract', COMMIT, done.
  -- Evaluate mapping to next state
  -- Validate mapped input against next state's input schema
  -- CAS transition on workflow_instances
  -- If CAS fails: set execution status = 'stale', COMMIT, done.
  -- Store validated output in workflow_state_outputs
  -- Insert audit event
  -- Create next state execution (status = 'pending')
COMMIT;
-- Then publish next execution_id to NATS (outside transaction)
```

Nothing else mutates workflow state. This constraint is enforced by making the mutation functions the only write path in the repository layer.

---

## 8. NATS Usage: Transport Only

NATS JetStream serves exactly one purpose: **delivering execution IDs to workers**.

### 8.1 What NATS Carries

```json
{
  "execution_id": "exec_abc123"
}
```

That is the entire message payload. Everything else is reloaded from PostgreSQL by the worker.

### 8.2 What NATS Does NOT Decide

| Concern | Decided By |
|---------|-----------|
| Retry count | PostgreSQL `workflow_retry_policy` table |
| Retry timing | PostgreSQL `next_retry_at` column + retry scheduler |
| Message ordering | PostgreSQL `revision` column + CAS |
| Execution validity | PostgreSQL `execution_token` verification |
| Ownership | PostgreSQL `FOR UPDATE SKIP LOCKED` |

### 8.3 Subject Design

```
wf.exec.<workflow_name>.<state>
```

Examples:

```
wf.exec.loan_onboarding.verify_identity
wf.exec.lead_nurture.send_welcome
wf.exec.order_fulfillment.check_inventory
```

This enables:

- **Selective scaling**: dedicated worker pools per state
- **Back-pressure control**: per-state `max_ack_pending` tuning
- **Monitoring**: per-state throughput and latency metrics

### 8.4 NATS JetStream Configuration

| Setting | Value | Rationale |
|---------|-------|-----------|
| Stream name | `wf-executions` | Single stream for all workflow executions |
| Subjects | `wf.exec.>` | Wildcard for all workflows and states |
| Retention | Limits (24h, 10GB max) | Short retention — NATS is transport, not storage |
| Storage | File | Survives NATS restarts |
| Consumer per state | Durable, explicit ACK | Worker pools per state type |
| Max deliver | 3 | Limited redelivery — retry logic is in PostgreSQL |
| ACK wait | 30s | Worker timeout before redelivery |
| Max ACK pending | Configurable per state | Back-pressure control |

### 8.5 Event Ingestion Stream (Separate)

The event ingestion stream from the original design remains:

```
wf.events.<tenant_id>.<event_type>
```

This handles form submissions, webhooks, and schedule triggers — routing them to the engine for workflow instance creation. This stream has longer retention (30 days) for replay and debugging.

---

## 9. Engine-Owned Retry and Backoff

Retry logic lives entirely in PostgreSQL, making it testable, observable, and deterministic.

### 9.1 Retry Policy Table

```sql
CREATE TABLE workflow_retry_policies (
    id              VARCHAR(50) PRIMARY KEY,
    tenant_id       VARCHAR(50) NOT NULL,
    partition_id    VARCHAR(50) NOT NULL,
    workflow_name   VARCHAR(255) NOT NULL,
    workflow_version INT NOT NULL,
    state           VARCHAR(255) NOT NULL,
    max_attempts    INT NOT NULL DEFAULT 3,
    backoff_strategy VARCHAR(20) NOT NULL DEFAULT 'exponential'
                    CHECK (backoff_strategy IN ('fixed', 'linear', 'exponential')),
    initial_delay   INTERVAL NOT NULL DEFAULT '1 second',
    max_delay       INTERVAL NOT NULL DEFAULT '5 minutes',
    retry_on        TEXT[] NOT NULL DEFAULT ARRAY['retryable', 'external_dependency'],
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    modified_at     TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE (tenant_id, workflow_name, workflow_version, state)
);

CREATE INDEX idx_wrp_tenant ON workflow_retry_policies(tenant_id, partition_id);
CREATE INDEX idx_wrp_lookup ON workflow_retry_policies(
    tenant_id, workflow_name, workflow_version, state
);
```

### 9.2 Retry Scheduler

A background process (running within the Frame service) polls for due retries:

```sql
SELECT execution_id, instance_id, state, attempt
FROM workflow_state_executions
WHERE status = 'retry_scheduled'
  AND next_retry_at <= NOW()
ORDER BY next_retry_at
FOR UPDATE SKIP LOCKED
LIMIT 100;
```

For each due retry:

1. Create new execution record (attempt + 1)
2. Mark previous execution as `stale`
3. Validate input against schema (input may have been re-mapped)
4. Set new execution to `pending`
5. Publish `execution_id` to NATS

### 9.3 Backoff Calculation

```go
func calculateDelay(policy *RetryPolicy, attempt int) time.Duration {
    switch policy.BackoffStrategy {
    case "fixed":
        return policy.InitialDelay
    case "linear":
        delay := policy.InitialDelay * time.Duration(attempt)
        return min(delay, policy.MaxDelay)
    case "exponential":
        delay := policy.InitialDelay * time.Duration(1<<uint(attempt-1))
        return min(delay, policy.MaxDelay)
    default:
        return policy.InitialDelay
    }
}
```

### 9.4 Error Classification

Workers **must** classify every error. The engine uses classification to determine behavior — never string matching.

```go
// ErrorClass is the exhaustive set of error classifications.
// Workers must return exactly one of these for every failure.
type ErrorClass string

const (
    // Retryable: transient failure, safe to retry (network timeout, 503, etc.)
    ErrorRetryable ErrorClass = "retryable"

    // Fatal: permanent failure, do not retry (400, invalid config, business logic rejection)
    ErrorFatal ErrorClass = "fatal"

    // Compensatable: failure that requires running a compensation workflow
    ErrorCompensatable ErrorClass = "compensatable"

    // ExternalDependency: third-party system is down, retry with longer backoff
    ErrorExternalDependency ErrorClass = "external_dependency"
)
```

The Worker SDK forces developers to classify errors at compile time:

```go
// Workers return typed results, not raw errors
func ExecuteVerifyIdentity(
    ctx context.Context,
    in VerifyIdentityInput,
) (*VerifyIdentityOutput, *ExecutionError) {
    // ExecutionError requires ErrorClass — cannot be constructed without it
}
```

---

## 10. Transition Validator: Compile-Time Correctness for Workflows

The transition validator is the most important quality gate in the system. No workflow definition is accepted unless it passes all validation phases.

### 10.1 Static Validation

Performed synchronously when a workflow definition is saved:

| Check | Failure Mode | Error |
|-------|-------------|-------|
| **DAG validation** | Cycles in state graph | `CYCLE_DETECTED: state_a → state_b → state_a` |
| **Reachability** | Orphaned states | `UNREACHABLE_STATE: state_x has no incoming transitions` |
| **Schema existence** | Missing schema definitions | `MISSING_SCHEMA: state verify_identity has no input schema` |
| **Mapping existence** | Missing state mappings | `MISSING_MAPPING: no mapping from state_a to state_b` |
| **Mapping compatibility** | Output fields don't cover input requirements | `INCOMPATIBLE_MAPPING: state_b requires 'user_id' but state_a output has no mapping for it` |
| **State name conflicts** | Duplicate state names | `DUPLICATE_STATE: 'verify_identity' appears twice` |
| **Expression compilation** | Invalid CEL syntax | `INVALID_EXPRESSION: line 1, col 15: undeclared reference 'foo'` |
| **Template validation** | Unresolvable `{{ }}` references | `INVALID_TEMPLATE: '{{ payload.missing }}' references undeclared field` |
| **Retry policy validation** | Invalid backoff configuration | `INVALID_RETRY: max_attempts must be > 0` |
| **Timeout validation** | State timeout > workflow timeout | `INVALID_TIMEOUT: state timeout (2h) exceeds workflow timeout (1h)` |

### 10.2 Simulation Validation

After static checks pass, the validator generates synthetic inputs and simulates the workflow:

```
1. Generate synthetic input from the workflow's initial input schema
   (using schema-aware generators: valid strings, numbers, objects)
2. For each state in topological order:
   a. Validate synthetic input against state input schema
   b. Generate synthetic output from state output schema
   c. Evaluate mapping to next state(s)
   d. Validate mapped result against next state's input schema
3. If any validation fails → block deployment with detailed error
```

This catches issues that static analysis cannot:

- Schema incompatibilities hidden behind optional fields
- Mapping expressions that produce the wrong types
- CEL expressions that reference fields that exist in the schema but are never mapped

### 10.3 Implementation Location

The transition validator lives in the `dsl/` package (zero infrastructure dependencies). It operates on in-memory data structures, not database rows. The business layer calls it before persisting workflow definitions.

---

## 11. Integration Acceleration Layer

### 11.1 Worker SDK Generator

From schema and state definitions, the system generates a typed Go SDK for each state:

```go
// Generated by orchestrator SDK generator
// DO NOT EDIT — regenerate with: orchestrator-sdk gen verify_identity

package verify_identity

// VerifyIdentityInput is the validated input for the verify_identity state.
// All fields match the registered input schema v3.
type VerifyIdentityInput struct {
    UserID      string `json:"user_id"`
    DocumentURL string `json:"document_url"`
    Country     string `json:"country"`
}

// VerifyIdentityOutput is the required output for the verify_identity state.
// All fields are validated against the registered output schema v3.
type VerifyIdentityOutput struct {
    Verified    bool    `json:"verified"`
    Score       float64 `json:"score"`
    RiskLevel   string  `json:"risk_level"`
    VerifiedAt  string  `json:"verified_at"`
}

// VerifyIdentityError is the structured error output for the verify_identity state.
type VerifyIdentityError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Details map[string]any `json:"details,omitempty"`
}

// Handler is the function signature that integrators implement.
// The SDK handles: heartbeat, execution token, retries, error classification,
// input/output validation. Integrators never touch envelopes.
type Handler func(ctx context.Context, in VerifyIdentityInput) (*VerifyIdentityOutput, *ExecutionError)
```

The SDK handles:

| Concern | SDK Responsibility |
|---------|-------------------|
| Execution command deserialization | Automatic |
| Input validation against schema | Automatic (before calling handler) |
| Output validation against schema | Automatic (after handler returns) |
| Execution token management | Automatic (presented on commit) |
| Heartbeat | Automatic (background goroutine) |
| Error classification | Forced by `ExecutionError` type |
| Retry semantics | Transparent (engine-driven) |
| OpenTelemetry context propagation | Automatic (trace_id from command) |

Integrators implement a single function with typed input and output. They never interact with the engine protocol directly.

### 11.2 Contract Test Harness

The system ships a test runner for state implementations:

```bash
orchestrator-test run verify_identity
```

This:

1. Boots a local engine (in-process, no external dependencies)
2. Loads the exact schemas for the target state
3. Feeds generated inputs (from schema generators)
4. Feeds recorded inputs (from production audit log, anonymized)
5. Validates produced outputs against output schema
6. Verifies error classification correctness
7. Tests retry semantics (simulated failures)
8. Reports schema compliance percentage

```bash
orchestrator-test run verify_identity

  ✓ Schema compliance: 100% (15/15 generated inputs)
  ✓ Output validation: 100% (15/15 outputs match schema)
  ✓ Error classification: 100% (5/5 errors properly classified)
  ✓ Retry behavior: exponential backoff, max 3 attempts
  ✗ Idempotency: WARN - handler produces different output for same input
    → Input: {"user_id": "u123", ...}
    → Output 1: {"score": 85.2, ...}
    → Output 2: {"score": 85.7, ...}

  14/15 checks passed. 1 warning.
```

This is the fastest way to onboard new integration teams. They can validate their implementation without deploying to any environment.

---

## 12. Workflow Definition Model

### 12.1 Enhanced DSL Structure

The DSL evolves from the original design to include contract declarations:

```json
{
  "version": "2.0",
  "name": "Loan Onboarding",
  "description": "End-to-end loan application processing",
  "timeout": "30d",
  "input_schema": {
    "type": "object",
    "required": ["applicant_id", "loan_amount"],
    "properties": {
      "applicant_id": { "type": "string" },
      "loan_amount": { "type": "number", "minimum": 0 }
    }
  },
  "states": [
    {
      "name": "verify_identity",
      "type": "action",
      "action": "identity.verify",
      "input_schema": { "$ref": "#/schemas/verify_identity_input" },
      "output_schema": { "$ref": "#/schemas/verify_identity_output" },
      "error_schema": { "$ref": "#/schemas/verify_identity_error" },
      "retry": {
        "max_attempts": 3,
        "backoff": "exponential",
        "initial_delay": "5s",
        "max_delay": "2m",
        "retry_on": ["retryable", "external_dependency"]
      },
      "timeout": "5m",
      "transitions": {
        "on_success": "assess_risk",
        "on_fatal": "manual_review"
      }
    },
    {
      "name": "assess_risk",
      "type": "action",
      "action": "risk.assess",
      "input_schema": { "$ref": "#/schemas/assess_risk_input" },
      "output_schema": { "$ref": "#/schemas/assess_risk_output" },
      "transitions": {
        "on_success": [
          {
            "condition": "output.risk_level == 'low'",
            "target": "auto_approve"
          },
          {
            "condition": "output.risk_level == 'medium'",
            "target": "manual_review"
          },
          {
            "condition": "output.risk_level == 'high'",
            "target": "reject"
          }
        ]
      }
    },
    {
      "name": "auto_approve",
      "type": "action",
      "action": "loan.approve",
      "transitions": { "on_success": "_end" }
    },
    {
      "name": "manual_review",
      "type": "wait_signal",
      "signal": "review_decision",
      "timeout": "7d",
      "transitions": {
        "on_signal": [
          { "condition": "signal.decision == 'approve'", "target": "auto_approve" },
          { "condition": "signal.decision == 'reject'", "target": "reject" }
        ],
        "on_timeout": "reject"
      }
    },
    {
      "name": "reject",
      "type": "action",
      "action": "loan.reject",
      "transitions": { "on_success": "_end" }
    }
  ],
  "mappings": {
    "verify_identity -> assess_risk": {
      "user_id": "$.user_id",
      "identity_score": "$.score",
      "loan_amount": "$workflow.input.loan_amount"
    },
    "assess_risk -> auto_approve": {
      "applicant_id": "$workflow.input.applicant_id",
      "loan_amount": "$workflow.input.loan_amount",
      "risk_level": "$.risk_level"
    }
  },
  "schemas": {
    "verify_identity_input": {
      "type": "object",
      "required": ["user_id", "document_url"],
      "properties": {
        "user_id": { "type": "string" },
        "document_url": { "type": "string", "format": "uri" }
      }
    },
    "verify_identity_output": {
      "type": "object",
      "required": ["verified", "score", "risk_level"],
      "properties": {
        "verified": { "type": "boolean" },
        "score": { "type": "number", "minimum": 0, "maximum": 100 },
        "risk_level": { "type": "string", "enum": ["low", "medium", "high"] }
      }
    }
  }
}
```

### 12.2 State Types

| Type | Purpose | Engine Behavior |
|------|---------|----------------|
| `action` | Execute a connector adapter | Dispatch to worker, validate I/O, commit result |
| `delay` | Durable wait | Engine creates timer record, scheduler fires at deadline |
| `condition` | Branch based on CEL expression | Engine evaluates expression, selects transition |
| `wait_signal` | Wait for external signal (approval) | Engine suspends instance, resumes on signal receipt |
| `parallel` | Execute multiple states concurrently | Engine dispatches all branches, waits for all/any completion |
| `foreach` | Iterate over a collection | Engine creates sub-executions per item |
| `sub_workflow` | Start a child workflow | Engine creates child instance, waits for completion |

### 12.3 Transition Types

| Transition | Trigger | Resolution |
|-----------|---------|-----------|
| `on_success` | State completed with valid output | Direct target or conditional list |
| `on_fatal` | State failed with fatal error | Direct target or `_end` |
| `on_timeout` | State exceeded timeout | Direct target or `_end` |
| `on_signal` | External signal received | Conditional list based on signal payload |
| `_end` | Workflow completion | Terminal state — set instance status |

---

## 13. Observability

### 13.1 Trace Model

Every execution creates a hierarchy of OpenTelemetry spans:

```
workflow.instance (root span)
  ├── workflow.dispatch (engine creates execution)
  ├── workflow.state.verify_identity
  │   ├── state.validate_input
  │   ├── state.dispatch (publish to NATS)
  │   ├── state.worker.execute (worker-side span)
  │   ├── state.validate_output
  │   ├── state.evaluate_mapping
  │   └── state.commit_transition
  ├── workflow.state.assess_risk
  │   └── ... (same structure)
  ├── workflow.retry.schedule (if retry needed)
  └── workflow.complete
```

The `trace_id` is stored in `workflow_state_executions.trace_id`, enabling:

- Correlation between engine logs, worker logs, and connector logs
- Trace search by execution_id, instance_id, or tenant_id
- End-to-end latency measurement per state and per workflow

### 13.2 Metrics (Cardinality-Safe)

Only low-cardinality labels are used. Never use instance_id, execution_id, or user-specific values as metric labels.

| Metric | Type | Labels | Purpose |
|--------|------|--------|---------|
| `engine.executions.total` | Counter | tenant, workflow, state, status | Execution volume |
| `engine.execution.latency_ms` | Histogram | tenant, workflow, state | Per-state execution time |
| `engine.transitions.total` | Counter | tenant, workflow, from_state, to_state | Transition volume |
| `engine.retries.total` | Counter | tenant, workflow, state, error_class | Retry volume by error type |
| `engine.contract_violations.total` | Counter | tenant, workflow, state, violation_type | Schema validation failures |
| `engine.stale_executions.total` | Counter | tenant, workflow, state | CAS conflict count |
| `engine.dispatch.latency_ms` | Histogram | tenant, workflow, state | Time from pending to dispatched |
| `engine.commit.latency_ms` | Histogram | tenant, workflow, state | Time from dispatched to committed |
| `engine.instances.active` | Gauge | tenant, workflow | Running instance count |
| `connector.calls.total` | Counter | tenant, connector_type, status | Connector invocation volume |
| `connector.latency_ms` | Histogram | tenant, connector_type | Connector response time |
| `events.ingested.total` | Counter | tenant, event_type | Event ingestion volume |
| `events.routed.total` | Counter | tenant, event_type, workflows_matched | Event routing fan-out |
| `quotas.exceeded.total` | Counter | tenant, quota_type | Quota rejections |

### 13.3 Deterministic Audit Stream

Every state transition produces an append-only audit event:

```sql
CREATE TABLE workflow_audit_events (
    id              VARCHAR(50) PRIMARY KEY,
    tenant_id       VARCHAR(50) NOT NULL,
    partition_id    VARCHAR(50) NOT NULL,
    instance_id     VARCHAR(50) NOT NULL,
    execution_id    VARCHAR(50),
    event_type      VARCHAR(50) NOT NULL,       -- dispatched, completed, failed, retried, transitioned, etc.
    state           VARCHAR(255),
    from_state      VARCHAR(255),
    to_state        VARCHAR(255),
    payload         JSONB DEFAULT '{}',         -- Event-specific data (never credentials or PII)
    trace_id        VARCHAR(64),
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_wae_tenant ON workflow_audit_events(tenant_id, partition_id);
CREATE INDEX idx_wae_instance ON workflow_audit_events(instance_id, created_at);
CREATE INDEX idx_wae_type ON workflow_audit_events(event_type);
CREATE INDEX idx_wae_created ON workflow_audit_events(created_at DESC);
```

Audit event types:

| Event Type | Meaning |
|-----------|---------|
| `instance.created` | New workflow instance started |
| `instance.completed` | Workflow reached terminal state |
| `instance.failed` | Workflow failed permanently |
| `instance.cancelled` | Workflow cancelled by user/system |
| `state.dispatched` | Execution dispatched to worker |
| `state.running` | Worker acknowledged execution |
| `state.completed` | Worker returned valid output |
| `state.failed` | Worker returned classified error |
| `state.retried` | Retry scheduled for failed execution |
| `state.timed_out` | Execution exceeded deadline |
| `state.contract_violation` | Input or output failed schema validation |
| `transition.committed` | State transition committed (CAS success) |
| `transition.rejected` | State transition rejected (CAS failure) |
| `signal.received` | External signal received for wait state |
| `mapping.evaluated` | Data mapped from state A to state B |

This stream feeds:

- **Live UI**: real-time workflow execution visualization
- **Debugging tools**: deterministic timeline reconstruction
- **Compliance exports**: immutable audit trail for regulated environments

---

## 14. Online Inspection and Replay-Free Debugging

Because all intermediate payloads are stored explicitly, you do not need replay to understand behavior.

### 14.1 Read-Only Query API

The `EventService` ConnectRPC handler exposes:

| RPC | Purpose |
|-----|---------|
| `GetInstanceTimeline` | Ordered list of audit events for an instance |
| `GetStateExecutions` | All execution attempts for a specific state |
| `GetExecutionDetail` | Input payload, output payload, schema versions, error details |
| `GetStateOutput` | Validated output of a specific state execution |
| `GetMappingResult` | Result of mapping evaluation between two states |
| `GetContractViolations` | All schema validation failures for an instance |
| `ListActiveInstances` | Running instances with current state and duration |

### 14.2 Why This Matters

All intermediate data is stored explicitly and self-describing (schema-tagged). Operators reconstruct exact data flows with standard PostgreSQL queries — no code replay, no specialized tools. For regulated environments (financial services, healthcare), this is a compliance advantage.

---

## 15. Scalability Model

### 15.1 Horizontal Engine Scaling

Engine nodes are stateless. All coordination uses PostgreSQL advisory locks and `FOR UPDATE SKIP LOCKED`.

```
┌─────────────┐  ┌─────────────┐  ┌─────────────┐
│  Engine #1  │  │  Engine #2  │  │  Engine #3  │
│             │  │             │  │             │
│ - Dispatch  │  │ - Dispatch  │  │ - Dispatch  │
│   scheduler │  │   scheduler │  │   scheduler │
│ - Retry     │  │ - Retry     │  │ - Retry     │
│   scheduler │  │   scheduler │  │   scheduler │
│ - Timer     │  │ - Timer     │  │ - Timer     │
│   scheduler │  │   scheduler │  │   scheduler │
│ - Event     │  │ - Event     │  │ - Event     │
│   router    │  │   router    │  │   router    │
└──────┬──────┘  └──────┬──────┘  └──────┬──────┘
       │                │                │
       └────────────────┼────────────────┘
                        │
                 ┌──────▼──────┐
                 │  PostgreSQL │
                 │  (single    │
                 │   writer)   │
                 └─────────────┘
```

Each scheduler independently polls for work:

```sql
-- Dispatch scheduler
SELECT execution_id FROM workflow_state_executions
WHERE status = 'pending'
ORDER BY created_at
FOR UPDATE SKIP LOCKED
LIMIT 50;

-- Retry scheduler
SELECT execution_id FROM workflow_state_executions
WHERE status = 'retry_scheduled' AND next_retry_at <= NOW()
ORDER BY next_retry_at
FOR UPDATE SKIP LOCKED
LIMIT 50;

-- Timer scheduler (for delay states)
SELECT id FROM workflow_timers
WHERE fires_at <= NOW() AND status = 'pending'
ORDER BY fires_at
FOR UPDATE SKIP LOCKED
LIMIT 50;
```

`SKIP LOCKED` ensures multiple engine nodes never process the same row. No distributed lock manager needed.

### 15.2 Partitioning Strategy

For large deployments, partition hot tables by time:

```sql
-- Partition workflow_state_executions by month
CREATE TABLE workflow_state_executions (
    ...
) PARTITION BY RANGE (created_at);

CREATE TABLE wse_2026_01 PARTITION OF workflow_state_executions
    FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');
CREATE TABLE wse_2026_02 PARTITION OF workflow_state_executions
    FOR VALUES FROM ('2026-02-01') TO ('2026-03-01');
```

Use partial indexes on hot statuses:

```sql
CREATE INDEX idx_wse_pending ON workflow_state_executions(created_at)
    WHERE status = 'pending';
CREATE INDEX idx_wse_retry ON workflow_state_executions(next_retry_at)
    WHERE status = 'retry_scheduled';
```

This keeps scheduler queries fast regardless of total table size.

### 15.3 Fast Execution Path

The hot path for a single state execution:

```
1. Read execution row (single indexed lookup)
2. Validate schema hash (in-memory cache hit)
3. Publish execution_id to NATS (single message)
4. Worker executes (external)
5. Commit transaction (single indexed update + insert)
```

There are no joins across history tables, no fan-out, and no scans on large tables in the hot path. All heavy inspection queries (timeline, audit, debugging) run against secondary indexes on cold data.

### 15.4 Worker Scaling

Workers scale independently per state type:

```
┌──────────────────────┐  ┌──────────────────────┐
│ Worker Pool:         │  │ Worker Pool:         │
│ verify_identity      │  │ send_email           │
│                      │  │                      │
│ 10 instances         │  │ 5 instances          │
│ NATS consumer:       │  │ NATS consumer:       │
│ wf.exec.*.verify_*   │  │ wf.exec.*.send_*     │
│ max_ack_pending: 20  │  │ max_ack_pending: 50  │
└──────────────────────┘  └──────────────────────┘
```

---

## 16. Versioning and Evolution

### 16.1 Three Version Axes

| Axis | What Changes | Running Instance Behavior |
|------|-------------|--------------------------|
| **Workflow version** | State graph, transitions, mappings | Running instances reference `(workflow_name, workflow_version)` — never change mid-execution |
| **State version** | Schema, retry policy, timeout | Engine uses state_version from execution record — immutable per execution |
| **Schema version** | Contract fields, types, constraints | Schema identified by `schema_hash` — immutable per execution |

### 16.2 Upgrade Strategy

Upgrades are done by routing, not mutation:

```
1. Create new workflow version (v2) with updated schemas and mappings
2. Update trigger bindings to point to v2 (new instances use v2)
3. Existing v1 instances continue executing with v1 schemas
4. Optionally: create migration workflow that reads v1 outputs and feeds v2
```

Definitions are never mutated. Old versions are retained indefinitely for:

- Running instances that reference them
- Audit trail reconstruction
- Debugging historical executions

---

## 17. Security and Tenant Isolation

### 17.1 Tenant Isolation Matrix

| Layer | Mechanism | Enforcement |
|-------|-----------|-------------|
| PostgreSQL | `tenant_id NOT NULL` on every table | Repository layer adds `WHERE tenant_id = ?` to every query |
| NATS execution subjects | `wf.exec.<workflow>.<state>` | Workers verify tenant_id in ExecutionCommand |
| NATS event subjects | `wf.events.<tenant_id>.<type>` | Subject hierarchy prevents cross-tenant reads |
| Valkey keys | `orch:<tenant_id>:*` prefix | Key construction includes tenant_id |
| Execution commands | `tenant_id` field in every command | Worker SDK verifies tenant_id matches claim |
| Audit events | `tenant_id` on every row | All audit queries scoped by tenant |

### 17.2 Execution Token Security

```
1. Engine generates cryptographically random 64-byte token at dispatch
2. Token stored in workflow_state_executions (hashed)
3. Worker receives token in ExecutionCommand
4. Worker presents token when committing result
5. Engine verifies token matches → commit proceeds
6. Token is single-use → consumed after commit
```

This prevents:

- Stale workers from committing results for re-dispatched executions
- Cross-worker result injection
- Replay attacks against the commit API

### 17.3 Credential Security

Unchanged from the original design:

- At rest: AES-256-GCM with versioned master key
- In transit: TLS for all connections
- In cache: Valkey with 5-minute TTL
- In logs: never logged (only field names, never values)
- In audit events: never stored (only credential_id reference)
- In NATS messages: never included (only execution_id)

---

## 18. Timer and Delay Implementation

The engine implements durable timers via PostgreSQL:

```sql
CREATE TABLE workflow_timers (
    id              VARCHAR(50) PRIMARY KEY,
    tenant_id       VARCHAR(50) NOT NULL,
    partition_id    VARCHAR(50) NOT NULL,
    instance_id     VARCHAR(50) NOT NULL REFERENCES workflow_instances(id),
    execution_id    VARCHAR(50) NOT NULL REFERENCES workflow_state_executions(execution_id),
    state           VARCHAR(255) NOT NULL,
    fires_at        TIMESTAMPTZ NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending', 'fired', 'cancelled')),
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_wt_tenant ON workflow_timers(tenant_id, partition_id);
CREATE INDEX idx_wt_fires ON workflow_timers(fires_at)
    WHERE status = 'pending';
CREATE INDEX idx_wt_instance ON workflow_timers(instance_id);
```

The timer scheduler polls:

```sql
SELECT id, instance_id, execution_id, state
FROM workflow_timers
WHERE status = 'pending' AND fires_at <= NOW()
ORDER BY fires_at
FOR UPDATE SKIP LOCKED
LIMIT 100;
```

For each fired timer:

1. Mark timer as `fired`
2. Complete the delay state execution
3. Evaluate mapping and create next state execution
4. Publish to NATS

Timer precision: ~1 second (polling interval). For workflows with delays of hours/days/weeks, this is more than sufficient.

---

## 19. Signal (Wait for External Input) Implementation

Human-in-the-loop approvals are implemented as signal waits:

```sql
CREATE TABLE workflow_signals (
    id              VARCHAR(50) PRIMARY KEY,
    tenant_id       VARCHAR(50) NOT NULL,
    partition_id    VARCHAR(50) NOT NULL,
    instance_id     VARCHAR(50) NOT NULL REFERENCES workflow_instances(id),
    signal_name     VARCHAR(255) NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'waiting'
                    CHECK (status IN ('waiting', 'received', 'timed_out', 'cancelled')),
    payload         JSONB,                         -- Signal payload (set on receive)
    timeout_at      TIMESTAMPTZ,                   -- When to timeout
    received_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_ws_tenant ON workflow_signals(tenant_id, partition_id);
CREATE INDEX idx_ws_waiting ON workflow_signals(instance_id, signal_name)
    WHERE status = 'waiting';
CREATE INDEX idx_ws_timeout ON workflow_signals(timeout_at)
    WHERE status = 'waiting' AND timeout_at IS NOT NULL;
```

**Sending a signal** (via ConnectRPC `SendSignal` RPC):

```
1. Look up waiting signal by (instance_id, signal_name, status = 'waiting')
2. If found:
   a. Update signal: status = 'received', payload = signal_data
   b. Complete the wait_signal state execution with signal payload as output
   c. Evaluate transition conditions against signal payload
   d. Create next state execution
   e. Publish to NATS
3. If not found: return NotFound error
```

**Signal timeout** (handled by timer scheduler):

```
1. Timer fires for signal timeout
2. Check if signal is still 'waiting'
3. If yes: mark signal as 'timed_out', follow on_timeout transition
4. If no (already received): no-op
```

---

## 20. Complete PostgreSQL Schema Summary

| Table | Purpose | Hot Path? |
|-------|---------|----------|
| `workflow_instances` | Running workflow state (CAS transitions) | Yes |
| `workflow_state_executions` | Each execution attempt with status tracking | Yes |
| `workflow_state_outputs` | Validated outputs stored explicitly | Yes (write), No (read) |
| `workflow_state_schemas` | Immutable schema registry | No (cached in Valkey) |
| `workflow_state_mappings` | Explicit data flow declarations | No (cached in Valkey) |
| `workflow_retry_policies` | Per-state retry configuration | No (cached in Valkey) |
| `workflow_timers` | Durable timers for delay states | Yes (scheduler) |
| `workflow_signals` | External signal wait/receive | Occasional |
| `workflow_audit_events` | Append-only audit trail | Write-heavy, read-cold |
| `workflow_definitions` | DSL document storage (from original design) | No |
| `form_schemas` | Form structure definitions (from original design) | No |
| `form_submissions` | Raw submission data (from original design) | Moderate |
| `event_log` | Event audit trail + outbox (from original design) | Yes (outbox) |
| `trigger_bindings` | Event → workflow mappings (from original design) | No (cached) |
| `connector_configs` | Per-tenant connector settings (from original design) | No |
| `connector_credentials` | Encrypted credential storage (from original design) | No (cached) |

---

## 21. Valkey Caching Strategy

| Key Pattern | Type | TTL | Purpose |
|-------------|------|-----|---------|
| `schema:{tenant_id}:{workflow}:{version}:{state}:{type}` | JSON | 10min | Schema cache |
| `mapping:{tenant_id}:{workflow}:{version}:{from}:{to}` | JSON | 10min | Mapping cache |
| `retry_policy:{tenant_id}:{workflow}:{version}:{state}` | JSON | 10min | Retry policy cache |
| `triggers:{tenant_id}:{event_type}` | JSON list | 60s | Trigger binding cache |
| `quota:{tenant_id}:events:daily:{date}` | Counter | 24h | Daily event quota |
| `quota:{tenant_id}:active_workflows` | Counter | 1h | Active workflow count |
| `ratelimit:{tenant_id}:events:{window}` | Counter | 1min | Event ingestion rate |
| `ratelimit:{tenant_id}:{connector_type}:{window}` | Counter | 1s | Connector call rate |
| `cred:{tenant_id}:{credential_id}` | Bytes | 5min | Decrypted credential |

Cache invalidation: TTL-based. Stale reads are acceptable within TTL windows. No distributed cache invalidation protocol needed.

---

## 22. Project Structure (Updated)

```
orchestrator/
├── apps/
│   └── default/                            # API + Engine + Workers
│       ├── cmd/main.go                     # Entry point
│       ├── config/config.go                # Configuration struct
│       ├── service/
│       │   ├── handlers/                   # ConnectRPC handlers
│       │   │   ├── form.go                # FormService
│       │   │   ├── workflow.go            # WorkflowService (CRUD + validation)
│       │   │   ├── connector.go           # ConnectorService
│       │   │   ├── event.go               # EventService (timeline, debug queries)
│       │   │   ├── trigger.go             # TriggerService
│       │   │   ├── signal.go              # SignalService (send signals to waiting states)
│       │   │   └── security.go            # Tenant extraction helpers
│       │   ├── business/                   # Business logic
│       │   │   ├── form.go                # Form submission processing
│       │   │   ├── workflow.go            # Workflow definition CRUD + validation
│       │   │   ├── engine.go              # State engine: dispatch, commit, transition
│       │   │   ├── event.go               # Event ingestion + outbox publishing
│       │   │   ├── trigger.go             # Trigger binding management
│       │   │   ├── connector.go           # Connector config + credential management
│       │   │   ├── signal.go              # Signal send/receive logic
│       │   │   ├── quota.go               # Rate limiting + quota enforcement
│       │   │   └── state_manager.go       # Valkey state manager (caching, rate limits)
│       │   ├── repository/                 # Data access layer
│       │   │   ├── migrate.go             # Migration runner
│       │   │   ├── form.go               # Form schema + submission CRUD
│       │   │   ├── workflow.go           # Workflow definition CRUD
│       │   │   ├── instance.go           # Workflow instance CRUD + CAS transitions
│       │   │   ├── execution.go          # State execution CRUD + scheduler queries
│       │   │   ├── schema_registry.go    # Schema CRUD + lookup
│       │   │   ├── mapping.go            # State mapping CRUD + lookup
│       │   │   ├── output.go             # State output storage
│       │   │   ├── timer.go              # Timer CRUD + scheduler queries
│       │   │   ├── signal.go             # Signal CRUD + matching
│       │   │   ├── audit.go              # Audit event append + queries
│       │   │   ├── retry_policy.go       # Retry policy CRUD + lookup
│       │   │   ├── event.go              # Event log + outbox queries
│       │   │   ├── trigger.go            # Trigger binding CRUD
│       │   │   └── connector.go          # Connector config + credential CRUD
│       │   ├── models/                     # Domain models
│       │   │   ├── form.go               # FormSchema, FormSubmission
│       │   │   ├── workflow.go           # WorkflowDefinition
│       │   │   ├── instance.go           # WorkflowInstance (with CAS revision)
│       │   │   ├── execution.go          # WorkflowStateExecution, ExecutionCommand
│       │   │   ├── schema.go             # WorkflowStateSchema
│       │   │   ├── mapping.go            # WorkflowStateMapping
│       │   │   ├── output.go             # WorkflowStateOutput
│       │   │   ├── timer.go              # WorkflowTimer
│       │   │   ├── signal.go             # WorkflowSignal
│       │   │   ├── audit.go              # WorkflowAuditEvent
│       │   │   ├── retry_policy.go       # WorkflowRetryPolicy
│       │   │   ├── event.go              # Event (event_log entry)
│       │   │   ├── trigger.go            # TriggerBinding
│       │   │   └── connector.go          # ConnectorConfig, ConnectorCredential
│       │   └── schedulers/                 # Background schedulers
│       │       ├── dispatch.go            # Dispatch pending executions
│       │       ├── retry.go              # Process due retries
│       │       ├── timer.go              # Fire due timers
│       │       ├── timeout.go            # Timeout overdue executions
│       │       ├── outbox.go             # Publish events to NATS
│       │       └── event_router.go       # Route events to workflow instances
│       ├── migrations/
│       │   └── 0001/migration.sql          # Initial schema (all tables)
│       ├── tests/
│       └── Dockerfile
│
├── dsl/                                    # DSL engine (zero infra deps, reusable)
│   ├── types.go                            # WorkflowSpec, StateSpec, TransitionSpec
│   ├── parser.go                           # JSON → WorkflowSpec parsing
│   ├── validator.go                        # Static validation (DAG, schemas, mappings)
│   ├── simulator.go                        # Simulation validation (synthetic execution)
│   ├── expression.go                       # CEL environment, compilation, evaluation
│   ├── template.go                         # {{ }} variable interpolation
│   ├── mapping.go                          # Mapping expression evaluation
│   └── schema.go                           # JSON Schema validation utilities
│
├── connector/                              # Connector framework (reusable)
│   ├── adapter.go                          # Adapter interface definition
│   ├── registry.go                         # In-memory adapter registry
│   ├── types.go                            # ExecuteRequest, ExecuteResponse, ErrorClass
│   └── adapters/                           # Built-in adapter implementations
│       ├── webhook.go                      # webhook.call adapter
│       ├── email.go                        # email.send adapter
│       └── http.go                         # http.request adapter
│
├── sdk/                                    # Worker SDK (generated + runtime)
│   ├── generator/                          # SDK code generator
│   │   ├── generate.go                    # Schema → Go type generation
│   │   └── templates/                     # Go code templates
│   ├── runtime/                            # SDK runtime library
│   │   ├── worker.go                      # Worker lifecycle (connect, heartbeat, commit)
│   │   ├── errors.go                      # ExecutionError, ErrorClass types
│   │   └── client.go                      # Engine client (commit result, heartbeat)
│   └── testharness/                        # Contract test harness
│       ├── runner.go                      # Test execution engine
│       ├── generator.go                   # Synthetic input generation from schemas
│       └── reporter.go                    # Test result reporting
│
├── pkg/                                    # Shared packages
│   ├── events/types.go                    # Event type constants (zero deps)
│   ├── telemetry/metrics.go               # Metrics + tracing definitions (OTel)
│   └── crypto/encrypt.go                  # Credential encryption (stdlib crypto)
│
├── proto/                                  # Protobuf definitions
│   ├── buf.yaml
│   ├── buf.gen.yaml
│   ├── form/v1/form.proto                 # FormService
│   ├── workflow/v1/workflow.proto          # WorkflowService
│   ├── connector/v1/connector.proto       # ConnectorService
│   ├── event/v1/event.proto               # EventService (includes timeline/debug)
│   ├── trigger/v1/trigger.proto           # TriggerService
│   └── signal/v1/signal.proto             # SignalService
│
├── docs/
│   ├── adr/                                # Architecture Decision Records
│   │   ├── 000-system-architecture.md
│   │   ├── 001-contract-driven-engine.md   # Contract-driven state transition engine
│   │   ├── 002-json-dsl-workflow-format.md
│   │   ├── 003-cel-expression-evaluation.md
│   │   ├── 004-event-first-architecture.md
│   │   ├── 005-connector-adapter-pattern.md
│   │   ├── 006-multi-tenant-isolation.md
│   │   ├── 007-credential-management.md
│   │   ├── 008-observability-opentelemetry.md
│   │   ├── 009-dsl-versioning.md
│   │   ├── 010-rate-limiting-backpressure.md
│   │   ├── 011-workflow-composition.md
│   │   ├── 012-event-schema-evolution.md
│   │   ├── 013-idempotency-exactly-once.md
│   │   ├── 014-api-versioning.md
│   │   ├── 015-plugin-extension-system.md
│   │   └── 016-schema-registry-design.md   # NEW: Schema registry and contract validation
│   ├── dsl-reference.md
│   └── sdk-guide.md                        # NEW: Worker SDK integration guide
│
├── docker-compose.yml                      # Local dev: PG + NATS + Valkey
├── .golangci.yaml
├── Makefile
├── go.mod
├── go.sum
├── CLAUDE.md
└── ARCHITECTURE.md
```

---

## 23. Failure Modes and Recovery

| Failure | Impact | Recovery |
|---------|--------|----------|
| **Engine process crash** | In-flight dispatches may be unACKed by NATS | NATS redelivers. Worker commit fails (stale token) → no harm. Schedulers on other nodes pick up pending work. |
| **PostgreSQL unavailable** | All state mutations stop | Service returns CodeUnavailable. Readiness probe fails. No data loss. Workers queue commits until recovery. |
| **NATS unavailable** | Cannot deliver execution IDs to workers | Executions remain `pending` in PostgreSQL. Dispatch scheduler retries on next poll. No data loss. |
| **Valkey unavailable** | Cannot check quotas or cache schemas | Fallback: load schemas from PostgreSQL directly. Rate limiting degraded but not blocking. |
| **Worker crash mid-execution** | Execution stays `dispatched` indefinitely | Timeout scheduler marks it `timed_out`. Retry scheduler creates new attempt if policy allows. |
| **Worker returns invalid output** | Schema validation fails | Execution marked `invalid_output_contract`. Retry does NOT help (same invalid code). Requires code fix. |
| **CAS conflict on transition** | Another process already advanced the instance | Execution marked `stale`. No data corruption. Audit event recorded. |
| **NATS message delivered twice** | Worker receives same execution_id twice | Second worker fails to commit (token already consumed). No harm. |
| **Clock skew between engine nodes** | Timers fire early/late by clock delta | Use `TIMESTAMPTZ` (UTC) everywhere. NTP sync. Timer precision is ~1s, so small skew is tolerable. |

### Recovery Invariant

After any crash or partial failure, the system self-heals through its schedulers:

1. **Dispatch scheduler**: picks up `pending` executions
2. **Retry scheduler**: picks up `retry_scheduled` executions past their `next_retry_at`
3. **Timeout scheduler**: marks `dispatched` executions past their deadline as `timed_out`
4. **Timer scheduler**: fires `pending` timers past their `fires_at`
5. **Outbox publisher**: publishes unpublished events from `event_log`

All schedulers use `FOR UPDATE SKIP LOCKED`, so multiple engine nodes coordinate without conflicts.

---

## 24. Implementation Phases

| Phase | Deliverables | Key Tables | New ADRs |
|-------|-------------|-----------|----------|
| **Phase 1: Core Engine** | State engine (CAS transitions), dispatch/commit/retry schedulers, schema registry, basic DSL v2 parser, contract test harness stub | `workflow_instances`, `workflow_state_executions`, `workflow_state_outputs`, `workflow_state_schemas`, `workflow_retry_policies`, `workflow_audit_events` | ADR-001 (contract-driven engine), ADR-016 (schema registry) |
| **Phase 2: Data Channeling** | Mapping engine, simulation validator, generated Worker SDK, mapping evaluation | `workflow_state_mappings` | |
| **Phase 3: Durability** | Timer scheduler, signal implementation, delay/wait_signal state types | `workflow_timers`, `workflow_signals` | |
| **Phase 4: Integration Acceleration** | Full SDK generator, contract test harness, SDK guide documentation | `sdk/` package | |
| **Phase 5: Advanced States** | Parallel execution, foreach iteration, sub-workflows, conditional transitions | | ADR-011 update |
| **Phase 6: Operational Hardening** | Partition strategy, read replicas, advanced metrics, quota management | | |

---

## 25. The Single Most Important Discipline

If you implement only one thing from this entire design, make it this:

> **No state executes unless its input has been validated against a registered schema and produced by a registered mapping from the previous state.**

That one rule is what keeps large automation platforms from silently decaying into unmaintainable, non-verifiable systems.
