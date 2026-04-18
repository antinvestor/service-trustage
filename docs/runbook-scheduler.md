# Scheduler Runbook

Operator-facing runbook for trustage scheduler + workflow lifecycle alerts.
One section per alert; anchor links match `runbook_url` annotations in the PrometheusRule.

---

## Operational overview

**Deployment.** trustage runs as a Kubernetes Deployment with HPA (1–12 pods) in the `trustage` namespace. Every pod starts a `CronScheduler` goroutine on startup. `FOR UPDATE SKIP LOCKED` in `ClaimAndFireBatch` ensures that concurrent pods claim disjoint rows — no double-fire, no coordination required.

**One transaction per sweep.** Each `RunOnce` call is a single DB transaction: claim a batch of due `schedule_definitions` rows, write `event_log` entries, advance `next_fire_at`, commit. If the transaction rolls back, no events are emitted and the rows remain due for the next sweep.

**Pre-deploy checklist.** After any schema change to `schedule_definitions`:
1. Run migrations.
2. Bounce pgBouncer pods in the `datastore` namespace to clear stale prepared statements: `kubectl rollout restart deployment/pgbouncer -n datastore`.

**Key config knobs** (all are Deployment env vars):

| Env var | Default | Purpose |
|---|---|---|
| `CRON_SCHEDULER_BATCH_SIZE` | `500` | Rows claimed per sweep |
| `CRON_SCHEDULER_INTERVAL_SECONDS` | `1` | Sweep tick frequency |
| `SCHEDULER_POOL_MAX_CONNS` | `10` | Max DB conns for the scheduler pool |
| `OUTBOX_PUBLISH_CONCURRENCY` | `16` | Parallel NATS publishes per outbox sweep |

**Emergency dial-down.** To throttle the cron scheduler without a redeploy:
```bash
kubectl set env deployment/trustage CRON_SCHEDULER_BATCH_SIZE=1 -n trustage
```
This reverts to per-row semantics with the existing code. Backlog will grow; monitor `scheduler_cron_backlog_seconds`.

---

## Alerts

### SchedulerCronSweepFailures {#schedulercronsweepfailures}

**Severity.** Page.

#### What it means

The cron scheduler's `ClaimAndFireBatch` transaction is returning errors consistently. Schedules are not firing. Events are not being written to `event_log`. Affected tenants will miss cron triggers until this is resolved.

#### Likely causes

1. **PostgreSQL unavailable or rejecting connections** — network partition, pod restart, certificate rotation, connection pool exhausted.
2. **Schema migration partially applied** — a column rename or removal broke the `ClaimAndFireBatch` query.
3. **pgBouncer pool saturation** — too many prepared-statement mismatches after a deploy without bouncing pgBouncer.
4. **`SCHEDULER_POOL_MAX_CONNS` too low** — pool exhausted under high pod count × concurrent sweeps.

#### Diagnose

```bash
# 1. Current sweep failure rate
promql: rate(scheduler_cron_fired_total{result="fail"}[2m]) > 0

# 2. Tail scheduler logs for the error message
kubectl logs -n trustage -l app=trustage --since=5m \
  | grep "cron scheduler: sweep failed"

# 3. Check PostgreSQL connectivity from a trustage pod
kubectl exec -n trustage deploy/trustage -- \
  psql "$DATABASE_URL" -c "SELECT 1"
```

#### Remediate

- **DB connectivity issue**: restore connectivity; pods will auto-recover on the next sweep tick.
- **Schema migration**: complete the migration and bounce the Deployment (`kubectl rollout restart deployment/trustage -n trustage`).
- **pgBouncer saturation**: `kubectl rollout restart deployment/pgbouncer -n datastore` then verify the error clears within 30 s.
- **Pool exhaustion**: increase `SCHEDULER_POOL_MAX_CONNS` (`kubectl set env deployment/trustage SCHEDULER_POOL_MAX_CONNS=20 -n trustage`).

#### Escalate

Page `#trustage-oncall` immediately if the failure rate is non-zero for more than 2 minutes. DB connectivity issues that cannot be resolved within 5 minutes escalate to the infra on-call.

---

### SchedulerCronBacklogGrowing {#schedulercronbackloggrowing}

**Severity.** Warning.

#### What it means

`scheduler_cron_backlog_seconds` is rising — the oldest due schedule has been waiting longer than the alert threshold. Cron fires are delayed; tenants may notice missed or late workflow triggers.

#### Likely causes

1. **Throughput insufficient** — `CRON_SCHEDULER_BATCH_SIZE` too small relative to the number of active schedules firing at the same time (e.g. top-of-hour stampede).
2. **DB slow** — high query latency on `schedule_definitions` (missing index, table bloat, autovacuum lag).
3. **Pod count too low** — HPA has not yet scaled up to handle load.
4. **Sweep errors mixed in** — partial sweep failures leaving rows unclaimed; check `SchedulerCronSweepFailures` simultaneously.

#### Diagnose

```bash
# 1. Current backlog age
promql: scheduler_cron_backlog_seconds

# 2. Sweep duration p99 (is the DB slow?)
promql: histogram_quantile(0.99,
  rate(scheduler_cron_sweep_duration_seconds_bucket[5m]))

# 3. Current pod count vs HPA limits
kubectl get hpa trustage -n trustage
kubectl get pods -n trustage -l app=trustage --no-headers | wc -l
```

#### Remediate

- **Throughput insufficient**: increase batch size temporarily:
  ```bash
  kubectl set env deployment/trustage CRON_SCHEDULER_BATCH_SIZE=2000 -n trustage
  ```
- **DB slow**: check `pg_stat_activity` and `pg_stat_user_tables` for `schedule_definitions`. Run `VACUUM ANALYZE schedule_definitions;` if bloat is high.
- **Pod count**: manually scale up if HPA has not reacted:
  ```bash
  kubectl scale deployment/trustage --replicas=8 -n trustage
  ```
- **Mixed errors**: resolve `SchedulerCronSweepFailures` first; backlog will drain once sweeps succeed.

#### Escalate

Escalate to `#trustage-oncall` if backlog exceeds 5 minutes and does not start declining within 10 minutes of remediation. Prolonged backlog causes missed cron fires, which may require manual re-trigger of affected tenant workflows.

---

### SchedulerCronSweepSlow {#schedulercronsweepslow}

**Severity.** Warning.

#### What it means

The p99 duration of `ClaimAndFireBatch` sweeps is above the threshold (typically 500 ms). Slow sweeps consume DB connections for longer, reduce effective throughput, and can cause the scheduler to fall behind on high-schedule tenants.

#### Likely causes

1. **Large batch size hitting DB row-lock contention** — many pods competing for the same rows.
2. **Table bloat or missing index** — autovacuum lag on `schedule_definitions`.
3. **DB overloaded** — shared with other heavy workloads; connection pool queuing.
4. **Network latency** between pod and DB (cross-AZ placement, DNS issues).

#### Diagnose

```bash
# 1. p99 sweep duration trend
promql: histogram_quantile(0.99,
  rate(scheduler_cron_sweep_duration_seconds_bucket[10m]))

# 2. Active DB connections used by trustage
kubectl exec -n trustage deploy/trustage -- \
  psql "$DATABASE_URL" -c \
  "SELECT count(*), state FROM pg_stat_activity
   WHERE application_name LIKE '%trustage%'
   GROUP BY state"

# 3. Check for lock waits
kubectl exec -n trustage deploy/trustage -- \
  psql "$DATABASE_URL" -c \
  "SELECT pid, wait_event_type, wait_event, query_start, query
   FROM pg_stat_activity WHERE wait_event_type = 'Lock'"
```

#### Remediate

- **Row-lock contention**: reduce `CRON_SCHEDULER_BATCH_SIZE` to lower lock-hold time per transaction:
  ```bash
  kubectl set env deployment/trustage CRON_SCHEDULER_BATCH_SIZE=100 -n trustage
  ```
- **Table bloat**: `VACUUM ANALYZE schedule_definitions;` — may need a maintenance window if bloat is severe.
- **DB overloaded**: reduce `SCHEDULER_POOL_MAX_CONNS` or shift non-critical workloads to off-peak.
- **Network latency**: verify pod-to-DB routing; check that pods and DB are in the same AZ.

#### Escalate

Escalate to `#trustage-oncall` if p99 stays above 2 s for more than 15 minutes. At that point lock contention or DB health requires direct DBA involvement.

---

### SchedulerInvalidCronPersisting {#schedulerinvalidcronpersisting}

**Severity.** Info.

#### What it means

`scheduler_cron_invalid_cron_total` has a non-zero rate, meaning one or more `schedule_definitions` rows contain an expression that cannot be parsed (`CronExpr`) or a timezone string rejected by the tz database. These rows are **parked** (their `next_fire_at` is cleared) and will never fire until corrected. The counter increments on every sweep for every affected row.

#### Likely causes

1. **Invalid cron expression stored** — user API accepted an expression that passes a loose validation but fails the strict parser (`dsl.ParseCron`). Source: `apps/default/service/schedulers/cron.go:136–148`.
2. **Invalid timezone string** — IANA timezone lookup fails (e.g. `"EST5EDT"` vs `"America/New_York"`). Source: `cron.go:155–167`.
3. **Data migration error** — bulk import of schedule rows with malformed data.

#### Diagnose

```bash
# 1. Which tenants have invalid rows
promql: increase(scheduler_cron_invalid_cron_total[10m]) > 0

# 2. Find offending rows in the DB
kubectl exec -n trustage deploy/trustage -- \
  psql "$DATABASE_URL" -c \
  "SELECT id, tenant_id, cron_expr, timezone, next_fire_at
   FROM schedule_definitions
   WHERE next_fire_at IS NULL AND active = true
   LIMIT 20"

# 3. Check logs for the offending schedule IDs
kubectl logs -n trustage -l app=trustage --since=10m \
  | grep "invalid cron\|invalid timezone"
```

#### Remediate

- **Invalid cron expr**: correct the expression via the workflow API (`ArchiveWorkflow` + `CreateWorkflow` with valid cron) or, if the row was bulk-inserted, update via the DB:
  ```sql
  UPDATE schedule_definitions
  SET cron_expr = '0 * * * *'   -- corrected expression
  WHERE id = '<offending-id>';
  ```
- **Invalid timezone**: replace with a valid IANA name (e.g. `UTC`, `America/New_York`).
- **Bulk data issue**: identify the source migration script and fix at the source; then update or delete affected rows.

No code change is required unless `dsl.ParseCron` or the API validation layer needs to be tightened.

#### Escalate

This alert is info-severity and does not require paging. File a ticket in `#trustage-oncall` if the affected tenant count exceeds 50 or the rows were inserted by a platform migration script (indicates a systemic issue).

---

### WorkflowLifecycleFailures {#workflowlifecyclefailures}

**Severity.** Warning.

#### What it means

`workflow_lifecycle_total{result="fail"}` has a non-zero rate, meaning `CreateWorkflow`, `ActivateWorkflow`, or `ArchiveWorkflow` is returning errors to callers. Affected tenants cannot create or manage workflow definitions. The metric is emitted via a `defer` in each lifecycle method (`apps/default/service/business/workflow.go:90,361,428`) so it captures every error path including panics.

#### Likely causes

1. **DB write failures** — connectivity, constraint violations, pool exhaustion.
2. **DSL validation rejecting input** — callers sending invalid DSL blobs; `result=fail` on `op=create` with a corresponding log.
3. **Concurrent activation conflict** — two callers activating the same workflow version simultaneously; one fails the status transition check.
4. **Schema registration failure** — `SchemaRegistry.RegisterSchema` hits a storage error.
5. **Schedule activation partial failure** — `ActivateByWorkflow` or `DeactivateByWorkflow` fails after the workflow row is updated.

#### Diagnose

```bash
# 1. Failure rate by operation
promql: rate(workflow_lifecycle_total{result="fail"}[5m]) by (op)

# 2. Error logs from the business layer
kubectl logs -n trustage -l app=trustage --since=5m \
  | grep -E "persist workflow|update workflow|activate schedules|deactivate schedules"

# 3. DB connectivity check
kubectl exec -n trustage deploy/trustage -- \
  psql "$DATABASE_URL" -c "SELECT count(*) FROM workflow_definitions LIMIT 1"
```

#### Remediate

- **DB connectivity**: restore and pods auto-recover; no manual step needed.
- **DSL validation errors** (`op=create`): these are caller errors (4xx equivalent); check logs for the offending DSL field. No remediation needed unless the API validator is too strict — that requires a code change.
- **Concurrent activation conflict**: transient; callers should retry. If persistent, check for a stuck workflow in an unexpected state:
  ```sql
  SELECT id, name, status FROM workflow_definitions
  WHERE name = '<name>' ORDER BY workflow_version DESC LIMIT 5;
  ```
- **Schema registration failure**: check `schema_definitions` table for constraint errors or storage quota issues.
- **Partial schedule activation**: the log line `"workflow ACTIVE but schedules stale; retry to reconcile"` indicates a retryable condition — re-calling `ActivateWorkflow` is safe and will reconcile.

#### Escalate

Escalate to `#trustage-oncall` if the failure ratio (fail / total) for any `op` exceeds 10 % for more than 5 minutes, or if `op=activate` failures persist after a retry attempt (indicates a stuck state machine that needs manual DB inspection).
