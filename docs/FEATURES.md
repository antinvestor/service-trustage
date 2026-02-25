# Trustage Feature Reference

**Version:** 0.1.1
**Last Updated:** 2026-02-19

Trustage is a contract-driven workflow orchestration engine for Stawi.dev. It accepts events (form submissions, webhooks, schedules), evaluates trigger bindings, executes durable workflows via a state transition engine, and calls external systems through typed connector adapters.

---

## Table of Contents

1. [HTTP API Endpoints](#1-http-api-endpoints)
2. [Event Ingestion](#2-event-ingestion)
3. [Workflow DSL](#3-workflow-dsl)
4. [Step Types](#4-step-types)
5. [CEL Expression Engine](#5-cel-expression-engine)
6. [Template Resolution](#6-template-resolution)
7. [Connector Adapters](#7-connector-adapters)
8. [State Engine](#8-state-engine)
9. [Schema Registry & Contract Validation](#9-schema-registry--contract-validation)
10. [Trigger Bindings & Event Routing](#10-trigger-bindings--event-routing)
11. [Schedulers](#11-schedulers)
12. [Queue Workers](#12-queue-workers)
13. [Retry Policies & Error Handling](#13-retry-policies--error-handling)
14. [Multi-Tenant Isolation](#14-multi-tenant-isolation)
15. [Caching](#15-caching)
16. [Security](#16-security)
17. [Observability](#17-observability)
18. [Data Models](#18-data-models)
19. [Configuration Reference](#19-configuration-reference)
20. [Deployment](#20-deployment)

---

## 1. HTTP API Endpoints

All endpoints return JSON. Authentication is via OIDC (tenant extracted from claims). Max request body: 1 MB.

### Workflow Management

| Method | Path | Purpose | Auth | Rate Limited |
|--------|------|---------|------|--------------|
| POST | `/api/v1/workflows` | Create workflow definition | OIDC | No |
| GET | `/api/v1/workflows/{id}` | Get workflow definition | OIDC | No |
| POST | `/api/v1/workflows/{id}/activate` | Activate workflow | OIDC | No |

### Event Ingestion

| Method | Path | Purpose | Auth | Rate Limited |
|--------|------|---------|------|--------------|
| POST | `/api/v1/events` | Ingest event | OIDC | Yes (100/min/tenant) |
| GET | `/api/v1/instances/{id}/timeline` | Get instance audit trail | OIDC | No |

### Form Capture

| Method | Path | Purpose | Auth | Rate Limited |
|--------|------|---------|------|--------------|
| POST | `/api/v1/forms/{form_id}/submit` | Submit form data | OIDC | Yes (100/min/tenant) |

### Webhook Reception

| Method | Path | Purpose | Auth | Rate Limited |
|--------|------|---------|------|--------------|
| POST | `/api/v1/webhooks/{webhook_id}` | Receive inbound webhook | OIDC | Yes (100/min/tenant) |

### Health Checks

| Method | Path | Purpose | Auth |
|--------|------|---------|------|
| GET | `/healthz` | Liveness probe | None |
| GET | `/readyz` | Readiness probe (checks DB) | None |

### Middleware

- **RequestID**: Generates or propagates `X-Request-ID` header for log correlation.
- **LimitBodySize**: Enforces 1 MB max request body on all endpoints.

---

## 2. Event Ingestion

Events are the primary way to trigger workflows. Three ingestion paths exist:

### Direct Event Ingestion (`POST /api/v1/events`)

```json
{
  "event_type": "order.created",
  "source": "shop-api",
  "idempotency_key": "order-12345",
  "payload": {
    "order_id": "12345",
    "amount": 500.00,
    "currency": "KES"
  }
}
```

**Response** (202 Accepted):
```json
{
  "event_id": "evt_abc123"
}
```

Idempotent: if `idempotency_key` matches an existing event, returns the existing event_id with `"idempotent": true`.

### Form Submission (`POST /api/v1/forms/{form_id}/submit`)

```json
{
  "fields": {
    "name": "John Doe",
    "email": "john@example.com",
    "amount": 1000
  },
  "submitter_id": "user_456",
  "idempotency_key": "form-sub-789"
}
```

Creates a `form.submitted` event with source `form:{form_id}`. The event payload includes `form_id`, `fields`, and optional `submitter_id`.

### Webhook Reception (`POST /api/v1/webhooks/{webhook_id}`)

Accepts arbitrary JSON body. Captures safe headers (`Content-Type`, `User-Agent`, `X-Request-ID`, `X-Webhook-Signature`, `X-Webhook-Event`). Creates a `webhook.received` event with source `webhook:{webhook_id}`.

### Event Processing Pipeline

All events follow the same pipeline:

```
Ingestion Endpoint → EventLog (PostgreSQL, published=false)
    → Outbox Scheduler (5s) → NATS JetStream
    → Event Router Worker → Trigger Binding Evaluation
    → Workflow Instance Creation → Execution Dispatch
```

This outbox pattern guarantees no events are lost, even if NATS is temporarily unavailable.

---

## 3. Workflow DSL

Workflows are defined as JSON documents with a declarative DSL.

### Structure

```json
{
  "version": "1.0",
  "name": "onboard-customer",
  "description": "Customer onboarding workflow",
  "input": {
    "customer_id": "string",
    "email": "string"
  },
  "config": {
    "notification_api": "https://notify.example.com/api/v1"
  },
  "timeout": "72h",
  "on_error": {
    "strategy": "retry",
    "fallback": []
  },
  "steps": [
    {
      "id": "validate",
      "type": "call",
      "name": "Validate Form Data",
      "call": {
        "action": "form.validate",
        "input": { "fields": "{{ payload.fields }}", "required_fields": ["name", "email"] }
      },
      "on_success": "check_risk",
      "on_failure": "notify_error"
    },
    {
      "id": "check_risk",
      "type": "if",
      "if": {
        "expr": "vars.validate.valid == true",
        "then": [{ "id": "send_welcome", "type": "call", "call": { "action": "notification.send", "input": {} } }],
        "else": [{ "id": "reject", "type": "call", "call": { "action": "log.entry", "input": {} } }]
      }
    }
  ]
}
```

### Workflow Lifecycle

| Status | Transitions To | Description |
|--------|---------------|-------------|
| `draft` | `active`, `archived` | Initial state, can be edited |
| `active` | `archived` | Accepting events, creating instances |
| `archived` | (terminal) | No longer active |

### Validation Rules

When a workflow is created, the DSL is validated for:

1. **Required fields**: `version`, `name`, at least one `step`
2. **Unique step IDs**: No duplicate IDs across the entire step tree
3. **Valid step types**: All types must be one of the 8 supported types
4. **Reference integrity**: `depends_on` references must point to existing steps
5. **DAG structure**: No circular dependencies (Kahn's algorithm)
6. **CEL expression syntax**: All expressions must compile
7. **Template syntax**: All `{{ }}` references must be syntactically valid
8. **Retry configuration**: `max_attempts >= 1`, valid durations
9. **Timeout consistency**: Step timeouts must not exceed workflow timeout

---

## 4. Step Types

### `call` — Invoke a Connector Adapter

Executes an external operation via a registered connector adapter.

```json
{
  "id": "send_sms",
  "type": "call",
  "call": {
    "action": "notification.send",
    "input": {
      "recipient": "{{ payload.phone }}",
      "channel": "sms",
      "body": "Welcome, {{ payload.name }}!"
    },
    "output_var": "sms_result"
  }
}
```

- `action`: Connector adapter type (e.g., `webhook.call`, `payment.initiate`)
- `input`: Parameters passed to the adapter (supports templates)
- `output_var`: Variable name to store the adapter's output

### `delay` — Durable Timer

Pauses workflow execution for a fixed duration or until a timestamp.

```json
{
  "id": "wait_24h",
  "type": "delay",
  "delay": {
    "duration": "24h"
  }
}
```

Or with a computed timestamp:

```json
{
  "id": "wait_until",
  "type": "delay",
  "delay": {
    "until": "payload.scheduled_time"
  }
}
```

Supports duration formats: `30s`, `5m`, `2h`, `7d` (days multiplied by 24h).

### `if` — Conditional Branching

Evaluates a CEL expression and executes `then` or `else` branch.

```json
{
  "id": "check_amount",
  "type": "if",
  "if": {
    "expr": "vars.payment.amount > 10000",
    "then": [
      { "id": "require_approval", "type": "call", "call": { "action": "approval.request", "input": {} } }
    ],
    "else": [
      { "id": "auto_approve", "type": "call", "call": { "action": "log.entry", "input": { "level": "info", "message": "Auto-approved" } } }
    ]
  }
}
```

### `sequence` — Ordered Execution

Executes sub-steps in order, sharing error handling scope.

```json
{
  "id": "onboard_steps",
  "type": "sequence",
  "sequence": {
    "steps": [
      { "id": "step_1", "type": "call", "call": { "action": "form.validate", "input": {} } },
      { "id": "step_2", "type": "call", "call": { "action": "notification.send", "input": {} } }
    ]
  }
}
```

### `parallel` — Concurrent Execution

Executes sub-steps concurrently.

```json
{
  "id": "notify_all",
  "type": "parallel",
  "parallel": {
    "steps": [
      { "id": "sms", "type": "call", "call": { "action": "notification.send", "input": { "channel": "sms" } } },
      { "id": "email", "type": "call", "call": { "action": "notification.send", "input": { "channel": "email" } } }
    ],
    "wait_all": true
  }
}
```

- `wait_all`: If `true` (default), waits for all steps. If `false`, continues on first completion.

### `foreach` — Loop Over Collection

Iterates over a list, executing sub-steps per item.

```json
{
  "id": "process_items",
  "type": "foreach",
  "foreach": {
    "items": "payload.line_items",
    "item_var": "item",
    "index_var": "idx",
    "max_concurrency": 5,
    "steps": [
      { "id": "process_item", "type": "call", "call": { "action": "payment.initiate", "input": { "amount": "{{ item.price }}" } } }
    ]
  }
}
```

- `items`: CEL expression yielding a list
- `item_var`: Variable name for current item (default: `"item"`)
- `index_var`: Variable name for current index (default: `"index"`)
- `max_concurrency`: Limit parallel iterations (0 = sequential)

### `signal_wait` — Wait for External Signal

Pauses execution until an external signal is received. Used for human approvals, external callbacks, or inter-workflow coordination.

```json
{
  "id": "await_approval",
  "type": "signal_wait",
  "signal_wait": {
    "signal_name": "approval_response",
    "timeout": "48h",
    "output_var": "approval"
  }
}
```

### `signal_send` — Send Signal to Another Workflow

Sends a signal to a running workflow instance, typically to unblock a `signal_wait` step.

```json
{
  "id": "notify_parent",
  "type": "signal_send",
  "signal_send": {
    "target_workflow_id": "{{ vars.parent_instance_id }}",
    "signal_name": "child_completed",
    "payload": { "result": "{{ vars.final_output }}" }
  }
}
```

### Transitions

Steps can define explicit transitions via `on_success` and `on_failure`:

**Static transition** (simple step ID):
```json
"on_success": "next_step_id"
```

**Conditional transition** (CEL-evaluated):
```json
"on_success": [
  { "condition": "vars.amount > 1000", "target": "high_value_path" },
  { "condition": "vars.amount > 0", "target": "standard_path" },
  { "condition": "", "target": "default_path" }
]
```

Empty condition acts as the default/fallback. First matching condition wins. If no `on_success` is defined, implicit sequential ordering is used (next step in array).

### Error Handling

Each step can define retry and error policies:

```json
{
  "id": "risky_call",
  "type": "call",
  "retry": {
    "max_attempts": 5,
    "initial_interval": "1s",
    "backoff_coefficient": 2.0,
    "max_interval": "5m"
  },
  "timeout": "30s",
  "on_error": {
    "strategy": "fallback",
    "fallback": [
      { "id": "handle_error", "type": "call", "call": { "action": "log.entry", "input": { "level": "error", "message": "Call failed" } } }
    ]
  }
}
```

Error strategies: `fail` (default), `continue`, `retry`, `fallback`.

---

## 5. CEL Expression Engine

All expressions use the [Common Expression Language (CEL)](https://github.com/google/cel-go). Cost budget: **10,000 max per expression**.

### Available Variables

| Variable | Type | Description |
|----------|------|-------------|
| `payload` | dynamic | Event payload data |
| `metadata` | dynamic | Event metadata |
| `vars` | dynamic | Accumulated step output variables |
| `env` | dynamic | Workflow config values |
| `now` | timestamp | Current timestamp (injected, never computed) |
| `item` | dynamic | Current foreach item (foreach context only) |
| `index` | int | Current foreach index (foreach context only) |
| `output` | dynamic | Output from current state |
| `signal` | dynamic | Signal payload (signal context only) |

### Supported Operations

- **Comparison**: `==`, `!=`, `<`, `<=`, `>`, `>=`
- **Logical**: `&&`, `||`, `!`
- **Arithmetic**: `+`, `-`, `*`, `/`, `%`
- **Ternary**: `condition ? true_value : false_value`
- **Membership**: `value in list`
- **Field access**: `object.field`, `map["key"]`
- **Built-in functions**: `size()`, `contains()`, `startsWith()`, `endsWith()`, `matches()` (regex), type conversions

### Examples

```cel
// Condition: high-value payment
payload.amount > 10000 && payload.currency == "KES"

// Conditional routing
vars.risk_score > 80 ? "manual_review" : "auto_approve"

// List filtering (foreach items)
payload.items.filter(i, i.status == "pending")

// String operations
payload.email.endsWith("@company.com")
```

---

## 6. Template Resolution

Templates use `{{ expression }}` syntax for variable substitution in connector inputs.

### Syntax

```
{{ payload.field }}        → Event payload field
{{ metadata.field }}       → Event metadata field
{{ vars.step_output.field }} → Previous step output
{{ env.field }}             → Workflow config value
{{ item.property }}         → Current foreach item
```

### Resolution

- Dot-notation path resolution through nested maps
- Recursive resolution for strings, maps, and slices
- Templates validated at definition save time
- Returns error if key doesn't exist or intermediate type is not a map

### Example

```json
{
  "recipient": "{{ payload.customer_email }}",
  "subject": "Order {{ payload.order_id }} confirmed",
  "body": "Total: {{ vars.payment_result.amount }} {{ payload.currency }}"
}
```

---

## 7. Connector Adapters

Connector adapters implement external integrations. Each adapter has typed input/config/output schemas.

### Adapter Interface

Every adapter implements:
- `Type()` — Unique identifier (e.g., `"webhook.call"`)
- `DisplayName()` — Human-readable name
- `InputSchema()` — JSON Schema for input
- `ConfigSchema()` — JSON Schema for configuration
- `OutputSchema()` — JSON Schema for output
- `Execute(ctx, req)` — Run the operation
- `Validate(req)` — Validate without executing

### Error Classification

Workers must classify every error:

| ErrorClass | Meaning | Engine Behavior |
|------------|---------|----------------|
| `retryable` | Transient failure (timeout, rate limit) | Retry with exponential backoff |
| `fatal` | Permanent error (validation, bad config) | Stop workflow |
| `compensatable` | Requires rollback | Trigger compensation workflow |
| `external_dependency` | External service failure | Retry with different strategy |

### Registered Adapters

#### `webhook.call` — Send Webhook

Send HTTP POST/PUT/PATCH to external URLs with SSRF protection.

**Input:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `url` | string (URI) | Yes | Target URL |
| `method` | string | No | POST (default), PUT, PATCH |
| `headers` | object | No | Custom HTTP headers |
| `body` | object | No | JSON request body |

**Output:** `{ "status_code": int, "body": object }`

---

#### `http.request` — Generic HTTP Request

Full HTTP client with GET/POST/PUT/PATCH/DELETE, query params, auth headers.

**Input:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `url` | string (URI) | Yes | Target URL |
| `method` | string | Yes | GET, POST, PUT, PATCH, DELETE |
| `headers` | object | No | Custom HTTP headers |
| `query` | object | No | URL query parameters |
| `body` | object | No | JSON request body |
| `auth_header` | string | No | Authorization header value |

**Output:** `{ "status_code": int, "headers": object, "body": object }`

---

#### `notification.send` — Send Notification

Dispatch SMS, email, or push notification via external service.

**Input:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `recipient` | string | Yes | Phone, email, or device token |
| `channel` | string | Yes | `sms`, `email`, or `push` |
| `subject` | string | Email only | Notification subject |
| `body` | string | Yes | Notification body text |
| `template_id` | string | No | Template identifier |
| `template_vars` | object | No | Template variables |

**Config:** `{ "api_url": "https://notify.example.com/api/v1" }`

**Output:** `{ "notification_id": string, "status": string, "channel": string }`

---

#### `notification.status` — Check Notification Status

Poll notification delivery status.

**Input:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `notification_id` | string | Yes | Notification to check |

**Config:** `{ "api_url": "https://notify.example.com/api/v1" }`

**Output:** `{ "notification_id": string, "status": "pending|sent|delivered|failed|bounced", "delivered_at": timestamp, "error": string }`

---

#### `payment.initiate` — Initiate Payment

Start a payment transaction.

**Input:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `amount` | number | Yes | Payment amount (>= 0) |
| `currency` | string | Yes | ISO 4217 code (3 chars) |
| `recipient` | string | Yes | Phone, account number, etc. |
| `reference` | string | Yes | Unique payment reference |
| `description` | string | No | Payment description |
| `method` | string | No | `mobile_money`, `bank_transfer`, `card` |

**Config:** `{ "api_url": "https://payments.example.com/api/v1" }`

**Output:** `{ "payment_id": string, "status": string, "reference": string }`

---

#### `payment.verify` — Verify Payment

Check payment completion status.

**Input:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `payment_id` | string | Yes | Payment to verify |

**Config:** `{ "api_url": "https://payments.example.com/api/v1" }`

**Output:** `{ "payment_id": string, "status": "pending|processing|completed|failed|reversed", "amount": number, "currency": string, "completed_at": timestamp, "error": string }`

---

#### `data.transform` — Transform Data

Reshape data between steps using CEL expressions. Pure computation, no I/O.

**Input:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `source` | object | Yes | Source data to transform |
| `expression` | string | No | Single CEL expression |
| `mappings` | object | No | Map of key → CEL expression pairs |

At least one of `expression` or `mappings` is required.

**Output:** `{ "result": any, "data": object }`

**Example:**
```json
{
  "source": { "first": "John", "last": "Doe", "scores": [85, 92, 78] },
  "expression": "source.first + ' ' + source.last",
  "mappings": {
    "full_name": "source.first + ' ' + source.last",
    "avg_score": "source.scores.reduce(s, 0, (a, b) => a + b) / source.scores.size()"
  }
}
```

---

#### `log.entry` — Audit Log Entry

Record a structured log entry. Pure computation, no I/O, always succeeds.

**Input:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `level` | string | Yes | `info`, `warn`, `error`, `debug` |
| `message` | string | Yes | Log message |
| `data` | object | No | Structured key-value data |

**Output:** `{ "logged": true, "timestamp": string, "level": string, "message": string }`

---

#### `form.validate` — Validate Form Data

Check form submission completeness and type correctness. Pure computation.

**Input:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `fields` | object | Yes | Form fields to validate |
| `required_fields` | array | Yes | Field names that must be present |
| `field_types` | object | No | Expected type per field (`string`, `number`, `boolean`, `array`, `object`) |

**Output:** `{ "valid": boolean, "errors": [string], "fields": object }`

Returns `ErrorFatal` with code `VALIDATION_FAILED` if validation fails.

---

#### `approval.request` — Request Human Approval

Send an approval request via notification service. Designed to pair with `signal_wait` for the response.

**Input:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `approver` | string | Yes | Approver identifier |
| `title` | string | Yes | Approval request title |
| `description` | string | No | Detailed description |
| `options` | array | No | Response options (default: `["approve", "reject"]`) |
| `callback_url` | string | No | URL for approver to respond |
| `expires_in` | string | No | Expiration duration |

**Config:** `{ "api_url": "https://approvals.example.com/api/v1" }`

**Output:** `{ "request_id": string, "status": "pending|sent|failed", "approver": string }`

Includes `execution_id` and `instance_id` in the API payload to enable signal-back from the approval response.

---

### SSRF Protection

All HTTP-based adapters validate URLs against:
- Blocked schemes: Only `http`/`https` allowed
- Blocked hostnames: `localhost`, `metadata.google.internal`, `*.internal`, `*.local`
- Blocked IP ranges: RFC 1918 private (10.x, 172.16.x, 192.168.x), loopback (127.x), link-local (169.254.x), carrier-grade NAT (100.64.x), IPv6 private/loopback

---

## 8. State Engine

The state engine is the core orchestration system managing workflow execution through three operations.

### Execution Lifecycle

```
Event Arrives → Trigger Binding Match → Create Instance + Initial Execution (pending)
    → Dispatch Scheduler (5s) → engine.Dispatch() → NATS JetStream
    → Execution Worker → Connector Adapter → engine.Commit()
    → CAS Transition → Next Execution (pending) → [repeat]
    → Terminal State → Instance Completed
```

### Operations

#### CreateInitialExecution
1. Validates input against registered schema
2. Generates cryptographically secure execution token (hash-only stored)
3. Creates `WorkflowStateExecution` with status `pending`
4. Returns `ExecutionCommand` with raw token for worker

#### Dispatch
1. Loads workflow instance
2. Generates new single-use token
3. Transitions execution: `pending` → `dispatched`
4. Returns `ExecutionCommand` for NATS publishing

#### Commit
1. **Token verification**: Verifies raw token matches stored hash, atomically consumes it (prevents replay)
2. **Output validation**: Validates output against registered schema
3. **State transition**: Resolves next step via DSL (static or conditional CEL transitions)
4. **CAS transition**: Atomically updates instance state with `WHERE revision = ? AND current_state = ? AND status = 'running'`
5. **Next execution**: Creates pending execution for next state
6. **Terminal check**: If no next step, completes the workflow instance

### CAS (Compare-And-Swap) Guarantees

```sql
UPDATE workflow_instances
SET current_state = ?, revision = revision + 1, modified_at = ?
WHERE id = ? AND tenant_id = ? AND current_state = ?
  AND revision = ? AND status = 'running'
```

- Optimistic locking via `revision` field
- State guard prevents drift
- Status guard prevents transitions on completed/failed instances
- 0 rows affected = concurrent modification detected (stale execution)

### Execution States

| Status | Description |
|--------|-------------|
| `pending` | Waiting for dispatch scheduler |
| `dispatched` | Sent to worker, awaiting commit |
| `completed` | Successfully executed |
| `failed` | Temporary execution failure |
| `fatal` | Permanent failure (retries exhausted or fatal error) |
| `timed_out` | Exceeded execution timeout |
| `invalid_input_contract` | Input schema validation failed |
| `invalid_output_contract` | Output schema validation failed |
| `stale` | CAS transition failed (concurrent modification) |
| `retry_scheduled` | Waiting for retry backoff to expire |

### Instance States

| Status | Transitions To |
|--------|---------------|
| `running` | `completed`, `failed`, `cancelled`, `suspended` |
| `suspended` | `running`, `cancelled` |
| `completed` | (terminal) |
| `failed` | (terminal) |
| `cancelled` | (terminal) |

---

## 9. Schema Registry & Contract Validation

Every workflow state declares input, output, and error schemas. All data is validated at runtime.

### Schema Types

- **Input Schema**: Validated before execution dispatch
- **Output Schema**: Validated before commit acceptance
- **Error Schema**: Optional, validated on error commits

### Immutability

Schemas are write-once, identified by SHA-256 content hash. This enables versioning and contract tracking across workflow versions.

### Three-Tier Caching

1. **L1**: In-process bounded LRU cache for compiled schemas (max 1000)
2. **L2**: Valkey distributed cache for schema blobs (10-minute TTL)
3. **L3**: PostgreSQL as source of truth

### Data Mapping

Data flows between states through declared, validated mappings (no implicit passthrough):

```json
{
  "user_id": "$.identity.user_id",
  "risk_score": "$.risk.score",
  "is_high_risk": "$.risk.score > 80"
}
```

Each mapping expression is a CEL expression evaluated against the previous state's output. Results are validated against the next state's input schema.

---

## 10. Trigger Bindings & Event Routing

Trigger bindings map events to workflows.

### Structure

| Field | Description |
|-------|-------------|
| `event_type` | Event type to match (e.g., `form.submitted`) |
| `event_filter` | CEL expression for filtering (empty = match all) |
| `workflow_name` | Target workflow definition |
| `workflow_version` | Target version |
| `input_mapping` | CEL mapping from event payload to workflow input |
| `active` | Enable/disable binding |

### Routing Flow

1. Event arrives via NATS (published by outbox scheduler)
2. Event router finds trigger bindings matching `event_type` + `tenant_id`
3. For each binding, evaluates `event_filter` CEL expression against event payload
4. If filter matches (or is empty), creates workflow instance with initial execution
5. Multiple bindings can match the same event (fan-out)

### CEL Filter Examples

```cel
// Match high-value orders
payload.amount > 10000

// Match specific form
payload.form_id == "kyc-verification"

// Match webhook from specific source
payload.headers["X-Webhook-Event"] == "payment.completed"
```

---

## 11. Schedulers

Six background schedulers manage the workflow lifecycle. All use `FOR UPDATE SKIP LOCKED` for safe multi-node operation.

### Dispatch Scheduler

| Property | Value |
|----------|-------|
| Interval | 5 seconds (configurable) |
| Batch Size | 100 (configurable) |
| Query | `status = 'pending' ORDER BY created_at` |
| Action | Transitions to `dispatched`, publishes to NATS |

### Retry Scheduler

| Property | Value |
|----------|-------|
| Interval | 10 seconds (configurable) |
| Batch Size | 50 (configurable) |
| Query | `status = 'retry_scheduled' AND next_retry_at <= NOW()` |
| Action | Creates new `pending` execution with `attempt + 1` |

### Timeout Scheduler

| Property | Value |
|----------|-------|
| Interval | 30 seconds (configurable) |
| Batch Size | 50 (configurable) |
| Query | `status = 'dispatched' AND created_at < NOW() - timeout` |
| Action | Marks `timed_out`, attempts retry or fails instance |

### Outbox Scheduler

| Property | Value |
|----------|-------|
| Interval | 5 seconds (configurable) |
| Batch Size | 100 (configurable) |
| Query | `published = false AND deleted_at IS NULL` |
| Action | Publishes event to NATS, marks as published (atomic transaction) |

### Cleanup Scheduler

| Property | Value |
|----------|-------|
| Interval | 6 hours (configurable) |
| Retention | 90 days (configurable) |
| Action | Hard-deletes old published events and audit events (batch of 1000) |

### Cron Scheduler

| Property | Value |
|----------|-------|
| Interval | 30 seconds (fixed) |
| Batch Size | 50 (fixed) |
| Query | `active = true AND next_fire_at <= NOW()` |
| Action | Creates `schedule.fired` event, computes next fire time |

Schedule definitions support duration-based expressions (e.g., `1h`, `30m`, `24h`, `7d`). Each schedule fires events with idempotency keys (`schedule_id:fired_at_timestamp`).

---

## 12. Queue Workers

### Execution Worker

Consumes `ExecutionCommand` messages from NATS JetStream.

**NATS Configuration:**
- Stream: `svc_trustage_executions` (workqueue retention)
- Consumer: `exec-worker` (durable, explicit ACK)
- Max delivery: 3, ACK wait: 30s, max ACK pending: 20

**Processing:**
1. Deserialize ExecutionCommand
2. Load workflow definition by name/version
3. Parse DSL, find current step
4. If `call` step: resolve adapter from registry, execute with idempotency key
5. Commit result (success output or classified error) via engine

### Event Router Worker

Consumes `IngestedEventMessage` from NATS JetStream.

**NATS Configuration:**
- Stream: `svc_trustage_events` (limits retention, 30-day max age)
- Consumer: `event-router` (durable, explicit ACK)
- Max delivery: 3, ACK wait: 10s

**Processing:**
1. Deserialize IngestedEventMessage
2. Route event to matching trigger bindings
3. Create workflow instances for each match

---

## 13. Retry Policies & Error Handling

### Retry Policy Model

| Field | Default | Description |
|-------|---------|-------------|
| `max_attempts` | 3 | Maximum retry attempts |
| `backoff_strategy` | `"exponential"` | Backoff algorithm |
| `initial_delay_ms` | 1000 | Initial delay (1s) |
| `max_delay_ms` | 300000 | Maximum delay (5m) |
| `retry_on` | `["retryable", "external_dependency"]` | Error classes to retry |

### Exponential Backoff with Full Jitter

```
delay = min(initial_delay * 2^(attempt-1), max_delay)
jittered_delay = random(0, delay)  // Full jitter prevents thundering herd
next_retry_at = now + jittered_delay
```

**Example with defaults:**
- Attempt 1: random [0, 1s]
- Attempt 2: random [0, 2s]
- Attempt 3: random [0, 4s]
- Attempt 4+: random [0, 5m] (capped)

### Error Flow

```
Worker Error → Classify (retryable/fatal/compensatable/external_dependency)
  → If retryable + retries remaining: status = retry_scheduled, compute backoff
  → If retryable + retries exhausted: status = fatal, instance = failed
  → If fatal: status = fatal, instance = failed
```

---

## 14. Multi-Tenant Isolation

Strict tenant isolation is enforced at every layer:

| Layer | Mechanism |
|-------|-----------|
| **Database** | `tenant_id NOT NULL` on every table; every query includes `WHERE tenant_id = ?` |
| **Indexes** | Composite indexes include `tenant_id`; unique constraints include `tenant_id` |
| **Cache** | All Valkey keys prefixed with `tenant_id` |
| **API** | Tenant extracted from OIDC claims |
| **Audit** | Every audit event includes `tenant_id` |

No cross-tenant data access is possible at the database layer.

---

## 15. Caching

### Architecture

| Layer | Backend | Scope | TTL |
|-------|---------|-------|-----|
| L1 | In-process LRU | Per-instance | Unbounded (bounded by size) |
| L2 | Valkey | Shared cluster | 10 minutes |
| L3 | PostgreSQL | Source of truth | Permanent |

### Cache Usage

- **Schema registry**: Compiled schemas (L1, max 1000), schema blobs (L2, 10min TTL)
- **DSL specs**: Parsed workflow specs (L1, max 200), DSL blobs (L2, 10min TTL)
- **CEL ASTs**: Compiled trigger filter expressions (L1, max 500)
- **Rate limiting**: Per-tenant counters with time-bucketed keys

### Fallback Behavior

If Valkey is unavailable at startup, the service falls back to in-memory cache and continues operating. Functionality is preserved but cache is not shared across instances.

---

## 16. Security

### Authentication

- OIDC-based via Frame's security manager
- Tenant ID extracted from claims on every authenticated request
- All endpoints except `/healthz` and `/readyz` require authentication

### Execution Tokens

- 64-byte cryptographically secure random tokens (128-char hex)
- Only SHA-256 hash stored in database; raw token never persisted
- Single-use: atomically consumed on commit (prevents replay)
- New token generated at each dispatch (not reused across retries)

### Credential Handling

- Credentials encrypted with AES-256-GCM
- Cached in Valkey with 5-minute TTL
- Never logged, never stored in messages or audit events

### SSRF Protection

All outbound HTTP requests validate URLs against private IP ranges, reserved hostnames, and non-HTTP schemes.

---

## 17. Observability

### OpenTelemetry Metrics

**Counters:**
| Metric | Description |
|--------|-------------|
| `engine.executions.total` | Total executions dispatched |
| `engine.transitions.total` | State transitions (by from/to) |
| `engine.retries.total` | Retry attempts |
| `engine.contract_violations.total` | Schema validation failures |
| `engine.stale_executions.total` | CAS failures |
| `connector.calls.total` | Connector adapter calls |
| `events.ingested.total` | Events received |
| `events.routed.total` | Events routed to triggers |

**Histograms (latency in ms):**
| Metric | Description |
|--------|-------------|
| `engine.dispatch.latency_ms` | Dispatch latency |
| `engine.commit.latency_ms` | Commit latency |
| `connector.latency_ms` | Connector call latency |

**Gauges:**
| Metric | Description |
|--------|-------------|
| `scheduler.pending_executions` | Pending execution count |
| `scheduler.retry_due_executions` | Retry-due count |
| `scheduler.dispatched_executions` | Dispatched count |
| `scheduler.unpublished_events` | Unpublished event count |

### Trace Spans

Every operation creates OpenTelemetry spans: `engine.dispatch`, `engine.commit`, `connector.execute`, `event.route`, `scheduler.dispatch`, `scheduler.retry`, `scheduler.timeout`, `scheduler.outbox`.

### Audit Trail

Append-only audit events with types: `instance.created`, `instance.completed`, `instance.failed`, `state.dispatched`, `state.completed`, `state.failed`, `state.retried`, `state.timed_out`, `transition.committed`, `signal.received`, `workflow.created`, `workflow.activated`, `trigger.matched`.

Queryable per-instance via `GET /api/v1/instances/{id}/timeline`.

### Logging

All logging uses `util.Log(ctx)` with structured fields: `tenant_id`, `workflow_id`, `step_id`, `event_id`, `execution_id`. Credentials and PII are never logged.

---

## 18. Data Models

### Core Models

| Model | Table | Key Fields |
|-------|-------|------------|
| WorkflowDefinition | `workflow_definitions` | Name, Version, Status, DSLBlob, TimeoutSeconds |
| WorkflowInstance | `workflow_instances` | WorkflowName, Version, CurrentState, Status, Revision, TriggerEventID |
| WorkflowStateExecution | `workflow_state_executions` | ExecutionID, InstanceID, State, Attempt, Status, ExecutionToken |
| WorkflowStateSchema | `workflow_state_schemas` | WorkflowName, Version, State, SchemaType, SchemaHash, SchemaBlob |
| WorkflowStateMapping | `workflow_state_mappings` | WorkflowName, Version, FromState, ToState, MappingExpr |
| WorkflowStateOutput | `workflow_state_outputs` | ExecutionID, InstanceID, State, SchemaHash, Payload |
| WorkflowRetryPolicy | `workflow_retry_policies` | WorkflowName, Version, State, MaxAttempts, BackoffStrategy |
| WorkflowAuditEvent | `workflow_audit_events` | InstanceID, ExecutionID, EventType, State, FromState, ToState |
| EventLog | `event_log` | EventType, Source, IdempotencyKey, Payload, Published |
| TriggerBinding | `trigger_bindings` | EventType, EventFilter, WorkflowName, Version, InputMapping, Active |
| ConnectorConfig | `connector_configs` | ConnectorType, Name, Config, Active |
| ConnectorCredential | `connector_credentials` | ConnectorType, CredentialBlob, KeyVersion |
| ScheduleDefinition | `schedule_definitions` | Name, CronExpr, WorkflowName, Version, InputPayload, Active, NextFireAt |

All models include `TenantID`, `PartitionID`, `CreatedAt`, `ModifiedAt`, `DeletedAt` (soft delete).

### Key Database Indexes

| Index | Table | Purpose |
|-------|-------|---------|
| `idx_wse_pending` | workflow_state_executions | Dispatch scheduler: `WHERE status = 'pending'` |
| `idx_wse_retry` | workflow_state_executions | Retry scheduler: `WHERE status = 'retry_scheduled'` |
| `idx_wse_dispatched` | workflow_state_executions | Timeout scheduler: `WHERE status = 'dispatched'` |
| `idx_el_unpublished` | event_log | Outbox scheduler: `WHERE published = false` |
| `idx_sd_due` | schedule_definitions | Cron scheduler: `WHERE active = true` |
| `idx_tb_event` | trigger_bindings | Event routing: `WHERE active = true` |

---

## 19. Configuration Reference

All configuration via environment variables:

### Server
| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | `:8080` | HTTP server port |

### Cache
| Variable | Default | Description |
|----------|---------|-------------|
| `VALKEY_CACHE_URL` | `redis://localhost:6379` | Valkey/Redis connection URL |

### Security
| Variable | Default | Description |
|----------|---------|-------------|
| `MASTER_ENCRYPTION_KEY` | (required) | AES-256 key for credential encryption |

### Scheduler Intervals
| Variable | Default | Description |
|----------|---------|-------------|
| `DISPATCH_INTERVAL_SECONDS` | 5 | Dispatch scheduler interval |
| `RETRY_INTERVAL_SECONDS` | 10 | Retry scheduler interval |
| `TIMEOUT_INTERVAL_SECONDS` | 30 | Timeout scheduler interval |
| `OUTBOX_INTERVAL_SECONDS` | 5 | Outbox publisher interval |
| `CLEANUP_INTERVAL_HOURS` | 6 | Cleanup scheduler interval |

### Scheduler Batch Sizes
| Variable | Default | Description |
|----------|---------|-------------|
| `DISPATCH_BATCH_SIZE` | 100 | Executions per dispatch cycle |
| `RETRY_BATCH_SIZE` | 50 | Retries per cycle |
| `TIMEOUT_BATCH_SIZE` | 50 | Timeouts per cycle |
| `OUTBOX_BATCH_SIZE` | 100 | Events per outbox cycle |

### Execution
| Variable | Default | Description |
|----------|---------|-------------|
| `DEFAULT_EXECUTION_TIMEOUT_SECONDS` | 300 | Max execution time (5 min) |
| `EVENT_INGEST_RATE_LIMIT` | 100 | Events per minute per tenant |
| `RETENTION_DAYS` | 90 | Audit/event retention |

### Queue (NATS JetStream)
| Variable | Description |
|----------|-------------|
| `QUEUE_EXEC_DISPATCH_NAME` | Execution dispatch publisher name |
| `QUEUE_EXEC_DISPATCH_URL` | NATS URL with JetStream stream config |
| `QUEUE_EXEC_WORKER_NAME` | Execution worker subscriber name |
| `QUEUE_EXEC_WORKER_URL` | NATS URL with consumer config |
| `QUEUE_EVENT_INGEST_NAME` | Event ingest publisher name |
| `QUEUE_EVENT_INGEST_URL` | NATS URL with JetStream stream config |
| `QUEUE_EVENT_ROUTER_NAME` | Event router subscriber name |
| `QUEUE_EVENT_ROUTER_URL` | NATS URL with consumer config |

---

## 20. Deployment

### Kubernetes Resources

Trustage is deployed via Flux CD with the following resources:

| Resource | Namespace | Description |
|----------|-----------|-------------|
| Namespace | `trustage` | Service namespace |
| HelmRelease | `trustage` | Colony chart v1.6.1, 2 replicas |
| Database (CNPG) | `datastore` | PostgreSQL database on `hub` cluster |
| NATS User | `trustage` | JetStream credentials with `svc.trustage.>` permissions |
| JetStream Streams | `trustage` | `svc_trustage_executions` (workqueue), `svc_trustage_events` (limits) |
| HTTPRoute | `trustage` | Gateway route for `trustage.stawi.dev` |
| ExternalSecrets | `trustage` | DB credentials, GHCR auth, OAuth2 client from Vault |

### Image

```
ghcr.io/antinvestor/service-trustage:v0.1.1
```

### Resource Limits

```yaml
requests:
  cpu: 50m
  memory: 128Mi
limits:
  cpu: 500m
  memory: 512Mi
```
