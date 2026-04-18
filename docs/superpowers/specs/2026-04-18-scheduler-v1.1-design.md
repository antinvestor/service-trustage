# Scheduler v1.1: batched fire, bug fixes, TZ, Archive

**Date:** 2026-04-18
**Status:** approved (revised)
**Scope:** `service-trustage` — builds on scheduler v1 (spec `2026-04-18-scheduler-v1-design.md`, shipped as `v0.3.34`). Direct-to-main commits.

## Why v1.1

The scheduler v1 release ships a correct exactly-once fire path but four production-relevant gaps surfaced under audit:

1. **Throughput ceiling is too low.** `ClaimAndFireBatch` opens one Postgres transaction per row. At 50 rows/batch × 30 s poll = 100 fires/min/pod = 20 fires/sec cluster-wide (12 pods HPA cap). Hundreds of millions of scheduled tasks per day requires two orders of magnitude more.

2. **Correctness bugs** that are tolerable at low load but unacceptable at scale:
   - `ListByWorkflow` omits tenant/partition filtering (cross-tenant read leak).
   - `CreateWorkflow` is not atomic across workflow + schedule inserts (orphan rows on partial failure).
   - User-supplied `input_payload` can shadow system fields in `schedule.fired` events.
   - Invalid cron at fire time still emits a ghost event before parking.

3. **No observability on the fire path.** `CronScheduler` has no trace span, no metrics. At scale, silent backlog is operationally unacceptable.

4. **`ActivateWorkflow` is O(n) round-trips** per schedule. At 10 k schedules per workflow, activation holds a tx for seconds.

Plus two allowed additions: per-schedule timezone, and `ArchiveWorkflow` RPC.

## Design principles (revised)

Two principles shape this release beyond the spec-is-truth one inherited from v1:

1. **Narrow the interface change to what's structurally required.** The batched fire path forces `ClaimAndFireBatch`'s fireFn signature to change — that is the one unavoidable interface shift. Everything else (observability, bugs, TZ, Archive) happens behind existing-shape methods or in new single-responsibility methods.

2. **Single-table transactions only, with one documented exception.** Cross-table transactions invite deadlocks that are hard to diagnose under load. Every tx in this release operates on exactly one table, with the single exception of the fire path (event_log + schedule_definitions) where exactly-once semantics demands atomicity across both. Lifecycle operations (Create / Activate / Archive) split into two sequential single-table tx with explicit ordering chosen for safe failure modes.

## Goals

- **100+ M fires/day** at the cluster layer (~6 k fires/sec at 12 pods, ~24 k/sec at 48 pods), DB capacity permitting.
- Exactly-once fire semantics preserved.
- All v1 audit correctness bugs resolved.
- Per-schedule IANA timezone (default UTC).
- `ArchiveWorkflow` RPC — DRAFT | ACTIVE → ARCHIVED, deactivating the workflow's schedules.
- No cross-table transactions in business logic. Each lifecycle op is two sequential single-table tx; the fire path is the one exception and lives inside the repository.
- Go-pattern adherence: `util.Log(ctx)`, wrapped errors, interface-first repos, table-driven tests, testcontainers (not mocks), typed event payloads in `pkg/events/`.

## Non-goals (hard exclusions)

- LISTEN/NOTIFY.
- Pause/resume a single schedule.
- Update one schedule's `cron_expr` without shipping a new workflow version. No `UpdateSchedule` RPC.
- Per-schedule missed-fire threshold override (global constant stays).
- Sub-minute cron precision (5-field parser only, no seconds).
- Reshaping every method into `-Tx` / tx-accepting variants. Changes are narrow.

---

## Transaction boundaries

**Rule**: each business operation is composed of one or more single-table transactions, invoked in a fixed order that yields a safe failure mode if the second tx fails.

**Sole exception**: `ClaimAndFireBatch` writes to both `schedule_definitions` and `event_log` atomically inside one repository-owned tx. Splitting would violate exactly-once:

- event_log insert first, then `next_fire_at` advance → if the second statement fails, the event is emitted but the schedule still reads as due → **double-fire** on the next sweep.
- `next_fire_at` advance first, then event_log insert → if the second statement fails, the schedule is advanced but no event is emitted → **silent lost fire**.

The fire tx is fully enclosed inside the repository: it acquires SKIP LOCKED on `schedule_definitions` first, then INSERTs into `event_log` by new UUID key. No other transaction takes locks on both tables in opposing order, so the deadlock risk is bounded.

### Lifecycle ordering

Every lifecycle op is two sequential single-table tx. The order is chosen so that a failure of the second tx leaves a **benign intermediate state**.

#### CreateWorkflow

```
Tx1 (workflow_definitions):  INSERT workflow row (DRAFT)
Tx2 (schedule_definitions):  INSERT schedule rows × N in one batch tx
```

Mid-failure: Tx1 succeeded, Tx2 failed → orphan DRAFT workflow with zero or partial schedules. **DRAFT does not fire**, so this is not a user-visible bad state. Retry blocks on `idx_wd_name_version`; operator either deletes the orphan workflow (admin path) or bumps version. Upstream DSL validation eliminates almost all failure modes before Tx2 runs, so this path is rare.

#### ActivateWorkflow

```
Tx1 (workflow_definitions):  UPDATE status = 'active'
Tx2 (schedule_definitions):  one batch tx that:
                               - DEACTIVATEs all OTHER versions' schedules
                               - ACTIVATEs this version's schedules (VALUES-join UPDATE with next_fire_at + jitter)
```

Mid-failure: Tx1 succeeded, Tx2 failed → workflow ACTIVE but new-version schedules not yet firing; the prior version's schedules are still active and still firing. Benign — the old behavior continues, operator re-invokes `ActivateWorkflow` (idempotent on workflow status, Tx2 retries). Alert on `workflow.status == ACTIVE && this-version schedules not active` so operators notice.

Ordering rationale: **workflow first** because the worst mid-failure state (prior version continues firing) is the same as pre-activation — no regression.

#### ArchiveWorkflow

```
Tx1 (schedule_definitions):  UPDATE SET active=false, next_fire_at=NULL WHERE workflow_name = ...
Tx2 (workflow_definitions):  UPDATE status = 'archived'
```

Ordering is **reversed** here: schedules off first. Mid-failure: Tx1 succeeded, Tx2 failed → schedules stopped, workflow still ACTIVE. Benign — no fires, temporary status mismatch; retry re-applies Tx1 (idempotent) and retries Tx2. The opposite order would leave workflow ARCHIVED with schedules still firing — user-visibly wrong behaviour.

---

## Architecture

Schema gains one column, one unique index, one predicate tightening. The scheduler fire path becomes one three-statement tx per sweep:

```
┌──────────────────────────────────────────────────────────────────┐
│ CronScheduler.RunOnce — one sweep (interval default 1s)          │
│                                                                   │
│  repo.ClaimAndFireBatch(ctx, planner, now, batch) → fired, err   │
│                                                                   │
│    Inside the repo, under one tx:                                 │
│      BEGIN                                                        │
│      1. SELECT ... FOR UPDATE SKIP LOCKED LIMIT $batch           │
│      2. planner.Plan(ctx, batch) → per-row (event, nextFire, jit)│
│      3. INSERT INTO event_log (...) × M                          │
│      4. UPDATE schedule_definitions s                             │
│           FROM (VALUES (...)) AS v(...)                           │
│           WHERE s.id = v.id AND s.tenant_id = v.tenant_id        │
│             AND s.partition_id = v.partition_id                   │
│      COMMIT                                                       │
│                                                                   │
│  Round-trips: 3 (plus BEGIN/COMMIT).                              │
│  planner runs in pure Go — NO DB access.                          │
└──────────────────────────────────────────────────────────────────┘
```

### Throughput model

| Configuration | Per pod | 12 pods | 48 pods |
|---|---|---|---|
| v1 (today): batch=50, interval=30s, per-row tx | 100/min | 20/sec | 80/sec |
| v1.1 (default): batch=500, interval=1s, batched tx | 500/sec | 6 k/sec | 24 k/sec |
| v1.1 (stress): batch=2000, interval=1s | 2 k/sec | 24 k/sec | 96 k/sec |

Default ≈ 520 M fires/day at 12 pods. Database (event_log insert, index updates) becomes the ceiling beyond that.

---

## Data model

### Model (`apps/default/service/models/schedule.go`)

One new field:

| Column | Type | Default | Purpose |
|---|---|---|---|
| `timezone` | `varchar(64) NOT NULL` | `'UTC'` | IANA zone used to evaluate the cron expression |

### Indexes (`apps/default/service/repository/migrate.go`)

1. **Tighten `idx_sd_due`** — add `next_fire_at IS NOT NULL` to the partial predicate so parked rows no longer consume index visits.

2. **New `idx_sd_workflow_unique`** — unique on `(tenant_id, partition_id, workflow_name, workflow_version, name) WHERE deleted_at IS NULL`. Makes schedule materialisation idempotent at the DB layer.

Migration path: the existing `idx_sd_due` is dropped once in `Migrate()` so AutoMigrate rebuilds it with the new predicate. AutoMigrate creates the new unique index if absent. The column additions are non-destructive.

---

## Repository interface — narrow

### `ScheduleRepository` (`apps/default/service/repository/schedule.go`)

Stays almost identical to v1. Only `ClaimAndFireBatch`'s fireFn signature changes; three new methods are added for batched lifecycle atomicity.

```go
type ScheduleRepository interface {
    // Existing — unchanged:
    Create(ctx context.Context, schedule *models.ScheduleDefinition) error
    ListByWorkflow(ctx context.Context, workflowName string, workflowVersion int) ([]*models.ScheduleDefinition, error)
    Pool() pool.Pool   // kept; v1.1 callers do not use it, but removing it is pure churn.

    // Signature change — batched fire (the only cross-table tx):
    ClaimAndFireBatch(ctx context.Context, plan SchedulePlanFn, now time.Time, limit int) (fired int, err error)

    // NEW — atomic single-table batch operations:
    CreateBatch(ctx context.Context, scheds []*models.ScheduleDefinition) error
    ActivateByWorkflow(
        ctx context.Context,
        workflowName string,
        workflowVersion int,
        tenantID, partitionID string,
        fires []ScheduleActivation,
    ) error
    DeactivateByWorkflow(ctx context.Context, workflowName, tenantID, partitionID string) error
}

// SchedulePlanFn is invoked per row by the repository inside the fire tx.
// It must be pure Go — no DB access, no I/O. It returns the event_log row
// to emit (may be nil to park the schedule), the next fire time (may be nil
// to park), and the jitter seconds stored on the schedule row.
type SchedulePlanFn func(ctx context.Context, sched *models.ScheduleDefinition) (
    event *models.EventLog,
    nextFire *time.Time,
    jitterSeconds int,
    err error,
)

// ScheduleActivation is a single row's activation plan — next_fire_at + jitter.
type ScheduleActivation struct {
    ID            string
    NextFireAt    time.Time
    JitterSeconds int
}
```

**What does NOT change:**
- `Pool()` stays. Not used by v1.1 code; kept so the interface doesn't churn for a reason unrelated to this release.
- `WorkflowDefinitionRepository` is untouched.
- `EventLogRepository` is untouched.
- No `-Tx` suffix methods, no `Transact(ctx, fn)`. Business composes sequential single-table ops, never a multi-table tx.

### Method behaviour

**`ClaimAndFireBatch(ctx, plan, now, limit)`** — inside one tx owned by the repo:

1. `SELECT ... FOR UPDATE SKIP LOCKED LIMIT $limit` from `schedule_definitions` with the due predicate.
2. For each row, call `plan(ctx, sched)` → `(event, nextFire, jitter, err)` in pure Go.
3. Multi-row `INSERT INTO event_log (...) VALUES (...)...` for every non-nil event (`CreateInBatches(events, 500)`).
4. `UPDATE schedule_definitions s SET last_fired_at=..., next_fire_at=v.next_fire_at, jitter_seconds=v.jitter_seconds, modified_at=... FROM (VALUES ...) AS v(...) WHERE s.id=v.id AND s.tenant_id=v.tenant_id AND s.partition_id=v.partition_id`.
5. COMMIT.

If `plan` returns an error for a row, that row is skipped (not included in the event INSERT or the UPDATE VALUES list); other rows in the batch still commit. This preserves partial progress on transient per-row issues.

**`CreateBatch(ctx, scheds)`** — one `tx.Create(scheds)` via GORM's slice-aware insert. Single-table. Atomic.

**`ActivateByWorkflow(ctx, workflowName, workflowVersion, tenantID, partitionID, fires)`** — one tx that:
1. `UPDATE schedule_definitions SET active=false, next_fire_at=NULL, modified_at=now WHERE workflow_name=? AND workflow_version != ? AND tenant_id=? AND partition_id=? AND deleted_at IS NULL` — deactivates sibling versions.
2. `UPDATE ... FROM (VALUES ...)` bulk-activates rows matching `fires`, setting `active=true, next_fire_at=v.next_fire_at, jitter_seconds=v.jitter_seconds, modified_at=now`. Filtered by tenant_id + partition_id to match the inbound `fires` values.

Single-table. Atomic.

**`DeactivateByWorkflow(ctx, workflowName, tenantID, partitionID)`** — one `UPDATE schedule_definitions SET active=false, next_fire_at=NULL, modified_at=now WHERE workflow_name=? AND tenant_id=? AND partition_id=? AND deleted_at IS NULL`. Deactivates every version's schedules. Single statement.

---

## Fire path implementation (`apps/default/service/schedulers/cron.go`)

`CronScheduler` exposes a `SchedulePlanFn`-shaped method and passes it to the repo. No DB access inside the method; pure Go.

```go
func (s *CronScheduler) planOne(
    ctx context.Context,
    sched *models.ScheduleDefinition,
) (event *models.EventLog, nextFire *time.Time, jitterSeconds int, err error) {

    now := s.clock.Now().UTC()  // clock injected for testability

    // Parse cron. Invalid → park (no event, no next fire).
    cronSched, parseErr := dsl.ParseCron(sched.CronExpr)
    if parseErr != nil {
        util.Log(ctx).WithError(parseErr).Error("invalid cron, parking",
            "schedule_id", sched.ID, "cron_expr", sched.CronExpr)
        s.metrics.IncrementSchedulerCronInvalid(ctx, sched)
        return nil, nil, 0, nil
    }

    // Missed-fire policy.
    base := now
    if sched.NextFireAt != nil && now.Sub(*sched.NextFireAt) <= cronMissedFireThreshold {
        base = *sched.NextFireAt
    }

    nominal, zoneErr := cronSched.NextInZone(base, sched.Timezone)
    if zoneErr != nil {
        util.Log(ctx).WithError(zoneErr).Error("invalid timezone, parking",
            "schedule_id", sched.ID, "timezone", sched.Timezone)
        s.metrics.IncrementSchedulerCronInvalid(ctx, sched)
        return nil, nil, 0, nil
    }

    jitter := dsl.JitterFor(sched.ID, cronSched, nominal)
    next := nominal.Add(jitter)

    event = buildEvent(sched, now)
    return event, &next, int(jitter / time.Second), nil
}

// buildEvent uses events.BuildScheduleFiredPayload so system fields cannot
// be shadowed by user input_payload.
```

`CronScheduler.RunOnce` opens a trace span, calls `scheduleRepo.ClaimAndFireBatch(ctx, s.planOne, now, batchSize)`, and records metrics.

---

## Business layer (`apps/default/service/business/workflow.go`)

All three lifecycle methods are **two sequential single-table calls** — no `Transact`, no `Pool()` access, no `*gorm.DB` handled by the business layer.

### CreateWorkflow

```go
func (b *workflowBusiness) CreateWorkflow(ctx context.Context, dslBlob json.RawMessage) (*models.WorkflowDefinition, error) {
    // Parse + validate upstream (unchanged).
    spec, err := dsl.Parse(dslBlob)
    if err != nil { return nil, fmt.Errorf("parse DSL: %w", err) }
    if res := dsl.Validate(spec); !res.Valid() { return nil, fmt.Errorf("%w: %w", ErrDSLValidationFailed, res.Error()) }
    if err := validateExecutableWorkflow(spec); err != nil { return nil, fmt.Errorf("%w: %w", ErrDSLValidationFailed, err) }
    if err := b.registerStepSchemas(ctx, spec); err != nil { return nil, fmt.Errorf("register schemas: %w", err) }

    def := buildDefinition(spec, dslBlob)

    // Tx1: workflow row.
    if err := b.defRepo.Create(ctx, def); err != nil {
        return nil, fmt.Errorf("persist workflow: %w", err)
    }

    // Tx2: schedule rows as one atomic batch.
    scheds, err := planSchedules(def, spec)
    if err != nil { return nil, err }
    if len(scheds) > 0 {
        if err := b.scheduleRepo.CreateBatch(ctx, scheds); err != nil {
            // Orphan DRAFT workflow. Harmless — DRAFT doesn't fire.
            util.Log(ctx).WithError(err).Error("schedule materialisation failed; workflow is orphan DRAFT",
                "workflow_id", def.ID, "name", def.Name)
            return nil, fmt.Errorf("materialise schedules (workflow %s created but schedules missing; retry blocked by unique index): %w", def.ID, err)
        }
    }

    return def, nil
}
```

### ActivateWorkflow

```go
func (b *workflowBusiness) ActivateWorkflow(ctx context.Context, id string) error {
    def, err := b.defRepo.GetByID(ctx, id)
    if err != nil { return fmt.Errorf("%w: %w", ErrWorkflowNotFound, err) }
    if err := def.TransitionTo(models.WorkflowStatusActive); err != nil {
        return fmt.Errorf("%w: %w", ErrInvalidWorkflowStatus, err)
    }

    // Tx1: workflow status update.
    if err := b.defRepo.Update(ctx, def); err != nil {
        return fmt.Errorf("update workflow: %w", err)
    }

    // Tx2: activate this version's schedules, deactivate sibling versions.
    fires, err := buildActivationFires(ctx, b.scheduleRepo, def)
    if err != nil { return err }
    if err := b.scheduleRepo.ActivateByWorkflow(
        ctx, def.Name, def.WorkflowVersion, def.TenantID, def.PartitionID, fires,
    ); err != nil {
        util.Log(ctx).WithError(err).Error("activate schedules failed; workflow is ACTIVE but schedules not rolled forward; retry to reconcile",
            "workflow_id", def.ID)
        return fmt.Errorf("activate schedules: %w", err)
    }
    return nil
}
```

`buildActivationFires` calls `scheduleRepo.ListByWorkflow` (single-table read) and computes the per-row `next_fire_at = cronNext(now) + jitter` in pure Go — no tx needed, no cross-repo access. Returns `[]ScheduleActivation`.

### ArchiveWorkflow (new)

```go
func (b *workflowBusiness) ArchiveWorkflow(ctx context.Context, id string) error {
    def, err := b.defRepo.GetByID(ctx, id)
    if err != nil { return fmt.Errorf("%w: %w", ErrWorkflowNotFound, err) }
    if err := def.TransitionTo(models.WorkflowStatusArchived); err != nil {
        return fmt.Errorf("%w: %w", ErrInvalidWorkflowStatus, err)
    }

    // Tx1: schedules off FIRST — safe failure mode.
    if err := b.scheduleRepo.DeactivateByWorkflow(ctx, def.Name, def.TenantID, def.PartitionID); err != nil {
        return fmt.Errorf("deactivate schedules: %w", err)
    }

    // Tx2: workflow status.
    if err := b.defRepo.Update(ctx, def); err != nil {
        util.Log(ctx).WithError(err).Error("workflow status update failed after schedules deactivated; retry to reconcile",
            "workflow_id", def.ID)
        return fmt.Errorf("update workflow status: %w", err)
    }
    return nil
}
```

### GetWorkflowWithSchedules — tenancy fix

The v1 audit found `ListByWorkflow` bypassed BaseRepository's tenancy scope. Fix in-place: use the project's idiomatic scoped-list helper (match `WorkflowDefinitionRepository.ListActiveByName`'s pattern). Tenancy + partition filtering applied.

---

## DSL changes (`dsl/`)

Unchanged from the prior revision:

- `ScheduleSpec.Timezone string` added (IANA, default `"UTC"`).
- `ScheduleSpec.Active *bool` removed.
- `CronSchedule.NextInZone(from, zone)` method added.
- `dsl/schedutil.go` with `JitterFor` + `CronMaxJitter` extracted.
- `validateSchedules` rejects non-loadable timezones.

---

## Event type (`pkg/events/schedule.go` — new)

Unchanged from the prior revision:

```go
const ScheduleFiredType = "schedule.fired"

type ScheduleFiredPayload struct {
    ScheduleID   string         `json:"schedule_id"`
    ScheduleName string         `json:"schedule_name"`
    FiredAt      string         `json:"fired_at"`
    Input        map[string]any `json:"input,omitempty"`
}

func BuildScheduleFiredPayload(id, name, firedAt string, userInput map[string]any) ScheduleFiredPayload
```

System fields live on typed struct fields — `maps.Copy` cannot shadow them.

---

## Proto (`proto/workflow/v1/workflow.proto`)

1. Add `timezone string` to the `ScheduleDefinition` proto message.
2. Add `ArchiveWorkflow` RPC:

```proto
rpc ArchiveWorkflow(ArchiveWorkflowRequest) returns (ArchiveWorkflowResponse) {
  option (common.v1.method_permissions) = { permissions: ["workflow_manage"] };
}

message ArchiveWorkflowRequest  { string id = 1; }
message ArchiveWorkflowResponse { WorkflowDefinition workflow = 1; }
```

Reuses existing `workflow_manage` permission.

---

## Configuration (`apps/default/config/config.go`)

```go
CronSchedulerBatchSize     int `env:"CRON_SCHEDULER_BATCH_SIZE"       envDefault:"500"`
CronSchedulerIntervalSecs  int `env:"CRON_SCHEDULER_INTERVAL_SECONDS" envDefault:"1"`

SchedulerPoolMaxConns      int `env:"SCHEDULER_POOL_MAX_CONNS" envDefault:"10"`
SchedulerPoolMinConns      int `env:"SCHEDULER_POOL_MIN_CONNS" envDefault:"2"`

OutboxPublishConcurrency   int `env:"OUTBOX_PUBLISH_CONCURRENCY" envDefault:"16"`
```

`main.go` creates a dedicated scheduler pool via `pool.NewPool(ctx) + AddConnection(...) + svc.DatastoreManager().AddPool(ctx, "scheduler", pool)`. The cron scheduler uses this pool; HTTP/RPC handlers keep the default pool.

---

## Outbox batch publish (`apps/default/service/schedulers/outbox.go`)

Investigate Frame's `QueueManager.Publish` signature during implementation. If batch publish is supported, use it. Otherwise use a bounded `workerpool` (size `OutboxPublishConcurrency`) to parallelise per-event publishes. Either approach eliminates the per-event round-trip ceiling at herd rates.

---

## Observability (`pkg/telemetry/metrics.go` + `apps/default/service/schedulers/cron.go`)

New:

```
SpanSchedulerCron                     // trace span wrapping each sweep
scheduler_cron_fired_total{result}    // counter: ok|fail
scheduler_cron_sweep_duration_seconds // histogram
scheduler_cron_invalid_cron_total     // counter (parse failure at fire time)
```

Wired on `telemetry.Metrics` struct; injected into `NewCronScheduler(scheduleRepo, cfg, metrics)`. A nil metrics is tolerated for tests.

---

## Testing

### Unit (no DB)

- `dsl/schedutil_test.go`: `JitterFor` determinism + cap.
- `dsl/schedule_test.go` (extended): `NextInZone` for America/New_York; invalid zone error path.
- `pkg/events/schedule_test.go`: payload JSON shape; `Input` namespaced so system keys cannot be shadowed.
- `apps/default/service/schedulers/cron_test.go`: `planOne` table-driven — normal, missed-fire, invalid cron (parks), invalid TZ (parks).

### Integration (testcontainers)

- `apps/default/service/repository/schedule_test.go`:
  - `ClaimAndFireBatch` exactly-once under 10 concurrent goroutines × 500 seeded rows.
  - `ClaimAndFireBatch` rollback on INSERT failure (inject unique-key violation on `event_log` idempotency_key): no rows advance, no events emitted.
  - `CreateBatch` atomicity: seed a failure (duplicate name), expect no rows persisted.
  - `ActivateByWorkflow`: seed v1 active, v2 draft-schedules; call `ActivateByWorkflow(v2)`; assert v1 deactivated, v2 activated with `next_fire_at` in future.
  - `DeactivateByWorkflow`: seeds 3 versions; call; all deactivated, all `next_fire_at = NULL`.
  - Unique-index: second `Create` with duplicate `(tenant, partition, workflow_name, workflow_version, name)` errors.
- `apps/default/service/business/workflow_integration_test.go`:
  - `CreateWorkflow_OrphanRecoveryBlocked`: simulate Tx2 failure (insert conflicting schedule name); assert Tx1's workflow row still exists; retry errors on unique index.
  - `ActivateWorkflow_RecoversAfterTx2Failure`: inject a Tx2 failure; workflow is ACTIVE; retry idempotently succeeds; prior-version schedules deactivate on successful retry.
  - `ArchiveWorkflow_SchedulesOffBeforeWorkflowArchived`: confirm the two-tx ordering by watching write-timestamps.
  - `ListByWorkflow_TenantIsolated`: tenant-A and tenant-B both have a `same-name` workflow; each sees only their own.
- `apps/default/service/schedulers/scheduler_test.go`:
  - Configurable batch/interval honored via `cfg`.
  - `planOne` integration — `CronScheduler` implements the `SchedulePlanFn` contract that `ClaimAndFireBatch` consumes.

### Contract

- `TestScheduleFiredPayload_ShapeStable` — golden fixture for the serialised `schedule.fired` payload.

All DB-touching tests use `frametests.FrameBaseTestSuite` + testcontainers.

---

## Rollout

Single release, direct-to-main.

1. Land code on `main`.
2. Tag `v0.3.35`; release workflow builds three images.
3. Flux image-automation promotes `v0.3.35`; colony chart's pre-install migration Job runs AutoMigrate:
   - Adds `timezone` column.
   - Drops v1 `idx_sd_due`, rebuilds with tightened predicate.
   - Creates `idx_sd_workflow_unique`.
4. Serving pods roll.
5. **Bounce pgBouncer pods** in `datastore` ns to flush cached plans (operational; same pattern as v0.3.34).
6. Observability verifies: `scheduler_cron_fired_total` counter increments, no `invalid_cron_total` for known-good schedules, sweep duration stable.

### Rollback

- **Emergency dial-down**: set `CRON_SCHEDULER_BATCH_SIZE=1` env on the Deployment and restart. Batched path still runs but with batch size 1 — effectively per-row semantics; returns to v1 throughput without redeploy.
- **Full rollback**: repin `v0.3.34` in `ImagePolicy` filter. Schema gains (new column, indexes) are additive; no schema rollback.
- **Revert main**: `git revert origin/main..HEAD^` and retag as `v0.3.36`.

---

## Risks

1. **`VALUES` typing in the bulk UPDATE.** NULLs in `next_fire_at` mixed with timestamps require explicit `::timestamptz` casts on the first tuple. Integration test asserts this.
2. **`CreateInBatches(events, 500)` behaviour** — GORM-level. Verified in staging before production scale.
3. **Pool sizing tradeoff.** `SCHEDULER_POOL_MAX_CONNS=10` is the default. If operators crank `CRON_SCHEDULER_BATCH_SIZE=2000` without bumping the pool, one in-flight sweep can hold a connection for ~100 ms, leaving only 9 for other ops. Documented.
4. **Outbox becomes the next bottleneck at herd rates.** Same release includes outbox batch/concurrent publish; monitor `scheduler_outbox_lag_seconds`.
5. **`maps.Copy` → typed payload breaking change.** Downstream consumers that read `input_payload` by merging into the root object instead of reading `input.*` will see a shape change. Mitigated by the typed `ScheduleFiredPayload` contract test (golden fixture).
6. **DSL breaking change**: `ScheduleSpec.Active` removed. Go silently ignores unknown fields, so in-code test helpers that set `Active: true` fail to compile; prod DSL consumers pass JSON, which is a no-op. Only the Go test helper path needed updating.
7. **Orphan DRAFT workflow after failed materialisation** (CreateWorkflow Tx2 failure). DRAFT doesn't fire, so impact is operator-facing only; retry is blocked by `idx_wd_name_version`. Document recovery in operator runbook.

---

## Success criteria

- All v1 audit bugs closed (verified by new tests).
- `TestClaimAndFireBatch_ExactlyOnceUnderConcurrency` passes at `count=5 batch=500 workers=10`.
- Staging benchmark: 12-pod cluster delivers ≥ 5 k fires/sec for 5 min without pod restart or DB saturation alerts.
- `ActivateWorkflow` with 1 000 schedules completes in < 500 ms (v1: ~2 s).
- HelmRelease `READY=True` ≥ 10 min post-deploy, zero restarts.
- Observability dashboards populate: fires/sec, sweep duration p99, invalid_cron counter steady at 0.

## References

- v1 spec: `docs/superpowers/specs/2026-04-18-scheduler-v1-design.md`
- Prior revision of this spec (wider interface reshape): commit `8e24921` (superseded)
- Current fire path: `apps/default/service/schedulers/cron.go`, `apps/default/service/repository/schedule.go`
- Lifecycle source: `apps/default/service/business/workflow.go`
- Tenancy memory: `/home/j/.claude/projects/-home-j-code-antinvestor-service-trustage/memory/feedback_tenancy_filters.md`
