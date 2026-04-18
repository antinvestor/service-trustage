# Scheduler v1.2 — full observability at scale

**Date:** 2026-04-18
**Status:** approved
**Scope:** additive observability for the scheduler / workflow lifecycle. No new user features, no schema changes, no interface reshape. Direct-to-main. Target release: `v0.3.37`.

## Why

Scheduler v1.1 shipped fire-rate, sweep-duration, fire-failure, and invalid-cron metrics. At millions-of-schedules scale the dominant failure mode is **slow degradation** — backlog creeping up, one tenant monopolising fires, lifecycle ops failing silently — none of which the v1.1 metrics catch. This release closes those gaps and ships alert rules so failures page, not just accumulate.

## Goals

- **Backlog visibility.** Operators can answer "how late is the scheduler?" from a single gauge.
- **Per-tenant attribution** on the fire counter (noisy-neighbor detection, per-tenant SLOs).
- **Lifecycle op counters.** Create/Activate/Archive failures are metered, not just logged.
- **Pool utilisation** visible if Frame exposes it cheaply (skip otherwise).
- **Alert rules** committed to the `deployments` repo, not documented prose.

## Non-goals

- Grafana dashboards-as-code — the repo has no existing dashboard framework; adding one is out of scope. Prometheus + Alertmanager is the primary observability path.
- No new feature surface. No DSL change, no proto change, no new RPC.
- No changes to the fire-path correctness model.

## Documentation deliverables (as important as the metrics)

Each shipped metric is useless without docs explaining what it means and what to do when it misbehaves.

- **`docs/scheduler-observability.md`** — developer-facing. Every metric listed with: semantics, label cardinality, PromQL examples, the source code path that emits it, and the tuning knob that shifts it.
- **`docs/runbook-scheduler.md`** — operator-facing. One section per alert in the PrometheusRule. Each section: what the alert means, likely causes, first three diagnostic commands, remediation steps, escalation path.
- **Inline documentation** — every new metric method and repo method has a Go docstring covering the invariant and cost. Alert rules in YAML carry `annotations.runbook` links to anchors in `runbook-scheduler.md`.

## Metrics to add

| Metric | Type | Labels | Source | Purpose |
|---|---|---|---|---|
| `scheduler_cron_backlog_seconds` | Gauge | none | sampled per sweep via `SELECT EXTRACT(EPOCH FROM (now() - MIN(next_fire_at)))...` | how late the scheduler is running |
| `scheduler_cron_fired_total` (revised) | Counter | `result={ok\|fail}`, `tenant_id` | aggregated from the sweep's per-tenant breakdown | per-tenant throughput + noisy-neighbor detection |
| `workflow_lifecycle_total` | Counter | `op={create\|activate\|archive}`, `result={ok\|fail}`, `tenant_id` | business layer, once per call | lifecycle op success/failure visible |
| `scheduler_pool_in_use_connections` / `scheduler_pool_max_connections` | Gauges | none | pgxpool.Stat() if exposed via Frame; else skip | pool saturation |

Existing metrics from v1.1 (`scheduler_cron_sweep_duration_seconds`, `scheduler_cron_invalid_cron_total`) stay as-is.

## Interface changes

One narrow repository interface change to support per-tenant attribution and backlog sampling:

```go
type ScheduleRepository interface {
    // … existing methods unchanged …

    // Backward-compatible signature change: ClaimAndFireBatch grows a second
    // return carrying per-tenant fire counts. fired == sum of firedByTenant values.
    // Callers that only care about total fired can ignore firedByTenant.
    ClaimAndFireBatch(
        ctx context.Context,
        plan SchedulePlanFn,
        now time.Time,
        limit int,
    ) (fired int, firedByTenant map[string]int, err error)

    // NEW — returns the age of the oldest due schedule in seconds, or 0 if
    // no rows are due. Single-table SELECT, tenancy-unscoped (backlog is a
    // global/per-pod signal).
    BacklogSeconds(ctx context.Context) (float64, error)
}
```

Business-layer callers of `ClaimAndFireBatch` don't exist (only the scheduler calls it), so the breakage surface is tiny. The scheduler iterates `firedByTenant` after the sweep and emits per-tenant counter increments.

No other interface changes. No `-Tx` methods added, no `Transact`, no `Pool()` removal. Matches the v1.1 narrow-surface principle.

## Implementation shape

### Repository (`apps/default/service/repository/schedule.go`)

- `ClaimAndFireBatch` tracks tenant_id per fired row (already has it from `sched.TenantID`) and returns the aggregated breakdown. No new SQL; just bookkeeping.
- `BacklogSeconds` runs one SELECT on the dedicated scheduler pool. Result is either a non-nil `*float64` (backlog in seconds) or nil (no due rows); the method returns `0.0` for the nil case so it's safe to Gauge-set directly.

### Telemetry (`pkg/telemetry/metrics.go`)

- Add instrument fields for the four new metrics.
- `scheduler_cron_fired_total` gains a `tenant_id` attribute; existing `result` attribute retained.
- Helper: `RecordSchedulerCronSweep(ctx, firedByTenant map[string]int, dur time.Duration, ok bool)` — iterates the map and increments the counter per tenant, plus records the histogram once. nil-tolerant.
- Helper: `ObserveSchedulerBacklog(ctx, seconds float64)`.
- Helper: `RecordWorkflowLifecycle(ctx, op string, tenantID string, ok bool)`.
- Pool gauge wiring only if Frame exposes `pgxpool.Stat()` via the `pool.Pool` interface or similar; if reaching pgxpool requires adding new public surface to Frame, skip for this release.

### Scheduler (`apps/default/service/schedulers/cron.go`)

- In `RunOnce`: sample `scheduleRepo.BacklogSeconds(ctx)` after each sweep, emit via `metrics.ObserveSchedulerBacklog`. One extra DB round-trip per sweep — negligible at 1s interval.
- Unpack `firedByTenant` from `ClaimAndFireBatch`; call `metrics.RecordSchedulerCronSweep(ctx, firedByTenant, dur, ok)` which does the per-tenant increments.

### Business (`apps/default/service/business/workflow.go`)

- Each of `CreateWorkflow`, `ActivateWorkflow`, `ArchiveWorkflow` calls `metrics.RecordWorkflowLifecycle(ctx, "create"|"activate"|"archive", tenantID, err == nil)` on function exit (defer-pattern with a pointer to err).
- Tenant ID comes from `security.ClaimsFromContext(ctx)` — already available in business.
- Metrics injection: grow `workflowBusiness` struct with `metrics *telemetry.Metrics` field; nil-tolerant in tests.

## Alert rules (new, in `deployments` repo)

Commit a `PrometheusRule` at `manifests/namespaces/trustage/common/scheduler-alerts.yaml` (or wherever the existing convention puts monitoring resources — the implementer checks). Rules:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: trustage-scheduler
  namespace: trustage
spec:
  groups:
    - name: scheduler.cron
      interval: 30s
      rules:
        - alert: SchedulerCronSweepFailures
          expr: rate(scheduler_cron_fired_total{result="fail"}[5m]) > 0.01
          for: 5m
          labels:
            severity: page
          annotations:
            summary: "Scheduler sweep failures > 0.6/min for 5m"
            runbook: "Check trustage pod logs; likely DB or pool issue."

        - alert: SchedulerCronBacklogGrowing
          expr: scheduler_cron_backlog_seconds > 300
          for: 2m
          labels:
            severity: warn
          annotations:
            summary: "Scheduler backlog > 5 min for 2m (fires arriving faster than pods drain)"
            runbook: "Check scheduler_cron_sweep_duration_seconds; may need to scale pods or raise CRON_SCHEDULER_BATCH_SIZE."

        - alert: SchedulerCronSweepSlow
          expr: histogram_quantile(0.99, rate(scheduler_cron_sweep_duration_seconds_bucket[5m])) > 2
          for: 5m
          labels:
            severity: warn
          annotations:
            summary: "Scheduler sweep p99 > 2s for 5m"
            runbook: "Check DB CPU + scheduler pool saturation."

        - alert: SchedulerInvalidCronPersisting
          expr: rate(scheduler_cron_invalid_cron_total[15m]) > 0
          for: 15m
          labels:
            severity: info
          annotations:
            summary: "Schedules with invalid cron parking for 15m"
            runbook: "Identify schedule_ids, contact tenant operator to fix DSL."

        - alert: WorkflowLifecycleFailures
          expr: rate(workflow_lifecycle_total{result="fail"}[5m]) > 0.01
          for: 5m
          labels:
            severity: warn
          annotations:
            summary: "Workflow lifecycle op failures > 0.6/min for 5m"
            runbook: "Check business-layer logs for Create/Activate/Archive errors."
```

Thresholds are starting points — refine based on staging observation.

## Testing

- Unit: `TestRecordSchedulerCronSweep_PerTenantCounts` — pass a `map["tenant-A"]2, "tenant-B"1`, verify counter increments by tenant. Pure metric-API test.
- Unit: `TestObserveSchedulerBacklog_Gauge` — set gauge, read back via `metric.WithOnlyCurrent()` or the meter's test helper.
- Integration (`apps/default/service/repository/schedule_test.go`): `TestBacklogSeconds_ReturnsOldestDueLag` — seed rows with varied `next_fire_at`, assert `BacklogSeconds` returns the oldest.
- Integration (`apps/default/service/business/workflow_integration_test.go`): `TestLifecycleCounters_RecordOnSuccessAndFailure` — stub metrics, run Create/Activate/Archive, assert increments.
- Integration (`apps/default/service/repository/schedule_test.go`): extend `TestClaimAndFireBatch_ExactlyOnceConcurrent` to verify `firedByTenant` totals equal the sum of parallel-worker `fired` returns.

## Rollout

- Same release pattern as v1.1: tag `v0.3.37`, Flux picks up, no migration (additive metrics only, no schema change).
- **No pgBouncer bounce needed** — no schema/index changes in this release. Lower risk than v1.1.
- PrometheusRule lands via `deployments` repo commit, reconciled by the existing `trustage-setup` Kustomization.

## Risks

1. **Per-tenant counter cardinality.** At ~hundreds of tenants, `scheduler_cron_fired_total{tenant_id="..."}` produces O(hundreds) series. At ~millions of tenants (theoretical upper bound of the platform), this becomes problematic. Mitigation: start with the label, monitor cardinality in Prometheus; if it ever climbs past 10k active series, strip the label and rely on top-N aggregation downstream. Not a launch blocker.
2. **`BacklogSeconds` query cost.** Runs `MIN(next_fire_at)` over the partial index (`idx_sd_due`) — index-only lookup, O(log n). Cheap. One extra round-trip per sweep at 1s interval is ~0.5% overhead on a saturated pod.
3. **Lifecycle counter breaks tests passing `nil` metrics.** All existing tests pass `nil` for metrics; helper must tolerate nil (same pattern as v1.1 `RecordSchedulerCronSweep`).
4. **Alert thresholds** are guesses. Expect to tune during the first week of production.

## Success criteria

- `scheduler_cron_backlog_seconds` series present in Prometheus, value ≈ 0 under normal load.
- `scheduler_cron_fired_total` split by `tenant_id` for any tenant that has active schedules.
- `workflow_lifecycle_total` increments observed after a single Create+Activate+Archive smoke test.
- PrometheusRule reconciled by Flux, alerts visible in Alertmanager/Grafana.
