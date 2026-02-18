# ADR-012: Event Schema Evolution and Compatibility

## Status

Accepted

## Context

Events are the primary input to the Orchestrator. Every form submission generates an event whose payload contains the submitted field data as a JSON object. Workflow trigger conditions and step expressions reference specific field names within these payloads (e.g., `payload.email`, `payload.order_total`). This creates a tight coupling between form schema and workflow logic.

Over time, form schemas change. Fields are added to capture new information, removed when no longer relevant, and renamed to improve clarity. A form that started with a `name` field may later split it into `first_name` and `last_name`. A field originally typed as a string may need to become a number. These changes are driven by business needs and happen independently of the workflows that consume the events.

This creates a compatibility problem at multiple levels. Active workflows must continue to function when form fields change beneath them. Historical events stored in `event_log` must remain replayable for audit and debugging. The DSL validator must provide useful feedback when workflow conditions reference fields that no longer exist in the current form schema, without blocking legitimate use cases where workflows intentionally handle multiple schema versions. The system must balance strictness (catching errors early) with flexibility (not breaking working workflows when forms evolve).

## Decision

Adopt an immutable-event, forward-compatible schema evolution strategy with the following rules:

### Rule 1: Events Are Immutable

Once an event is created and stored in `event_log`, its payload is never modified. The payload represents the exact data submitted at the exact moment of submission. No backfilling, no migration, no transformation after the fact.

### Rule 2: Schema Version in Metadata

Every event carries the form schema version in its metadata. This enables version-aware workflow logic when strict matching is needed.

**Event payload structure:**

```json
{
  "id": "evt_01HQXYZ...",
  "type": "form.submitted",
  "tenant_id": "tenant_abc",
  "payload": {
    "email": "user@example.com",
    "first_name": "Jane",
    "last_name": "Doe",
    "order_total": 149.99
  },
  "metadata": {
    "form_id": "form_contact_v3",
    "form_version": 7,
    "schema_fields": ["email", "first_name", "last_name", "order_total"],
    "source": "web",
    "ip": "203.0.113.42",
    "submitted_at": "2025-03-15T10:30:00Z"
  }
}
```

### Rule 3: Graceful Missing-Field Handling

Workflows must handle missing fields without failing. The CEL expression engine provides two mechanisms:

- `has(payload.field)` for existence checks before access.
- `payload.field.orValue("fallback")` for default values on possibly-missing fields.

### Rule 4: Soft Validation Warnings

The DSL validator warns (but does not error) when a workflow condition references a field name not present in the current form schema. This catches likely typos and stale references without blocking workflows that intentionally handle multiple schema versions.

### Compatibility Guidelines for Workflow Authors

| Pattern | CEL Expression | Use Case |
|---------|---------------|----------|
| Check field exists | `has(payload.email)` | Guard against missing fields after form change |
| Default value | `payload.priority.orValue("normal")` | Handle field that may not exist in older events |
| Version-specific logic | `metadata.form_version >= 5 ? payload.first_name : payload.name` | Bridge across a field rename |
| Pin to form version | `metadata.form_version == 7` | Only process events from a specific schema version |

### Schema Change Impact Matrix

| Change Type | Impact on Existing Workflows | Mitigation |
|-------------|------------------------------|------------|
| Add field | None (new field is ignored by existing workflows) | No action needed |
| Remove field | Workflows referencing removed field get nil/zero value | Add `has()` guard or default value |
| Rename field | Workflows referencing old name get nil/zero value | Version-specific expression or update workflow |
| Change field type | Type mismatch in expressions | CEL type coercion or version-specific logic |

## Alternatives Considered

| Option | Pros | Cons |
|--------|------|------|
| **Immutable events + soft validation (chosen)** | Simple model. Events are truthful records. Workflows handle evolution explicitly. No data migration needed. Full audit trail preserved. | Puts burden on workflow authors to handle schema changes. Can lead to subtle bugs if authors forget `has()` guards. |
| **Schema registry with strict validation** | Catches all mismatches at publish time. Strong type safety. | Heavyweight infrastructure (Confluent-style registry). Blocks event publishing when schema evolves. Incompatible with rapid form iteration. |
| **Event payload migration on read** | Workflows always see latest schema. No `has()` guards needed. | Destroys audit trail (what was actually submitted?). Complex migration logic. Retried states may see different data than original execution. |
| **Event versioned endpoints** | Each schema version has its own event type. Clean separation. | Explosion of event types. Workflows must subscribe to multiple types. Routing complexity grows quadratically. |

## Rationale

1. **Immutable events are the foundation of auditability.** The `event_log` must answer "what exactly was submitted?" for compliance, debugging, and dispute resolution. Mutating events after the fact destroys this guarantee.

2. **Form version in metadata enables version-aware workflow logic without infrastructure overhead.** A simple integer version and field list in the metadata gives workflow authors everything they need to write version-conditional expressions, without requiring a separate schema registry service.

3. **Graceful missing-field handling prevents cascade failures when forms evolve.** CEL's `has()` and `orValue()` are lightweight, expressive, and familiar to developers. A workflow that checks `has(payload.email)` before accessing it will not fail regardless of which form version generated the event.

4. **Soft validation warnings strike the right balance between safety and flexibility.** Hard errors would block legitimate use cases (workflows intentionally handling multiple schema versions). No validation at all would miss typos and stale references. Warnings surface likely issues without blocking saves.

5. **Stable event data is required for consistent processing.** If event payloads were migrated on read, a retried workflow state could see different data than the original execution, producing incorrect results.

## Consequences

**Positive:**

- Events are truthful, immutable records suitable for audit and compliance
- No event migration infrastructure to build or maintain
- Workflows can explicitly handle multiple schema versions via CEL expressions
- DSL validator catches likely issues without blocking legitimate patterns
- Retry correctness is preserved (events never change)
- Form authors can iterate on schemas without coordinating with workflow authors

**Negative:**

- Workflow authors must proactively handle schema evolution (add `has()` guards)
- Subtle bugs possible if workflows assume fields always exist
- Soft validation warnings may be ignored, leading to runtime failures
- No automatic schema compatibility checking (e.g., "will this form change break any workflows?")
- Historical events with old schemas accumulate indefinitely in `event_log`

## Implementation Notes

### Event Storage Schema

```sql
CREATE TABLE event_log (
    id              TEXT PRIMARY KEY,       -- ULID
    tenant_id       TEXT NOT NULL,
    event_type      TEXT NOT NULL,
    payload         JSONB NOT NULL,         -- Raw form data, never modified
    metadata        JSONB NOT NULL,         -- form_id, form_version, schema_fields, source, ip, submitted_at
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
);

CREATE INDEX idx_event_log_tenant_type ON event_log (tenant_id, event_type);
CREATE INDEX idx_event_log_created ON event_log (tenant_id, created_at DESC);
```

### CEL Environment Configuration

The CEL environment is configured with the `has()` macro enabled by default. Custom helper functions are registered for common patterns:

```go
env, err := cel.NewEnv(
    cel.Variable("payload", cel.DynType),
    cel.Variable("metadata", cel.MapType(cel.StringType, cel.DynType)),
    cel.Variable("trigger", cel.DynType),
)
```

### DSL Validator Warning Example

When a workflow condition references `payload.phone` but the current form schema does not include a `phone` field:

```json
{
  "warnings": [
    {
      "step_id": "check_phone",
      "field": "condition",
      "message": "Expression references 'payload.phone' but field 'phone' is not in the current schema for form 'form_contact_v3' (version 7). This may cause nil access at runtime. Consider adding a has(payload.phone) guard.",
      "severity": "warning"
    }
  ]
}
```

### Extensibility

| Capability | Timeline | Description |
|------------|----------|-------------|
| Schema diff on form publish | Phase 3 | Show which active workflows are affected when a form schema changes |
| Auto-guard insertion | Phase 4 | Suggest or auto-insert `has()` guards when validator detects missing field references |
| Event replay with schema context | Phase 3 | Replay UI shows which schema version generated each event |
| Breaking change detection | Phase 4 | Block form publish if it would break workflows without `has()` guards (opt-in strict mode) |
