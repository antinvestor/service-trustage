# ADR-006: Multi-Tenant Isolation Strategy

## Status
Accepted

## Context
Orchestrator is a multi-tenant SaaS platform where multiple organizations share the same underlying infrastructure. This shared model introduces critical security and reliability requirements: we must prevent data leakage between tenants, guard against resource exhaustion from noisy neighbors, ensure credentials are never exposed across tenant boundaries, and block unauthorized workflow execution.

The existing ecosystem already provides foundational patterns for multi-tenancy. Frame's `BaseModel` enforces `tenant_id` and `partition_id` columns on every database table, ensuring tenant ownership is baked into the data model from the start. Authentication flows use `security.ClaimsFromContext()` to extract tenant identity from OIDC tokens on every request. Foundry's concurrency model uses per-tenant counters in Valkey to enforce limits.

The challenge is extending these patterns across all layers of Orchestrator's architecture: PostgreSQL storage, NATS JetStream event streaming, the state engine's workflow execution, Valkey caching, ConnectRPC API handlers, and external connector integrations. Each layer has its own isolation mechanisms and failure modes, and a gap in any single layer could compromise the entire isolation model.

## Decision
We adopt row-level isolation across all layers, augmented with per-tenant resource quotas to prevent noisy-neighbor problems. Every layer enforces tenant boundaries independently so that no single layer is a single point of failure for isolation.

### Isolation Per Layer

| Layer | Isolation Mechanism |
|-------|-------------------|
| PostgreSQL | `tenant_id NOT NULL` on every table; all repository methods include `tenant_id` in `WHERE` clauses |
| NATS JetStream | Subject hierarchy `orchestrator.events.{tenant_id}.{event_type}` scopes all event routing |
| State Engine | `tenant_id` column on every workflow table (`workflow_instances`, `workflow_state_executions`, etc.); all queries include `tenant_id` filter |
| Valkey | Key prefix `orch:{tenant_id}:*` for all cached data, counters, and rate limit windows |
| ConnectRPC | Auth interceptor extracts `TenantID` from OIDC claims via `security.ClaimsFromContext()` and injects into request context |
| Connectors | Credentials stored with `tenant_id` foreign key; encrypted with tenant-scoped encryption keys |
| Workers | `TenantID` included in execution command; workers load tenant-specific configuration and credentials |

### Resource Quotas

| Resource | Default Limit | Notes |
|----------|--------------|-------|
| Active workflows | Configurable per plan | Enforced via Valkey counter |
| Events per day | Configurable per plan | Sliding window in Valkey |
| Connector calls per minute | Configurable per plan | Per-connector rate limiting |
| Form submissions per day | Configurable per plan | Sliding window in Valkey |
| Workflow definitions | Configurable per plan | Enforced at repository layer |
| Concurrent streaming connections | Configurable per plan | Tracked in Valkey |

## Alternatives Considered

| Option | Pros | Cons |
|--------|------|------|
| Row-level isolation (chosen) | Shared infrastructure keeps costs low; consistent with Frame/Foundry patterns; simple connection management; cross-tenant admin queries possible | Must ensure every query includes `tenant_id`; data leak risk if repository has bugs |
| Schema-per-tenant | Stronger isolation at database level; tenant-specific schema migrations possible | Migration complexity across hundreds of schemas; connection pool explosion; incompatible with Frame BaseModel |
| Database-per-tenant | Strongest isolation; per-tenant backup/restore | Extreme operational complexity; connection management nightmare; cost scales linearly with tenants |
| Partition-per-tenant (engine) | Strong execution isolation; per-tenant scheduling configuration | Management overhead for partition lifecycle; cannot query across tenants for admin tooling; resource waste for low-volume tenants |

## Rationale
1. Row-level isolation is a proven pattern already established across the ecosystem through Frame's `BaseModel` and Foundry's tenant-aware repositories.
2. Repository-layer enforcement acts as the primary safety boundary, ensuring tenant scoping happens at the data access layer regardless of caller.
3. Resource quotas prevent noisy-neighbor problems without requiring infrastructure-level isolation, keeping operational costs proportional to actual usage.
4. PostgreSQL indexes on `tenant_id` enable cross-tenant administrative queries (platform health, usage analytics) while maintaining per-tenant isolation for normal operations.
5. Enterprise tenants can upgrade to dedicated database partitions or separate schemas when regulatory or contractual requirements demand it, without changing the application layer.

## Consequences

**Positive:**
- Shared infrastructure keeps per-tenant costs low and operations simple
- Consistent with Foundry patterns, reducing cognitive overhead for developers
- Resource quotas prevent abuse and provide clear plan-tier differentiation
- Enterprise partition isolation is available as an upgrade path without architectural changes

**Negative:**
- Every query and repository method must include `tenant_id`; missing it is a data leak vulnerability
- Quota tuning requires per-plan-tier configuration and ongoing adjustment based on usage patterns
- Data leak risk exists if a repository method omits the `tenant_id` filter; requires code review discipline and integration tests

## Implementation Notes

### Tenant Extraction Pattern

```go
func (s *WorkflowService) CreateWorkflow(
    ctx context.Context,
    req *connect.Request[v1.CreateWorkflowRequest],
) (*connect.Response[v1.CreateWorkflowResponse], error) {
    claims, err := security.ClaimsFromContext(ctx)
    if err != nil {
        return nil, connect.NewError(connect.CodeUnauthenticated, err)
    }
    tenantID := claims.TenantID

    // All downstream calls scoped to tenantID
    wf, err := s.repo.CreateWorkflow(ctx, tenantID, req.Msg)
    // ...
}
```

### Quota Check Pattern

```go
func (q *QuotaChecker) CheckEventQuota(ctx context.Context, tenantID string) error {
    key := fmt.Sprintf("orch:%s:quota:events:%s", tenantID, currentWindow())

    count, err := q.valkey.Incr(ctx, key).Result()
    if err != nil {
        return fmt.Errorf("quota check failed: %w", err)
    }

    // Set TTL on first increment
    if count == 1 {
        q.valkey.Expire(ctx, key, windowDuration)
    }

    limit := q.planLimits.EventsPerWindow(tenantID)
    if count > int64(limit) {
        return connect.NewError(
            connect.CodeResourceExhausted,
            fmt.Errorf("event quota exceeded: %d/%d", count, limit),
        )
    }

    return nil
}
```
