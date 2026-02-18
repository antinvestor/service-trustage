# Orchestrator - Development Conventions

## Project Overview

Workflow automation engine for Stawi.dev. Accepts events (form submissions, webhooks, schedules), evaluates trigger bindings, executes durable workflows via a contract-driven state transition engine, and calls external systems through typed connector adapters.

**Read ARCHITECTURE.md for the full system design.** This file covers development conventions only.

## Technology Stack

| Layer | Technology | Access Via |
|-------|-----------|-----------|
| Framework | github.com/pitabwire/frame | Direct |
| API | ConnectRPC | Generated from proto/ |
| Database | PostgreSQL | Frame `datastore.Manager` (NEVER direct sql) |
| Message Queue | NATS JetStream | Frame `queue.Manager` (NEVER direct nats) |
| Cache | Valkey | Frame `cache.Manager` (NEVER direct redis/valkey client) |
| Workflow Engine | Contract-driven state engine | PostgreSQL (CAS transitions) + NATS (transport) |
| Expressions | CEL | `github.com/google/cel-go` |
| Auth | OIDC | Frame `security.Manager` |

## Architecture Layers

```
ConnectRPC Handlers → Business Logic → Repository (PostgreSQL)
                          ↓
                    Event Publishing (NATS via outbox)
                          ↓
                    Event Router → State Engine (creates workflow instances)
                          ↓
                    Dispatch Scheduler → NATS (execution_id only)
                          ↓
                    Worker (executes domain logic) → Commit API (validates output, advances state)
```

## Mandatory Patterns

### Models
- ALL models embed `data.BaseModel` (provides ID, TenantID, PartitionID, CreatedAt, ModifiedAt, DeletedAt)
- NEVER define manual ID, TenantID, or timestamp fields
- State transitions use explicit validation methods
- Proto conversion via `.ToAPI()` methods

### Repositories
- ALL repositories use `datastore.BaseRepository[T]` or raw pool with interface
- ALL repository methods receive context and enforce tenant_id filtering
- Interface-based for testability
- Return concrete types, never `any`

### Business Logic
- Interface-based with constructor dependency injection
- Extract tenant from `security.ClaimsFromContext(ctx)`
- Validate inputs before persistence
- Wrap errors: `fmt.Errorf("context: %w", err)`
- Emit events via Frame's event/queue abstractions

### Handlers (ConnectRPC)
- Extract tenant from OIDC claims
- Validate request fields
- Delegate to business layer (no business logic in handlers)
- Convert domain models to proto via `.ToAPI()`
- Classify errors to ConnectRPC codes (NotFound, InvalidArgument, etc.)
- Record telemetry spans and metrics

### Logging
- ALWAYS use `util.Log(ctx)` -- NEVER `log.Println`, `slog`, or `fmt.Printf`
- Include structured fields: tenant_id, workflow_id, step_id, event_id
- NEVER log credentials, PII, or form submission values

### HTTP Clients
- ALWAYS use `svc.HTTPClientManager().Client(ctx)` -- NEVER `&http.Client{}`

### Async Processing
- Quick internal work (<100ms, no I/O): Frame Events
- Durable work / external consumption: Frame Queue (NATS JetStream)
- Bounded parallelism with results: Frame WorkerPool
- NEVER use raw goroutines for critical work

## State Engine Rules

### State Transitions
- All state mutations happen via CAS (Compare-And-Swap) on `workflow_instances.revision`
- Only two transaction types: Dispatch (create execution) and Commit (validate output, advance state)
- Execution tokens are single-use: generated at dispatch, verified at commit
- PostgreSQL is the single source of truth for all workflow state
- NATS carries only `execution_id` — all state loaded from PostgreSQL by workers

### Workers
- Workers receive structured `ExecutionCommand` with typed input
- Workers must classify every error using `ErrorClass` (retryable, fatal, compensatable, external_dependency)
- Workers commit results via the engine's Commit API with execution token
- Workers must be idempotent (ADR-013)

### Schedulers
- All background schedulers use `FOR UPDATE SKIP LOCKED` for safe multi-node operation
- Dispatch scheduler: picks up `pending` executions
- Retry scheduler: picks up `retry_scheduled` executions past `next_retry_at`
- Timer scheduler: fires `pending` timers past `fires_at`
- Timeout scheduler: marks overdue `dispatched` executions as `timed_out`
- Outbox publisher: publishes unpublished events from `event_log`

### Schema Validation
- Every state has registered InputSchema, OutputSchema, and ErrorSchema
- Input validated before dispatch, output validated before commit
- Schemas are immutable (write-once, identified by content hash)
- Data flows between states through declared, validated mappings (no implicit passthrough)

## DSL Engine Rules (dsl/)

### Zero infrastructure dependencies
- The `dsl/` package must NOT import Frame, NATS, or any infrastructure
- It is a pure Go library: parse, validate, evaluate
- This enables reuse in other products

### CEL Expressions
- All custom functions must be pure (no side effects), deterministic, and bounded-cost
- `now` is always an injected variable, never computed
- Cost budget: 10,000 max per expression

### Template Resolution
- `{{ payload.field }}` resolved against vars map before connector execution
- Templates validated at definition save time (not execution time)

## Connector Rules (connector/)

### Adapter Interface
- Every adapter implements `connector.Adapter` interface
- Adapters registered in `connector.Registry` at startup
- JSON Schema for input/config/output enables AI discovery and visual builder forms
- Adapters handle their own input validation

### Credential Handling
- Credentials decrypted in engine dispatch phase or worker execution
- Credentials cached in Valkey with 5min TTL
- Credentials NEVER logged, NEVER stored in any message or audit event

## Multi-Tenant Isolation

- `tenant_id NOT NULL` on EVERY table
- EVERY repository query includes tenant_id in WHERE clause
- EVERY workflow instance has tenant_id in all related tables
- EVERY Valkey key includes tenant_id in prefix
- EVERY NATS subject includes tenant_id in hierarchy (event subjects)

## File Organization

| Directory | Purpose | Dependencies |
|-----------|---------|-------------|
| `apps/default/` | Main service (handlers, business, repository, models, schedulers) | Frame, proto gen, dsl, connector, sdk |
| `dsl/` | DSL parsing, validation, CEL evaluation, templates, mappings | ZERO infrastructure deps |
| `connector/` | Adapter interface, registry, built-in adapters | Minimal (only what adapters need) |
| `sdk/` | Worker SDK generator, runtime, contract test harness | dsl, connector |
| `pkg/events/` | Event type constants and message schemas | ZERO infrastructure deps |
| `pkg/telemetry/` | Metrics and tracing definitions | OpenTelemetry |
| `pkg/crypto/` | Credential encryption utilities | Go stdlib crypto |
| `proto/` | Protobuf definitions | buf.build toolchain |

## Testing

- Use testcontainers for PostgreSQL, NATS, Valkey
- Use contract test harness for worker/state validation
- Table-driven tests with testify suites
- Test tenant isolation in every repository test
- Test CEL expression safety (cost limits, termination)
- Test DSL validation (DAG, schemas, mappings, simulation)
- Test CAS transition correctness (concurrent updates)
- Test scheduler behavior (dispatch, retry, timeout, timer)

## Commands

```bash
make tests          # Run all tests with race detection
make lint           # golangci-lint
make format         # goimports + golangci-lint fix
make proto-gen      # buf generate
make docker-setup   # Start local dev dependencies
make docker-stop    # Stop local dev dependencies
make build          # Build service binary
```

## Key Dependencies

```
github.com/pitabwire/frame         # Service framework
github.com/pitabwire/util          # Utility functions (logging, IDs, crypto)
connectrpc.com/connect             # RPC framework
github.com/google/cel-go           # CEL expression engine
google.golang.org/protobuf         # Protobuf runtime
github.com/stretchr/testify        # Testing
```

## Reference Projects

- `/home/j/code/stawi.dev/foundry` -- Primary reference for Frame patterns, ConnectRPC handlers, repository layer, telemetry, events, queue workers
- `/home/j/code/stawi.dev/gitvault` -- Simpler reference for single-service Frame patterns
