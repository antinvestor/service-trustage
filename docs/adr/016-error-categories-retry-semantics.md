# ADR-016: Error Categories and Default Retry Semantics

## Status

Accepted

## Context

The Orchestrator executes workflows whose individual states call external systems (APIs, webhooks, email providers, CRM platforms) and internal domain logic. These operations fail in qualitatively different ways. A network timeout is fundamentally different from an invalid API key, which is fundamentally different from a third-party outage, which is fundamentally different from a business rule violation that requires compensating actions.

ADR-001 established the four error classes (`retryable`, `fatal`, `compensatable`, `external_dependency`) and the retry policy table. ADR-013 established idempotency guarantees that make retries safe. However, neither document fully specifies:

1. What concrete failure conditions map to each error class
2. What the default retry behavior is when no explicit policy is configured
3. How error classes interact with state transitions, audit events, and alerting
4. How workers should classify ambiguous errors (e.g., HTTP 429, partial failures)
5. How the engine handles error class escalation (e.g., retryable exhaustion → fatal)

Workers must classify every error at compile time (enforced by the `ExecutionError` type), but without a clear taxonomy of what belongs in each class, different teams will classify the same failure differently. A partner team might classify an HTTP 429 as `fatal` (because the API rejected the request) when it should be `retryable` (because the rejection is transient). This inconsistency undermines the engine's ability to apply correct retry behavior.

This ADR codifies the error taxonomy, assigns default retry semantics to each class, and defines the engine's behavior for each classification.

## Decision

### Error Class Taxonomy

The engine recognizes exactly four error classes. This is a closed set — no additional classes may be added without an ADR.

#### `retryable`

Transient failures where the same request, with the same input, is expected to succeed on a subsequent attempt.

| Signal | Examples |
|--------|----------|
| HTTP 408 (Request Timeout) | API did not respond within deadline |
| HTTP 429 (Too Many Requests) | Rate limit exceeded, retry after cooldown |
| HTTP 502 (Bad Gateway) | Upstream proxy error |
| HTTP 503 (Service Unavailable) | Server temporarily overloaded |
| HTTP 504 (Gateway Timeout) | Upstream timeout |
| Connection refused | Target service not yet available |
| Connection reset | Network interruption mid-request |
| DNS resolution failure | Transient DNS issue |
| TLS handshake timeout | Slow TLS negotiation |
| Context deadline exceeded | Worker's own timeout fired |
| Database connection pool exhausted | Transient resource contention |
| Serialization conflict (PostgreSQL 40001) | Concurrent transaction conflict |

**Default engine behavior:**

| Parameter | Default Value |
|-----------|---------------|
| Max attempts | 3 |
| Backoff strategy | Exponential |
| Initial delay | 1 second |
| Max delay | 5 minutes |
| Jitter | ±20% of calculated delay |

**On exhaustion:** When all retry attempts are exhausted, the error **escalates to `fatal`**. The engine follows the `on_fatal` transition if defined, otherwise marks the workflow instance as `failed`.

#### `fatal`

Permanent failures where retrying with the same input will never succeed. The worker has determined that the failure is deterministic and non-recoverable.

| Signal | Examples |
|--------|----------|
| HTTP 400 (Bad Request) | Malformed request payload |
| HTTP 401 (Unauthorized) | Invalid or expired credentials |
| HTTP 403 (Forbidden) | Insufficient permissions |
| HTTP 404 (Not Found) | Target resource does not exist |
| HTTP 409 (Conflict) | Business-level conflict (e.g., duplicate entity) |
| HTTP 422 (Unprocessable Entity) | Semantic validation failure |
| JSON unmarshalling error | Response not parseable |
| Schema validation failure | Worker output does not match output schema |
| Business rule rejection | Domain logic determined the operation is invalid |
| Invalid configuration | Connector config is malformed or missing required fields |
| Authentication failure | OAuth token refresh failed permanently |

**Default engine behavior:**

| Parameter | Default Value |
|-----------|---------------|
| Max attempts | 1 (no retry) |
| Backoff strategy | N/A |

**On occurrence:** The engine immediately follows the `on_fatal` transition if defined. If no `on_fatal` transition exists, the workflow instance status is set to `failed`. An audit event of type `execution.fatal` is recorded with the full error detail.

#### `external_dependency`

The external system that the worker depends on is experiencing a sustained outage. Distinguished from `retryable` because the expected recovery time is longer (minutes to hours rather than milliseconds to seconds).

| Signal | Examples |
|--------|----------|
| HTTP 503 persisting across multiple attempts | Sustained service outage |
| Circuit breaker open | Too many consecutive failures to the same endpoint |
| External status page reports incident | Known third-party outage |
| DNS SERVFAIL or NXDOMAIN for a known-good domain | DNS infrastructure issue |
| TLS certificate errors on a previously working endpoint | Certificate rotation issue |
| Consecutive connection timeouts (>3) | Network path failure |

**Default engine behavior:**

| Parameter | Default Value |
|-----------|---------------|
| Max attempts | 10 |
| Backoff strategy | Exponential |
| Initial delay | 30 seconds |
| Max delay | 30 minutes |
| Jitter | ±20% of calculated delay |

**On exhaustion:** Escalates to `fatal`. The engine follows the `on_fatal` transition. An alert-level audit event `execution.dependency_exhausted` is emitted, indicating that operator intervention may be required.

**Distinction from `retryable`:** Workers SHOULD start with `retryable` for transient errors. If a worker detects a pattern of repeated failures (e.g., circuit breaker opens, or the worker itself is on its Nth retry attempt and still seeing 503s), it SHOULD classify the error as `external_dependency` to signal longer backoff. The engine also automatically escalates: if a `retryable` execution fails 3 consecutive times with the same HTTP status, the retry scheduler reclassifies remaining attempts under the `external_dependency` policy.

#### `compensatable`

The operation partially succeeded or produced side effects that must be reversed. The failure requires running a compensation workflow rather than simply retrying.

| Signal | Examples |
|--------|----------|
| Partial batch processing | 80 of 100 records processed, 20 failed |
| Payment captured but fulfillment failed | Money collected, goods not shipped |
| External entity created but local commit failed | CRM contact created, workflow state not updated |
| Multi-step operation partially completed | First API call succeeded, second failed |
| Webhook delivered but response indicates downstream failure | Partner system received data but could not process it |

**Default engine behavior:**

| Parameter | Default Value |
|-----------|---------------|
| Max attempts | 1 (no retry of the original operation) |
| Compensation | Engine looks up `compensation_workflow` on the state definition |

**On occurrence:** The engine does NOT retry the original execution. Instead:

1. Records the partial result in `workflow_state_outputs` with `status = 'compensating'`
2. If a `compensation_workflow` is defined on the state, creates a new workflow instance for the compensation workflow, passing the original execution's output and error as input
3. If no `compensation_workflow` is defined, marks the execution as `fatal` and follows `on_fatal`
4. An audit event of type `execution.compensating` is recorded

### Default Retry Policy

When a state definition does not include an explicit `retry` block, the engine applies a default retry policy based on the error class:

```go
var DefaultRetryPolicies = map[ErrorClass]RetryPolicy{
    ErrorRetryable: {
        MaxAttempts:     3,
        BackoffStrategy: "exponential",
        InitialDelay:    1 * time.Second,
        MaxDelay:        5 * time.Minute,
    },
    ErrorFatal: {
        MaxAttempts: 1,
    },
    ErrorCompensatable: {
        MaxAttempts: 1,
    },
    ErrorExternalDependency: {
        MaxAttempts:     10,
        BackoffStrategy: "exponential",
        InitialDelay:    30 * time.Second,
        MaxDelay:        30 * time.Minute,
    },
}
```

Explicit per-state `retry` blocks in the DSL override these defaults entirely. The `retry_on` array in the policy controls which error classes trigger retry:

```json
{
  "retry": {
    "max_attempts": 5,
    "backoff": "exponential",
    "initial_delay": "10s",
    "max_delay": "10m",
    "retry_on": ["retryable", "external_dependency"]
  }
}
```

If `retry_on` does not include an error class, the engine treats that class as if `max_attempts = 1` (no retry).

### Jitter

All backoff calculations include ±20% jitter to prevent thundering herd when multiple executions retry against the same external system:

```go
func calculateDelayWithJitter(policy *RetryPolicy, attempt int) time.Duration {
    base := calculateDelay(policy, attempt)
    jitterRange := float64(base) * 0.2
    jitter := (rand.Float64()*2 - 1) * jitterRange // [-20%, +20%]
    return time.Duration(float64(base) + jitter)
}
```

### Retry Exhaustion and Escalation

When retry attempts are exhausted:

```
retryable (exhausted)         → fatal → on_fatal transition
external_dependency (exhausted) → fatal → on_fatal transition + alert
compensatable                 → compensation workflow → on_compensation_complete / on_compensation_failed
fatal                         → on_fatal transition (immediate, no retry)
```

The engine records the escalation chain in the audit log:

```json
{
  "event_type": "execution.retry_exhausted",
  "execution_id": "exec_01HQX...",
  "original_error_class": "retryable",
  "escalated_to": "fatal",
  "total_attempts": 3,
  "first_failure_at": "2026-02-14T10:00:00Z",
  "last_failure_at": "2026-02-14T10:00:06Z",
  "error_summary": "HTTP 503: service temporarily unavailable"
}
```

### Error Classification Decision Tree for Workers

Workers MUST follow this decision tree when classifying errors:

```
1. Did the operation produce side effects that need reversal?
   YES → compensatable
   NO  → continue

2. Is the error caused by invalid input, bad config, or business logic?
   YES → fatal
   NO  → continue

3. Is the external system experiencing a sustained outage?
   (circuit breaker open, multiple consecutive failures, known incident)
   YES → external_dependency
   NO  → continue

4. Is the error transient?
   (timeout, rate limit, temporary network issue)
   YES → retryable
   NO  → fatal (when in doubt, fail fast and let operators investigate)
```

### HTTP Status Code Quick Reference

| Status Code | Default Classification | Notes |
|-------------|----------------------|-------|
| 400 | `fatal` | Bad request — fix the input |
| 401 | `fatal` | Credentials invalid — cannot retry with same creds |
| 403 | `fatal` | Permission denied — structural issue |
| 404 | `fatal` | Resource not found — will not appear on retry |
| 408 | `retryable` | Server timeout — transient |
| 409 | `fatal` | Conflict — business-level duplicate |
| 422 | `fatal` | Validation failure — fix the input |
| 429 | `retryable` | Rate limited — respect `Retry-After` header if present |
| 500 | `retryable` | Server error — may be transient |
| 502 | `retryable` | Bad gateway — proxy issue, usually transient |
| 503 | `retryable` or `external_dependency` | Start as `retryable`; escalate to `external_dependency` if persistent |
| 504 | `retryable` | Gateway timeout — transient |

**Special case: HTTP 429 with `Retry-After` header.** When a 429 response includes a `Retry-After` header, the worker SHOULD parse it and pass the delay hint to the engine via `ExecutionError.RetryAfter`. The engine uses `max(calculated_backoff, retry_after)` as the actual delay.

### Timeout as an Error Class

Execution timeouts (the timeout scheduler marking an execution as `timed_out`) are treated as `retryable` by default. The engine creates a retry attempt if the retry policy allows it. If the same execution times out across all retry attempts, it escalates to `fatal`.

The per-state `timeout` in the DSL sets the maximum wall-clock time for a single execution attempt:

```json
{
  "timeout": "5m"
}
```

Default timeout when not specified: **5 minutes**. Workers SHOULD set their own context deadline slightly below the state timeout to allow time for error classification and commit.

### Schema Validation Failures

Input and output schema validation failures are always `fatal` and are never retried:

- `invalid_input_contract`: The input failed schema validation before dispatch. The mapping or upstream state produced invalid data. This is a workflow definition bug, not a runtime issue.
- `invalid_output_contract`: The worker's output failed schema validation at commit. The worker produced data that does not match its declared output schema. This is a worker implementation bug.

These are recorded as distinct execution statuses (not error classes) because they are engine-detected, not worker-classified.

## Alternatives Considered

| Option | Pros | Cons | Verdict |
|--------|------|------|---------|
| **Four closed error classes with defaults (chosen)** | Simple taxonomy. Workers make one decision. Engine behavior is deterministic and predictable. Default policies cover common cases. | Less granular than per-error-code policies. Workers must map nuanced failures to four buckets. | **Chosen** |
| **Open error class set (extensible enum)** | Teams can define domain-specific error classes. More precise classification. | Engine must handle unknown classes. Policy configuration explodes. Different teams invent overlapping classes. No consistent behavior guarantees. | Rejected |
| **Per-HTTP-status-code retry policy** | Maximum precision for HTTP-based connectors. | Doesn't generalize to non-HTTP workers (database, file system, internal logic). Explosion of configuration surface. Most status codes map cleanly to the four classes anyway. | Rejected |
| **No default policies (all explicit)** | Every state must declare retry behavior. No implicit behavior. | Boilerplate explosion. Every workflow definition must include retry blocks on every state. Increases onboarding friction for partners. | Rejected |
| **Engine-side automatic classification (no worker input)** | Workers don't need to think about error classes. | Engine cannot distinguish between a 500 that means "server bug" and a 500 that means "your request caused an edge case." Workers have domain context the engine lacks. | Rejected |

## Rationale

1. **Four error classes are sufficient for the target workloads.** The Orchestrator automates business processes involving API calls, emails, webhooks, and approvals. These operations fail in four ways: transiently, permanently, with side effects, or due to dependency outage. Finer-grained classification adds configuration burden without changing engine behavior.

2. **Workers must classify errors because they have domain context.** An HTTP 500 from a payment API might mean "your card number is invalid" (fatal) or "our server crashed" (retryable). Only the worker, which understands the API's error response format, can make this distinction. The engine applies policy; workers apply judgment.

3. **Default retry policies reduce boilerplate without hiding behavior.** Most states need the same retry behavior: retry transient errors 3 times with exponential backoff, don't retry permanent errors, use longer backoff for dependency outages. Making these the defaults means workflow definitions only need explicit `retry` blocks when they deviate from the norm.

4. **Jitter prevents thundering herd.** When a third-party API recovers from an outage, all workflows retrying against it would otherwise fire at exactly the same backoff intervals, creating a retry storm. ±20% jitter spreads the load.

5. **Retry exhaustion escalates to `fatal` rather than leaving executions in limbo.** A `retryable` error that persists through all attempts is, by definition, no longer transient. Escalating to `fatal` ensures the workflow either follows its error transition or terminates cleanly. Leaving executions in `retry_scheduled` forever would leak resources and confuse operators.

6. **`external_dependency` deserves separate treatment from `retryable`.** A 503 that resolves in 200ms is different from a third-party outage lasting 45 minutes. The longer initial delay (30s vs 1s) and higher max attempts (10 vs 3) reflect this reality. The separate class also enables targeted alerting: `external_dependency` exhaustion triggers operator notification because it likely requires coordination with the external provider.

## Consequences

**Positive:**

- Workers have a clear decision tree for error classification — no ambiguity
- Default retry policies cover 90%+ of states without explicit configuration
- HTTP status codes have a documented default mapping — connector authors don't guess
- Retry exhaustion has deterministic behavior (escalation to `fatal` with audit trail)
- Jitter prevents retry storms against recovering services
- `external_dependency` class enables longer backoff without penalizing `retryable` errors with excessive delays
- Compensation workflows are triggered automatically for `compensatable` errors
- Schema validation failures are never retried (correctness over availability)
- Timeout handling is integrated into the retry flow

**Negative:**

- Four classes may feel coarse for some failure modes (e.g., distinguishing "quota exceeded" from "rate limited" — both map to `retryable`)
- Workers bear the classification burden — incorrect classification leads to incorrect engine behavior
- Default policies may not be optimal for all workloads (e.g., a state calling a slow-to-recover API may need more than 3 retries but less than 10)
- Automatic escalation from `retryable` to `external_dependency` after 3 consecutive same-status failures adds complexity to the retry scheduler
- `Retry-After` header parsing adds HTTP-specific logic to the otherwise protocol-agnostic error model

## Implementation Notes

### ExecutionError Type

```go
type ExecutionError struct {
    Class      ErrorClass      `json:"class"`
    Code       string          `json:"code"`       // Machine-readable error code (e.g., "SMTP_TIMEOUT")
    Message    string          `json:"message"`     // Human-readable description
    Detail     json.RawMessage `json:"detail"`      // Structured error data (validated against error_schema)
    RetryAfter *time.Duration  `json:"retry_after"` // Optional hint from external system
}

// Constructor functions enforce classification at compile time
func NewRetryableError(code, message string, detail json.RawMessage) *ExecutionError {
    return &ExecutionError{Class: ErrorRetryable, Code: code, Message: message, Detail: detail}
}

func NewFatalError(code, message string, detail json.RawMessage) *ExecutionError {
    return &ExecutionError{Class: ErrorFatal, Code: code, Message: message, Detail: detail}
}

func NewCompensatableError(code, message string, detail json.RawMessage) *ExecutionError {
    return &ExecutionError{Class: ErrorCompensatable, Code: code, Message: message, Detail: detail}
}

func NewExternalDependencyError(code, message string, detail json.RawMessage) *ExecutionError {
    return &ExecutionError{Class: ErrorExternalDependency, Code: code, Message: message, Detail: detail}
}
```

### Engine Commit Handling by Error Class

```go
func (e *Engine) handleExecutionError(ctx context.Context, exec *Execution, err *ExecutionError) error {
    policy := e.getRetryPolicy(ctx, exec)

    switch err.Class {
    case ErrorRetryable:
        if exec.Attempt < policy.MaxAttempts {
            return e.scheduleRetry(ctx, exec, policy, err)
        }
        return e.escalateToFatal(ctx, exec, err, "retry_exhausted")

    case ErrorExternalDependency:
        depPolicy := e.getExternalDependencyPolicy(ctx, exec)
        if exec.Attempt < depPolicy.MaxAttempts {
            return e.scheduleRetry(ctx, exec, depPolicy, err)
        }
        return e.escalateToFatal(ctx, exec, err, "dependency_exhausted")

    case ErrorFatal:
        return e.transitionOnFatal(ctx, exec, err)

    case ErrorCompensatable:
        return e.triggerCompensation(ctx, exec, err)
    }

    return fmt.Errorf("unknown error class: %s", err.Class)
}
```

### Monitoring and Alerting

| Metric | Description | Alert Threshold |
|--------|-------------|-----------------|
| `orchestrator_execution_errors_total{class}` | Errors by class | Informational |
| `orchestrator_retry_attempts_total{class}` | Retry attempts by class | > 1000/min (retry storm) |
| `orchestrator_retry_exhausted_total{class}` | Exhausted retries by class | > 10/min (systematic failure) |
| `orchestrator_compensation_triggered_total` | Compensation workflows started | > 5/min (partial failure pattern) |
| `orchestrator_external_dependency_exhausted_total` | Dependency outage retries exhausted | > 0 (requires operator attention) |
| `orchestrator_execution_timeout_total` | Executions timed out | > 50/min (timeout too aggressive or service degraded) |
