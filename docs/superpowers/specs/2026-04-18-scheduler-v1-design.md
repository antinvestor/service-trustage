# Scheduler v1: declarative, cron-based, horizontally scalable

**Date:** 2026-04-18
**Status:** approved
**Scope:** `service-trustage` — all code changes under `apps/default/`, `dsl/`, and `proto/`. No `deployments` repo changes. Commits go direct to `main`.

## Problem

The scheduler subsystem ships with several gaps that become acute at scale:

1. **No declarative writer.** `WorkflowSpec` (the JSON accepted by `CreateWorkflow`) has no `schedule` field. The only code that writes rows to `schedule_definitions` is the test helper `scheduleRepo.Create()`. There is no `ScheduleService` RPC. Operators cannot define schedules without writing Go.

2. **`CronExpr` is not cron.** Despite the column name, `apps/default/service/schedulers/cron.go:147` calls `dsl.ParseDuration`, so values must be Go-style durations (`"1h"`, `"7d"`). Real cron expressions (`"0 */5 * * *"`) error out silently by setting `next_fire_at = nil`. Operators lose all typical cron capabilities (specific minute, day-of-week, hour ranges).

3. **`fireSchedule` is not transactional.** At `cron.go:96-142`, `eventRepo.Create` and `scheduleRepo.UpdateFireTimes` run as two independent auto-commit statements. `FindDue`'s `FOR UPDATE SKIP LOCKED` lock is released the moment the SELECT returns (no explicit tx wraps the read and the writes). In a multi-pod deployment two pods polling within the same millisecond **can fire the same schedule twice** — the lock window does not cover the writes. At one pod the bug is latent; scaling out makes it reproducible.

4. **No thundering-herd protection.** `computeNextFire` is `now + duration`, so schedules created near the same instant drift together and fire at the same second forever. Cron's `0 */5 * * *` puts every schedule on `:00`, `:05`, `:10`. Tens of thousands of schedules firing in the same second is a write spike neither the DB nor the outbox benefit from.

5. **Scan cost grows with table size, not due rows.** The current `FindDue` query (`active AND deleted_at IS NULL AND next_fire_at IS NOT NULL AND next_fire_at <= now()`) has no partial index. At millions of rows PostgreSQL still has to walk or bitmap-scan to find due ones every 30 s.

6. **No missed-fire policy.** If a pod is down for two hours and comes back up, every affected schedule sees `next_fire_at` two hours in the past; the scheduler will fire them back-to-back in a tight loop trying to catch up, which is almost never what anyone wanted.

## Goals

- Operators define schedules declaratively via RPC and as part of workflow definitions in JSON/YAML.
- Real cron expressions (5-field, UTC evaluation).
- Exactly-once fire semantics across any number of scheduler-running pods.
- Scan cost independent of table size up to 10 M+ schedules.
- Horizontal scaling needs **zero configuration** — any pod added to the deployment starts contributing immediately. No shard assignment, no leader election, no partition map.
- Thundering-herd flattening via deterministic per-schedule jitter.
- Sane behaviour after pod downtime: fire once, skip-forward to the next slot.

## Non-goals (deferred)

- LISTEN/NOTIFY for sub-second wake. Frame exposes only GORM; driving `pgx` LISTEN directly requires a dedicated connection outside the pool and a reconnection loop. Operational cost > benefit at v1. 30 s poll + partial index is sufficient for workflow schedules.
- Per-schedule timezone. UTC only in v1.
- Per-schedule missed-fire-threshold override. One global policy in v1.
- Schedule fire history / audit log. The `event_log` entry per fire is sufficient for now.
- Web UI for schedule management.
- Dropping the schedule_definitions row on workflow archive/delete. Kept as-is; lifecycle remains explicit via RPC.

## Architecture

Nothing about the topology changes. Every `trustage` pod already runs `CronScheduler.Start` (`apps/default/cmd/main.go:208`), and `FindDue` already uses `FOR UPDATE SKIP LOCKED`. The "zero-config horizontal scale" property is **already present**; this spec hardens the row contract around it (transactional fire + partial index) and adds the declarative surface. Adding pods requires no config change after this work, just as it requires none now.

```
┌──────────────────────────────────────────────────────────────────┐
│ trustage pods (N replicas, HPA 1-12)                              │
│                                                                   │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐             │
│  │ ScheduleSvc  │  │ CronScheduler│  │ WorkflowSvc  │             │
│  │ (RPC writer) │  │ (30s sweep)  │  │ (embeds      │             │
│  │              │  │              │  │  ScheduleSpec)│            │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘             │
│         │  writes         │  scans+fires    │  materialises       │
│         ▼                 ▼                 ▼                     │
└─────────┼─────────────────┼─────────────────┼─────────────────────┘
          │                 │                 │
          ▼                 ▼                 ▼
┌──────────────────────────────────────────────────────────────────┐
│ PostgreSQL (CNPG hub)                                            │
│                                                                   │
│   schedule_definitions                event_log (outbox)          │
│     id, tenant_id, cron_expr,         (unchanged)                 │
│     next_fire_at, active, ...                                     │
│     partial idx on (next_fire_at)                                 │
│     WHERE active AND deleted_at IS NULL                          │
│                                                                   │
│   Fires: FOR UPDATE SKIP LOCKED → insert event_log → UPDATE       │
│   next_fire_at + last_fired_at, all in ONE tx per schedule.       │
└──────────────────────────────────────────────────────────────────┘
```

## Data model

### Schema changes

Add two columns to `schedule_definitions`. Both nullable-default-safe so `AutoMigrate` is a no-op on existing empty tables.

| Column | Type | Default | Purpose |
|---|---|---|---|
| `timezone` | `varchar(64) NOT NULL` | `'UTC'` | Reserved for v2; present so v1 migrations don't need a schema change. Evaluation ignores it. |
| `jitter_seconds` | `integer NOT NULL` | `0` | Amount of jitter baked into `next_fire_at` (materialised, not computed at scan). Purely observability. |

Rename is *not* performed. `cron_expr` keeps its name but the semantics change: it now stores a real 5-field cron expression, not a duration. The only existing writer of `cron_expr` is a test helper (per the user's initial message), so there is no production data to migrate. Tests are updated in the same commit that drops `computeNextFire`.

### Partial index

```
CREATE INDEX IF NOT EXISTS idx_schedule_due
    ON schedule_definitions (next_fire_at ASC)
    WHERE active = true
      AND deleted_at IS NULL
      AND next_fire_at IS NOT NULL;
```

GORM's `Migrator` supports this via the existing `migrationIndexes()` pattern (`apps/default/service/repository/migrate.go`). The index is the single most important scaling change in the spec: it makes every pod's 30 s scan O(rows due) regardless of total table size.

## Cron parsing

Add dependency: `github.com/robfig/cron/v3`.

Use `cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)` — strict 5-field, no seconds, no descriptors. Rejecting seconds keeps the DSL honest (30 s poll floor) and avoids a whole class of surprise.

Introduce `dsl/schedule.go` with:

```go
type CronSchedule struct {
    Expr string // canonical 5-field cron expression, validated at parse time
}

func ParseCron(expr string) (CronSchedule, error) { /* robfig parser */ }

// Next returns the first fire time strictly after `from` for this schedule.
func (s CronSchedule) Next(from time.Time) time.Time { /* delegates to parsed schedule */ }
```

`dsl/schedule.go` stays infrastructure-free (per CLAUDE.md: "`dsl/` package must NOT import Frame, NATS, or any infrastructure").

### Jitter

```go
// Deterministic per-schedule offset, stable across restarts.
// h = fnv64a(scheduleID)
// period = cronSchedule.Next(base).Sub(base)
// jitter = h mod min(period/10, 30s)
// returned next_fire_at = cronSchedule.Next(base) + jitter
```

Stored `jitter_seconds = jitter / time.Second` for visibility. The `+ jitter` is baked into `next_fire_at` so no cost on the hot path.

Rationale for `min(period/10, 30s)`: on a 5-minute cron, spread is up to 30 s; on an hourly cron, 6 min; on a daily cron, capped at 30 s anyway because the next poll interval is 30 s and larger jitter doesn't help. Keeps the actual fire within one poll window of the nominal cron time.

### Missed-fire policy

If `next_fire_at` is more than `scheduleStaleThreshold = 5 * time.Minute` in the past at scan time:

1. Fire **once** at `now` (emit one event_log row with `fired_at = now`).
2. Set `last_fired_at = now`.
3. Set `next_fire_at = cronSchedule.Next(now) + jitter`.

Never emit a catch-up burst. This is the behaviour operators always want for workflow triggers ("pick up from where we are, don't replay the backlog").

Threshold lives as a scheduler constant in v1. Per-schedule override deferred to v2.

## Transactional fire

The repository grows one method:

```go
// ClaimAndFireBatch scans for due schedules under one tx, invokes fireFn
// for each, and commits atomically. fireFn receives the schedule and a DB
// handle bound to the same tx so event_log inserts and next_fire_at updates
// participate in the same transaction as the row lock.
//
// Returns the number of schedules fired (fireFn did not error).
ClaimAndFireBatch(
    ctx context.Context,
    now time.Time,
    limit int,
    fireFn func(ctx context.Context, tx *gorm.DB, sched *models.ScheduleDefinition) (nextFire *time.Time, err error),
) (int, error)
```

Implementation:

1. `db.Transaction(func(tx *gorm.DB) error { … })` (GORM-managed tx).
2. Inside: `SELECT ... FOR UPDATE SKIP LOCKED LIMIT N` against the partial index.
3. For each row, call `fireFn(ctx, tx, sched)`. The caller inserts into `event_log` using the `tx` handle and returns the computed `nextFire`.
4. After `fireFn` returns, the repo issues `UPDATE schedule_definitions SET last_fired_at=now, next_fire_at=<nextFire>, jitter_seconds=<jitter> WHERE id=<id>` on the same `tx`.
5. On `fireFn` error for any schedule: log, skip that one, continue batch. The tx still commits the successful ones.

The tx boundary is what restores exactly-once. The `FOR UPDATE SKIP LOCKED` lock is held across the full fire, so no second pod can see the same row until commit.

`CronScheduler.fireSchedule` is rewritten to be the `fireFn` callback. It no longer calls `UpdateFireTimes` directly — the repo does, inside the tx.

`ScheduleRepository.FindDue` and `ScheduleRepository.UpdateFireTimes` are **removed** (no other callers). Replaced fully by `ClaimAndFireBatch` + the new writer methods below.

### New writer methods

```go
GetByID(ctx, id)                   // + tenancy filter via BaseRepository
ListByTenant(ctx, activeOnly, limit, offset)
Update(ctx, sched)                 // recomputes next_fire_at if cron_expr changed
SetActive(ctx, id, active)         // pause/resume; recomputes next_fire_at if activating
SoftDelete(ctx, id)
```

Standard BaseRepository shape, mirrors `WorkflowDefinitionRepository`.

## Proto: ScheduleService

New file: `proto/schedule/v1/schedule.proto`.

```proto
syntax = "proto3";
package schedule.v1;

import "common/v1/common.proto";
import "common/v1/permissions.proto";
import "gnostic/openapi/v3/annotations.proto";
import "google/protobuf/struct.proto";
import "google/protobuf/timestamp.proto";

option go_package = "github.com/antinvestor/service-trustage/gen/go/schedule/v1;schedulev1";

service ScheduleService {
  option (common.v1.service_permissions) = {
    namespace: "service_trustage"
    permissions: ["schedule_view", "schedule_manage"]
  };

  rpc CreateSchedule(CreateScheduleRequest) returns (CreateScheduleResponse) {
    option (common.v1.method_permissions) = { permissions: ["schedule_manage"] };
  }
  rpc GetSchedule(GetScheduleRequest) returns (GetScheduleResponse) {
    option (common.v1.method_permissions) = { permissions: ["schedule_view"] };
  }
  rpc ListSchedules(ListSchedulesRequest) returns (ListSchedulesResponse) {
    option (common.v1.method_permissions) = { permissions: ["schedule_view"] };
  }
  rpc UpdateSchedule(UpdateScheduleRequest) returns (UpdateScheduleResponse) {
    option (common.v1.method_permissions) = { permissions: ["schedule_manage"] };
  }
  rpc PauseSchedule(PauseScheduleRequest) returns (PauseScheduleResponse) {
    option (common.v1.method_permissions) = { permissions: ["schedule_manage"] };
  }
  rpc ResumeSchedule(ResumeScheduleRequest) returns (ResumeScheduleResponse) {
    option (common.v1.method_permissions) = { permissions: ["schedule_manage"] };
  }
  rpc DeleteSchedule(DeleteScheduleRequest) returns (DeleteScheduleResponse) {
    option (common.v1.method_permissions) = { permissions: ["schedule_manage"] };
  }
}

message ScheduleDefinition {
  string id = 1;
  string name = 2;
  string cron_expr = 3;
  string workflow_name = 4;
  int32 workflow_version = 5;
  google.protobuf.Struct input_payload = 6;
  bool active = 7;
  google.protobuf.Timestamp next_fire_at = 8;
  google.protobuf.Timestamp last_fired_at = 9;
  int32 jitter_seconds = 10;
  google.protobuf.Timestamp created_at = 11;
  google.protobuf.Timestamp updated_at = 12;
}

message CreateScheduleRequest {
  string name = 1;
  string cron_expr = 2;
  string workflow_name = 3;
  int32 workflow_version = 4;
  google.protobuf.Struct input_payload = 5;
  bool active = 6;
}
message CreateScheduleResponse { ScheduleDefinition schedule = 1; }

message GetScheduleRequest { string id = 1; }
message GetScheduleResponse { ScheduleDefinition schedule = 1; }

message ListSchedulesRequest {
  string workflow_name = 1;
  bool active_only = 2;
  common.v1.SearchRequest search = 3;
}
message ListSchedulesResponse {
  repeated ScheduleDefinition items = 1;
  common.v1.PageCursor next_cursor = 2;
}

message UpdateScheduleRequest {
  string id = 1;
  string cron_expr = 2;
  google.protobuf.Struct input_payload = 3;
  string name = 4;
}
message UpdateScheduleResponse { ScheduleDefinition schedule = 1; }

message PauseScheduleRequest  { string id = 1; }
message PauseScheduleResponse { ScheduleDefinition schedule = 1; }
message ResumeScheduleRequest { string id = 1; }
message ResumeScheduleResponse{ ScheduleDefinition schedule = 1; }
message DeleteScheduleRequest { string id = 1; }
message DeleteScheduleResponse{}
```

Permissions `schedule_view` and `schedule_manage` are new for this namespace; registered at service startup alongside the existing four (per `apps/default/cmd/main.go:297` — the `frame.WithPermissionRegistration` block).

Buf generation runs via existing `make proto-gen`.

## DSL: WorkflowSpec.Schedule

Add to `dsl.WorkflowSpec`:

```go
type WorkflowSpec struct {
    // ...existing fields...
    Schedules []*ScheduleSpec `json:"schedules,omitempty"`
}

type ScheduleSpec struct {
    Name         string         `json:"name"`
    CronExpr     string         `json:"cron_expr"`
    InputPayload map[string]any `json:"input_payload,omitempty"`
    Active       *bool          `json:"active,omitempty"` // default true
}
```

`dsl.Validate(spec)` grows a `validateSchedules(spec)` branch:
- Names are non-empty and unique within the spec.
- Each `CronExpr` parses via `ParseCron`.
- No infrastructure imports (stays inside `dsl/`).

When `CreateWorkflow` receives a `WorkflowSpec` with schedules, the business layer (in `apps/default/service/business/workflow.go`) materialises them into `schedule_definitions` rows *after* the workflow definition row is persisted — same transaction, tenant_id/partition_id copied from the workflow's claims. If no schedules in the spec, no rows written.

`UpdateWorkflow` (if/when it exists as a write path) is **out of scope** — users manage schedule lifecycle via `ScheduleService` after the initial create. This keeps the "spec as seed, service as ongoing manager" contract clean.

## Scheduler changes

`CronScheduler.Start` stays unchanged structurally. `RunOnce` becomes a thin wrapper that calls the repo's `ClaimAndFireBatch` with an inline `fireFn`:

```go
func (s *CronScheduler) RunOnce(ctx context.Context) int {
    now := time.Now().UTC()
    n, err := s.scheduleRepo.ClaimAndFireBatch(ctx, now, cronSchedulerBatchSize,
        func(ctx context.Context, tx *gorm.DB, sched *models.ScheduleDefinition) (*time.Time, error) {
            // 1. Build event_log row (same as today's fireSchedule payload shape).
            // 2. Insert into event_log via tx.Create(&eventLog).
            // 3. Parse sched.CronExpr, compute nextFire:
            //    - normal:       cronSched.Next(sched.NextFireAt) + jitter
            //    - missed-fire:  cronSched.Next(now) + jitter
            //      (triggered when now - sched.NextFireAt > staleThreshold)
            // 4. Return nextFire.
        })
    if err != nil { /* log */ }
    return n
}
```

`computeNextFire` and its caller are removed. Duration-style `CronExpr` values in tests are updated to real cron expressions.

## Testing

1. **Unit tests** (`dsl/schedule_test.go`): `ParseCron` round-trips, rejects 6-field/seconds/descriptors, returns monotonically increasing `Next()` times.
2. **Repository tests** (`apps/default/service/repository/schedule_test.go`):
   - `ClaimAndFireBatch` is exactly-once under concurrent claimants: spawn 10 goroutines each calling the method, sum the fired counts, assert = row count.
   - Missed-fire policy: insert a row with `next_fire_at = now - 1h`, fire once, observe `last_fired_at ≈ now` and `next_fire_at` ≈ the next cron slot, not `now - 55m`.
   - Jitter determinism: two calls on the same row produce the same `jitter_seconds`.
   - Partial index is used: `EXPLAIN` the scan query, assert the plan mentions `idx_schedule_due`.
3. **Scheduler test** (`apps/default/service/schedulers/cron_test.go`, existing file): update to cron-based expressions.
4. **Business tests** (`apps/default/service/business/workflow_test.go` or new): materialising a `WorkflowSpec.Schedules` block creates corresponding rows, rolled back atomically if workflow create fails.
5. **Handler tests** for each `ScheduleService` RPC: create, get, list (tenancy-scoped), pause, resume, delete, update.
6. **Contract test**: create a workflow with a 1-minute cron schedule via `CreateWorkflow`, observe a `schedule_definitions` row appear and `next_fire_at` set correctly.

All DB-touching tests use the existing `frametests.FrameBaseTestSuite` with testcontainers; no mocks for the database.

## Verification

Local:

```
make proto-gen          # regenerates schedule/v1 and workflow/v1 with Schedule field
make tests              # all suites, race detection
make lint               # strict
go build ./apps/default/cmd/...
```

Post-deploy (same verification shape as the reliability work):

```
curl -sS -o /dev/null -w "%{http_code}\n" \
  -H "Authorization: Bearer $TOKEN" \
  https://api.stawi.dev/trustage/schedule.v1.ScheduleService/ListSchedules \
  -X POST -d '{}'                                       # 200 with empty items[]

# Create a schedule that fires every minute
curl ... /schedule.v1.ScheduleService/CreateSchedule -d '{"name":"smoke","cron_expr":"*/1 * * * *","workflow_name":"noop","workflow_version":1,"active":true}'

# After ≤ 90 s, verify event_log accrual
kubectl -n trustage exec deploy/trustage -- psql ... \
  -c "SELECT count(*) FROM event_log WHERE event_type='schedule.fired' AND source LIKE 'schedule:%';"
```

## Rollout

Single release, direct-to-main, same workflow as the reliability work. Tag `v0.3.34` once the plan's final task completes and tests are green. Flux image-automation picks up the new tag within 15 minutes; HelmRelease upgrade is non-disruptive because `migration.enabled: true` (from the reliability work) runs the migration Job with the new index creation, then serving pods roll.

No `deployments` repo change unless the permission registration requires an OPL namespace update — TBD at implementation time (likely not, since new permissions reuse the existing `service_trustage` namespace).

## Risks

1. **Breaking change to `CronExpr` semantics.** Any existing row with a duration-format value would stop producing valid `next_fire_at` after the switch. The user confirmed only the test suite writes schedules today, so production impact is zero. This assumption is load-bearing — plan step 1 verifies it by querying production `schedule_definitions` for any duration-like values before migrating.

2. **`jitter_seconds` drift on `UpdateSchedule`.** When cron_expr changes, we recompute jitter from scratch. Existing in-flight schedules may jump. Acceptable: schedules that mutate expression are explicitly re-planned.

3. **`robfig/cron/v3` evaluation cost.** Parsing is O(expr-length); `Next()` is O(1) amortised for typical expressions. Millions of schedules each parsing once per fire is negligible (nanoseconds). Not a concern.

4. **Partial-index write amplification.** Every `UPDATE next_fire_at` touches the index. At 100+ fires/sec sustained this is still < 1 % of typical write throughput on CNPG's hardware profile. Flagged, not blocking.

5. **Permissions registration failure in migration job.** The chart's migration Job runs `cmd main` with `DO_MIGRATION=true`, which already registers the permission namespaces (`apps/default/cmd/main.go:297`). Two new permissions (`schedule_view`, `schedule_manage`) ship with the same release. Keto will auto-reconcile the OPL policy. If the rewrite fails, falls back to "permission denied until retry" — same failure mode the other trustage services already have.

## References

- Current scheduler: `apps/default/service/schedulers/cron.go`
- Current model / repository: `apps/default/service/models/schedule.go`, `apps/default/service/repository/schedule.go`
- Reference for RPC shape: `proto/workflow/v1/workflow.proto`, `apps/default/service/handlers/workflow.go`
- Reference for materialisation: `apps/default/service/business/workflow.go`
- Reference for migration+index pattern: `apps/default/service/repository/migrate.go`
- Cron library: `github.com/robfig/cron/v3`
