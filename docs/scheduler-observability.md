# Scheduler Observability Reference

Developer-facing reference for every metric the cron scheduler and workflow business layer emit.
Assumes familiarity with Prometheus but not with the trustage codebase.

---

## Architecture: fire-path observability flow

```
CronScheduler.RunOnce                          (cron.go:84)
  → telemetry.StartSpan(TracerScheduler, SpanSchedulerCron)   ← span scheduler.cron.sweep
  → repo.ClaimAndFireBatch(planOne, now, batchSize)
      ↳ planOne(): validate cron expr + timezone, compute next_fire_at
          • invalid cron/tz → SchedulerCronInvalid.Add (+ park row)
  → metrics.RecordSchedulerCronSweep(firedByTenant, dur, ok)
      ↳ SchedulerCronSweepDuration.Record  (histogram, 1 obs per sweep)
      ↳ SchedulerCronFired.Add             (counter, 1 add per tenant)
  → sampleBacklog(ctx)
      ↳ repo.BacklogSeconds()              (1 lightweight DB query)
      ↳ metrics.ObserveSchedulerBacklog    (gauge set)
  → telemetry.EndSpan
```

### Span `scheduler.cron.sweep`

One span per `RunOnce` call, created with tracer `trustage.scheduler`.
Wraps the full sweep including the DB transaction and backlog sample.
Span constant: `telemetry.SpanSchedulerCron` (`pkg/telemetry/metrics.go:48`).
Useful for distributed-trace correlation when a sweep fires cross-tenant.

---

## Metrics

### `scheduler_cron_fired_total`

| Field | Value |
|---|---|
| **Type** | Counter |
| **Labels** | `result` (`ok` \| `fail`), `tenant_id` (string \| `""`) |
| **Instrument** | `Metrics.SchedulerCronFired` |
| **Source** | `pkg/telemetry/metrics.go:154` (`RecordSchedulerCronSweep`) called by `apps/default/service/schedulers/cron.go:98,103` |

**Semantics.** Counts schedule rows successfully fired (i.e. events written to `event_log` and `next_fire_at` advanced) per tenant per sweep. On failure (`result=fail`) a single add of 0 is emitted with `tenant_id=""` so the fail timeseries always exists and rate queries return 0 (not "no data").

**Labels detail.**
- `result=ok` — sweep completed; value = number of rows fired for that tenant this sweep.
- `result=fail` — `ClaimAndFireBatch` returned an error; value is always 0.
- `tenant_id` — the tenant UUID of the fired schedules. Empty string on failure or on an idle sweep (no rows due).

**Sampling cadence.** Once per sweep tick. Default tick: 1 s (`CRON_SCHEDULER_INTERVAL_SECONDS`).

**PromQL examples.**

```promql
# Rows fired per second, all tenants
rate(scheduler_cron_fired_total{result="ok"}[1m])

# Sweep failure rate (1 = 100% failing)
rate(scheduler_cron_fired_total{result="fail"}[5m])
  /
rate(scheduler_cron_fired_total[5m])
```

**Tuning knobs.**

| Env var | Default | Effect |
|---|---|---|
| `CRON_SCHEDULER_INTERVAL_SECONDS` | `1` | Sweep frequency; lower → more metric points, higher DB load |
| `CRON_SCHEDULER_BATCH_SIZE` | `500` | Rows claimed per sweep; larger batch → fewer sweeps needed at high volume |
| `SCHEDULER_POOL_MAX_CONNS` | `10` | DB connection ceiling for the scheduler pool |

**Cardinality warning.** The `tenant_id` label creates one timeseries per active-tenant per result. At 10 000 tenants with active schedules this adds ~20 000 series. If Prometheus scrape intervals are short (< 15 s) and tenants number in the hundreds of thousands, consider dropping or hashing `tenant_id` via a relabelling rule:

```yaml
metric_relabel_configs:
  - source_labels: [__name__, tenant_id]
    regex: 'scheduler_cron_fired_total;.+'
    target_label: tenant_id
    replacement: "aggregated"
```

---

### `scheduler_cron_sweep_duration_seconds`

| Field | Value |
|---|---|
| **Type** | Histogram |
| **Labels** | `result` (`ok` \| `fail`) |
| **Instrument** | `Metrics.SchedulerCronSweepDuration` |
| **Source** | `pkg/telemetry/metrics.go:165` (`RecordSchedulerCronSweep`) called by `apps/default/service/schedulers/cron.go:98,103` |

**Semantics.** Records end-to-end wall-clock duration of one `ClaimAndFireBatch` call (DB transaction + row planning). Does NOT include the subsequent `BacklogSeconds` call. One observation per sweep regardless of how many tenants or rows were fired.

**Sampling cadence.** Once per sweep. Default: 1 s.

**PromQL examples.**

```promql
# p99 sweep duration over the last 5 minutes
histogram_quantile(0.99,
  rate(scheduler_cron_sweep_duration_seconds_bucket[5m])
)

# Mean sweep duration
rate(scheduler_cron_sweep_duration_seconds_sum[1m])
  /
rate(scheduler_cron_sweep_duration_seconds_count[1m])
```

**Tuning knobs.** Same as `scheduler_cron_fired_total`. Large `CRON_SCHEDULER_BATCH_SIZE` increases per-sweep duration. High `SCHEDULER_POOL_MAX_CONNS` reduces wait time for a connection under concurrent schedulers.

---

### `scheduler_cron_invalid_cron_total`

| Field | Value |
|---|---|
| **Type** | Counter |
| **Labels** | `tenant_id` (string) |
| **Instrument** | `Metrics.SchedulerCronInvalid` |
| **Source** | `apps/default/service/schedulers/cron.go:143,162` (`planOne`) |

**Semantics.** Incremented whenever `planOne` encounters a schedule row that cannot be parsed (invalid cron expression) **or** whose timezone string is rejected by `cronSched.NextInZone`. Either path parks the row (sets `next_fire_at = NULL`, emits no event) and increments this counter. A non-zero rate means persistent bad data in `schedule_definitions` — it does not self-heal.

**Sampling cadence.** Emitted inside `ClaimAndFireBatch` once per problematic row per sweep. The same bad row increments the counter on every sweep until the row is corrected or deleted.

**PromQL examples.**

```promql
# Rate of invalid-cron hits across all tenants
rate(scheduler_cron_invalid_cron_total[5m])

# Tenants with persisting bad cron rows
increase(scheduler_cron_invalid_cron_total[10m]) > 0
```

**Tuning knobs.** No tuning knob controls this counter. Fixing it requires correcting or deleting the offending `schedule_definitions` row. See runbook `#schedulerinvalidcronpersisting`.

**Cardinality.** One series per affected tenant. Typically low cardinality because invalid rows are usually caught at workflow create/activate time.

---

### `scheduler_cron_backlog_seconds`

| Field | Value |
|---|---|
| **Type** | Gauge |
| **Labels** | none |
| **Instrument** | `Metrics.SchedulerCronBacklog` |
| **Source** | `apps/default/service/schedulers/cron.go:125` (`sampleBacklog`) → `pkg/telemetry/metrics.go:191` (`ObserveSchedulerBacklog`) |

**Semantics.** Age in seconds of the oldest schedule row that is currently due but not yet fired (`next_fire_at <= now AND active = true`). A value of `0` means no schedules are currently overdue. A steadily rising value means the scheduler cannot drain the ready queue — either the batch size is too small, the interval is too long, or the DB is under pressure.

The underlying `repo.BacklogSeconds()` runs one lightweight `SELECT EXTRACT(EPOCH FROM (now() - MIN(next_fire_at)))` query against `schedule_definitions`. It executes even on a failed sweep so operators can see lag growing independently of sweep health.

**Sampling cadence.** Once per sweep (after `RecordSchedulerCronSweep`). One extra DB round-trip per sweep cycle.

**PromQL examples.**

```promql
# Current backlog
scheduler_cron_backlog_seconds

# Alert: backlog growing consistently (example threshold 60 s)
scheduler_cron_backlog_seconds > 60
```

**Tuning knobs.**

| Env var | Default | Effect |
|---|---|---|
| `CRON_SCHEDULER_BATCH_SIZE` | `500` | Increase to drain more rows per sweep |
| `CRON_SCHEDULER_INTERVAL_SECONDS` | `1` | Decrease to sweep more often |
| `SCHEDULER_POOL_MAX_CONNS` | `10` | More connections → less DB queue wait |

**Emergency dial-down.** Set `CRON_SCHEDULER_BATCH_SIZE=1` on the Deployment env to revert to per-row semantics while keeping the running code, without redeploy.

---

### `workflow_lifecycle_total`

| Field | Value |
|---|---|
| **Type** | Counter |
| **Labels** | `op` (`create` \| `activate` \| `archive`), `result` (`ok` \| `fail`), `tenant_id` (string) |
| **Instrument** | `Metrics.WorkflowLifecycleTotal` |
| **Source** | `apps/default/service/business/workflow.go:90,361,428` — deferred at end of `CreateWorkflow`, `ActivateWorkflow`, `ArchiveWorkflow` |

**Semantics.** Counts workflow lifecycle operations by operation type and outcome. Emitted via a `defer` so it fires on every return path including panics recovered by the framework. A rising `result=fail` rate indicates persistent errors in the persistence or validation layer.

**Labels detail.**
- `op=create` — `CreateWorkflow` returned (DSL parse, schema registration, DB persist).
- `op=activate` — `ActivateWorkflow` returned (status transition + schedule activation).
- `op=archive` — `ArchiveWorkflow` returned (schedule deactivation + status update).
- `result=ok` / `result=fail` — whether `err == nil` at function exit.
- `tenant_id` — extracted from OIDC claims via `security.ClaimsFromContext(ctx)`.

**Sampling cadence.** Event-driven; emitted once per API call, not on a fixed interval.

**PromQL examples.**

```promql
# Workflow creation rate per second
rate(workflow_lifecycle_total{op="create", result="ok"}[5m])

# Failure ratio across all lifecycle operations
rate(workflow_lifecycle_total{result="fail"}[5m])
  /
rate(workflow_lifecycle_total[5m])
```

**Tuning knobs.** No direct knob. Failure rate reduction requires code or infrastructure changes. Cardinality: 3 ops × 2 results × N tenants. Same tenant_id explosion risk as `scheduler_cron_fired_total` at large tenant counts.

---

## Cardinality summary

| Metric | Max series (N tenants, normal ops) |
|---|---|
| `scheduler_cron_fired_total` | `2 × N + 2` (ok+fail per tenant + idle + failure tombstone) |
| `scheduler_cron_sweep_duration_seconds` | `2` (ok + fail) |
| `scheduler_cron_invalid_cron_total` | `K` (affected tenants only) |
| `scheduler_cron_backlog_seconds` | `1` |
| `workflow_lifecycle_total` | `6 × N` (3 ops × 2 results × N tenants) |

At N > 50 000 tenants with regular activity, apply relabelling or recording rules to aggregate `tenant_id` before federation.
