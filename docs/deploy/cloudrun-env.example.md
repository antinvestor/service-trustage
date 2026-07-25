# Cloud Run deployment example (trustage)

Primary target from design Rev 2.2: **Pub/Sub + Cloud Tasks + Cloud Scheduler**.

## Services

| Cloud Run service | `SERVICE_ROLE` | Notes |
|-------------------|----------------|-------|
| `trustage-api` | `api` | ConnectRPC + form/webhook; publishes `event-ingest` only |
| `trustage-worker` | `worker` | Push subscribers; `SERVICE_ROLE_RECONCILE_IN_WORKER=true` |
| Job `trustage-migrate` | any | `DO_DATABASE_MIGRATE=true` then exit |

## Common env

```bash
SERVER_PORT=8080   # or map Cloud Run PORT
CACHE_REQUIRE_VALKEY=true
VALKEY_CACHE_URL=redis://...
PROGRESS_DRIVER=external_only
ENABLE_WORK_NOTIFIER=true
CLOUD_RUN_MAX_STEP_SECONDS=300
DEFAULT_EXECUTION_TIMEOUT_SECONDS=300
# Worker request timeout: 300s
```

## Queue URLs (illustrative)

```bash
# Publish (api + worker)
QUEUE_EVENT_INGEST_URL=gcppubsub://PROJECT/wf-events
QUEUE_EXEC_DISPATCH_URL=gcppubsub://PROJECT/wf-executions

# Subscribe (worker) — push demux
QUEUE_EVENT_ROUTER_URL=push://event-router
QUEUE_EXEC_WORKER_URL=push://exec-worker
QUEUE_SCHED_RECONCILE_URL=push://sched-reconcile
QUEUE_SCHED_CRON_URL=push://sched-cron
QUEUE_SCHED_CLEANUP_URL=push://sched-cleanup
QUEUE_SCHED_DISPATCH_URL=push://sched-dispatch
QUEUE_SCHED_RETRY_URL=push://sched-retry
QUEUE_SCHED_TIMER_URL=push://sched-timer
QUEUE_SCHED_TIMEOUT_URL=push://sched-timeout
QUEUE_SCHED_SIGNAL_URL=push://sched-signal

# Delayed wakes (worker)
CLOUD_TASKS_DELAYED_URL_TEMPLATE='cloudtasks:///projects/PROJECT/locations/REGION/queues/wf-delayed?url=https://WORKER_HOST/_frame/queue/{ref}&oidc_sa=tasks-invoker@PROJECT.iam.gserviceaccount.com'
```

Point each Pub/Sub **push** subscription and Cloud Tasks queue HTTP target at:

`https://WORKER_HOST/_frame/queue/{ref}`

with `FRAME_QUEUE_PUSH_AUTH=oidc` and allowed invoker SA emails.

## Cloud Scheduler

| Job | Cadence | Target |
|-----|---------|--------|
| reconcile | every 1–2 min | `POST .../_frame/queue/sched-reconcile` body `{"kind":"reconcile"}` |
| cron | **every 60s** | `POST .../_frame/queue/sched-cron` |
| cleanup | every 6h | `POST .../_frame/queue/sched-cleanup` |

## GKE secondary

Use `PROGRESS_DRIVER=multi_sweep` + `RECONCILER_MULTI_SWEEP_INTERVAL_SECONDS=5` when Cloud Tasks is unavailable. NATS pull URLs remain valid for local/non-GCP only long-term.
