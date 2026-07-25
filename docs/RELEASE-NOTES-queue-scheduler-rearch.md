# Release notes: queue + external scheduler re-architecture

**Date:** 2026-07-25  
**Status:** Greenfield — no legacy ticker path

## Summary

trustage-api is a **role-split binary** driven by Frame `queue.Manager` and either:

- `PROGRESS_DRIVER=multi_sweep` — single in-process reconcile loop, or  
- `PROGRESS_DRIVER=external_only` — Cloud Scheduler / Cloud Tasks push only  

There is **no** multi-ticker legacy mode and **no** age-based execution timeout fallback. Dispatch always sets absolute `timeout_at`.

## Operational requirements

| Change | Action |
|--------|--------|
| Serving pods **do not migrate** | Run migrate Job: `DO_DATABASE_MIGRATE=true` |
| Progress drivers only | `multi_sweep` or `external_only` |
| `CACHE_REQUIRE_VALKEY=true` in prod | Fail-closed if Valkey missing |
| Cron lag SLO **60s** | Cloud Scheduler every minute for `sched-cron` |
| Worker Cloud Run timeout **300s** | Align with step budget |

## Env (see `docs/deploy/`)

- `SERVICE_ROLE`, `SERVICE_ROLE_RECONCILE_IN_WORKER`, `WORKER_EXPOSE_API`
- `PROGRESS_DRIVER`, `ENABLE_WORK_NOTIFIER`
- `RECONCILER_MULTI_SWEEP_INTERVAL_SECONDS`, `CLOUD_RUN_MAX_STEP_SECONDS`
- `CLOUD_TASKS_DELAYED_URL_TEMPLATE`, `CACHE_REQUIRE_VALKEY`
- `QUEUE_SCHED_*_NAME/URL` for reconcile/wake refs

## Rollout

1. Deploy migrate Job (AutoMigrate `timeout_at`).
2. Deploy worker + api (never `SERVICE_ROLE=all` in prod).
3. Wire Pub/Sub push + Cloud Tasks + Cloud Scheduler.
4. Use `PROGRESS_DRIVER=external_only` on Cloud Run; `multi_sweep` for local/GKE without Tasks.
5. Set `CACHE_REQUIRE_VALKEY=true` in prod.

## Design

See `docs/design/2026-07-25-queue-external-scheduler-rearchitecture.md`.
