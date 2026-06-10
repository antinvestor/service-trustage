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

package telemetry

import (
	"context"
	"time"

	frametelemetry "github.com/pitabwire/frame/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Tracer name constants.
const (
	TracerEngine    = "trustage.engine"
	TracerConnector = "trustage.connector"
	TracerEvent     = "trustage.event"
	TracerScheduler = "trustage.scheduler"
)

// Span name constants.
const (
	SpanDispatch          = "engine.dispatch"
	SpanCommit            = "engine.commit"
	SpanValidateInput     = "state.validate_input"
	SpanValidateOutput    = "state.validate_output"
	SpanEvaluateMapping   = "state.evaluate_mapping"
	SpanExecuteConnector  = "connector.execute"
	SpanRouteEvent        = "event.route"
	SpanSchedulerDispatch = "scheduler.dispatch"
	SpanSchedulerRetry    = "scheduler.retry"
	SpanSchedulerTimeout  = "scheduler.timeout"
	SpanSchedulerOutbox   = "scheduler.outbox"
	SpanSchedulerCron     = "scheduler.cron.sweep"
	SpanCreateWorkflow    = "workflow.create"
)

// Attribute key constants.
const (
	AttrStatus        = "status"
	AttrTenantID      = "tenant_id"
	AttrWorkflow      = "workflow"
	AttrState         = "state"
	AttrFromState     = "from_state"
	AttrToState       = "to_state"
	AttrErrorClass    = "error_class"
	AttrViolationType = "violation_type"
	AttrConnectorType = "connector_type"
	AttrEventType     = "event_type"
)

// Metrics holds all OTel instruments for the engine. Instruments come from
// frame's BusinessMetrics factory, so every measurement transparently carries
// tenant_id/partition_id from the claims context. Scheduler sweeps run on a
// system context (no claims) and attribute tenants explicitly where the
// recorded tenant is not the caller's tenant.
type Metrics struct {
	ExecutionsTotal            frametelemetry.Counter
	ExecutionLatency           frametelemetry.Histogram
	TransitionsTotal           frametelemetry.Counter
	RetriesTotal               frametelemetry.Counter
	ContractViolationsTotal    frametelemetry.Counter
	StaleExecutionsTotal       frametelemetry.Counter
	DispatchLatency            frametelemetry.Histogram
	CommitLatency              frametelemetry.Histogram
	ConnectorCallsTotal        frametelemetry.Counter
	ConnectorLatency           frametelemetry.Histogram
	EventsIngestedTotal        frametelemetry.Counter
	EventsRoutedTotal          frametelemetry.Counter
	SchedulerPendingGauge      frametelemetry.Gauge
	SchedulerRetryDueGauge     frametelemetry.Gauge
	SchedulerDispatchedGauge   frametelemetry.Gauge
	SchedulerOutboxGauge       frametelemetry.Gauge
	SchedulerCronFired         frametelemetry.Counter
	SchedulerCronSweepDuration frametelemetry.Histogram
	SchedulerCronInvalid       frametelemetry.Counter
	// v1.2 observability — backlog gauge, tenant-attributed fire counter, lifecycle counter.
	SchedulerCronBacklog   frametelemetry.FloatGauge // scheduler_cron_backlog_seconds
	WorkflowLifecycleTotal frametelemetry.Counter    // workflow_lifecycle_total
}

// NewMetrics creates and registers all OTel instruments.
func NewMetrics() *Metrics {
	bm := frametelemetry.NewBusinessMetrics("trustage")

	return &Metrics{
		ExecutionsTotal:         bm.Counter("engine.executions.total", "Total workflow state executions dispatched"),
		ExecutionLatency:        bm.Histogram("engine.execution.latency_ms", "End-to-end execution latency"),
		TransitionsTotal:        bm.Counter("engine.transitions.total", "Total state transitions committed"),
		RetriesTotal:            bm.Counter("engine.retries.total", "Total execution retries scheduled"),
		ContractViolationsTotal: bm.Counter("engine.contract_violations.total", "Total schema contract violations"),
		StaleExecutionsTotal: bm.Counter(
			"engine.stale_executions.total",
			"Total stale execution mutations rejected",
		),
		DispatchLatency:     bm.Histogram("engine.dispatch.latency_ms", "Dispatch latency"),
		CommitLatency:       bm.Histogram("engine.commit.latency_ms", "Commit latency"),
		ConnectorCallsTotal: bm.Counter("connector.calls.total", "Total connector invocations"),
		ConnectorLatency:    bm.Histogram("connector.latency_ms", "Connector call latency"),
		EventsIngestedTotal: bm.Counter("events.ingested.total", "Total events ingested"),
		EventsRoutedTotal:   bm.Counter("events.routed.total", "Total events routed to workflow instances"),
		SchedulerPendingGauge: bm.Gauge(
			"scheduler.pending_executions",
			"Pending executions seen by the dispatch sweep",
		),
		SchedulerRetryDueGauge: bm.Gauge(
			"scheduler.retry_due_executions",
			"Retry-due executions seen by the retry sweep",
		),
		SchedulerDispatchedGauge: bm.Gauge(
			"scheduler.dispatched_executions",
			"Overdue dispatched executions seen by the timeout sweep",
		),
		SchedulerOutboxGauge: bm.Gauge(
			"scheduler.unpublished_events",
			"Unpublished events claimed by the outbox sweep",
		),
		SchedulerCronFired: bm.Counter(
			"scheduler_cron_fired_total",
			"Schedules fired per sweep, attributed per tenant",
		),
		SchedulerCronSweepDuration: bm.Histogram(
			"scheduler_cron_sweep_duration_seconds",
			"Duration of one ClaimAndFireBatch sweep",
			metric.WithUnit("s"),
		),
		SchedulerCronInvalid: bm.Counter(
			"scheduler_cron_invalid_cron_total",
			"Schedules parked due to invalid cron/timezone",
		),
		SchedulerCronBacklog: bm.FloatGauge(
			"scheduler_cron_backlog_seconds",
			"Age (in seconds) of the oldest due schedule. 0 if no schedules are currently due.",
			metric.WithUnit("s"),
		),
		WorkflowLifecycleTotal: bm.Counter(
			"workflow_lifecycle_total",
			"Count of workflow lifecycle operations (create|activate|archive) by result and tenant.",
		),
	}
}

// RecordSchedulerCronSweep emits per-tenant fire counters and the sweep-duration
// histogram after one ClaimAndFireBatch sweep completes. firedByTenant may be
// empty (no rows fired this sweep). ok=false marks a sweep that returned an
// error; firedByTenant should be empty in that case.
//
// The sweep runs on a system context (no claims), so the wrapper cannot infer
// a tenant; the per-tenant attribution from firedByTenant is the only source
// and stays explicit. Explicit attrs are appended after ctx attrs, so they
// win regardless.
func (m *Metrics) RecordSchedulerCronSweep(
	ctx context.Context, firedByTenant map[string]int, dur time.Duration, ok bool,
) {
	if m == nil {
		return
	}

	result := "ok"
	if !ok {
		result = "fail"
	}

	// Histogram: one observation per sweep.
	m.SchedulerCronSweepDuration.Record(ctx, dur.Seconds(),
		attribute.String("result", result))

	// Counter: one increment per tenant in the sweep. On failure, emit a
	// single fail counter with empty tenant so the fail rate is always visible
	// even if no rows were fired.
	if !ok || len(firedByTenant) == 0 {
		m.SchedulerCronFired.Add(ctx, 0,
			attribute.String("result", result),
			attribute.String("tenant_id", ""),
		)
		return
	}

	for tenantID, count := range firedByTenant {
		m.SchedulerCronFired.Add(ctx, int64(count),
			attribute.String("result", result),
			attribute.String("tenant_id", tenantID),
		)
	}
}

// ObserveSchedulerBacklog sets the scheduler_cron_backlog_seconds gauge to the
// most recent sampled value. Called once per sweep by CronScheduler.
func (m *Metrics) ObserveSchedulerBacklog(ctx context.Context, seconds float64) {
	if m == nil {
		return
	}
	m.SchedulerCronBacklog.Record(ctx, seconds)
}

// RecordWorkflowLifecycle increments workflow_lifecycle_total for the given
// operation (create|activate|archive) and result. Called by the business
// layer at the end of each lifecycle method on the caller's request context,
// so tenant_id/partition_id are attached transparently by the wrapper.
func (m *Metrics) RecordWorkflowLifecycle(ctx context.Context, op string, ok bool) {
	if m == nil {
		return
	}
	result := "ok"
	if !ok {
		result = "fail"
	}
	m.WorkflowLifecycleTotal.Add(ctx, 1,
		attribute.String("op", op),
		attribute.String("result", result),
	)
}

// StartSpan starts a new OTel span.
func StartSpan(
	ctx context.Context,
	tracerName, spanName string,
	attrs ...attribute.KeyValue,
) (context.Context, trace.Span) {
	tracer := otel.Tracer(tracerName)

	opts := []trace.SpanStartOption{
		trace.WithAttributes(attrs...),
	}

	return tracer.Start(ctx, spanName, opts...) //nolint:spancheck // callers manage span lifecycle via EndSpan
}

// EndSpan ends a span, recording any error.
func EndSpan(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
	}

	span.End()
}
