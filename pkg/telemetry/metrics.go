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

// Metrics holds all OTel instruments for the engine.
type Metrics struct {
	ExecutionsTotal            metric.Int64Counter
	ExecutionLatency           metric.Float64Histogram
	TransitionsTotal           metric.Int64Counter
	RetriesTotal               metric.Int64Counter
	ContractViolationsTotal    metric.Int64Counter
	StaleExecutionsTotal       metric.Int64Counter
	DispatchLatency            metric.Float64Histogram
	CommitLatency              metric.Float64Histogram
	ConnectorCallsTotal        metric.Int64Counter
	ConnectorLatency           metric.Float64Histogram
	EventsIngestedTotal        metric.Int64Counter
	EventsRoutedTotal          metric.Int64Counter
	SchedulerPendingGauge      metric.Int64Gauge
	SchedulerRetryDueGauge     metric.Int64Gauge
	SchedulerDispatchedGauge   metric.Int64Gauge
	SchedulerOutboxGauge       metric.Int64Gauge
	SchedulerCronFired         metric.Int64Counter
	SchedulerCronSweepDuration metric.Float64Histogram
	SchedulerCronInvalid       metric.Int64Counter
}

// NewMetrics creates and registers all OTel instruments.
func NewMetrics() *Metrics {
	meter := otel.Meter("trustage")

	execTotal, _ := meter.Int64Counter("engine.executions.total")
	execLatency, _ := meter.Float64Histogram("engine.execution.latency_ms")
	transTotal, _ := meter.Int64Counter("engine.transitions.total")
	retriesTotal, _ := meter.Int64Counter("engine.retries.total")
	violationsTotal, _ := meter.Int64Counter("engine.contract_violations.total")
	staleTotal, _ := meter.Int64Counter("engine.stale_executions.total")
	dispatchLatency, _ := meter.Float64Histogram("engine.dispatch.latency_ms")
	commitLatency, _ := meter.Float64Histogram("engine.commit.latency_ms")
	connectorTotal, _ := meter.Int64Counter("connector.calls.total")
	connectorLatency, _ := meter.Float64Histogram("connector.latency_ms")
	eventsIngested, _ := meter.Int64Counter("events.ingested.total")
	eventsRouted, _ := meter.Int64Counter("events.routed.total")
	pendingGauge, _ := meter.Int64Gauge("scheduler.pending_executions")
	retryDueGauge, _ := meter.Int64Gauge("scheduler.retry_due_executions")
	dispatchedGauge, _ := meter.Int64Gauge("scheduler.dispatched_executions")
	outboxGauge, _ := meter.Int64Gauge("scheduler.unpublished_events")
	cronFired, _ := meter.Int64Counter("scheduler_cron_fired_total")
	cronSweepDuration, _ := meter.Float64Histogram("scheduler_cron_sweep_duration_seconds")
	cronInvalid, _ := meter.Int64Counter("scheduler_cron_invalid_cron_total")

	return &Metrics{
		ExecutionsTotal:            execTotal,
		ExecutionLatency:           execLatency,
		TransitionsTotal:           transTotal,
		RetriesTotal:               retriesTotal,
		ContractViolationsTotal:    violationsTotal,
		StaleExecutionsTotal:       staleTotal,
		DispatchLatency:            dispatchLatency,
		CommitLatency:              commitLatency,
		ConnectorCallsTotal:        connectorTotal,
		ConnectorLatency:           connectorLatency,
		EventsIngestedTotal:        eventsIngested,
		EventsRoutedTotal:          eventsRouted,
		SchedulerPendingGauge:      pendingGauge,
		SchedulerRetryDueGauge:     retryDueGauge,
		SchedulerDispatchedGauge:   dispatchedGauge,
		SchedulerOutboxGauge:       outboxGauge,
		SchedulerCronFired:         cronFired,
		SchedulerCronSweepDuration: cronSweepDuration,
		SchedulerCronInvalid:       cronInvalid,
	}
}

// RecordSchedulerCronSweep records a single cron sweep: fired count,
// sweep duration, and success/failure result label.
func (m *Metrics) RecordSchedulerCronSweep(ctx context.Context, fired int, dur time.Duration, ok bool) {
	if m == nil {
		return
	}
	result := "ok"
	if !ok {
		result = "fail"
	}
	m.SchedulerCronFired.Add(ctx, int64(fired), metric.WithAttributes(attribute.String("result", result)))
	m.SchedulerCronSweepDuration.Record(ctx, dur.Seconds(), metric.WithAttributes(attribute.String("result", result)))
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
