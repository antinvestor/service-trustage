# Scheduler Runbook

Operator-facing runbook for trustage scheduler + workflow lifecycle alerts.

---

## Operational overview

**Deployment.** trustage runs as role-split processes (same binary: `SERVICE_ROLE=api|worker|reconciler|all`). There is **no** in-process multi-ticker engine.

| Mode | Env | Behaviour |
|------|-----|-----------|
| **Cloud Run (primary)** | `PROGRESS_DRIVER=external_only` | Cloud Scheduler → Frame push `sched-cron` / `sched-reconcile`; Cloud Tasks delayed wakes |
| **Multi-sweep** | `PROGRESS_DRIVER=multi_sweep` | One in-process loop (≤5–10s) for timer/retry/dispatch/outbox/cron/cleanup |

Cron lag SLO: **60s** max (Cloud Scheduler every minute). `FOR UPDATE SKIP LOCKED` in `ClaimAndFireBatch` keeps multi-pod cron safe.

**One transaction per cron sweep.** Each `RunOnce` is a single DB transaction: claim due `schedule_definitions`, write `event_log`, advance `next_fire_at`, commit.

**Key config knobs:**

| Env var | Default | Purpose |
|---|---|---|
| `PROGRESS_DRIVER` | `multi_sweep` | `multi_sweep` or `external_only` |
| `CRON_SCHEDULER_BATCH_SIZE` | `500` | Rows claimed per cron sweep |
| `RECONCILER_MULTI_SWEEP_INTERVAL_SECONDS` | `5` | Multi-sweep tick |
| `SCHEDULER_POOL_MAX_CONNS` | `10` | Dedicated scheduler DB pool |
| `OUTBOX_PUBLISH_CONCURRENCY` | `16` | Parallel publishes per outbox sweep |

---

## Alerts

### SchedulerCronSweepFailures {#schedulercronsweepfailures}

**Severity.** Page.

Cron `ClaimAndFireBatch` errors — schedules not firing.

**Remediate:** restore DB connectivity; complete migrations; bounce pooler; raise `SCHEDULER_POOL_MAX_CONNS`.

### SchedulerCronBacklogGrowing {#schedulercronbackloggrowing}

**Severity.** Warning.

`scheduler_cron_backlog_seconds` rising — increase batch size, scale workers/reconcilers, or fix DB lag.

### SchedulerCronSweepSlow {#schedulercronsweepslow}

**Severity.** Warning.

p99 sweep duration high — reduce contention, vacuum `schedule_definitions`, check pool sizing.
