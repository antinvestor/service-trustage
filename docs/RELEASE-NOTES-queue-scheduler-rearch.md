# Release notes: queue + external scheduler re-architecture

**Date:** 2026-07-25  
**Status:** Ready for staged rollout

## Summary

trustage-api is no longer an always-on multi-ticker monolith. It is a **role-split binary** driven by Frame `queue.Manager` (config URL schemes), with dual-path-safe CAS for dispatch/retry/timeout, per-step `timeout_at`, and external or multi-sweep progress.

## Breaking / operational changes

| Change | Action |
|--------|--------|
| Serving pods **do not migrate** | Run migrate Job: `DO_DATABASE_MIGRATE=true` |
| Default progress is `multi_sweep`, not 9 tickers | Set `PROGRESS_DRIVER=external_only` on Cloud Run |
| `CACHE_REQUIRE_VALKEY=true` in prod | Fail-closed if Valkey missing |
| Cron lag SLO **60s** | Cloud Scheduler every minute for `sched-cron` |
| Worker Cloud Run timeout **300s** | Align with step budget |

## New env (see `docs/deploy/`)

- `SERVICE_ROLE`, `SERVICE_ROLE_RECONCILE_IN_WORKER`, `WORKER_EXPOSE_API`
- `PROGRESS_DRIVER`, `ENABLE_LEGACY_TICKERS`, `ENABLE_WORK_NOTIFIER`
- `RECONCILER_MULTI_SWEEP_INTERVAL_SECONDS`, `CLOUD_RUN_MAX_STEP_SECONDS`
- `CLOUD_TASKS_DELAYED_URL_TEMPLATE`, `CACHE_REQUIRE_VALKEY`
- `QUEUE_SCHED_*_NAME/URL` for reconcile/wake refs

## Rollout order

1. Deploy migrate Job (adds `timeout_at` via AutoMigrate).
2. Deploy worker + api with dual registration; keep multi-sweep or legacy until push/Tasks verified.
3. Point Pub/Sub push + Cloud Scheduler + Cloud Tasks.
4. Set `PROGRESS_DRIVER=external_only`, `ENABLE_LEGACY_TICKERS=false`.
5. Set `CACHE_REQUIRE_VALKEY=true` in prod.

## Rollback

- `PROGRESS_DRIVER=legacy_tickers` on `worker`/`reconciler`/`all` (never `api`).
- Or `PROGRESS_DRIVER=multi_sweep` without external schedulers.

## Design

See `docs/design/2026-07-25-queue-external-scheduler-rearchitecture.md`.
