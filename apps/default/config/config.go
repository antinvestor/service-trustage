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

import "github.com/pitabwire/frame/config"

// Config holds all configuration for the Orchestrator service.
type Config struct {
	config.ConfigurationDefault

	// Server.
	ServerPort string `env:"SERVER_PORT" envDefault:"8080"`

	// Valkey.
	ValkeyCacheURL string `env:"VALKEY_CACHE_URL" envDefault:"redis://localhost:6379"`

	// Encryption.
	MasterEncryptionKey string `env:"MASTER_ENCRYPTION_KEY"`

	// Cron scheduler.
	CronSchedulerBatchSize    int `env:"CRON_SCHEDULER_BATCH_SIZE"       envDefault:"500"`
	CronSchedulerIntervalSecs int `env:"CRON_SCHEDULER_INTERVAL_SECONDS" envDefault:"1"`

	// Scheduler intervals (seconds).
	DispatchIntervalSeconds int `env:"DISPATCH_INTERVAL_SECONDS" envDefault:"5"`
	RetryIntervalSeconds    int `env:"RETRY_INTERVAL_SECONDS"    envDefault:"10"`
	TimerIntervalSeconds    int `env:"TIMER_INTERVAL_SECONDS"    envDefault:"5"`
	SignalIntervalSeconds   int `env:"SIGNAL_INTERVAL_SECONDS"   envDefault:"5"`
	ScopeIntervalSeconds    int `env:"SCOPE_INTERVAL_SECONDS"    envDefault:"5"`
	TimeoutIntervalSeconds  int `env:"TIMEOUT_INTERVAL_SECONDS"  envDefault:"30"`
	OutboxIntervalSeconds   int `env:"OUTBOX_INTERVAL_SECONDS"   envDefault:"5"`
	OutboxClaimTTLSeconds   int `env:"OUTBOX_CLAIM_TTL_SECONDS"  envDefault:"30"`
	TimerClaimTTLSeconds    int `env:"TIMER_CLAIM_TTL_SECONDS"   envDefault:"30"`
	SignalClaimTTLSeconds   int `env:"SIGNAL_CLAIM_TTL_SECONDS"  envDefault:"30"`
	ScopeClaimTTLSeconds    int `env:"SCOPE_CLAIM_TTL_SECONDS"   envDefault:"30"`

	// Scheduler batch sizes.
	DispatchBatchSize          int `env:"DISPATCH_BATCH_SIZE"            envDefault:"100"`
	RetryBatchSize             int `env:"RETRY_BATCH_SIZE"               envDefault:"50"`
	TimerBatchSize             int `env:"TIMER_BATCH_SIZE"               envDefault:"100"`
	SignalBatchSize            int `env:"SIGNAL_BATCH_SIZE"              envDefault:"100"`
	ScopeBatchSize             int `env:"SCOPE_BATCH_SIZE"               envDefault:"100"`
	TimeoutBatchSize           int `env:"TIMEOUT_BATCH_SIZE"             envDefault:"50"`
	OutboxBatchSize            int `env:"OUTBOX_BATCH_SIZE"              envDefault:"100"`
	DispatchMaxBatchesPerSweep int `env:"DISPATCH_MAX_BATCHES_PER_SWEEP"     envDefault:"100"`
	OutboxMaxBatchesPerSweep   int `env:"OUTBOX_MAX_BATCHES_PER_SWEEP"       envDefault:"100"`
	OutboxPublishConcurrency   int `env:"OUTBOX_PUBLISH_CONCURRENCY"         envDefault:"16"`

	// Execution timeout (seconds) - default timeout for dispatched executions.
	DefaultExecutionTimeoutSeconds int `env:"DEFAULT_EXECUTION_TIMEOUT_SECONDS" envDefault:"300"`

	// Rate limiting (per tenant, per minute).
	EventIngestRateLimit int `env:"EVENT_INGEST_RATE_LIMIT" envDefault:"100"`

	// Data retention.
	CleanupIntervalHours int `env:"CLEANUP_INTERVAL_HOURS" envDefault:"6"`
	RetentionDays        int `env:"RETENTION_DAYS"         envDefault:"90"`

	// Queue: Execution Dispatch (publisher).
	QueueExecDispatchName string `env:"QUEUE_EXEC_DISPATCH_NAME" envDefault:"exec-dispatch"`
	QueueExecDispatchURL  string `env:"QUEUE_EXEC_DISPATCH_URL"  envDefault:"nats://localhost:4222?jetstream=true&stream_name=wf-executions&stream_subjects=wf.exec.%3E&stream_retention=limits&stream_max_age=24h&stream_storage=file&stream_num_replicas=1&subject=wf.exec.dispatch"`

	// Queue: Execution Worker (subscriber).
	QueueExecWorkerName string `env:"QUEUE_EXEC_WORKER_NAME" envDefault:"exec-worker"`
	QueueExecWorkerURL  string `env:"QUEUE_EXEC_WORKER_URL"  envDefault:"nats://localhost:4222?jetstream=true&stream_name=wf-executions&stream_subjects=wf.exec.%3E&stream_retention=limits&stream_max_age=24h&stream_storage=file&stream_num_replicas=1&consumer_durable_name=exec-worker&consumer_ack_policy=explicit&consumer_max_deliver=3&consumer_ack_wait=30s&consumer_max_ack_pending=5000&consumer_deliver_policy=all&subject=wf.exec.dispatch"`

	// Queue: Event Ingest (publisher).
	QueueEventIngestName string `env:"QUEUE_EVENT_INGEST_NAME" envDefault:"event-ingest"`
	QueueEventIngestURL  string `env:"QUEUE_EVENT_INGEST_URL"  envDefault:"nats://localhost:4222?jetstream=true&stream_name=wf-events&stream_subjects=wf.events.%3E&stream_retention=limits&stream_max_age=720h&stream_storage=file&stream_num_replicas=1&subject=wf.events.%3E"`

	// Queue: Event Router (subscriber).
	QueueEventRouterName string `env:"QUEUE_EVENT_ROUTER_NAME" envDefault:"event-router"`
	QueueEventRouterURL  string `env:"QUEUE_EVENT_ROUTER_URL"  envDefault:"nats://localhost:4222?jetstream=true&stream_name=wf-events&stream_subjects=wf.events.%3E&stream_retention=limits&stream_max_age=720h&stream_storage=file&stream_num_replicas=1&consumer_durable_name=event-router&consumer_ack_policy=explicit&consumer_max_deliver=3&consumer_ack_wait=10s&consumer_max_ack_pending=10000&consumer_deliver_policy=all&subject=wf.events.%3E"`
}
