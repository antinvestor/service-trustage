// Copyright 2023-2026 Ant Investor Ltd
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"net/url"
	"strconv"

	"github.com/pitabwire/frame/v2/config"
)

// Config holds all configuration for the Orchestrator service.
type Config struct {
	config.ConfigurationDefault

	// Server.
	ServerPort string `env:"SERVER_PORT" envDefault:"8080"`

	// Process role and progress driver (see role.go).
	ServiceRole string `env:"SERVICE_ROLE" envDefault:"all"`
	// When true, worker also hosts reconcile/cron/cleanup handlers.
	ServiceRoleReconcileInWorker bool `env:"SERVICE_ROLE_RECONCILE_IN_WORKER" envDefault:"true"`
	// Optional ConnectRPC on worker role (discouraged).
	WorkerExposeAPI bool `env:"WORKER_EXPOSE_API" envDefault:"false"`
	// multi_sweep | external_only.
	ProgressDriver string `env:"PROGRESS_DRIVER" envDefault:"multi_sweep"`
	// Primary-path notify after durable writes.
	EnableWorkNotifier bool `env:"ENABLE_WORK_NOTIFIER" envDefault:"true"`
	// Interval for multi_sweep progress driver.
	ReconcilerMultiSweepIntervalS int `env:"RECONCILER_MULTI_SWEEP_INTERVAL_SECONDS" envDefault:"5"`
	// Cap for per-step execution timeout (Cloud Run budget).
	CloudRunMaxStepSeconds int `env:"CLOUD_RUN_MAX_STEP_SECONDS" envDefault:"300"`

	// Valkey.
	ValkeyCacheURL string `env:"VALKEY_CACHE_URL" envDefault:"redis://localhost:6379"`
	// Fail closed when true (required for multi-instance prod).
	CacheRequireValkey bool `env:"CACHE_REQUIRE_VALKEY" envDefault:"false"`

	// Encryption.
	MasterEncryptionKey string `env:"MASTER_ENCRYPTION_KEY"`

	// Cloud Tasks delayed publisher (optional; empty = Noop delayed).
	CloudTasksDelayedURLTemplate string `env:"CLOUD_TASKS_DELAYED_URL_TEMPLATE"`
	// Max schedule horizon for delayed wakes (hours). Default 720 = 30 days.
	CloudTasksMaxHorizonHours int `env:"CLOUD_TASKS_MAX_HORIZON_HOURS" envDefault:"720"`

	// Cron scheduler.
	CronSchedulerBatchSize int `env:"CRON_SCHEDULER_BATCH_SIZE" envDefault:"500"`

	// Primary DB pool sizing (shared by all HTTP/RPC handlers and most repositories).
	// Default 50 — sized to absorb NATS worker bursts (exec-worker 500 + event-router 200)
	// without exhausting the PostgreSQL server's connection limit.
	DatabasePoolMaxConns int `env:"DATABASE_POOL_MAX_CONNS" envDefault:"50"`

	// Scheduler pool sizing (dedicated pool isolates fire-path from HTTP/RPC handlers).
	SchedulerPoolMaxConns int `env:"SCHEDULER_POOL_MAX_CONNS" envDefault:"10"`
	SchedulerPoolMinConns int `env:"SCHEDULER_POOL_MIN_CONNS" envDefault:"2"`

	// Claim TTLs for SKIP LOCKED lease sweeps (RunOnce).
	OutboxClaimTTLSeconds int `env:"OUTBOX_CLAIM_TTL_SECONDS" envDefault:"30"`
	// Timer lease TTL.
	TimerClaimTTLSeconds int `env:"TIMER_CLAIM_TTL_SECONDS" envDefault:"30"`
	// Signal wait lease TTL.
	SignalClaimTTLSeconds int `env:"SIGNAL_CLAIM_TTL_SECONDS" envDefault:"30"`
	// Scope lease TTL.
	ScopeClaimTTLSeconds int `env:"SCOPE_CLAIM_TTL_SECONDS" envDefault:"30"`

	// Scheduler batch sizes.
	// Outbox and dispatch defaults reduced (100→20/50) to prevent cluster-burst storms
	// across 12+ pods simultaneously hammering NATS and the primary DB pool.
	DispatchBatchSize          int `env:"DISPATCH_BATCH_SIZE"            envDefault:"50"`
	RetryBatchSize             int `env:"RETRY_BATCH_SIZE"               envDefault:"50"`
	TimerBatchSize             int `env:"TIMER_BATCH_SIZE"               envDefault:"100"`
	SignalBatchSize            int `env:"SIGNAL_BATCH_SIZE"              envDefault:"100"`
	ScopeBatchSize             int `env:"SCOPE_BATCH_SIZE"               envDefault:"100"`
	TimeoutBatchSize           int `env:"TIMEOUT_BATCH_SIZE"             envDefault:"50"`
	OutboxBatchSize            int `env:"OUTBOX_BATCH_SIZE"              envDefault:"20"`
	DispatchMaxBatchesPerSweep int `env:"DISPATCH_MAX_BATCHES_PER_SWEEP" envDefault:"50"`
	OutboxMaxBatchesPerSweep   int `env:"OUTBOX_MAX_BATCHES_PER_SWEEP"   envDefault:"50"`
	OutboxPublishConcurrency   int `env:"OUTBOX_PUBLISH_CONCURRENCY"     envDefault:"16"`
	ReconcileMaxBatchesPerJob  int `env:"RECONCILE_MAX_BATCHES_PER_JOB"  envDefault:"1"`

	// Execution timeout (seconds) - default timeout for dispatched executions.
	DefaultExecutionTimeoutSeconds int `env:"DEFAULT_EXECUTION_TIMEOUT_SECONDS" envDefault:"300"`

	// Rate limiting (per tenant, per minute).
	EventIngestRateLimit int `env:"EVENT_INGEST_RATE_LIMIT" envDefault:"100"`

	// Event router binding fanout cap. A single event type with more matching
	// bindings than this limit would lock the NATS handler (consumer_ack_wait=10s)
	// and trigger redelivery storms. 200 is sized to match consumer_max_ack_pending.
	EventRouterBindingLimit int `env:"EVENT_ROUTER_BINDING_LIMIT" envDefault:"200"`

	// Data retention.
	CleanupIntervalHours      int `env:"CLEANUP_INTERVAL_HOURS"       envDefault:"6"`
	RetentionDays             int `env:"RETENTION_DAYS"               envDefault:"90"`
	WorkflowRowRetentionHours int `env:"WORKFLOW_ROW_RETENTION_HOURS" envDefault:"720"`

	// Adapter HTTP timeout — used by connector adapters making outbound HTTP calls.
	AdapterHTTPTimeoutSeconds int `env:"ADAPTER_HTTP_TIMEOUT_SECONDS" envDefault:"30"`

	// NATS consumer back-pressure — caps the number of in-flight unacked messages
	// per subscriber. Upper bound on goroutines a slow downstream can pin. Must
	// be > 0 to take effect; zero leaves the baked-in URL value.
	ExecWorkerMaxAckPending  int `env:"EXEC_WORKER_MAX_ACK_PENDING"  envDefault:"1000"`
	EventRouterMaxAckPending int `env:"EVENT_ROUTER_MAX_ACK_PENDING" envDefault:"1000"`

	// Queue: Execution Dispatch (publisher).
	QueueExecDispatchName string `env:"QUEUE_EXEC_DISPATCH_NAME" envDefault:"exec-dispatch"`
	QueueExecDispatchURL  string `env:"QUEUE_EXEC_DISPATCH_URL"  envDefault:"nats://localhost:4222?jetstream=true&stream_name=wf-executions&stream_subjects=wf.exec.%3E&stream_retention=limits&stream_max_age=24h&stream_storage=file&stream_num_replicas=1&subject=wf.exec.dispatch"`

	// Queue: Execution Worker (subscriber).
	QueueExecWorkerName string `env:"QUEUE_EXEC_WORKER_NAME" envDefault:"exec-worker"`
	QueueExecWorkerURL  string `env:"QUEUE_EXEC_WORKER_URL"  envDefault:"nats://localhost:4222?jetstream=true&stream_name=wf-executions&stream_subjects=wf.exec.%3E&stream_retention=limits&stream_max_age=24h&stream_storage=file&stream_num_replicas=1&consumer_durable_name=exec-worker&consumer_ack_policy=explicit&consumer_max_deliver=3&consumer_ack_wait=30s&consumer_max_ack_pending=1000&consumer_deliver_policy=all&subject=wf.exec.dispatch"`

	// Queue: Event Ingest (publisher).
	QueueEventIngestName string `env:"QUEUE_EVENT_INGEST_NAME" envDefault:"event-ingest"`
	QueueEventIngestURL  string `env:"QUEUE_EVENT_INGEST_URL"  envDefault:"nats://localhost:4222?jetstream=true&stream_name=wf-events&stream_subjects=wf.events.%3E&stream_retention=limits&stream_max_age=720h&stream_storage=file&stream_num_replicas=1&subject=wf.events.%3E"`

	// Queue: Event Router (subscriber).
	QueueEventRouterName string `env:"QUEUE_EVENT_ROUTER_NAME" envDefault:"event-router"`
	QueueEventRouterURL  string `env:"QUEUE_EVENT_ROUTER_URL"  envDefault:"nats://localhost:4222?jetstream=true&stream_name=wf-events&stream_subjects=wf.events.%3E&stream_retention=limits&stream_max_age=720h&stream_storage=file&stream_num_replicas=1&consumer_durable_name=event-router&consumer_ack_policy=explicit&consumer_max_deliver=3&consumer_ack_wait=10s&consumer_max_ack_pending=1000&consumer_deliver_policy=all&subject=wf.events.%3E"`

	// Reconcile / wake push subscribers (Frame push:// or pull URLs).
	// Defaults use mem:// for local safety; production sets push:// or nats/gcppubsub.
	QueueSchedReconcileName string `env:"QUEUE_SCHED_RECONCILE_NAME" envDefault:"sched-reconcile"`
	QueueSchedReconcileURL  string `env:"QUEUE_SCHED_RECONCILE_URL"  envDefault:"mem://sched-reconcile"`
	QueueSchedCronName      string `env:"QUEUE_SCHED_CRON_NAME"      envDefault:"sched-cron"`
	QueueSchedCronURL       string `env:"QUEUE_SCHED_CRON_URL"       envDefault:"mem://sched-cron"`
	QueueSchedCleanupName   string `env:"QUEUE_SCHED_CLEANUP_NAME"   envDefault:"sched-cleanup"`
	QueueSchedCleanupURL    string `env:"QUEUE_SCHED_CLEANUP_URL"    envDefault:"mem://sched-cleanup"`
	QueueSchedDispatchName  string `env:"QUEUE_SCHED_DISPATCH_NAME"  envDefault:"sched-dispatch"`
	QueueSchedDispatchURL   string `env:"QUEUE_SCHED_DISPATCH_URL"   envDefault:"mem://sched-dispatch"`
	QueueSchedRetryName     string `env:"QUEUE_SCHED_RETRY_NAME"     envDefault:"sched-retry"`
	QueueSchedRetryURL      string `env:"QUEUE_SCHED_RETRY_URL"      envDefault:"mem://sched-retry"`
	QueueSchedTimerName     string `env:"QUEUE_SCHED_TIMER_NAME"     envDefault:"sched-timer"`
	QueueSchedTimerURL      string `env:"QUEUE_SCHED_TIMER_URL"      envDefault:"mem://sched-timer"`
	QueueSchedTimeoutName   string `env:"QUEUE_SCHED_TIMEOUT_NAME"   envDefault:"sched-timeout"`
	QueueSchedTimeoutURL    string `env:"QUEUE_SCHED_TIMEOUT_URL"    envDefault:"mem://sched-timeout"`
	QueueSchedSignalName    string `env:"QUEUE_SCHED_SIGNAL_NAME"    envDefault:"sched-signal"`
	QueueSchedSignalURL     string `env:"QUEUE_SCHED_SIGNAL_URL"     envDefault:"mem://sched-signal"`
}

// injectQueryParam replaces or adds a single query parameter on a URL string.
// Returns the original string unchanged if parse fails (URL stays usable; the
// default baked-in value applies).
func injectQueryParam(raw, key string, value int) string {
	if value <= 0 {
		return raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	q := u.Query()
	q.Set(key, strconv.Itoa(value))
	u.RawQuery = q.Encode()
	return u.String()
}

// ApplyQueueOverrides rewrites the consumer URLs with the configured
// back-pressure values. Safe to call once after env loading.
func (c *Config) ApplyQueueOverrides() {
	c.QueueExecWorkerURL = injectQueryParam(
		c.QueueExecWorkerURL, "consumer_max_ack_pending", c.ExecWorkerMaxAckPending,
	)
	c.QueueEventRouterURL = injectQueryParam(
		c.QueueEventRouterURL, "consumer_max_ack_pending", c.EventRouterMaxAckPending,
	)
}
