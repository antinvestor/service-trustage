# ADR-000: Workflow Automation System Architecture

## Status
Accepted

## Context

Stawi.dev needs a workflow automation platform that enables non-technical users to define and execute multi-step automations triggered by events (form submissions, webhooks, schedules). The system must support workflows that run for minutes, hours, or weeks, survive infrastructure failures, and interact with external systems through typed connectors.

### Problem Statement

Users of Stawi.dev forms need to trigger actions when submissions occur: send emails, call webhooks, update CRMs, start multi-step sequences with delays and conditional logic. Currently, no automation layer exists. Building one requires:

1. Durable workflow execution (survives restarts, handles multi-day delays)
2. A user-friendly workflow definition format (authorable by visual builders and AI)
3. Safe expression evaluation for conditions and filters
4. Event-driven architecture for decoupled trigger binding
5. Extensible connector system for external integrations
6. Multi-tenant isolation at every layer
7. Encrypted credential storage for connector authentication
8. Full observability across the entire execution path

### Key Constraints

- **Frame-native**: All services must use the Frame framework with its built-in abstractions for database, messaging, caching, security, and observability.
- **NATS JetStream for events**: The existing ecosystem uses NATS JetStream for async messaging. The Orchestrator must integrate with this, not replace it.
- **Multi-tenant SaaS**: Multiple organizations share infrastructure. Strict isolation is mandatory.
- **No user code execution**: Workflow definitions are declarative DSL, not imperative code. The system never executes arbitrary user code.
- **AI-compatible**: Workflow definitions must be generatable by LLMs via structured output.
- **Extensible**: New event sources, step types, and connectors must be addable without architectural changes.

## Decision

Build an **event-driven workflow automation platform** with these components:

### System Architecture

```
Events (Forms, Webhooks, Schedules)
    → NATS JetStream (event ingestion + fan-out)
    → Event Router (trigger matching + CEL filter evaluation)
    → State Engine (contract-driven state transitions in PostgreSQL)
    → Workers (execute domain logic via typed SDK)
    → Connector Adapters (external system calls)
```

### Component Responsibilities

| Component | Technology | Responsibility |
|-----------|-----------|---------------|
| API layer | ConnectRPC + Frame | Form management, workflow CRUD, trigger binding, connector config |
| Event bus | NATS JetStream via Frame | Event ingestion, fan-out, subject-based routing |
| Event router | Go service (Frame) | Match events to triggers, evaluate CEL filters, create workflow instances |
| State engine | PostgreSQL (CAS transitions) | Contract-driven state transitions, schema validation, execution tracking |
| Schedulers | Go (Frame) | Dispatch, retry, timer, timeout, and outbox scheduling via `FOR UPDATE SKIP LOCKED` |
| DSL interpreter | Go (pure library) | Parse and validate JSON DSL, evaluate CEL expressions |
| Expression engine | CEL (google/cel-go) | Safe condition evaluation, trigger filtering |
| Connector framework | Go interface + registry | Typed adapters for external system calls |
| Data layer | PostgreSQL via Frame | Definitions, event log, triggers, credentials, workflow state, schemas, mappings |
| Cache layer | Valkey via Frame | Trigger cache, rate limits, quotas, credential cache |

### Single Binary Architecture

All components run within a single Go binary using Frame:
- ConnectRPC handlers serve the API
- NATS subscribers run the event router and outbox publisher
- Engine schedulers run as managed goroutines, sharing the process lifecycle
- This keeps deployment simple while maintaining logical separation

## Alternatives Considered

| Option | Pros | Cons |
|--------|------|------|
| **Contract-driven state engine** (chosen) | Machine-verifiable data contracts. Schema-validated data flow. Generated SDK. Replay-free debugging. Single infrastructure (PostgreSQL + NATS). | Must implement timer, retry, and dispatch scheduling. ~1s timer precision. |
| Custom state machine on NATS | No new infrastructure. Consistent with Foundry. | Must build timer persistence, replay, signals, search. 3-6 months of engine work before features. |
| Serverless functions (Lambda) | Zero ops, auto-scale | Execution time limits. No durable timers. No workflow state. Vendor lock-in. |
| Apache Airflow | Mature, visual DAGs | Batch-oriented, not event-driven. Python. Poor long-running wait support. |

## Rationale

1. **Contract enforcement is the core value proposition.** Data flowing between automation states is formally specified and machine-verifiable. The engine refuses to advance state without validated data, making correctness structural rather than optional.

2. **PostgreSQL is already the system's source of truth.** Adding execution state, schemas, and mappings to the same database eliminates an entire infrastructure dependency and simplifies transactions that span definitions and executions.

3. **The DSL interpreter pattern contains engine complexity.** One generic interpreter reads any DSL definition. Users, the visual builder, and AI never touch engine code.

4. **Frame provides everything outside workflow execution.** HTTP serving, authentication, database access, caching, messaging, observability -- Frame handles all of it.

5. **Single binary keeps operations simple.** The API, event router, and engine schedulers share one process, one deployment, one set of health checks.

## Consequences

**Positive:**
- Users define workflows via visual builder or AI without touching infrastructure
- Workflows survive restarts, deployments, and infrastructure failures
- Events from any source (forms, webhooks, schedules) use the same pipeline
- Full audit trail via event log and workflow execution history in PostgreSQL
- Connectors are extensible without DSL or engine changes
- Multi-tenant isolation enforced at every layer
- Replay-free debugging: reconstruct exact data flows with standard SQL

**Negative:**
- Must implement engine scheduling (dispatch, retry, timer, timeout)
- Timer precision is ~1 second (sufficient for hours/days/weeks workflows)
- CEL expression language has a learning curve for users

## Related ADRs

| ADR | Decision |
|-----|----------|
| [ADR-001](001-contract-driven-state-engine.md) | Contract-driven state transition engine |
| [ADR-002](002-json-dsl-workflow-format.md) | JSON DSL as workflow definition format |
| [ADR-003](003-cel-expression-evaluation.md) | CEL for safe expression evaluation |
| [ADR-004](004-event-first-architecture.md) | Event-first architecture with typed event bus |
| [ADR-005](005-connector-adapter-pattern.md) | Connector adapter pattern with typed schemas |
| [ADR-006](006-multi-tenant-isolation.md) | Multi-tenant isolation strategy |
| [ADR-007](007-credential-management.md) | Credential management and secret encryption |
| [ADR-008](008-observability-opentelemetry.md) | Observability strategy (OpenTelemetry) |
