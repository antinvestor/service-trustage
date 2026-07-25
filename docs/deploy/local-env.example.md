# Local docker-compose style env

```bash
SERVICE_ROLE=all
PROGRESS_DRIVER=multi_sweep
RECONCILER_MULTI_SWEEP_INTERVAL_SECONDS=5
ENABLE_WORK_NOTIFIER=true
CACHE_REQUIRE_VALKEY=false
VALKEY_CACHE_URL=redis://localhost:6379

# Hot path (NATS or mem)
QUEUE_EVENT_INGEST_URL=mem://event-ingest
QUEUE_EVENT_ROUTER_URL=mem://event-ingest
QUEUE_EXEC_DISPATCH_URL=mem://exec-dispatch
QUEUE_EXEC_WORKER_URL=mem://exec-dispatch

# Wakes / reconcile (mem)
QUEUE_SCHED_RECONCILE_URL=mem://sched-reconcile
QUEUE_SCHED_CRON_URL=mem://sched-cron
QUEUE_SCHED_CLEANUP_URL=mem://sched-cleanup
QUEUE_SCHED_DISPATCH_URL=mem://sched-dispatch
QUEUE_SCHED_RETRY_URL=mem://sched-retry
QUEUE_SCHED_TIMER_URL=mem://sched-timer
QUEUE_SCHED_TIMEOUT_URL=mem://sched-timeout
QUEUE_SCHED_SIGNAL_URL=mem://sched-signal

# Empty = Noop delayed (multi-sweep covers timers)
CLOUD_TASKS_DELAYED_URL_TEMPLATE=
```

Migrate once:

```bash
DO_DATABASE_MIGRATE=true ./default
```
