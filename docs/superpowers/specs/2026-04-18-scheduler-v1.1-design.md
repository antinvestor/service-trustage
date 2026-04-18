# Scheduler v1.1: batched fire, bug fixes, TZ, Archive

**Date:** 2026-04-18
**Status:** approved
**Scope:** `service-trustage` — builds on scheduler v1 (spec `2026-04-18-scheduler-v1-design.md`, shipped as `v0.3.34`). Direct-to-main commits.

## Why v1.1

The scheduler v1 release ships a correct exactly-once fire path but four production-relevant gaps surfaced under audit:

1. **Throughput ceiling is too low for the platform's target.** `ClaimAndFireBatch` opens one Postgres transaction per row. At 50 rows/batch × 30 s poll = 100 fires/min/pod = 20 fires/sec cluster-wide (12 pods HPA cap). Hundreds of millions of scheduled tasks per day requires two orders of magnitude more.

2. **Correctness bugs** that are tolerable at low load but unacceptable at scale:
   - `ListByWorkflow` omits tenant/partition filtering (cross-tenant read leak).
   - `CreateWorkflow` is not atomic across workflow + schedule inserts (orphan rows on partial failure).
   - User-supplied `input_payload` can shadow system fields in `schedule.fired` events.
   - Invalid cron at fire time still emits a ghost event before parking.

3. **Stateful hooks that would be misused later.** Today's interface exposes `ScheduleRepository.Pool()` (a layering leak whose only point is to enable LISTEN/NOTIFY) and `ScheduleSpec.Active *bool` (a tri-state flag whose only purpose is to enable pause/resume). Neither feature is allowed — so the hooks must go before someone tries.

4. **No observability on the fire path.** Compare to `outbox.go:107` which has a trace span + gauge. `CronScheduler` has neither. At scale, silent backlog is operationally unacceptable.

Plus two allowed additions: per-schedule timezone, and `ArchiveWorkflow` RPC.

## Goals

- **100+ M fires/day** achievable at cluster layer with DB capable of the sustained write rate (~6 k fires/sec at 12 pods, ~24 k/sec at 48 pods).
- Exactly-once fire semantics preserved.
- All v1 audit bugs resolved.
- Interface surface narrow enough that LISTEN/NOTIFY, pause/resume, and single-schedule update are structurally harder to add than to design correctly (i.e. they require new abstractions, not a tweak).
- Timezone support (per-schedule IANA, default UTC).
- `ArchiveWorkflow` RPC — DRAFT | ACTIVE → ARCHIVED, atomically deactivating the workflow's schedules.
- Full Go-pattern adherence: `util.Log(ctx)`, wrapped errors, interface-first repos, table-driven tests, testcontainers (not mocks), no raw goroutines for critical work, typed event payloads in `pkg/events/`.

## Non-goals (hard exclusions)

- LISTEN/NOTIFY. Removing `ScheduleRepository.Pool()` from the interface is the concrete step that prevents its easy addition.
- Pause/resume a single schedule. Removing `ScheduleSpec.Active *bool` is the concrete step that prevents its easy addition.
- Update one schedule's `cron_expr` without shipping a new workflow version. No `UpdateSchedule` RPC, no mutable `cron_expr` business method.
- Per-schedule missed-fire threshold override (global constant stays).
- Sub-minute cron precision (5-field parser only, no seconds).

---

## Architecture

No new services, no new tables. Schema gains one column, one unique index, one predicate tightening. The scheduler loop becomes a single three-statement tx per sweep:

```
┌────────────────────────────────────────────────────────────────────┐
│ CronScheduler.RunOnce — one sweep (configurable interval, default 1s)
│                                                                    │
│  tx BEGIN                                                          │
│    1. SELECT … FOR UPDATE SKIP LOCKED LIMIT $batch  (one roundtrip)│
│    2. INSERT INTO event_log (...), (...)... ($batch rows)          │
│    3. UPDATE schedule_definitions s                                │
│         SET last_fired_at, next_fire_at, jitter_seconds, modified  │
│       FROM (VALUES (id,tenant,partition,…), …) AS v(…)             │
│       WHERE s.id=v.id AND s.tenant_id=v.tenant_id                  │
│         AND s.partition_id=v.partition_id                          │
│  COMMIT                                                            │
│                                                                    │
│  Round-trips per sweep: 3 (plus BEGIN/COMMIT)                      │
│  Batch size: configurable (default 500)                            │
└────────────────────────────────────────────────────────────────────┘
```

Correctness guarantees:

- SKIP LOCKED holds row locks for the entire tx. No second pod sees claimed rows until COMMIT/ROLLBACK.
- On crash or failure, the whole batch rolls back; schedules remain due; next sweep retries.
- `IdempotencyKey = scheduleID + ":" + RFC3339Nano(now)` where `now` is the single batch timestamp — unique per row across the batch and across batches.
- Tenant/partition filtered throughout.

---

## Throughput model

| Configuration | Per pod | 12 pods | 48 pods |
|---|---|---|---|
| v1 (today): batch=50, interval=30s, per-row tx | 100/min | 20/sec | 80/sec |
| v1.1 (default): batch=500, interval=1s, batched tx | 500/sec | 6 k/sec | 24 k/sec |
| v1.1 (stress): batch=2000, interval=1s | 2 k/sec | 24 k/sec | 96 k/sec |

v1.1 ceiling (default) ≈ 520 M fires/day at 12 pods, which is the headline number for the platform target. The DB — event_log inserts + partial-index updates — is the next bottleneck, and that's where the user said "if the database can cope up" kicks in.

---

## Data model changes

### Schema (`apps/default/service/models/schedule.go`)

Add one column:

| Column | Type | Default | Purpose |
|---|---|---|---|
| `timezone` | `varchar(64) NOT NULL` | `'UTC'` | IANA zone used to evaluate the cron expression |

Existing rows receive the default on AutoMigrate — no data migration script required.

### Indexes (`apps/default/service/repository/migrate.go`)

Two changes:

1. **Tighten `idx_sd_due`** predicate to include `next_fire_at IS NOT NULL`:
   ```sql
   CREATE INDEX idx_sd_due ON schedule_definitions (next_fire_at ASC)
     WHERE active = true AND deleted_at IS NULL AND next_fire_at IS NOT NULL;
   ```
   Eliminates parked (NULL `next_fire_at`) rows from index visits.

2. **Add unique idempotency index** preventing duplicate schedule materialisation:
   ```sql
   CREATE UNIQUE INDEX idx_sd_workflow_unique
     ON schedule_definitions (tenant_id, partition_id, workflow_name, workflow_version, name)
     WHERE deleted_at IS NULL;
   ```
   `materialiseSchedules` retries no longer silently duplicate rows; PG enforces uniqueness. Combined with the atomicity fix (B2), the write is idempotent at both layers.

---

## Repository interface (`apps/default/service/repository/schedule.go`)

Shrinks in visible surface, grows in tx-acceptance:

```go
type ScheduleRepository interface {
    // Writes — all accept an optional *gorm.DB for cross-repo tx composition.
    CreateTx(ctx context.Context, tx *gorm.DB, schedule *models.ScheduleDefinition) error
    // Create is the convenience wrapper for callers that don't need a shared tx.
    Create(ctx context.Context, schedule *models.ScheduleDefinition) error

    // Reads — tenant/partition filtered via BaseRepository scope (enforced).
    ListByWorkflow(ctx context.Context, workflowName string, workflowVersion int) ([]*models.ScheduleDefinition, error)

    // Lifecycle — tx-bound; callers pass their own tx.
    SetActiveByWorkflowTx(
        ctx context.Context, tx *gorm.DB,
        workflowName string, workflowVersion int,   // workflowVersion < 0 no longer valid; panic
        tenantID, partitionID string,
        active bool,
        seedNextFireAt *time.Time,
        seedJitterSeconds int,
    ) error

    // Archive — deactivates all versions of the named workflow in one tx.
    ArchiveWorkflowTx(
        ctx context.Context, tx *gorm.DB,
        workflowName, tenantID, partitionID string,
    ) error

    // Fire hot path — fully-batched, no callback, returns the plan for metrics.
    ClaimAndFireBatch(ctx context.Context, planner FirePlanner, now time.Time, limit int) (fired int, err error)

    // Expose a pool-derived tx starter so business can compose cross-repo transactions
    // WITHOUT exposing the raw pool.Pool. One allowed tx-starter method, no direct
    // connection access.
    Transact(ctx context.Context, fn func(tx *gorm.DB) error) error
}

// FirePlanner builds the per-row fire plan in pure Go — no DB access inside.
type FirePlanner interface {
    PlanFires(ctx context.Context, batch []*models.ScheduleDefinition, now time.Time) (fires []ScheduledFire)
}

type ScheduledFire struct {
    Schedule      *models.ScheduleDefinition  // source row (for audit)
    Event         *models.EventLog            // prepared event_log row, partition info copied
    NextFireAt    *time.Time                  // nil = park the schedule
    JitterSeconds int
}
```

**Design decisions:**

- **`Pool()` is gone.** Business composes cross-repo transactions by calling `scheduleRepo.Transact(ctx, fn)` which gives them a scoped `*gorm.DB`. The raw `pool.Pool` never leaves the repository layer.
- **`SetActiveByWorkflow(workflowVersion=-1)` is gone.** Archive has its own method `ArchiveWorkflowTx` with clear semantics.
- **`FirePlanner` is pure.** The scheduler implements it without DB access — parses cron, computes jitter, builds event rows. The repo does the two bulk writes. Unit-testable without a DB.
- **All tx-accepting methods have a `-Tx` suffix**, following Go conventions for dual-mode APIs (see stdlib `sql` package's `Tx`-bound methods).

---

## Fire-path implementation (`apps/default/service/schedulers/cron.go`)

The scheduler becomes a thin orchestrator:

```go
type CronScheduler struct {
    scheduleRepo repository.ScheduleRepository
    cfg          *config.Config
    metrics      *telemetry.Metrics
}

func (s *CronScheduler) RunOnce(ctx context.Context) int {
    ctx, span := telemetry.StartSpan(ctx, telemetry.TracerEngine, telemetry.SpanSchedulerCron)
    defer span.End()

    now := time.Now().UTC()
    start := now

    fired, err := s.scheduleRepo.ClaimAndFireBatch(ctx, s, now, s.cfg.CronSchedulerBatchSize)
    if err != nil {
        util.Log(ctx).WithError(err).Error("cron scheduler: sweep failed")
        s.metrics.SchedulerCronFired.Add(ctx, 0, attrsFail())
        return 0
    }

    dur := time.Since(start)
    s.metrics.SchedulerCronFired.Add(ctx, int64(fired), attrsOK())
    s.metrics.SchedulerCronSweepDuration.Record(ctx, dur.Seconds())

    return fired
}

// PlanFires implements FirePlanner — pure Go, no DB.
func (s *CronScheduler) PlanFires(
    ctx context.Context,
    batch []*models.ScheduleDefinition,
    now time.Time,
) []repository.ScheduledFire {
    fires := make([]repository.ScheduledFire, 0, len(batch))
    for _, sched := range batch {
        fire, ok := planOne(ctx, sched, now)
        if !ok {
            // Invalid cron: park the schedule (no event emission).
            s.metrics.SchedulerCronInvalid.Add(ctx, 1, attrsSchedule(sched))
            fires = append(fires, repository.ScheduledFire{
                Schedule: sched, Event: nil, NextFireAt: nil, JitterSeconds: 0,
            })
            continue
        }
        fires = append(fires, fire)
    }
    return fires
}
```

`planOne` is the unit-testable core — parses cron, computes next fire + jitter (via `dsl/schedutil`), builds `models.EventLog` with fixed system fields preceded by user `input_payload` (so system fields win on key collisions), returns a `ScheduledFire`. Pure function.

`ClaimAndFireBatch` inside the repo:

```go
func (r *scheduleRepository) ClaimAndFireBatch(
    ctx context.Context,
    planner FirePlanner,
    now time.Time,
    limit int,
) (fired int, err error) {
    err = r.Transact(ctx, func(tx *gorm.DB) error {
        // 1. Claim.
        var batch []*models.ScheduleDefinition
        if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
            Where("active = ? AND deleted_at IS NULL AND next_fire_at IS NOT NULL AND next_fire_at <= ?", true, now).
            Order("next_fire_at ASC").
            Limit(limit).
            Find(&batch).Error; err != nil {
            return fmt.Errorf("claim due batch: %w", err)
        }
        if len(batch) == 0 {
            return nil
        }

        // 2. Plan (pure, no DB).
        fires := planner.PlanFires(ctx, batch, now)

        // 3. Multi-row INSERT event_log.
        events := make([]*models.EventLog, 0, len(fires))
        for _, f := range fires {
            if f.Event != nil {
                events = append(events, f.Event)
            }
        }
        if len(events) > 0 {
            if err := tx.CreateInBatches(events, 500).Error; err != nil {
                return fmt.Errorf("batch insert event_log: %w", err)
            }
        }

        // 4. Single UPDATE with VALUES join.
        if err := applyFireUpdates(tx, fires, now); err != nil {
            return fmt.Errorf("batch update schedules: %w", err)
        }

        fired = len(fires)
        return nil
    })
    return
}
```

`applyFireUpdates` constructs:

```sql
UPDATE schedule_definitions s
   SET last_fired_at  = v.last_fired_at,
       next_fire_at   = v.next_fire_at,
       jitter_seconds = v.jitter_seconds,
       modified_at    = v.modified_at
  FROM (
    VALUES
      ($1::uuid, $2::text, $3::text, $4::timestamptz, $5::timestamptz, $6::int, $4::timestamptz),
      ($7::uuid, $8::text, $9::text, $4::timestamptz, $10::timestamptz, $11::int, $4::timestamptz),
      …
  ) AS v(id, tenant_id, partition_id, last_fired_at, next_fire_at, jitter_seconds, modified_at)
 WHERE s.id = v.id AND s.tenant_id = v.tenant_id AND s.partition_id = v.partition_id;
```

Explicit casts on the first row are required because `VALUES` infers types from the first row and subsequent mismatched NULLs can error.

---

## Business-layer changes (`apps/default/service/business/workflow.go`)

### `CreateWorkflow` — atomic

```go
func (b *workflowBusiness) CreateWorkflow(ctx context.Context, dslBlob json.RawMessage) (*models.WorkflowDefinition, error) {
    spec, err := dsl.Parse(dslBlob)
    if err != nil { return nil, fmt.Errorf("parse DSL: %w", err) }
    if res := dsl.Validate(spec); !res.Valid() {
        return nil, fmt.Errorf("%w: %w", ErrDSLValidationFailed, res.Error())
    }
    if err := validateExecutableWorkflow(spec); err != nil {
        return nil, fmt.Errorf("%w: %w", ErrDSLValidationFailed, err)
    }

    def := buildDefinition(spec, dslBlob)

    if err := b.registerStepSchemas(ctx, spec); err != nil {
        return nil, fmt.Errorf("register schemas: %w", err)
    }

    txErr := b.scheduleRepo.Transact(ctx, func(tx *gorm.DB) error {
        if err := b.defRepo.CreateTx(ctx, tx, def); err != nil {
            return fmt.Errorf("persist workflow: %w", err)
        }
        for _, sspec := range spec.Schedules {
            sched, err := materialiseOne(def, sspec)
            if err != nil {
                return err
            }
            if err := b.scheduleRepo.CreateTx(ctx, tx, sched); err != nil {
                return fmt.Errorf("create schedule %s: %w", sspec.Name, err)
            }
        }
        return nil
    })
    if txErr != nil { return nil, txErr }
    return def, nil
}
```

Requires adding `CreateTx(ctx, tx, def)` to `WorkflowDefinitionRepository`.

### `ActivateWorkflow` — O(1) bulk update

Replaces the per-schedule UPDATE loop with one VALUES-join statement matching the fire-path shape. Same pattern: SELECT this version's schedules, build the plan (cronNext + jitter) in Go, then one bulk UPDATE — all inside the lifecycle tx. Per-schedule round-trips collapse to one regardless of schedule count.

At 10 k schedules per workflow, activation goes from ~20 s (10 k × 2 ms) to ~50 ms.

### `ArchiveWorkflow` — new

```go
func (b *workflowBusiness) ArchiveWorkflow(ctx context.Context, id string) error {
    return b.scheduleRepo.Transact(ctx, func(tx *gorm.DB) error {
        def, err := b.defRepo.GetByIDTx(ctx, tx, id)
        if err != nil {
            return fmt.Errorf("%w: %w", ErrWorkflowNotFound, err)
        }
        if err := def.TransitionTo(models.WorkflowStatusArchived); err != nil {
            return fmt.Errorf("%w: %w", ErrInvalidWorkflowStatus, err)
        }
        if err := b.defRepo.UpdateTx(ctx, tx, def); err != nil {
            return fmt.Errorf("update workflow: %w", err)
        }
        return b.scheduleRepo.ArchiveWorkflowTx(ctx, tx, def.Name, def.TenantID, def.PartitionID)
    })
}
```

### `GetWorkflowWithSchedules` — tenancy-safe

`ListByWorkflow` is already tenancy-scoped via `BaseRepository` — the audit finding (B1) is addressed by delegating to `r.BaseRepository.Pool().DB(ctx, false).Where(...)` through the scoped session, NOT the raw pool. The fix is one-line: use BaseRepository-provided tenancy scope instead of bypassing it.

---

## DSL changes (`dsl/`)

### `WorkflowSpec.Schedules` + `ScheduleSpec`

```go
type ScheduleSpec struct {
    Name         string         `json:"name"`
    CronExpr     string         `json:"cron_expr"`
    Timezone     string         `json:"timezone,omitempty"`     // IANA; default "UTC"
    InputPayload map[string]any `json:"input_payload,omitempty"`
    // Active is REMOVED in v1.1 — DRAFT workflows never fire until activated,
    // and there is no pause/resume RPC. Dead field in v1.
}
```

Validator change: `validateSchedules` parses `Timezone` via `time.LoadLocation` and rejects invalid zones. Empty → `"UTC"`.

### `dsl/schedule.go` — timezone-aware

```go
// Next returns the first fire time strictly after `from`, evaluated in the
// schedule's declared timezone. Timezone-less schedules are evaluated in UTC.
func (s CronSchedule) NextInZone(from time.Time, zone string) (time.Time, error) {
    loc, err := time.LoadLocation(zone)
    if err != nil {
        return time.Time{}, fmt.Errorf("load zone %q: %w", zone, err)
    }
    next := s.schedule.Next(from.In(loc))
    return next.UTC(), nil
}
```

Legacy `Next(from)` is kept as a thin alias for `NextInZone(from, "UTC")`.

### `dsl/schedutil.go` — extracted shared helper (new file)

```go
package dsl

import (
    "hash/fnv"
    "time"
)

const (
    CronMaxJitter           = 30 * time.Second
    cronJitterPeriodDivisor = 10
)

// JitterFor returns a deterministic per-schedule offset to flatten thundering
// herds at common cron boundaries. Cap = min(period/10, CronMaxJitter).
func JitterFor(scheduleID string, cronSched CronSchedule, nominal time.Time) time.Duration {
    following := cronSched.Next(nominal)
    period := following.Sub(nominal)
    if period <= 0 {
        return 0
    }
    maxDur := period / cronJitterPeriodDivisor
    if maxDur > CronMaxJitter {
        maxDur = CronMaxJitter
    }
    if maxDur <= 0 {
        return 0
    }
    h := fnv.New64a()
    _, _ = h.Write([]byte(scheduleID))
    return time.Duration(int64(h.Sum64() % uint64(maxDur)))
}
```

Callers in `schedulers/cron.go` and `business/workflow.go` replaced with `dsl.JitterFor(...)`.

---

## Event types (`pkg/events/schedule.go` — new)

```go
package events

const ScheduleFiredType = "schedule.fired"

// ScheduleFiredPayload is the wire shape of `schedule.fired` events.
// System fields are set LAST so user-supplied input_payload cannot shadow them.
type ScheduleFiredPayload struct {
    ScheduleID   string         `json:"schedule_id"`
    ScheduleName string         `json:"schedule_name"`
    FiredAt      string         `json:"fired_at"` // RFC3339
    Input        map[string]any `json:"input,omitempty"`
}
```

`planOne` in the scheduler builds this struct directly — no `maps.Copy`, no shadow risk — then JSON-marshals it for `event_log.Payload`.

---

## Proto changes (`proto/workflow/v1/workflow.proto`)

1. Add `timezone string` field to `ScheduleDefinition` proto message.
2. Add `ArchiveWorkflow` RPC:

```proto
rpc ArchiveWorkflow(ArchiveWorkflowRequest) returns (ArchiveWorkflowResponse) {
  option (common.v1.method_permissions) = { permissions: ["workflow_manage"] };
}

message ArchiveWorkflowRequest  { string id = 1; }
message ArchiveWorkflowResponse { WorkflowDefinition workflow = 1; }
```

No new permissions (reuses `workflow_manage`).

Regeneration via `make proto-gen` updates `gen/go/workflow/v1/workflow.pb.go` and `workflowv1connect/`.

---

## Configuration (`apps/default/config/config.go`)

Add:

```go
type Config struct {
    // existing fields...

    // Scheduler tuning.
    CronSchedulerBatchSize     int `env:"CRON_SCHEDULER_BATCH_SIZE"       envDefault:"500"`
    CronSchedulerIntervalSecs  int `env:"CRON_SCHEDULER_INTERVAL_SECONDS" envDefault:"1"`

    // Dedicated scheduler DB pool sizing.
    SchedulerPoolMaxConns      int `env:"SCHEDULER_POOL_MAX_CONNS"        envDefault:"10"`
    SchedulerPoolMinConns      int `env:"SCHEDULER_POOL_MIN_CONNS"        envDefault:"2"`
}
```

`main.go` wiring:

```go
// Primary pool for handlers — unchanged.
ctx, svc := frame.NewServiceWithContext(ctx,
    frame.WithName(cfg.Name()),
    frame.WithConfig(&cfg),
    frame.WithDatastore(
        pool.WithPreferSimpleProtocol(true),
        pool.WithPreparedStatements(false),
    ),
)

// Dedicated scheduler pool.
schedulerPool := pool.NewPool(ctx)
if err := schedulerPool.AddConnection(ctx,
    pool.WithConnection(cfg.DatabasePrimary[0], false),
    pool.WithPreparedStatements(false),
    pool.WithPreferSimpleProtocol(true),
    pool.WithMaxConnections(cfg.SchedulerPoolMaxConns),
    pool.WithMinConnections(cfg.SchedulerPoolMinConns),
); err != nil {
    log.WithError(err).Fatal("scheduler pool init")
}
svc.DatastoreManager().AddPool(ctx, "scheduler", schedulerPool)

// Scheduler repositories use the dedicated pool.
scheduleRepoForScheduler := repository.NewScheduleRepository(schedulerPool)
```

---

## Outbox batch publish (`apps/default/service/schedulers/outbox.go`)

Current: loop over claimed events, call `queueMgr.Publish(subject, payload)` once per event.

Change: collect a batch of serialised messages, call `queueMgr.Publish(subject, payloads ...[]byte)`. Verify Frame's `QueueManager.Publish` signature supports batched publish; if not, look for a batch variant or submit one goroutine per message via `workerpool` bounded by a config knob `OutboxPublishConcurrency`. The goal is to stop being the per-event NATS round-trip bottleneck.

(Per Frame patterns: check if `QueueManager` exposes `PublishBatch` or similar; fall back to `workerpool.SubmitJob` with a bounded pool if not.)

---

## Observability (`apps/default/service/schedulers/cron.go` + `pkg/telemetry/`)

Add constants to `pkg/telemetry/metrics.go`:

```go
const (
    SpanSchedulerCron = "scheduler.cron.sweep"

    MetricSchedulerCronFired          = "scheduler_cron_fired_total"
    MetricSchedulerCronSweepDuration  = "scheduler_cron_sweep_duration_seconds"
    MetricSchedulerCronBacklog        = "scheduler_cron_backlog_seconds"
    MetricSchedulerCronInvalidCron    = "scheduler_cron_invalid_cron_total"
)
```

Wire them on `telemetry.Metrics`:
- `SchedulerCronFired` — counter keyed by `result ∈ {ok, fail}`.
- `SchedulerCronSweepDuration` — histogram of sweep wall time in seconds.
- `SchedulerCronBacklog` — gauge of `now - MIN(next_fire_at) WHERE active=true`, sampled every sweep via a `SELECT MIN(next_fire_at)`.
- `SchedulerCronInvalid` — counter keyed by `schedule_id, tenant_id`.

Pass `*telemetry.Metrics` into `NewCronScheduler`; update `main.go` wiring.

---

## Testing

### Unit (no DB)

- `dsl/schedutil_test.go`: `JitterFor` determinism + cap.
- `dsl/schedule_test.go` (extended): `NextInZone` for America/New_York and non-existent times (DST forward leap) — assert deterministic.
- `pkg/events/schedule_test.go`: payload build ordering — user `Input` map never overwrites `ScheduleID`, `ScheduleName`, `FiredAt`.
- `apps/default/service/schedulers/cron_test.go`: `planOne` table-driven — normal, missed-fire, invalid cron, TZ aware.

### Integration (testcontainers)

- `apps/default/service/repository/schedule_test.go`:
  - `TestClaimAndFireBatch_ExactlyOnceUnderConcurrency` — 10 goroutines × 500 rows, every row fires exactly once. (Extend existing.)
  - `TestClaimAndFireBatch_BatchedTxSemantics` — one INSERT + one UPDATE per sweep (use a `QueryMatcher` or statement counter).
  - `TestClaimAndFireBatch_RollbackOnFailure` — inject a UPDATE failure; assert no event_log rows committed, no next_fire_at advanced.
  - `TestUniqueIndex_PreventsDuplicateMaterialise` — insert two schedules with same (tenant, partition, workflow, version, name); second errors with unique violation.
- `apps/default/service/business/workflow_integration_test.go`:
  - `TestCreateWorkflow_Atomicity` — inject a schedule-insert failure; assert workflow row also rolled back.
  - `TestArchiveWorkflow_DeactivatesAllVersions` + cross-tenant isolation.
  - `TestActivateWorkflow_BulkUpdatePerformance` — 1000 schedules; activation completes in one tx (count statements via gorm callback).
- `apps/default/service/schedulers/scheduler_test.go`:
  - Backpressure: verify `RunOnce` blocking doesn't pile goroutines.
  - Configurable batch/interval honored.
- `apps/default/tests/e2e_test.go` (if exists): end-to-end create → activate → fire → observe event_log row with correct schema.

### Contract

- `TestScheduleFiredPayload_ShapeStable` — assert serialised shape matches a golden fixture so downstream consumers aren't broken by additions.

All DB-touching tests use `frametests.FrameBaseTestSuite` with real Postgres (testpostgres). No mocks for database.

---

## Rollout

1. Land the code on `main`. All commits direct; already approved path.
2. Tag `v0.3.35`; release workflow builds three images.
3. Flux image-automation promotes `v0.3.35`; HelmRelease pre-install migration Job runs AutoMigrate which:
   - Adds `timezone` column.
   - Replaces `idx_sd_due` with the tightened partial index.
   - Creates `idx_sd_workflow_unique`.
4. Migration Job completes; serving pods roll.
5. **Bounce pgBouncer pods in `datastore` ns** to flush any cached plans (operational, same as v0.3.34). Absent that, the first sweep may hit `SQLSTATE 0A000` once; harmless but surfaces in logs.
6. Observability verifies: `scheduler_cron_fired_total` counter increments, backlog gauge ≈ 0, no invalid-cron counter events for known-good schedules.

### Rollback

- Revert `v0.3.35` tag and re-roll `v0.3.34`. Schema-wise the new column and indexes remain (additive); no schema rollback required or safe.
- If the batched UPDATE surfaces a regression in production, the simplest mitigation is env `CRON_SCHEDULER_BATCH_SIZE=1` — reverts functional behaviour to per-row semantics while keeping the new code path. No restart needed if env is reloaded.

---

## Risks & mitigations

1. **Batched UPDATE `VALUES` typing surprises on Postgres.** Partial NULLs in `next_fire_at` require explicit `::timestamptz` casts on the first VALUES row. Integration test covers it.
2. **`CreateInBatches(events, 500)` GORM nuances** — GORM-level batching. If GORM fragments at N < 500 we still get good throughput; if it single-statements we see one INSERT. Measure in staging before scaling.
3. **Connection pool sizing** — default `SCHEDULER_POOL_MAX_CONNS=10` is the cluster-visible cap. If operators crank `CRON_SCHEDULER_BATCH_SIZE=2000` without bumping the pool, a single sweep holds a connection for the full sweep duration; pool underrun is possible under herd. Documented tuning guidance in `apps/default/config/config.go` comments.
4. **Outbox publish bottleneck shifts to NATS** — we unlock the scheduler but push the load onto the outbox + NATS cluster. Outbox batching (same release) mitigates. Monitor `scheduler_outbox_lag_seconds`.
5. **Breaking change to `ScheduleRepository` interface**: `Pool()` removed, `SetActiveByWorkflow` wildcard gone, new `FirePlanner` / `Transact` surface. All callers are in this repo; change is enclosed. No external SDK change.
6. **DSL breaking change**: `ScheduleSpec.Active` removed. `json:"active"` still parses (Go ignores unknown fields) — but any caller today setting it is accepting silent no-op. User confirmed only the Go test helper writes schedules; no external consumers.

---

## Success criteria

- All audit bugs closed (verified by new tests in the relevant packages).
- `TestClaimAndFireBatch_ExactlyOnceUnderConcurrency` passes at `count=5 batch=500 workers=10`.
- Staging sustained-fire benchmark: 12-pod cluster delivers ≥ 5 k fires/sec for 5 minutes without DB saturation or pod restart.
- `ActivateWorkflow` with 1000 schedules completes in < 500 ms (currently ~2 s).
- Cluster HelmRelease `READY=True` for ≥ 10 min post-deploy with zero restarts.
- Observability dashboards populate: fires/sec, sweep duration p99, backlog gauge steady near 0.

## References

- v1 spec (prior): `docs/superpowers/specs/2026-04-18-scheduler-v1-design.md`
- Audit findings (this conversation): three-axis audit — correctness/design/scale
- Current fire path: `apps/default/service/schedulers/cron.go`, `apps/default/service/repository/schedule.go`
- Activate lifecycle: `apps/default/service/business/workflow.go:300-387`
- Memory (user feedback): `/home/j/.claude/projects/-home-j-code-antinvestor-service-trustage/memory/feedback_tenancy_filters.md`
