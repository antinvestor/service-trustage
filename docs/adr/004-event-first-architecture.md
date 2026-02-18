# ADR-004: Event-First Architecture with Typed Event Bus

## Status

Accepted

## Context

The Orchestrator must handle multiple event sources:

1. **Form submissions**: A user submits a Stawi.dev form
2. **Webhook ingestion**: An external system sends a webhook payload
3. **API calls**: A client explicitly triggers a workflow via the API
4. **Scheduled triggers**: A cron schedule fires at a configured time
5. **Internal events**: A workflow completes and triggers another workflow
6. **Cross-service events**: Another Stawi.dev service (Foundry, etc.) emits an event

Each event source may trigger zero, one, or many workflows depending on tenant configuration. The mapping between events and workflows must be:

- **Decoupled**: Event producers do not know about workflow consumers
- **Filterable**: Trigger bindings can include CEL expressions to filter events
- **Fan-out capable**: One event can trigger multiple workflows
- **Replayable**: Events can be re-processed to re-trigger workflows
- **Auditable**: Every event is persisted with full context for debugging and compliance

## Decision

Adopt an **event-first architecture** where every external stimulus is normalized into a typed event before any workflow processing occurs.

### Event Shape

```json
{
  "id": "evt_01HQ3K5X7Y...",
  "type": "form.submitted.frm_01HQ...",
  "tenant_id": "ten_01HQ...",
  "partition_id": "prt_01HQ...",
  "source": "forms",
  "source_id": "frm_01HQ...",
  "payload": { "...source-specific data" },
  "metadata": { "ip": "...", "user_agent": "...", "headers": {} },
  "created_at": "2025-01-15T10:30:00Z"
}
```

### Event Flow

Events flow through three stages:

```
1. PERSIST  -> event_log table (audit trail, replay source)
2. PUBLISH  -> NATS JetStream subject: orchestrator.events.{tenant_id}.{event_type}
3. ROUTE    -> EventRouterWorker matches events to trigger_bindings, creates workflow instances via state engine
```

### Event Type Naming Convention

| Pattern | Example | Source |
|---------|---------|--------|
| `form.submitted.{form_id}` | `form.submitted.frm_01HQ3K5X7Y` | Form submission |
| `webhook.received.{hook_id}` | `webhook.received.whk_01HQ8M2N4P` | Webhook ingestion |
| `schedule.fired.{schedule_id}` | `schedule.fired.sch_01HQBR6T9W` | Cron schedule |
| `workflow.completed.{workflow_def_id}` | `workflow.completed.wfd_01HQ5P8V3Z` | Workflow completion |
| `connector.callback.{connector_id}` | `connector.callback.con_01HQDR4K7M` | Async connector callback |
| `external.{service}.{event_name}` | `external.foundry.build_complete` | Cross-service event |

### Trigger Bindings

Trigger bindings are stored in PostgreSQL and define the mapping from events to workflows:

```sql
CREATE TABLE trigger_bindings (
    id          TEXT PRIMARY KEY,
    tenant_id   TEXT NOT NULL,
    event_type  TEXT NOT NULL,
    workflow_id TEXT NOT NULL,
    workflow_version INT NOT NULL DEFAULT 0,
    filter_expr TEXT,          -- CEL expression, NULL means "match all"
    priority    INT NOT NULL DEFAULT 0,
    enabled     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (tenant_id, event_type, workflow_id)
);
```

When an event arrives, the router queries all enabled trigger bindings for the `(tenant_id, event_type)` pair, evaluates any `filter_expr` CEL expressions against the event payload, and creates a workflow instance via the state engine for each matching binding (ordered by priority).

## Alternatives Considered

| Option | Pros | Cons | Verdict |
|--------|------|------|---------|
| **Event bus + trigger bindings** | Full decoupling between producers and consumers. Fan-out is first-class. Filter expressions enable fine-grained control. Event log enables replay. Any source uses the same pipeline. | Must ensure reliable publishing (transactional outbox). Routing adds latency (<100ms). Trigger binding cache required for performance. | **Chosen** |
| **Direct form-to-workflow binding** | Simple. Low latency. No event bus required. | No fan-out (one form, one workflow). No filtering. Cannot support non-form event sources. Tight coupling between forms and workflows. | Rejected |
| **Workflow polling** | No event infrastructure needed. Workflows check for new data on schedule. | High latency (polling interval). Wasted resources when no events. Does not scale with event volume. Poor user experience. | Rejected |
| **Database triggers + LISTEN/NOTIFY** | Uses existing PostgreSQL. Low latency for DB changes. | Not durable (missed if listener disconnected). No replay. No fan-out beyond PostgreSQL clients. Does not handle external events. Single point of failure. | Rejected |

## Rationale

1. **Decoupling is the foundation of extensibility.** By normalizing every stimulus into a typed event, new event sources (IoT sensors, payment webhooks, third-party integrations) plug into the same pipeline without changes to the workflow engine, trigger system, or existing event sources.

2. **Trigger bindings are the user's control surface.** Users configure which events trigger which workflows, with optional CEL filters for fine-grained control. This is the primary configuration mechanism exposed in the UI.

3. **NATS JetStream provides durable subscriptions, back-pressure, subject routing, and replay.** JetStream's subject hierarchy (`orchestrator.events.{tenant_id}.{event_type}`) enables efficient routing. Durable consumers ensure no events are lost. Back-pressure prevents the router from being overwhelmed.

4. **The event log enables replay without re-publishing.** When a workflow definition is updated or a new trigger binding is created, historical events can be replayed from the `event_log` table to backfill workflow executions.

5. **Fan-out is first-class.** One form submission can trigger a welcome email workflow, a CRM sync workflow, and an analytics pipeline -- each configured independently via trigger bindings.

## Consequences

**Positive:**

- Any event source (forms, webhooks, schedules, cross-service) uses the same pipeline
- Flexible trigger composition: one event to many workflows, CEL filtering, priority ordering
- Full audit trail via `event_log` table with 100% event capture
- Event replay enables backfill, debugging, and disaster recovery
- NATS JetStream provides durable delivery, back-pressure, and subject-based routing
- Trigger bindings are a clean, user-facing configuration surface

**Negative:**

- Must implement reliable publishing via transactional outbox pattern to avoid event loss
- Event routing adds <100ms latency between event occurrence and workflow start
- Trigger binding cache is required to avoid database queries on every event
- Two persistence layers for events (PostgreSQL event_log + NATS JetStream) must stay consistent

## Implementation Notes

### NATS JetStream Configuration

```
Stream:     orchestrator-events
Subjects:   orchestrator.events.>
Retention:  Limits (30 days, 50GB)
Storage:    File
Replicas:   3 (production), 1 (development)
Discard:    Old (discard oldest when limits reached)
```

Consumer for the event router:

```
Consumer:   event-router
Durable:    yes
AckPolicy:  Explicit
MaxDeliver: 5
AckWait:    10s
FilterSubject: orchestrator.events.>
```

### Event Router

The `EventRouterWorker` is a NATS subscriber that:

1. Receives events from the `orchestrator.events.>` subject
2. Loads matching trigger bindings from Valkey cache (falling back to PostgreSQL)
3. Evaluates CEL filter expressions against the event payload
4. Creates a workflow instance via the state engine for each matching trigger binding
5. ACKs the NATS message after all instances are created (or NAKs on failure)

### Trigger Binding Cache

To avoid querying PostgreSQL on every event, trigger bindings are cached in Valkey:

```
Key:    triggers:{tenant_id}:{event_type}
Value:  JSON array of trigger bindings
TTL:    60 seconds
```

Cache is invalidated (deleted) on any trigger binding CRUD operation for the affected `(tenant_id, event_type)` pair. On cache miss, the router queries PostgreSQL and populates the cache.

### Reliable Publishing (Transactional Outbox)

To ensure events are not lost between database persistence and NATS publishing:

1. Within a single database transaction:
   - Insert the source record (e.g., `form_submissions`)
   - Insert into `event_log` with `event_published = false`
2. A background outbox worker polls for unpublished events:
   ```sql
   SELECT * FROM event_log
   WHERE event_published = false
   ORDER BY created_at ASC
   LIMIT 100
   FOR UPDATE SKIP LOCKED;
   ```
3. For each unpublished event, publish to NATS JetStream
4. On successful publish ACK, update `event_published = true`
5. On failure, the event remains unpublished and is retried on next poll cycle

The outbox worker runs on a configurable interval (default: 100ms) and processes events in batches.

### Future Event Sources

| Source | Event Type Pattern | Trigger |
|--------|-------------------|---------|
| Forms | `form.submitted.{form_id}` | User submits a Stawi.dev form |
| Webhooks | `webhook.received.{hook_id}` | External system POSTs to webhook endpoint |
| Cron schedules | `schedule.fired.{schedule_id}` | Scheduled time arrives |
| Foundry builds | `external.foundry.build_complete` | Foundry build pipeline completes |
| Payments | `external.payments.payment_received` | Payment processor confirms charge |
| Connector callbacks | `connector.callback.{connector_id}` | Async connector operation completes |
| Manual triggers | `manual.triggered.{workflow_def_id}` | User manually starts a workflow |
| Workflow completion | `workflow.completed.{workflow_def_id}` | A workflow execution finishes |
| Data changes | `data.changed.{entity_type}` | A database record is created/updated/deleted |
| IoT sensors | `external.iot.{device_type}.{event}` | IoT device reports a reading or alert |
