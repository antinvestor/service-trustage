# Scheduler v1: spec-driven, cron-based, horizontally scalable

**Date:** 2026-04-18
**Status:** approved
**Scope:** `service-trustage` — code changes under `apps/default/`, `dsl/`, and `proto/workflow/v1`. No new proto package, no new RPC service. Commits go direct to `main`.

## Problem

The scheduler subsystem ships with several gaps that become acute at scale:

1. **No declarative writer.** `dsl.WorkflowSpec` has no `schedule` field. The only code that writes rows to `schedule_definitions` today is the Go test helper `scheduleRepo.Create()`. Operators cannot define schedules without writing Go.

2. **`CronExpr` is not cron.** Despite the column name, `apps/default/service/schedulers/cron.go:147` calls `dsl.ParseDuration`, so values must be Go-style durations (`"1h"`, `"7d"`). Real cron expressions (`"0 */5 * * *"`) error silently by setting `next_fire_at = nil`.

3. **`fireSchedule` is not transactional.** At `cron.go:96-142`, `eventRepo.Create` and `scheduleRepo.UpdateFireTimes` run as two independent auto-commit statements. `FindDue`'s `FOR UPDATE SKIP LOCKED` lock is released the moment the SELECT returns (no explicit tx wraps the read and the writes). In a multi-pod deployment two pods polling within the same millisecond **can fire the same schedule twice**. At one pod the bug is latent; scaling out makes it reproducible.

4. **No thundering-herd protection.** Schedules created near the same instant drift together and fire at the same second forever. Cron's `0 */5 * * *` puts every schedule on `:00`, `:05`, `:10`. Tens of thousands of schedules firing in the same second is a write spike neither the DB nor the outbox benefit from.

5. **Scan cost grows with table size, not due rows.** The current `FindDue` query has no partial index. At millions of rows PostgreSQL still has to walk or bitmap-scan to find due ones every 30 s.

6. **No missed-fire policy.** If a pod is down for two hours and comes back up, affected schedules have `next_fire_at` two hours in the past; the scheduler will fire them back-to-back trying to catch up, which is almost never wanted.

## Design principle: spec is the source of truth

Schedules are a property of a workflow definition, not a separately managed resource. They are declared inside a `WorkflowSpec`, materialised into `schedule_definitions` when `CreateWorkflow` runs, and follow the workflow's lifecycle for the rest of their life. There is **no `ScheduleService` RPC**, no pause/resume/update/delete endpoints, no schedule-level CRUD anywhere. If you need to change a schedule, you ship a new workflow version; if you need to stop one, you archive the workflow.

This keeps the API surface small, the invariants easy to reason about, and the hot path free of mutation concerns that are hard to scale.

## Goals

- Schedules defined declaratively inside a workflow JSON/YAML spec.
- Real cron expressions (5-field, UTC evaluation).
- Exactly-once fire across any number of scheduler-running pods.
- Scan cost independent of table size up to 10 M+ schedules.
- Horizontal scaling needs **zero configuration** — any pod added to the deployment starts contributing immediately. No shard assignment, no leader election, no partition map.
- Thundering-herd flattening via deterministic per-schedule jitter.
- Sane behaviour after pod downtime: fire once, skip-forward to the next slot.
- Schedule lifecycle derived entirely from workflow lifecycle.

## Non-goals (deferred)

- Any runtime schedule mutation RPC (the whole point).
- LISTEN/NOTIFY for sub-second wake. Frame exposes only GORM; driving `pgx` LISTEN directly requires a dedicated connection outside the pool and a reconnection loop. Not required for correctness; coupling the system to a feature that is hard to scale buys nothing.
- Per-schedule timezone. UTC only.
- Per-schedule missed-fire-threshold override. One global policy.
- Schedule fire history / audit. The `event_log` row per fire is sufficient.
- Web UI.
- Partitioning `schedule_definitions` by tenant.

## Architecture

Nothing about the topology changes. Every `trustage` pod already runs `CronScheduler.Start` (`apps/default/cmd/main.go:208`), and `FindDue` already uses `FOR UPDATE SKIP LOCKED`. The "zero-config horizontal scale" property is **already present**; this spec hardens the row contract around it (transactional fire + partial index) and adds the declarative path. Adding pods requires no config change after this work, just as it requires none now.

```
┌──────────────────────────────────────────────────────────────────┐
│ trustage pods (N replicas, HPA 1-12)                              │
│                                                                   │
│  ┌───────────────────────┐        ┌──────────────────────────┐    │
│  │ WorkflowService       │        │ CronScheduler            │    │
│  │ - CreateWorkflow      │        │ - 30s sweep              │    │
│  │   materialises        │        │ - ClaimAndFireBatch      │    │
│  │   Spec.Schedules      │        │   (one tx per fire)      │    │
│  │ - ActivateWorkflow    │        │                          │    │
│  │   flips schedules     │        │                          │    │
│  │   active=true         │        │                          │    │
│  │ - ArchiveWorkflow     │        │                          │    │
│  │   flips active=false  │        │                          │    │
│  └──────────┬────────────┘        └────────────┬─────────────┘    │
│             │  writes                          │  scans+fires     │
│             ▼                                  ▼                  │
└─────────────┼──────────────────────────────────┼──────────────────┘
              │                                  │
              ▼                                  ▼
┌──────────────────────────────────────────────────────────────────┐
│ PostgreSQL (CNPG hub)                                            │
│                                                                   │
│   schedule_definitions                event_log (outbox)          │
│     id, tenant_id, cron_expr,         (unchanged)                 │
│     workflow_name, workflow_version,                              │
│     active, next_fire_at, ...                                     │
│     partial idx on (next_fire_at)                                 │
│     WHERE active AND deleted_at IS NULL                           │
│                                                                   │
│   Fires: FOR UPDATE SKIP LOCKED → insert event_log → UPDATE       │
│   next_fire_at + last_fired_at, all in ONE tx per schedule.       │
└──────────────────────────────────────────────────────────────────┘
```

## Data model

### Schema changes

One additive column to `schedule_definitions`:

| Column | Type | Default | Purpose |
|---|---|---|---|
| `jitter_seconds` | `integer NOT NULL` | `0` | Observable jitter baked into `next_fire_at` at materialisation time. |

`cron_expr` keeps its name; semantics change from "Go duration" to "5-field cron expression". The user confirmed only the test suite writes `schedule_definitions` today, so the breaking change has no production data footprint. The plan's first step verifies this against the live DB.

### Partial index

```
CREATE INDEX IF NOT EXISTS idx_schedule_due
    ON schedule_definitions (next_fire_at ASC)
    WHERE active = true
      AND deleted_at IS NULL
      AND next_fire_at IS NOT NULL;
```

Created via the existing `migrationIndexes()` pattern in `apps/default/service/repository/migrate.go`. Single most important scaling change in the spec — makes every pod's 30 s scan O(rows due) regardless of total table size.

## Cron parsing

Add dependency: `github.com/robfig/cron/v3`.

Strict 5-field parser:

```go
parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
```

No seconds, no descriptors (`@yearly` etc.). Rejecting seconds keeps the DSL honest — the 30 s poll is the floor.

New file `dsl/schedule.go`, infrastructure-free (per CLAUDE.md):

```go
type CronSchedule struct {
    expr     string                // canonical text
    schedule cron.Schedule         // parsed; robfig type
}

func ParseCron(expr string) (CronSchedule, error) { /* ... */ }

// Next returns the first fire time strictly after `from`.
func (s CronSchedule) Next(from time.Time) time.Time { return s.schedule.Next(from) }

func (s CronSchedule) Expr() string { return s.expr }
```

### Jitter

```go
// Deterministic per-schedule offset, stable across restarts.
// h = fnv64a(scheduleID)
// period = cronSchedule.Next(base).Sub(base)
// jitter = time.Duration(h mod min(period/10, 30s).Nanoseconds()) * time.Nanosecond
// nextFire = cronSchedule.Next(base) + jitter
// stored: jitter_seconds = int(jitter / time.Second)
```

Cap at 30 s matches the poll interval — larger jitter doesn't reduce herd pressure any further. Baked into `next_fire_at` at materialisation and on every fire, so zero cost on the scan hot path.

### Missed-fire policy

If `now - sched.NextFireAt > staleThreshold` (constant: 5 min) at fire time:

1. Fire **once** at `now` (one event_log row).
2. Set `last_fired_at = now`.
3. Set `next_fire_at = cronSched.Next(now) + jitter` (not `Next(old_next_fire)`).

Never emit a catch-up burst. Per-schedule override deferred to v2.

## Transactional fire

One repository method replaces both `FindDue` and `UpdateFireTimes`:

```go
// ClaimAndFireBatch scans for due schedules under one tx, invokes fireFn
// for each, and commits atomically. fireFn receives the schedule and a DB
// handle bound to the same tx so event_log inserts and next_fire_at updates
// participate in the same transaction as the row lock.
//
// fireFn returns the new next_fire_at and the jitter applied. The repository
// persists those onto the row before committing.
//
// Returns the number of schedules fired (fireFn did not error).
ClaimAndFireBatch(
    ctx context.Context,
    now time.Time,
    limit int,
    fireFn func(ctx context.Context, tx *gorm.DB, sched *models.ScheduleDefinition) (nextFire *time.Time, jitterSeconds int, err error),
) (int, error)
```

Implementation:

1. `db.Transaction(func(tx *gorm.DB) error { … })` (GORM-managed tx).
2. `SELECT ... FOR UPDATE SKIP LOCKED LIMIT N` against the partial index.
3. For each row, call `fireFn(ctx, tx, sched)`. The caller inserts into `event_log` using the `tx` handle and returns the new `nextFire` + jitter.
4. After `fireFn` returns, the repo issues `UPDATE schedule_definitions SET last_fired_at=now, next_fire_at=<nextFire>, jitter_seconds=<jitter>, modified_at=now WHERE id=<id>` on the same `tx`.
5. On `fireFn` error for any schedule: log, skip that one, continue the batch. The tx still commits the successful ones.

The tx boundary is what restores exactly-once. The `FOR UPDATE SKIP LOCKED` lock is held across the full fire, so no second pod can see the row until commit.

`CronScheduler.fireSchedule` is rewritten as the `fireFn` callback. It inserts into `event_log` and returns the next fire time:

```go
func (s *CronScheduler) fireOne(ctx context.Context, tx *gorm.DB, sched *models.ScheduleDefinition) (*time.Time, int, error) {
    now := time.Now().UTC()

    // 1. Build event_log payload (schedule_id, schedule_name, fired_at, merged input_payload).
    // 2. tx.Create(&eventLog)
    // 3. Parse sched.CronExpr via dsl.ParseCron.
    // 4. Pick baseline:
    //      if now - sched.NextFireAt > staleThreshold:  base = now        (missed-fire)
    //      else:                                         base = *sched.NextFireAt  (normal)
    // 5. nextFire = cronSched.Next(base) + jitterFor(sched.ID, base, cronSched)
    // 6. return &nextFire, int(jitter / time.Second), nil
}
```

## DSL: WorkflowSpec.Schedules

Additions to `dsl/types.go`:

```go
type WorkflowSpec struct {
    // ...existing fields...
    Schedules []*ScheduleSpec `json:"schedules,omitempty"`
}

type ScheduleSpec struct {
    Name         string         `json:"name"`
    CronExpr     string         `json:"cron_expr"`
    InputPayload map[string]any `json:"input_payload,omitempty"`
    Active       *bool          `json:"active,omitempty"` // default true; a spec can ship a canary schedule intentionally disabled
}
```

`dsl.Validate(spec)` grows a `validateSchedules(spec)` branch:
- Names are non-empty and unique within the spec.
- Each `CronExpr` parses via `ParseCron`.
- Stays infrastructure-free.

## Lifecycle binding

Schedules follow workflow state. No schedule-level controls.

### `CreateWorkflow`

Business layer (`apps/default/service/business/workflow.go`) materialises each `Spec.Schedules[i]` into a `schedule_definitions` row inside the same transaction that persists the workflow row:

- `Name = spec.Name`
- `CronExpr = spec.CronExpr`
- `WorkflowName = workflow.Name`
- `WorkflowVersion = workflow.Version`
- `InputPayload = JSON(spec.InputPayload)`
- `Active = false` (workflows are created in DRAFT)
- `NextFireAt = nil` (only computed when workflow activates)
- `jitter_seconds = 0` (only computed at activation)
- `tenant_id`/`partition_id` copied from claims

Rolled back atomically if the workflow create fails.

### `ActivateWorkflow`

When a workflow transitions to ACTIVE:

1. For every schedule in the newly-active (workflow_name, version): set `Active = true`, compute initial `NextFireAt = cronSched.Next(now) + jitter`, set `jitter_seconds`.
2. For every schedule of any **other** version of the same `workflow_name`: set `Active = false`. (Previous version stops firing; row kept for audit.)
3. Honour `Active=false` from the DSL: if the operator shipped a schedule explicitly disabled, it stays disabled even in an ACTIVE workflow.

All in one tx.

### Stopping a schedule

There is no `ArchiveWorkflow` or `DeleteWorkflow` RPC in v1 (only `Create`, `Get`, `List`, `Activate` — see `proto/workflow/v1/workflow.proto`). The supported way to stop a schedule is:

1. `CreateWorkflow` with a new version of the same `workflow_name` that omits or disables the schedule.
2. `ActivateWorkflow` on the new version → the new version's schedules become active; every previous version's schedules with matching `workflow_name` flip to `Active=false` in the same tx (step 2 of the Activate lifecycle above).

Emergency stop is a direct SQL `UPDATE schedule_definitions SET active = false WHERE id = …;`. Operator escape hatch, not an API.

When an `ArchiveWorkflow`/`DeleteWorkflow` RPC is added later, it will reuse the same `SetActiveByWorkflow` repo method to flip rows off in the archive transaction.

### `GetWorkflow` response

Grows a `schedules []ScheduleDefinition` field — the persisted rows filtered by (workflow.Name, workflow.Version). No new RPC; the existing `GetWorkflowResponse` message is extended with a repeated field.

`ListWorkflows` is **not** extended (keeps list scans cheap). Full detail is a `GetWorkflow` away.

## Repository surface

```go
type ScheduleRepository interface {
    // Used by CreateWorkflow materialisation.
    Create(ctx context.Context, schedule *models.ScheduleDefinition) error

    // Used by GetWorkflow to populate the schedules[] field.
    ListByWorkflow(ctx context.Context, workflowName string, workflowVersion int) ([]*models.ScheduleDefinition, error)

    // Used by ActivateWorkflow / ArchiveWorkflow lifecycle hooks.
    SetActiveByWorkflow(ctx context.Context, tx *gorm.DB, workflowName string, workflowVersion int, active bool, activatedAt time.Time) error

    // Used by CronScheduler.
    ClaimAndFireBatch(
        ctx context.Context,
        now time.Time,
        limit int,
        fireFn func(ctx context.Context, tx *gorm.DB, sched *models.ScheduleDefinition) (nextFire *time.Time, jitterSeconds int, err error),
    ) (int, error)
}
```

`FindDue` and `UpdateFireTimes` removed (no callers).

## Testing

1. **Unit** (`dsl/schedule_test.go`): `ParseCron` round-trip, reject 6-field/seconds/descriptors, `Next` monotonically increases.
2. **Unit** (`dsl/validator_test.go`): validates schedule blocks (non-empty names, unique, parseable cron).
3. **Repository** (`apps/default/service/repository/schedule_test.go`):
   - `ClaimAndFireBatch` is exactly-once under concurrent claimants: spawn 10 goroutines each calling the method on an N-row dataset, sum the fired counts, assert = N.
   - Missed-fire policy: seed a row with `next_fire_at = now - 1h`, fire once, observe `last_fired_at ≈ now` and `next_fire_at ≈ next cron slot from now`, not `now - 55m`.
   - Jitter determinism: two calls on the same row produce the same `jitter_seconds`.
   - Partial index is used: `EXPLAIN` the scan, assert `idx_schedule_due` is in the plan.
4. **Business** (`apps/default/service/business/workflow_test.go`, extended):
   - `CreateWorkflow` with `schedules[]` in the DSL produces matching rows (`Active=false`, `NextFireAt=nil`).
   - `ActivateWorkflow` flips the newest version's rows to `Active=true`, sets `NextFireAt`, flips the previous version's rows to `Active=false`.
   - Rollback on workflow create failure leaves no orphan schedule rows.
5. **Scheduler** (`apps/default/service/schedulers/cron_test.go`, existing file): updated to cron expressions, asserts the transactional path.
6. **Handler** (`apps/default/service/handlers/workflow_test.go`, extended): `GetWorkflow` response includes `schedules[]`.
7. **Contract**: end-to-end — `CreateWorkflow` with a `*/1 * * * *` schedule, activate, observe `schedule.fired` events in `event_log` within 90 s.

All DB-touching tests use `frametests.FrameBaseTestSuite` + testcontainers; no mocks for the database.

## Verification (post-deploy)

```
# Create a workflow carrying a schedule
curl -sS -H "Authorization: Bearer $TOKEN" \
  https://api.stawi.dev/trustage/workflow.v1.WorkflowService/CreateWorkflow \
  -X POST -d '{
    "dsl": {
      "version":"v1","name":"noop","steps":[{"id":"s","type":"delay","delay":{"duration":"1s"}}],
      "schedules":[{"name":"smoke","cron_expr":"*/1 * * * *"}]
    }
  }'
# → 200, schedules[] in response (active=false)

# Activate
curl ... /WorkflowService/ActivateWorkflow -d '{"id":"<from above>"}'
# → 200, next_fire_at populated

# After ≤ 90 s, event_log accrual
kubectl -n trustage exec deploy/trustage -- psql ... \
  -c "SELECT count(*) FROM event_log WHERE event_type='schedule.fired';"
# → > 0
```

## Rollout

Single release, direct-to-main, same workflow as the reliability work. Tag `v0.3.34` once all plan tasks complete and tests are green. Flux image-automation picks up the new tag; HelmRelease upgrade is non-disruptive because the colony chart's pre-install migration Job (from the reliability work) runs the new AutoMigrate with the partial index, then serving pods roll.

No `deployments` repo change — no new permissions, no new RPC surface, no new env vars.

## Risks

1. **Breaking change to `CronExpr` semantics.** Plan's first step verifies production `schedule_definitions` is empty or uses cron syntax before migrating; aborts if duration-format values exist.
2. **`jitter_seconds` drift on workflow re-activation.** When a workflow is re-activated after archive, we recompute jitter. Accepted: re-activation is an operator action and "fires at a slightly different second" is fine.
3. **`robfig/cron/v3` parsing cost.** Nanosecond-scale per fire; not a concern.
4. **Partial-index write amplification.** Every fire touches the index. Negligible on CNPG hardware.
5. **ActivateWorkflow semantics around multi-version schedules.** When version N+1 activates, version N's schedules flip off in the same tx. If a scheduler pod was mid-fire of a version-N schedule, the row lock ensures that fire completes; the `Active=false` write happens after it. No race, no lost fire.

## References

- Current scheduler: `apps/default/service/schedulers/cron.go`
- Current model / repository: `apps/default/service/models/schedule.go`, `apps/default/service/repository/schedule.go`
- Workflow business / materialisation target: `apps/default/service/business/workflow.go`
- Workflow proto (extending `GetWorkflowResponse`): `proto/workflow/v1/workflow.proto`
- Migration + index pattern: `apps/default/service/repository/migrate.go`
- Cron library: `github.com/robfig/cron/v3`
