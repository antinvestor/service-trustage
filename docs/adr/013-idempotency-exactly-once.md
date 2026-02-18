# ADR-013: Idempotency and Exactly-Once Delivery

## Status

Accepted

## Context

The Orchestrator processes events through multiple layers: NATS JetStream for event ingestion, an event router for trigger evaluation, the state engine for workflow execution, and connector adapters for external API calls. Each layer has its own retry and failure-recovery mechanisms. NATS redelivers messages when consumers do not acknowledge within the timeout. The event router may crash mid-processing and restart. The engine's retry scheduler retries failed executions with configurable backoff. External APIs may timeout without confirming whether the operation succeeded.

At every boundary between these layers, there is a window where duplicate processing can occur. A NATS consumer processes an event, creates a workflow instance, but crashes before acknowledging the message. NATS redelivers. Now two instances are attempted for the same event. Or a connector worker calls an email API, the API sends the email, but the HTTP response is lost due to a network partition. The engine retries the execution. Now two emails are sent.

The consequences of duplicate processing range from annoying (duplicate notification emails) to severe (duplicate charges, duplicate CRM entries, duplicate webhook deliveries to customer systems). For a workflow automation platform handling business-critical processes, idempotency is not optional. Every layer must either guarantee exactly-once processing or provide mechanisms for the next layer to deduplicate.

## Decision

Implement idempotency at each processing layer, using the strongest mechanism available at that layer. The strategy is defense-in-depth: no single layer relies on another for deduplication.

### Layer-by-Layer Idempotency

| Layer | Mechanism | Dedup Key | Window |
|-------|-----------|-----------|--------|
| Event publishing | NATS JetStream message deduplication | `Nats-Msg-Id` header = `event_log.id` (ULID) | 2 minutes (configurable via `MaxAge` on dedup) |
| Event routing | Deterministic workflow instance ID (unique constraint) | `wf-{tenant_id}-{workflow_def_id}-{event_id}` | Permanent (instance ID is globally unique) |
| Workflow execution | Execution tokens (single-use, CAS transitions) | `execution_token` per attempt | Lifetime of execution attempt |
| Connector calls | Adapter-level idempotency keys | External API's idempotency mechanism (where supported) | Varies by API |

### Instance ID Format

```
wf-{tenant_id}-{workflow_def_id}-{event_id}
```

**Example:** `wf-tenant_abc-wfdef_order_proc-evt_01HQXYZ`

This format is deterministic: the same event processed by the same workflow definition always produces the same instance ID. When the event router calls the engine to create an instance with an ID that already exists, the unique constraint prevents a duplicate. This is the primary deduplication mechanism.

### NATS JetStream Deduplication

Events are published to NATS with the `Nats-Msg-Id` header set to the `event_log.id`:

```go
_, err := js.Publish(subject, payload, nats.MsgId(event.ID))
```

JetStream tracks message IDs within the deduplication window and silently discards duplicates. This prevents duplicate events from entering the stream even before they reach the event router.

### Execution Token Deduplication

Each state execution receives a single-use `execution_token` at dispatch time. When a worker commits results via the Commit API, the token is verified and consumed. If a duplicate execution attempts to commit with the same token, the verification fails. Combined with CAS transitions on `workflow_instances`, this prevents double-advancement of workflow state.

### Connector-Level Idempotency

Each connector adapter is responsible for implementing idempotency using the mechanisms provided by the external API:

| Connector | Idempotency Mechanism |
|-----------|-----------------------|
| Webhook | `X-Idempotency-Key` header = `{instance_id}-{step_id}-{attempt}` |
| Email (SendGrid/SES) | Dedup via message ID in metadata |
| Stripe | `Idempotency-Key` header (native support) |
| CRM (HubSpot/Salesforce) | Upsert by external ID rather than create |
| Slack | Check for existing message before posting (best-effort) |

For APIs that do not support idempotency keys, the connector operates in at-least-once mode. The worker records its result in a side table (`connector_call_log`) and checks for existing results before making the external call.

### Connector Call Log

```sql
CREATE TABLE connector_call_log (
    idempotency_key TEXT PRIMARY KEY,   -- {instance_id}-{step_id}
    connector_type  TEXT NOT NULL,
    request_hash    TEXT NOT NULL,       -- SHA-256 of request payload
    response        JSONB,
    status          TEXT NOT NULL,       -- pending, success, failed
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at    TIMESTAMPTZ
);
```

Before making an external call, the worker checks this table. If a record exists with status `success`, it returns the cached response without calling the external API.

## Alternatives Considered

| Option | Pros | Cons |
|--------|------|------|
| **Multi-layer defense-in-depth (chosen)** | Each layer independently prevents duplicates. No single point of failure for dedup. Uses native mechanisms at each layer (strongest guarantees). | Multiple dedup mechanisms to implement and maintain. Connector-level dedup varies by API. Some APIs have no idempotency support. |
| **Single dedup at event router only** | Simple. One place to implement and reason about. | If event router dedup fails (crash between dedup check and instance creation), duplicates leak through. No protection at connector layer. |
| **Exactly-once NATS consumers (no further dedup)** | NATS JetStream provides exactly-once consumer semantics. Minimal code. | Exactly-once in NATS means exactly-once delivery to consumer, not exactly-once processing. Consumer can still crash after processing but before ack. Does not cover engine retry at all. |
| **Global dedup service** | Centralized dedup logic. Consistent behavior across all layers. | Additional infrastructure. Single point of failure. Latency on every operation for dedup check. Over-engineered for the actual failure modes. |

## Rationale

1. **Deterministic instance ID uniqueness is the primary and most reliable dedup mechanism.** A deterministic instance ID derived from `tenant_id + workflow_def_id + event_id` means the same event can never start the same workflow twice. PostgreSQL's unique constraint enforces this at the database level, regardless of how many times the event router attempts to create the instance.

2. **NATS JetStream dedup is a belt-and-suspenders measure for event publishing.** It prevents duplicate events from entering the stream, reducing unnecessary work for the event router. The 2-minute dedup window covers the realistic retry window for publisher failures.

3. **Execution tokens prevent duplicate state advancement.** Each execution attempt has a single-use token that is consumed at commit time. Combined with CAS transitions, this ensures that even if a worker retries, only one commit succeeds.

4. **Connector-level idempotency is an adapter responsibility, not a framework guarantee.** External APIs have wildly different idempotency capabilities. Stripe has native idempotency keys. Slack has none. The framework provides the building blocks (`instance_id + step_id` as a key, `connector_call_log` as a cache), but each adapter must implement the appropriate strategy for its target API.

5. **Defense-in-depth is justified by the severity of duplicate side effects.** A duplicate email is annoying. A duplicate charge is a support incident. A duplicate webhook to a customer's system can trigger duplicate downstream processing. The cost of implementing dedup at each layer is low compared to the cost of duplicate business operations.

## Consequences

**Positive:**

- Same event + same workflow definition = same instance ID (deterministic, guaranteed by PostgreSQL unique constraint)
- NATS dedup prevents duplicate events from entering the stream
- Execution tokens prevent duplicate state transitions
- Connector adapters provide idempotency using the strongest mechanism available for each API
- `connector_call_log` provides best-effort dedup even for APIs without native idempotency
- No duplicate workflow instances, regardless of event router retries or crashes
- Audit trail in `connector_call_log` shows all external API calls with their idempotency keys

**Negative:**

- Multiple dedup mechanisms to implement, test, and maintain across layers
- Connector-level idempotency varies by API (some are exactly-once, others are best-effort)
- `connector_call_log` adds a database write per connector call (latency overhead)
- NATS dedup window (2 minutes) is finite; very late retries may not be caught at this layer
- Deterministic instance ID format means one workflow definition processes one event exactly once (cannot intentionally re-process without a new event ID)

## Implementation Notes

### Event Router Dedup Flow

```go
func (r *EventRouter) createWorkflowInstance(ctx context.Context, event Event, binding TriggerBinding) error {
    instanceID := fmt.Sprintf("wf-%s-%s-%s", event.TenantID, binding.WorkflowDefID, event.ID)

    instance := &WorkflowInstance{
        ID:              instanceID,
        TenantID:        event.TenantID,
        WorkflowName:    binding.WorkflowName,
        WorkflowVersion: binding.WorkflowVersion,
        Status:          "running",
        CurrentState:    binding.InitialState,
        Revision:        0,
    }

    err := r.engine.CreateInstance(ctx, instance, event.Payload)
    if err != nil {
        // Instance already exists — this is expected for dedup, not an error
        if isUniqueConstraintViolation(err) {
            r.logger.Info("workflow instance already exists for event", "instance_id", instanceID)
            return nil
        }
        return fmt.Errorf("create workflow instance: %w", err)
    }

    r.logger.Info("created workflow instance", "instance_id", instanceID)
    return nil
}
```

### Connector Worker Dedup Pattern

```go
func (w *WebhookWorker) Execute(ctx context.Context, cmd ExecutionCommand) (*ExecutionResult, *ExecutionError) {
    idempotencyKey := fmt.Sprintf("%s-%s", cmd.InstanceID, cmd.State)

    // Check connector_call_log for existing result
    existing, err := w.callLog.Get(ctx, idempotencyKey)
    if err == nil && existing.Status == "success" {
        return existing.Response, nil // Return cached result
    }

    // Make the external call with idempotency key
    resp, err := w.httpClient.Post(input.URL, input.Payload, map[string]string{
        "X-Idempotency-Key": idempotencyKey,
    })

    // Record result in connector_call_log
    w.callLog.Put(ctx, idempotencyKey, resp, err)

    return resp, err
}
```

### Monitoring and Alerting

| Metric | Description | Alert Threshold |
|--------|-------------|-----------------|
| `orchestrator_dedup_nats_duplicates_total` | Events rejected by NATS JetStream dedup | > 100/min (indicates publisher retry storm) |
| `orchestrator_dedup_instance_already_exists_total` | Instance creation rejected by unique constraint | > 50/min (indicates event router retry storm) |
| `orchestrator_dedup_connector_cache_hits_total` | Connector calls served from `connector_call_log` | Informational (indicates execution retries) |
| `orchestrator_connector_call_duration_seconds` | External API call latency | p99 > 10s (indicates timeout risk) |
