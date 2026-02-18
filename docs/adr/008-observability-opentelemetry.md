# ADR-008: Observability Strategy (OpenTelemetry)

## Status
Accepted

## Context
Orchestrator has a complex execution path that spans multiple subsystems: ConnectRPC handlers receive requests, Frame business logic processes them, NATS JetStream carries events, the event router evaluates triggers, the state engine orchestrates workflow execution, workers invoke external systems through connectors, and AI services generate and validate DSL. A single user action (such as submitting a form) can traverse all of these layers before producing a final result.

Without correlated observability across these layers, debugging production issues becomes a matter of guesswork. A failed workflow step could be caused by a connector timeout, a malformed event payload, a rate limit hit, or a worker crash. Operators need the ability to follow a single request from ingestion through final completion, with metrics to understand system health and logs to diagnose specific failures.

Foundry establishes the observability patterns for the ecosystem: named tracers per domain, structured span attributes, OTel counters/histograms/gauges for business metrics, and `util.Log(ctx)` for structured context-aware logging. Orchestrator must follow these patterns for consistency while adding workflow-specific telemetry that captures the unique execution model of a workflow automation platform.

## Decision
We adopt OpenTelemetry as the sole observability framework, following Foundry's established patterns and extending them with workflow-specific telemetry.

### Tracing

**Named tracers per domain:**

| Tracer Name | Scope |
|-------------|-------|
| `orchestrator.form` | Form rendering, submission, validation |
| `orchestrator.workflow` | Workflow lifecycle: start, step execution, completion |
| `orchestrator.connector` | External system calls via connector adapters |
| `orchestrator.event` | Event publishing, routing, trigger evaluation |
| `orchestrator.engine` | State engine operations, scheduler lifecycle |
| `orchestrator.ai` | DSL generation, validation, refinement |

**Span hierarchy for a typical execution:**

```
form.submit
└── event.publish
    └── event.route
        └── workflow.start
            ├── workflow.step.condition
            ├── workflow.step.connector
            │   └── connector.execute.send_email
            ├── workflow.step.delay
            └── workflow.step.ai_generate
```

**Trace context propagation:**

| Boundary | Mechanism |
|----------|-----------|
| ConnectRPC | Automatic via OTel gRPC/Connect interceptors |
| NATS JetStream | Manual inject/extract using message headers with `propagation.TextMapCarrier` |
| State Engine | Engine-generated spans at dispatch, commit, and transition boundaries |

### Metrics

| Metric Name | Type | Labels | Purpose |
|-------------|------|--------|---------|
| `form.submissions.total` | Counter | `tenant_id`, `form_id`, `status` | Track form submission volume and success rate |
| `form.validation.errors` | Counter | `tenant_id`, `form_id`, `field` | Identify problematic form fields |
| `events.published.total` | Counter | `tenant_id`, `event_type`, `source` | Event ingestion volume |
| `events.routed.total` | Counter | `tenant_id`, `event_type`, `matched` | Trigger matching effectiveness |
| `events.routing.latency_ms` | Histogram | `tenant_id`, `event_type` | Time from event publish to workflow trigger |
| `workflows.started.total` | Counter | `tenant_id`, `workflow_id`, `trigger_type` | Workflow activation volume |
| `workflows.completed.total` | Counter | `tenant_id`, `workflow_id`, `status` | Workflow completion and failure rates |
| `workflows.duration_ms` | Histogram | `tenant_id`, `workflow_id`, `status` | End-to-end workflow execution time |
| `workflows.active` | Gauge | `tenant_id` | Current active workflow count per tenant |
| `steps.executed.total` | Counter | `tenant_id`, `step_type`, `status` | Step execution volume by type |
| `steps.duration_ms` | Histogram | `tenant_id`, `step_type` | Per-step execution time |
| `connectors.calls.total` | Counter | `tenant_id`, `connector_type`, `action`, `status` | External API call volume |
| `connectors.latency_ms` | Histogram | `tenant_id`, `connector_type`, `action` | External API response time |
| `connectors.errors.total` | Counter | `tenant_id`, `connector_type`, `error_type` | External API failure classification |
| `ai.generations.total` | Counter | `tenant_id`, `model`, `status` | AI generation request volume |
| `ai.generation.latency_ms` | Histogram | `tenant_id`, `model` | AI generation response time |
| `ai.validation.failures` | Counter | `tenant_id`, `failure_type` | AI output validation failure classification |
| `triggers.matched.total` | Counter | `tenant_id`, `event_type` | Events that matched at least one trigger |
| `triggers.filtered.total` | Counter | `tenant_id`, `event_type` | Events filtered out by trigger conditions |
| `quotas.exceeded.total` | Counter | `tenant_id`, `quota_type` | Rate limit and quota violations |
| `nats.queue.depth` | Gauge | `stream`, `consumer` | NATS consumer pending message count |

### Logging

All logging uses `util.Log(ctx)` for automatic context propagation. Every log entry in a request path includes:

| Field | Source |
|-------|--------|
| `tenant_id` | Extracted from auth claims |
| `workflow_id` | Set when workflow context is established |
| `step_id` | Set during step execution |
| `event_id` | Set during event processing |
| `connector_type` | Set during connector execution |
| `trace_id` | Automatic from OTel context |
| `span_id` | Automatic from OTel context |

**Rules:**
- Error wrapping with context at every layer boundary
- Sensitive data (credentials, PII, full payloads) is never logged
- Log levels: `DEBUG` for step-level execution detail, `INFO` for lifecycle events, `WARN` for retries and quota near-limits, `ERROR` for failures requiring investigation

## Alternatives Considered

| Option | Pros | Cons |
|--------|------|------|
| OpenTelemetry (chosen) | Vendor-neutral; consistent with Foundry; unified traces/metrics/logs; broad ecosystem | SDK maturity varies by language; configuration complexity |
| Datadog APM native | Excellent UI; automatic instrumentation; built-in alerting | Vendor lock-in; cost at scale; diverges from Foundry patterns |
| Prometheus + Jaeger + ELK | Mature individual tools; open source | Three separate systems to operate; no unified correlation; manual trace context propagation |
| Custom instrumentation | Tailored to exact needs; no dependency overhead | Maintenance burden; no ecosystem tooling; inconsistent with Foundry |

## Rationale
1. Consistency with Foundry's established observability patterns reduces cognitive overhead for developers working across the ecosystem.
2. The state engine generates spans at every transition boundary (dispatch, commit, validation, mapping), providing fine-grained tracing across the execution path.
3. Business metrics (form submissions, workflow completions, connector calls) drive product decisions around plan limits, feature adoption, and reliability targets.
4. Correlated traces from form submission through workflow completion enable rapid diagnosis of failures in a multi-layered execution model.
5. Vendor-neutral OTel allows the platform to switch between observability backends (Grafana, Datadog, Honeycomb) without changing application code.

## Consequences

**Positive:**
- Full request tracing from API ingestion through workflow completion to external connector calls
- Business metrics enable data-driven plan tier tuning and feature development
- Consistent patterns across the ecosystem reduce onboarding time
- Vendor-neutral backend choice preserves operational flexibility

**Negative:**
- Trace context must be manually propagated across NATS message boundaries
- High-cardinality labels (per-workflow, per-tenant) require careful metric design to avoid storage explosion
- Metric instrumentation adds code to every layer; must be maintained as the codebase evolves
- OTel SDK configuration adds startup complexity

## Implementation Notes

### NATS Trace Context Injection

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/propagation"
    "github.com/nats-io/nats.go"
)

type natsHeaderCarrier nats.Header

func (c natsHeaderCarrier) Get(key string) string   { return nats.Header(c).Get(key) }
func (c natsHeaderCarrier) Set(key, val string)      { nats.Header(c).Set(key, val) }
func (c natsHeaderCarrier) Keys() []string           { /* enumerate keys */ }

func publishWithTrace(ctx context.Context, js nats.JetStreamContext, subject string, data []byte) error {
    msg := &nats.Msg{
        Subject: subject,
        Data:    data,
        Header:  nats.Header{},
    }

    propagator := otel.GetTextMapPropagator()
    propagator.Inject(ctx, natsHeaderCarrier(msg.Header))

    _, err := js.PublishMsg(msg)
    return err
}
```

### Metrics Struct Pattern

```go
// internal/telemetry/metrics.go

type Metrics struct {
    FormSubmissions     metric.Int64Counter
    EventsPublished    metric.Int64Counter
    EventsRouted       metric.Int64Counter
    EventRoutingLatency metric.Float64Histogram
    WorkflowsStarted   metric.Int64Counter
    WorkflowsCompleted metric.Int64Counter
    WorkflowDuration   metric.Float64Histogram
    WorkflowsActive    metric.Int64UpDownCounter
    StepsExecuted      metric.Int64Counter
    StepDuration       metric.Float64Histogram
    ConnectorCalls     metric.Int64Counter
    ConnectorLatency   metric.Float64Histogram
    ConnectorErrors    metric.Int64Counter
    AIGenerations      metric.Int64Counter
    AILatency          metric.Float64Histogram
    QuotasExceeded     metric.Int64Counter
}

// NewMetrics creates all metric instruments at startup.
// The returned Metrics struct is injected into all service layers.
func NewMetrics(meter metric.Meter) (*Metrics, error) {
    m := &Metrics{}
    var err error

    m.FormSubmissions, err = meter.Int64Counter("form.submissions.total",
        metric.WithDescription("Total form submissions"),
    )
    if err != nil {
        return nil, err
    }

    // ... remaining instruments ...

    return m, nil
}
```

### Span Naming Convention

All spans follow the pattern `{domain}.{operation}`:

- `form.submit`, `form.validate`, `form.render`
- `event.publish`, `event.route`, `event.evaluate_trigger`
- `workflow.start`, `workflow.step.condition`, `workflow.step.connector`
- `connector.execute.send_email`, `connector.execute.create_ticket`
- `ai.generate`, `ai.validate`, `ai.refine`
