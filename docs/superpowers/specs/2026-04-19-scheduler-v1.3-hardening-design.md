# Scheduler v1.3 — correctness + scale hardening

**Date:** 2026-04-19
**Status:** approved
**Scope:** `service-trustage` + one HelmRelease env update in `deployments`. Direct-to-main. Target release: `v0.3.38`.

## Why

Full-system audit surfaced one correctness bug, two correctness gaps, three load failure points, and several non-critical caveats. None require redesign; all are tunable or localised code changes. This release closes them in one cohesive patch.

## Goals

1. Primary DB pool size is operator-configurable.
2. Dispatch → publish is atomic from the execution's perspective (no stranded dispatches).
3. Event-router does not fan out O(N) DB ops synchronously inside a NATS message handler.
4. Slow third-party endpoints cannot pin worker goroutines indefinitely.
5. Minor correctness gaps closed (Timeout tx atomicity, Cleanup SKIP LOCKED, `TransitionTo` idempotency, Cleanup retention for workflow rows).
6. Tuning defaults appropriate for the 12-pod cluster.

## Non-goals

- Circuit breakers per destination (future).
- Rewrite of event-router architecture (future).
- Per-workflow adaptive retry backoff (future).

---

## Changes

### A. Primary DB pool sizing (C1)

**File:** `apps/default/config/config.go` + `apps/default/cmd/main.go`.

Add `DatabasePoolMaxConns int \`env:"DATABASE_POOL_MAX_CONNS" envDefault:"50"\``.

In `main.go`, before `frame.WithDatastore(...)`: there is no direct way to inject `WithMaxOpen` into Frame's default pool via `WithDatastore` options — the pool created there is internal. Two options:

1. **Replace the primary pool** with an explicit `pool.NewPool(ctx) + AddConnection(..., WithMaxOpen(n))` + `svc.DatastoreManager().AddPool(ctx, datastore.DefaultPoolName, primary)`. Requires re-wiring all repos that use the default pool to accept the explicit pool (they already do — `defRepo := repository.NewWorkflowDefinitionRepository(dbPool)` etc.). This is the cleanest path.

2. If Frame's `WithDatastore` accepts a `pool.Option` variadic chain, pass `pool.WithMaxOpen(cfg.DatabasePoolMaxConns)` directly. Check Frame source during implementation.

Go with whichever matches Frame's current exposed API.

### B. Dispatch failure handling (C2)

**File:** `apps/default/service/schedulers/dispatch.go`.

Current shape (around lines 120-136):
```go
if err := b.engine.Dispatch(ctx, exec); err != nil { /* handle */ }
if err := queueMgr.Publish(ctx, dispatchName, command); err != nil {
    // Execution is already 'dispatched'. Sits until timeout.
    log.WithError(err).Error("dispatch publish failed")
}
```

Fix: on publish failure, revert the execution status back to `pending`:

```go
if publishErr := queueMgr.Publish(ctx, dispatchName, command); publishErr != nil {
    if revertErr := b.engine.RevertDispatch(ctx, exec.ID); revertErr != nil {
        log.WithError(revertErr).Error("dispatch publish failed AND revert failed — execution stranded until timeout")
    } else {
        log.WithError(publishErr).Warn("dispatch publish failed, execution reverted to pending")
    }
    continue // next execution; don't count this one as fired
}
```

New `engine.RevertDispatch(ctx, executionID) error` method — sets `status = 'pending'`, clears `started_at` and `token_hash`, leaves `attempt` unchanged. Tenancy-scoped via BaseRepository.

### C. Event-router fanout (C3)

**File:** `apps/default/service/repository/trigger_binding.go` + `apps/default/service/queues/event_router_worker.go` + `apps/default/service/business/event_router.go`.

Two changes:

1. **`FindByEventType` grows a `limit int` parameter.** Caller passes a sensible cap (default 200). If a tenant legitimately has 200+ bindings for one event type, they need a different architecture — not our problem for v1.3, but the limit prevents runaway fanout.

2. **`EventRouterWorker.Handle` dispatches instance creation async** via `frame.WorkerPool` with bounded concurrency. NATS ack returns fast; instance creation happens in the background. This decouples NATS in-flight from DB fanout.

Alternatively (simpler for v1.3): keep the handler synchronous but (a) cap bindings via the new limit and (b) do instance creation in batches with `CreateInBatches`. Bulk-insert `workflow_instances` + `workflow_state_executions` in one tx per batch. This eliminates the sequential-query problem without introducing the async/worker-pool complexity.

Prefer (b). It's smaller and gets us most of the win.

### D. Per-adapter HTTP timeout (C4 part 1)

**Files:** `connector/adapters/*.go`.

Every adapter that issues an HTTP call currently uses the injected `*http.Client` without a per-call timeout. Add:

```go
const defaultAdapterCallTimeout = 30 * time.Second

// Inside adapter Execute:
ctx, cancel := context.WithTimeout(ctx, defaultAdapterCallTimeout)
defer cancel()
// use ctx for the Do call
```

Make the timeout configurable via config (`ADAPTER_HTTP_TIMEOUT_SECONDS` default 30) so operators can override without a code change.

### E. NATS consumer capacity (C4 part 2)

**File:** `apps/default/config/config.go`.

Change the env defaults in the two consumer URLs:

- `QUEUE_EXEC_WORKER_URL`: `consumer_max_ack_pending=5000` → **500**.
- `QUEUE_EVENT_ROUTER_URL`: `consumer_max_ack_pending=10000` → **200**.

These are embedded in default URL strings (`config.go:81,89`). Update the `envDefault` values.

### F. `ActivateWorkflow` idempotency (I1)

**File:** `apps/default/service/models/workflow_definition.go` (around `TransitionTo`).

Confirm `TransitionTo(current == new)` returns `nil` (idempotent). If not, add the guard. Either way, add a test in `workflow_definition_test.go` for `ACTIVE→ACTIVE` and `ARCHIVED→ARCHIVED`.

### G. `TimeoutScheduler` tx atomicity (I2)

**File:** `apps/default/service/schedulers/timeout.go`.

The two statements (`UpdateStatus(timed_out)` + `Create(retry exec)`) must be in a single tx. Add a repo method `MarkTimedOutAndCreateRetry(ctx, oldID, newExec *WorkflowStateExecution) error` that wraps both in `db.Transaction`. Scheduler calls the new atomic method.

### H. `CleanupScheduler` SKIP LOCKED (I3)

**Files:** `apps/default/service/repository/event_log.go`, `audit_event.go`.

Add `Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"})` to the SELECT that precedes the batch DELETE. One-line change each.

### I. Cleanup retention for workflow rows (I4)

**File:** `apps/default/service/schedulers/cleanup.go` + `workflow_execution.go`, `workflow_timer.go`, `workflow_signal_wait.go` repositories.

New repo methods (each in its respective file):
- `DeleteCompletedBefore(ctx, cutoff time.Time, limit int) (int64, error)` on `workflow_execution`, `workflow_timer`, `workflow_signal_wait`.

Cleanup scheduler grows three new sweep passes (after the existing event_log and audit_event ones). Retention cutoff from config: `WORKFLOW_ROW_RETENTION_HOURS` default **720 (30 days)**.

### J. Event_log payload size constraint (I5)

**File:** `apps/default/service/models/event_log.go` + repository/migrate.go.

Add a Go-side validation at `EventLog.BeforeCreate` (GORM hook):
```go
const MaxEventLogPayloadBytes = 1 << 20 // 1 MiB

func (e *EventLog) BeforeCreate(tx *gorm.DB) error {
    if len(e.Payload) > MaxEventLogPayloadBytes {
        return fmt.Errorf("event_log payload exceeds %d bytes", MaxEventLogPayloadBytes)
    }
    return nil
}
```

Also reduce `OUTBOX_BATCH_SIZE` default from **100 → 20**. Reduce `OUTBOX_MAX_BATCHES_PER_SWEEP` from **100 → 50**. Reduce `DISPATCH_BATCH_SIZE` / `DISPATCH_MAX_BATCHES_PER_SWEEP` from **100/100 → 50/50**.

### K. Deployment tuning (deployments repo)

**File:** `deployments/manifests/namespaces/trustage/api/trustage-api.yaml`.

Add env entries for:
- `DATABASE_POOL_MAX_CONNS=50`
- `OUTBOX_BATCH_SIZE=20`
- `OUTBOX_MAX_BATCHES_PER_SWEEP=50`
- `DISPATCH_BATCH_SIZE=50`
- `DISPATCH_MAX_BATCHES_PER_SWEEP=50`
- `ADAPTER_HTTP_TIMEOUT_SECONDS=30`
- `WORKFLOW_ROW_RETENTION_HOURS=720`

(Code defaults match these; env pins them explicitly for production clarity.)

---

## Testing

- Unit: `TestTransitionTo_Idempotent` for ACTIVE→ACTIVE and ARCHIVED→ARCHIVED.
- Integration: `TestDispatchScheduler_RevertsOnPublishFailure` — stub `queueMgr.Publish` to error, assert exec goes back to `pending`.
- Integration: `TestEventRouter_RespectsBindingLimit` — seed 300 bindings, call handler, assert at most 200 processed.
- Integration: `TestEventRouter_BatchCreatesInstances` — seed 50 bindings, assert 1-2 SQL round-trips for instance+execution creation (via GORM callback statement counter).
- Integration: `TestTimeoutScheduler_AtomicTxOnCrash` — simulate tx rollback, assert status+retry revert together.
- Integration: `TestCleanup_SkipLockedConcurrent` — spawn 10 cleanup goroutines, assert no duplicate-delete errors.
- Integration: `TestCleanup_DeletesOldWorkflowRows` — seed completed executions with old `finished_at`, assert deleted after sweep.
- Integration: `TestEventLog_RejectsOversizedPayload` — attempt insert > 1 MiB, assert BeforeCreate blocks.

All integration tests use existing `frametests.FrameBaseTestSuite` + testcontainers.

---

## Rollout

Direct-to-main. Tag `v0.3.38`. No schema migration — all changes additive (new env vars, new GORM hook, new repo methods). No pgBouncer bounce needed.

HelmRelease rolls, pods pick up new env vars on restart.

## Rollback

- Per-env rollback: unset the new env vars on the Deployment → falls back to code defaults (which match prod values anyway).
- Full rollback: re-pin `v0.3.37` in ImagePolicy.

## Risks

1. **Reducing `consumer_max_ack_pending` from 5000 → 500** may surface a backlog if the cluster was silently relying on burst capacity. Monitor queue lag for 24h post-deploy.
2. **`OUTBOX_BATCH_SIZE=20`** reduces per-sweep throughput; if a burst of 10k events arrives at once, drain time increases. At 12 pods × 50 batches × 20 = 12k events per sweep cycle — still adequate for most cases.
3. **New `WORKFLOW_ROW_RETENTION_HOURS`** default of 30 days means operators who need longer retention must set it explicitly. Documented in runbook.

## Success criteria

- All audit findings C1-C4 + I1-I5 closed.
- Integration tests pass.
- Post-deploy: queue lag stable, no increase in dispatch-stranded executions, no OOM events.
