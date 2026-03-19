# ADR-010: Rate Limiting and Back-Pressure Strategy

## Status
Accepted

## Context
Without rate limiting, a single tenant could submit millions of events per hour, trigger thousands of concurrent workflows, and flood external APIs with connector calls. This would degrade the platform for all tenants, exhaust infrastructure resources, and potentially cause cascading failures across NATS, the state engine, and downstream systems. Rate limiting is not optional in a multi-tenant workflow automation platform; it is a core reliability requirement.

Foundry implements a three-layer concurrency model using Valkey counters with TTL-based safety mechanisms. The pattern is proven: atomic increments for counting, TTL expiry as a safety net against counter drift, and plan-tier-based limits for commercial differentiation. Orchestrator must extend this model to cover its unique resource types: events, workflows, connector calls, and streaming connections.

The challenge is that rate limiting must happen at multiple points in the execution pipeline. Event ingestion must be limited before events enter NATS. Workflow concurrency must be bounded before the engine schedules new executions. Connector calls must respect per-API rate limits. And the entire pipeline needs back-pressure mechanisms so that bursts are absorbed gracefully rather than causing failures. Each layer protects a different resource and has different failure modes.

## Decision
We adopt a layered admission-control architecture, with each layer protecting a
specific resource type and using the mechanism best suited to that layer.

### Limiter Selection Rules

The service must not treat all overload problems as "rate limiting." Different
failure modes require different controls:

| Failure mode | Correct mechanism | Why |
|--------------|-------------------|-----|
| A tenant or caller can submit too many requests per time window | `frame/ratelimiter.LeasedWindowLimiter` or `WindowLimiter` | Protects a shared distributed request budget |
| Downstream backlog is already too high | `frame/ratelimiter.QueueDepthLimiter` | Stops admitting more work until the system can drain safely |
| Too many expensive operations are running at once in a single worker process | `frame/ratelimiter.ConcurrencyLimiter` | Protects local finite execution capacity |
| Too many active workflow instances exist fleet-wide for a tenant | Distributed Valkey/Postgres-backed concurrency accounting | This is a cross-process business quota, not a local in-process concurrency cap |

These controls are complementary. They should be layered where needed rather
than forced into one limiter type.

### Layer 1: Event Ingestion Rate Limiting

**Resource protected:** NATS JetStream and downstream processing pipeline

| Parameter | Value |
|-----------|-------|
| Mechanism | Sliding window counter in Valkey |
| Key format | `ratelimit:{tenant_id}:events:{window}` |
| Default limit | 1,000 events/minute, 50,000 events/day |
| Configurability | Per plan tier via `plan_limits` table |
| Exceeded behavior | HTTP 429 / ConnectRPC `ResourceExhausted` |
| TTL safety | Window key expires after window duration + buffer |

Events are rate-limited at the API ingestion point, before they enter NATS JetStream. This prevents a burst of events from overwhelming the event router and trigger evaluation pipeline.

### Layer 2: Workflow Concurrency Limiting

**Resource protected:** Engine scheduler and workflow execution resources

| Parameter | Value |
|-----------|-------|
| Mechanism | Atomic counter in Valkey |
| Key format | `active_workflows:{tenant_id}` |
| Default limit | 100 concurrent active workflows |
| Configurability | Per plan tier |
| Exceeded behavior | Event NACKed back to NATS (retried after backoff) |
| TTL safety | Counter key has 1-hour TTL; background reconciliation |

The counter is incremented when a workflow starts and decremented when it completes (success, failure, or cancellation). If a tenant is at their concurrency limit, new triggering events are NACKed back to NATS JetStream, where they wait in the stream and are redelivered after the configured backoff period.

### Layer 3: Connector Rate Limiting

**Resource protected:** External APIs and third-party service quotas

| Parameter | Value |
|-----------|-------|
| Mechanism | Sliding window counter in Valkey |
| Key format | `ratelimit:{tenant_id}:{connector_type}:{window}` |
| Default limit | 10 requests/second per connector type |
| Configurability | Per `connector_configs.rate_limit_rps` |
| Exceeded behavior | Worker returns retryable error; retried with exponential backoff per retry policy |
| TTL safety | Window key expires after window duration |

Connector rate limits are enforced within workers. When a limit is exceeded, the worker returns a retryable error, and the engine's retry scheduler handles the backoff per the configured retry policy. This prevents Orchestrator from overwhelming external APIs and respects per-connector rate limit configurations.

### Layer 4: NATS JetStream Back-Pressure

**Resource protected:** Event processing pipeline and worker memory

| Parameter | Value |
|-----------|-------|
| Mechanism | NATS consumer configuration |
| `max_ack_pending` | 50 messages per consumer |
| `max_deliver` | 5 attempts before dead-letter |
| Behavior | Messages beyond `max_ack_pending` wait in the stream |

NATS JetStream provides natural back-pressure through its consumer acknowledgment model. By setting `max_ack_pending=50`, the consumer will only have 50 in-flight messages at a time. Additional messages remain in the stream and are delivered as in-flight messages are acknowledged. This prevents the event router from being overwhelmed during traffic spikes.

### Counter Safety Mechanisms

All Valkey counters implement safety mechanisms to prevent drift:

| Mechanism | Purpose |
|-----------|---------|
| TTL on all counter keys | Prevents counters from persisting indefinitely if decrement is missed (1-hour TTL) |
| Floor at zero on decrement | Prevents counters from going negative due to race conditions or duplicate completions |
| Background reconciliation job | Periodically syncs Valkey counters with actual workflow instance counts in PostgreSQL to correct drift |
| Atomic operations | All increment/decrement operations use Valkey atomic commands (INCR/DECR) |

## Alternatives Considered

| Option | Pros | Cons |
|--------|------|------|
| Four-layer Valkey + NATS (chosen) | Each layer protects specific resource; tenant-configurable; uses engine retry scheduling; NATS provides natural back-pressure | Must tune limits per plan tier; counter safety mechanisms add complexity; reconciliation job needed |
| Single global rate limiter | Simple implementation; single configuration point | Cannot differentiate between resource types; event limits would also limit connector calls; too coarse |
| Token bucket algorithm | Smooth rate limiting; handles bursts well | More complex implementation; harder to inspect current state; Valkey implementation less straightforward |
| External rate limiting service (e.g., Envoy) | Battle-tested; feature-rich; protocol-aware | Additional infrastructure; cannot rate-limit internal engine executions; overkill for application-level limits |
| No rate limiting (rely on infrastructure scaling) | Zero application complexity | Noisy neighbor problem; unbounded costs; external API abuse; cascading failures |

## Rationale
1. Each layer protects a different resource with different failure characteristics, requiring layer-specific rate limiting rather than a single global mechanism.
2. Tenant-configurable limits enable commercial plan differentiation (free tier: 100 events/min; enterprise: 10,000 events/min) without code changes.
3. The engine's retry scheduler handles connector rate limiting naturally, converting rate limit violations into delayed retries per the configured retry policy.
4. NATS JetStream's `max_ack_pending` provides back-pressure without any application code, leveraging the streaming platform's built-in flow control.
5. Valkey atomic counters with TTL safety nets provide a proven pattern from Foundry that handles the edge cases of distributed counting (missed decrements, process crashes, network partitions).

## Consequences

**Positive:**
- Resource protection at every layer of the execution pipeline
- Tenant-configurable limits support plan-tier differentiation
- Engine retry scheduler handles connector retry scheduling automatically
- NATS provides infrastructure-level back-pressure without application code
- Counter safety mechanisms (TTL, floor, reconciliation) prevent drift from causing permanent issues

**Negative:**
- Must tune default limits per plan tier based on actual usage patterns; initial values are estimates
- Counter safety mechanisms (TTL, reconciliation) add operational complexity
- Background reconciliation job must be deployed and monitored
- Rate limit errors must be surfaced clearly to users (not just logged) to be actionable

## Implementation Notes

### Event Rate Limit Check

```go
func (r *RateLimiter) CheckEventRate(ctx context.Context, tenantID string) error {
    window := time.Now().Truncate(time.Minute).Unix()
    key := fmt.Sprintf("ratelimit:%s:events:%d", tenantID, window)

    count, err := r.valkey.Incr(ctx, key).Result()
    if err != nil {
        // Fail open: allow the event if Valkey is unavailable
        util.Log(ctx).Warn("rate limit check failed, allowing event",
            "error", err, "tenant_id", tenantID)
        return nil
    }

    if count == 1 {
        r.valkey.Expire(ctx, key, 2*time.Minute) // window + buffer
    }

    limit := r.planLimits.EventsPerMinute(tenantID)
    if count > int64(limit) {
        r.metrics.QuotasExceeded.Add(ctx, 1,
            metric.WithAttributes(
                attribute.String("tenant_id", tenantID),
                attribute.String("quota_type", "events_per_minute"),
            ),
        )
        return connect.NewError(connect.CodeResourceExhausted,
            fmt.Errorf("event rate limit exceeded: %d/%d per minute", count, limit))
    }

    return nil
}
```

### Workflow Concurrency Guard

```go
func (r *RateLimiter) AcquireWorkflowSlot(ctx context.Context, tenantID string) (release func(), err error) {
    key := fmt.Sprintf("active_workflows:%s", tenantID)

    count, err := r.valkey.Incr(ctx, key).Result()
    if err != nil {
        return nil, fmt.Errorf("acquire workflow slot: %w", err)
    }

    // Safety TTL: reset counter if decrement is never called
    r.valkey.Expire(ctx, key, time.Hour)

    limit := r.planLimits.ActiveWorkflows(tenantID)
    if count > int64(limit) {
        // Release the slot we just took
        r.valkey.Decr(ctx, key)
        return nil, fmt.Errorf("workflow concurrency limit exceeded: %d/%d", count, limit)
    }

    release = func() {
        result, err := r.valkey.Decr(ctx, key).Result()
        if err != nil {
            util.Log(ctx).Error("failed to release workflow slot",
                "error", err, "tenant_id", tenantID)
            return
        }
        // Floor at zero
        if result < 0 {
            r.valkey.Set(ctx, key, 0, time.Hour)
        }
    }

    return release, nil
}
```

### Connector Rate Limit in Worker

```go
func (w *ConnectorWorker) Execute(ctx context.Context, cmd ExecutionCommand) (*ExecutionResult, *ExecutionError) {
    key := fmt.Sprintf("ratelimit:%s:%s:%d",
        cmd.TenantID, cmd.ConnectorType, time.Now().Unix())

    count, err := w.valkey.Incr(ctx, key).Result()
    if err == nil && count == 1 {
        w.valkey.Expire(ctx, key, 2*time.Second)
    }

    limit := w.connectorConfig.RateLimitRPS(cmd.TenantID, cmd.ConnectorType)
    if err == nil && count > int64(limit) {
        // Return retryable error: engine retry scheduler will retry with backoff
        return nil, &ExecutionError{
            Class:   ErrorClassRetryable,
            Message: fmt.Sprintf("connector rate limit exceeded: %d/%d rps", count, limit),
            Code:    "RATE_LIMITED",
        }
    }

    // Proceed with connector call
    return w.adapter.Execute(ctx, cmd)
}
```

### Future Enhancements
- **Layer 0 - Global platform protection:** Platform-wide rate limits that protect infrastructure regardless of tenant allocation (e.g., total events/second across all tenants).
- **Per-form rate limiting:** Spam prevention on public-facing forms with CAPTCHA integration at high submission rates.
- **Per-connector-instance quotas:** Rate limits scoped to specific connector instances (e.g., a specific Slack workspace) rather than connector type.
- **Cost-based limiting:** Rate limits based on estimated cost (AI generation tokens, premium connector calls) rather than raw request count.
- **Geographic rate limiting:** Different limits based on request origin for abuse prevention.
