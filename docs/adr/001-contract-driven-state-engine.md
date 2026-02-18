# ADR-001: Contract-Driven State Transition Engine

## Status

Accepted

## Context

The Orchestrator must execute workflows that:

1. Run for minutes, hours, days, or weeks
2. Survive process restarts and deployments
3. Support conditional branching, parallel execution, and iteration
4. Provide exactly-once step execution semantics
5. Enable human-in-the-loop approvals via signals
6. Scale horizontally across multiple workers
7. Enforce formal data contracts between every state boundary
8. Enable machine-verifiable correctness before deployment
9. Support generated Worker SDKs and contract test harnesses for fast integration onboarding

The platform's primary value proposition is that data flowing between automation states is formally specified and machine-verifiable. Partners integrating with the system need compile-time guarantees that their state handler will receive correctly typed inputs and that their outputs will be validated before advancing the workflow. The system must generate typed Worker SDKs from schema definitions, ship contract test harnesses, and allow partners to validate their implementations offline.

In production, operators need to reconstruct the exact data that flowed between states without re-executing workflow code. The contract-driven approach stores all intermediate payloads explicitly — every state's validated input and output are queryable rows in PostgreSQL. No replay, no specialized expertise, no risk of divergence due to code changes.

The system is not building "a workflow engine." It is building:

> A **distributed, contract-driven state transition system** with externally implemented actions.

This reframing clarifies every design decision: the engine's job is to enforce contracts, validate data, manage transitions, and provide observability. Workers implement domain logic against typed interfaces. The engine never executes domain logic itself.

## Decision

Build a **PostgreSQL-native, contract-driven state transition engine** that enforces formal data contracts at every state boundary. The engine owns all state transitions. No external system can unilaterally advance workflow state.

### Core Invariant

> **No state executes unless its input has been validated against a registered schema and produced by a registered mapping from the previous state.**

This single rule is the architectural foundation. Everything else follows from it.

### Integration Flow

```
ConnectRPC Handlers
    → Business Logic
        → State Engine (CAS transitions in PostgreSQL)
            → Dispatch Scheduler (FOR UPDATE SKIP LOCKED)
                → NATS (delivers execution_id only)
                    → Worker (executes domain logic)
                        → Commit API (validates output, advances state)
            → Retry Scheduler (policy-driven, PostgreSQL-owned)
            → Timer Scheduler (durable delays, PostgreSQL-polled)
```

- **Frame** owns the service layer (API, database, caching, observability).
- **NATS JetStream** owns event ingestion and fan-out.
- **PostgreSQL** owns all workflow state, schemas, mappings, and execution history.

### Component Responsibilities

| Component | Responsibility |
|-----------|---------------|
| **Schema Registry** | Stores immutable input/output/error schemas per (workflow, version, state). Schemas are write-once. |
| **Mapping Engine** | Evaluates explicit field mappings between states. No implicit data passthrough. |
| **Transition Validator** | Static analysis (DAG, schemas, mappings) + simulation validation before definition acceptance. |
| **State Engine** | CAS transitions on `workflow_instances`, execution token generation, output validation. |
| **Dispatch Scheduler** | Polls `pending` executions via `FOR UPDATE SKIP LOCKED`, publishes `execution_id` to NATS. |
| **Retry Scheduler** | Polls `retry_scheduled` executions past `next_retry_at`, creates new attempts. |
| **Timer Scheduler** | Polls `workflow_timers` past `fires_at`, completes delay states. |
| **Timeout Scheduler** | Marks overdue `dispatched` executions as `timed_out`. |
| **Worker SDK** | Generated typed Go code per state. Handles envelope, validation, heartbeat, error classification. |
| **Contract Test Harness** | Boots local engine, feeds synthetic inputs from schemas, validates outputs. |

### NATS Role (Transport Only)

NATS carries a single field per message: `execution_id`. All state, schemas, payloads, and policies are loaded from PostgreSQL by the worker. NATS never decides retries, ordering, validity, or ownership.

### PostgreSQL as Single Source of Truth

All workflow state lives in PostgreSQL:

| Table | Purpose |
|-------|---------|
| `workflow_instances` | Running workflow state with CAS `revision` counter |
| `workflow_state_executions` | Each execution attempt with status tracking |
| `workflow_state_outputs` | Validated outputs stored explicitly (no implicit passthrough) |
| `workflow_state_schemas` | Immutable schema registry |
| `workflow_state_mappings` | Explicit data flow declarations between states |
| `workflow_retry_policies` | Per-state retry configuration (max attempts, backoff strategy) |
| `workflow_timers` | Durable timers for delay states |
| `workflow_signals` | External signal wait/receive for human-in-the-loop |
| `workflow_audit_events` | Append-only audit trail of every transition |

### Strict Transition CAS

Every state transition uses optimistic concurrency:

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

Zero rows affected → stale execution. No distributed locks needed.

### Two Transaction Types (Only Mutations)

**Dispatch transaction**: Create execution record, validate input against schema, set status to `pending`.

**Commit transaction**: Verify execution token, validate output against schema, evaluate mapping, CAS transition on instance, store output, create next execution.

Nothing else mutates workflow state.

### Error Taxonomy

Workers must classify every error using the engine's type system:

| Class | Engine Behavior |
|-------|----------------|
| `retryable` | Schedule retry per policy (exponential backoff, max attempts) |
| `fatal` | No retry. Follow `on_fatal` transition. |
| `compensatable` | Trigger compensation workflow if defined. |
| `external_dependency` | Retry with longer backoff (third-party system down). |

The Worker SDK enforces classification at compile time — errors cannot be constructed without a class.

### Horizontal Scaling

Engine nodes are stateless. All schedulers use `FOR UPDATE SKIP LOCKED` for distributed coordination without locks:

```sql
SELECT execution_id FROM workflow_state_executions
WHERE status = 'pending'
ORDER BY created_at
FOR UPDATE SKIP LOCKED
LIMIT 50;
```

Multiple engine nodes safely process different rows concurrently.

## Alternatives Considered

| Option | Pros | Cons | Verdict |
|--------|------|------|---------|
| **Contract-driven PostgreSQL engine** | Machine-verifiable correctness. Schema-validated data flow. Generated SDK. Simpler ops. Replay-free debugging. All intermediate data stored. | Must implement timers, signals, retry scheduling. ~1s timer precision. | **Chosen** |
| **Purpose-built workflow engine (e.g., Temporal)** | Proven durability. Deterministic replay. Signals, search, child workflows, cron, continue-as-new. | No data contracts between steps. Free-form JSON payloads. Hand-written activities. Debugging requires replay. Additional infrastructure (3+ nodes). Two systems to reason about. | Rejected |
| **Custom state machine on NATS** | No new infrastructure beyond NATS. | NATS is not a database. No transactional guarantees. Must build everything a workflow engine provides AND everything the contract engine provides. | Rejected |
| **Serverless functions** | Zero ops, auto-scale. | Execution time limits. No durable timers. No workflow state. Vendor lock-in. | Rejected |
| **Apache Airflow** | Mature, visual DAGs. | Batch-oriented, not event-driven. Python. Poor long-running wait support. | Rejected |

## Rationale

1. **Contract enforcement must be an engine primitive, not middleware.** If schema validation is a layer on top of the execution engine, it can be bypassed, misconfigured, or silently fail. When the engine itself refuses to advance state without validated data, correctness is structural.

2. **PostgreSQL is already the system's source of truth.** The Orchestrator already depends on PostgreSQL for workflow definitions, event log, trigger bindings, connector configs, and credentials. Adding execution state to the same database eliminates an entire infrastructure dependency and simplifies transactions that span definitions and executions.

3. **Explicit data storage enables replay-free debugging.** Storing every state's validated input and output as queryable rows means operators can reconstruct exact data flows with standard SQL. No code replay, no specialized expertise, no risk of divergence due to code changes.

4. **Generated SDKs and contract tests accelerate integrations.** When the engine owns the schema registry, it can generate typed Go code for each state handler and ship test harnesses that validate implementations against registered schemas. Partners integrate in days, not weeks.

5. **`FOR UPDATE SKIP LOCKED` provides sufficient distributed coordination.** The engine's scheduling needs (dispatch pending work, retry failed work, fire timers) are well-served by PostgreSQL's row-level locking. This pattern is battle-tested at scale in job queue systems (Que, River, PGMQ).

6. **Timer precision of ~1 second is acceptable.** The platform's workflows involve delays of hours, days, and weeks. Sub-millisecond timer precision provides no user-visible benefit.

7. **The trade-off on deterministic replay is acceptable.** Failed states retry from scratch. For the target workloads (form → API → email → wait → follow-up), retrying a single state is fast and cheap.

## Consequences

**Positive:**

- Machine-checkable data contracts at every state boundary (input validation, output validation, mapping validation)
- All intermediate data stored explicitly — no hidden state, no implicit passthrough
- Static + simulation validation catches workflow definition errors before deployment
- Generated Worker SDK reduces integration time from weeks to days
- Contract test harness enables offline validation of state implementations
- Simpler infrastructure: PostgreSQL + NATS only
- Replay-free debugging: reconstruct exact data flows with standard SQL queries
- Deterministic audit trail: every transition, validation, and mapping recorded as append-only events
- Engine-owned retry logic is testable and observable
- Horizontal scaling via stateless engine nodes + `FOR UPDATE SKIP LOCKED`
- Full SQL query capability for workflow instances

**Negative:**

- Must implement durable timer scheduling (PostgreSQL polling, ~1s precision)
- Must implement signal/wait mechanism (PostgreSQL-backed, ConnectRPC API)
- Must implement retry scheduling with backoff policies
- Must implement timeout detection for overdue executions
- No deterministic replay — interrupted states retry from scratch
- No code-level workflow authoring (DSL only — intentional design choice)
- No continue-as-new for infinite-duration workflows (not needed for target workloads)
- Engine development effort: timer scheduler, retry scheduler, dispatch scheduler, CAS transition logic, schema validation, mapping evaluation
- PostgreSQL becomes the performance bottleneck for scheduling queries (mitigated by partial indexes and table partitioning)

## Implementation Notes

### Schema Registry

Schemas are immutable and identified by content hash:

```sql
CREATE TABLE workflow_state_schemas (
    ...
    schema_hash     VARCHAR(64) NOT NULL,       -- SHA-256 of schema_blob
    schema_blob     JSONB NOT NULL,             -- JSON Schema document
    UNIQUE (tenant_id, workflow_name, workflow_version, state, schema_type)
);
```

### Execution Command Envelope

Workers receive a structured command, not a raw payload:

```go
type ExecutionCommand struct {
    ExecutionID     string          `json:"execution_id"`
    InstanceID      string          `json:"instance_id"`
    TenantID        string          `json:"tenant_id"`
    Workflow        string          `json:"workflow"`
    WorkflowVersion int             `json:"workflow_version"`
    State           string          `json:"state"`
    Attempt         int             `json:"attempt"`
    InputPayload    json.RawMessage `json:"input_payload"`
    InputSchemaHash string          `json:"input_schema_hash"`
    ExecutionToken  string          `json:"execution_token"`
}
```

The `ExecutionToken` is single-use, generated at dispatch, and verified at commit. Stale workers cannot commit results.

### Self-Healing Schedulers

After any crash, the system recovers through five independent schedulers:

| Scheduler | Picks Up | Query |
|-----------|----------|-------|
| Dispatch | `status = 'pending'` | `ORDER BY created_at FOR UPDATE SKIP LOCKED` |
| Retry | `status = 'retry_scheduled' AND next_retry_at <= NOW()` | `ORDER BY next_retry_at FOR UPDATE SKIP LOCKED` |
| Timer | `workflow_timers WHERE fires_at <= NOW() AND status = 'pending'` | `ORDER BY fires_at FOR UPDATE SKIP LOCKED` |
| Timeout | `status = 'dispatched' AND dispatched_at + timeout < NOW()` | Periodic sweep |
| Outbox | `event_log WHERE event_published = FALSE` | `ORDER BY created_at LIMIT 100` |

All use `FOR UPDATE SKIP LOCKED` for safe multi-node operation.

### Transition Validator Pipeline

```
1. Static validation
   ├── DAG validation (no cycles)
   ├── Reachability analysis (no orphaned states)
   ├── Schema existence (every state has input/output/error schemas)
   ├── Mapping existence (every transition has a declared mapping)
   ├── Mapping compatibility (output fields cover input requirements)
   ├── Expression compilation (CEL expressions parse without error)
   └── Template validation ({{ }} references resolve)

2. Simulation validation
   ├── Generate synthetic inputs from initial input schema
   ├── For each state in topological order:
   │   ├── Validate input against state schema
   │   ├── Generate synthetic output from state schema
   │   ├── Evaluate mapping to next state(s)
   │   └── Validate mapped result against next state's input schema
   └── Report any validation failures (blocks deployment)
```

### Worker SDK Generation

From registered schemas:

```go
// Generated: verify_identity/types.go
type VerifyIdentityInput struct {
    UserID      string `json:"user_id"`
    DocumentURL string `json:"document_url"`
}

type VerifyIdentityOutput struct {
    Verified  bool    `json:"verified"`
    Score     float64 `json:"score"`
    RiskLevel string  `json:"risk_level"`
}

// Integrators implement this single function:
type Handler func(ctx context.Context, in VerifyIdentityInput) (*VerifyIdentityOutput, *ExecutionError)
```

The SDK handles envelope deserialization, schema validation, heartbeat, execution token, and error classification. Integrators never touch engine internals.

### Extensibility

| Capability | Engine Primitive | Use Case |
|------------|-----------------|----------|
| Durable timers | `workflow_timers` table + timer scheduler | Scheduled delays (hours, days, weeks) |
| Child workflows | `sub_workflow` state type + child instance tracking | Parallel fan-out, sub-workflow composition |
| Signal wait/receive | `workflow_signals` table + ConnectRPC `SendSignal` API | Approval flows, external callbacks |
| Cron scheduling | External cron emits events into event bus | Scheduled report generation, periodic cleanup |
| Saga compensation | `compensatable` error class + compensation workflow refs | Multi-step rollback on failure |
| Batch processing | `foreach` + child instances | Process collections in parallel |
| SLA monitoring | Timer + signal race | Escalate if approval not received within deadline |
| Workflow versioning | Three-axis versioning (workflow, state, schema) | Safe evolution of definitions |
