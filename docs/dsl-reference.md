# Orchestrator DSL Reference

**Version:** 1.0
**Audience:** Workflow authors, AI prompt context, visual builder developers

---

## Overview

The Orchestrator DSL (Domain-Specific Language) is a JSON format for defining workflow automations. It is:

- **Declarative**: Describes what should happen, not how
- **Not executable**: An intermediate representation interpreted at runtime
- **Versionable**: Every document has a version envelope
- **Validatable**: Static analysis catches errors before execution
- **AI-friendly**: JSON structured output produces reliable DSL documents

Current runtime support: the grammar accepts all documented step types, and the
runtime executes `call`, `delay`, `if`, `sequence`, `parallel`, `foreach`,
`signal_wait`, and `signal_send` with durable waiting, branch reconciliation,
and signal delivery.

---

## Document Structure

```json
{
  "version": "1.0",
  "name": "Workflow Name (required)",
  "description": "Optional description",
  "input": {
    "field_name": "type_hint"
  },
  "config": {},
  "timeout": "30d",
  "on_error": {
    "strategy": "fail",
    "fallback": []
  },
  "triggers": [],
  "signals": [],
  "steps": []
}
```

### Top-Level Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `version` | string | Yes | DSL version (currently `"1.0"`) |
| `name` | string | Yes | Human-readable workflow name |
| `description` | string | No | Workflow description |
| `input` | object | No | Expected input fields with type hints |
| `config` | object | No | Workflow-level configuration values |
| `timeout` | string | No | Maximum workflow duration (e.g., `"30d"`, `"24h"`, `"30m"`) |
| `on_error` | object | No | Workflow-level error handling policy |
| `triggers` | array | No | Events that start an instance (see below) |
| `signals` | array | No | Events delivered to running instances as signals (see below) |
| `steps` | array | Yes | Ordered list of step objects |

### Triggers and Signal Routes

`triggers` start instances from ingested events; `signals` deliver ingested
events to instances that are already running. Both are part of the version:
they take effect when the version is activated.

```json
"triggers": [
  {"event_type": "proposal.created",
   "filter": "payload.amount > 0",
   "key": "payload.proposal_id"}
],
"signals": [
  {"event_type": "proposal.cancelled", "key": "payload.proposal_id"},
  {"event_type": "intent.finalized", "signal_name": "finalized",
   "instance": "payload.meta.instance_id"}
]
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `triggers[].event_type` | string | Yes | Event type that starts an instance |
| `triggers[].filter` | CEL (bool) | No | Over `payload`; the trigger fires only when it holds |
| `triggers[].key` | CEL | No | Over the trigger `payload`; the value is stored on the instance as its correlation key |
| `signals[].event_type` | string | Yes | Event type to deliver as a signal |
| `signals[].signal_name` | string | No | Signal name to deliver; defaults to `event_type` |
| `signals[].filter` | CEL (bool) | No | Over `payload`; the route applies only when it holds |
| `signals[].key` | CEL | One of `key`/`instance` | Delivers to running instances whose correlation key equals the value |
| `signals[].instance` | CEL | One of `key`/`instance` | Delivers to the running instance whose id is the value |

**Lifecycle.** Activating version V of workflow W deactivates the trigger
bindings of W's other versions and upserts one binding per trigger of V, plus
V's signal routes. Archiving V deactivates V's bindings but keeps its signal
routes, so instances still running on V keep receiving signals. Submitting a
DSL identical to an ACTIVE version (`CreateWorkflow`) re-asserts its bindings
and routes. All of this is idempotent. The event router ignores a binding
whose version is not ACTIVE.

**Correlation key.** A key expression's value becomes a string: strings as
is, numbers and booleans formatted (`42` → `"42"`). A null, list, map or an
evaluation error gives an empty key (logged); an empty key never matches a
key route. Branch children of `parallel`/`foreach` inherit the key.

**Delivery.** A route only delivers to running instances of exactly its own
workflow name and version. An instance route ignores ids of other workflows,
versions or tenants, and finished instances. A signal that arrives before the
instance reaches its `signal_wait` is buffered and consumed when the wait
starts. Delivery is idempotent per (event id, instance id, signal name): a
redelivered event never buffers a second signal. Key fan-out is capped at
the router's binding limit (`EVENT_ROUTER_BINDING_LIMIT`, default 200).

**Validation.** `event_type` is required; `filter` must compile to a bool;
`key` and `instance` must compile; a route sets exactly one of `key` and
`instance`; a `key` route requires every trigger to declare `key`; every
route's signal name must be waited on by some `signal_wait` in the workflow
(including nested and `on_error.fallback` steps).

`triggers` replaces the free-form `config.triggers` convention. Documents
without `triggers`/`signals` are unchanged.

### Duration Strings

Durations use Go-style format: `"30s"`, `"5m"`, `"2h"`, `"72h"`, `"7d"`, `"30d"`.

Compound durations are not supported. Use hours for multi-day durations: `"168h"` (7 days).

---

## Step Structure

Every step has a common base plus type-specific fields:

```json
{
  "id": "unique_step_id",
  "type": "call|delay|if|sequence|parallel|foreach|signal_wait|signal_send",
  "name": "Optional Human Label",
  "depends_on": "other_step_id",
  "retry": {
    "max_attempts": 3,
    "initial_interval": "1s",
    "backoff_coefficient": 2.0,
    "max_interval": "5m"
  },
  "timeout": "30s",
  "on_error": {
    "strategy": "fail|continue|retry|fallback",
    "fallback": []
  }
}
```

### Common Step Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | Yes | Unique identifier within the workflow (no duplicates across entire tree) |
| `type` | string | Yes | Step type (see below) |
| `name` | string | No | Human-readable label for display |
| `depends_on` | string | No | Step ID that must complete before this step |
| `retry` | object | No | Retry policy for this step |
| `timeout` | string | No | Maximum step duration |
| `on_error` | object | No | Error handling for this step |

### Error Handling Strategies

| Strategy | Behavior |
|----------|----------|
| `fail` | Step failure fails the workflow (default) |
| `continue` | Step failure is logged but workflow continues |
| `retry` | Step is retried according to retry policy |
| `fallback` | Execute fallback steps instead |

---

## Step Types

### `call` -- Invoke a Connector

Calls an external system through a registered connector adapter.

```json
{
  "id": "send_welcome_email",
  "type": "call",
  "name": "Send Welcome Email",
  "call": {
    "action": "email.send",
    "input": {
      "to": "{{ payload.email }}",
      "subject": "Welcome, {{ payload.name }}!",
      "body": "Thank you for signing up."
    },
    "output_var": "email_result"
  }
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `call.action` | string | Yes | Connector type (e.g., `"email.send"`, `"webhook.call"`) |
| `call.input` | object | Yes | Input fields for the connector (supports `{{ }}` templates) |
| `call.output_var` | string | No | Variable name to store the connector's response |

**Template syntax:** `{{ a.b.c }}` dot paths resolve against the step's input:
- `{{ payload.field }}` -- the step input (the trigger payload for the first step)
- `{{ field }}` -- a top-level key of the step input
- `{{ metadata.<name> }}` -- the execution context:

| Variable | Value |
|----------|-------|
| `metadata.instance_id` | The workflow instance id |
| `metadata.execution_id` | The id of this step execution |
| `metadata.workflow_name` | The workflow name |
| `metadata.workflow_version` | The workflow version |
| `metadata.correlation_key` | The instance's correlation key (empty when none) |

These override same-named keys of a `metadata` object carried in the input;
other keys of that object stay readable. The same variables are available to
`signal_send.target_workflow_id` and `signal_send.payload`. When the previous
step sets `output_var`, the engine resolves the next call's templates against
that output at commit time and leaves `{{ metadata.* }}` for the worker.

### `delay` -- Durable Wait

Pauses the workflow for a specified duration. Survives process restarts.

```json
{
  "id": "wait_3_days",
  "type": "delay",
  "name": "Wait 3 Days",
  "delay": {
    "duration": "72h"
  }
}
```

Or wait until a computed timestamp:

```json
{
  "id": "wait_until_deadline",
  "type": "delay",
  "delay": {
    "until": "now + duration('48h')"
  }
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `delay.duration` | string | One of duration/until | Fixed duration string |
| `delay.until` | string | One of duration/until | CEL expression evaluating to a timestamp |

### `if` -- Conditional Branching

Evaluates a CEL expression and executes one of two branches.

```json
{
  "id": "check_amount",
  "type": "if",
  "if": {
    "expr": "payload.amount > 100",
    "then": [
      {
        "id": "notify_sales",
        "type": "call",
        "call": { "action": "slack.post", "input": { "message": "High-value lead: {{ payload.email }}" } }
      }
    ],
    "else": [
      {
        "id": "add_to_drip",
        "type": "call",
        "call": { "action": "email.send", "input": { "to": "{{ payload.email }}", "subject": "Thanks for your interest" } }
      }
    ]
  }
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `if.expr` | string | Yes | CEL expression that evaluates to boolean |
| `if.then` | array | Yes | Steps to execute if expression is true |
| `if.else` | array | No | Steps to execute if expression is false |

### `sequence` -- Ordered Execution

Executes sub-steps in order. Useful for grouping steps with shared error handling.

```json
{
  "id": "onboarding_sequence",
  "type": "sequence",
  "name": "Onboarding Flow",
  "on_error": { "strategy": "continue" },
  "sequence": {
    "steps": [
      { "id": "step_a", "type": "call", "call": { "action": "email.send", "input": { "to": "{{ payload.email }}" } } },
      { "id": "step_b", "type": "delay", "delay": { "duration": "24h" } },
      { "id": "step_c", "type": "call", "call": { "action": "email.send", "input": { "to": "{{ payload.email }}" } } }
    ]
  }
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `sequence.steps` | array | Yes | Ordered list of sub-steps |

### `parallel` -- Concurrent Execution (v1.1)

Executes sub-steps concurrently.

```json
{
  "id": "notify_all",
  "type": "parallel",
  "parallel": {
    "steps": [
      { "id": "email_user", "type": "call", "call": { "action": "email.send", "input": {} } },
      { "id": "slack_team", "type": "call", "call": { "action": "slack.post", "input": {} } },
      { "id": "update_crm", "type": "call", "call": { "action": "crm.upsert", "input": {} } }
    ],
    "wait_all": true
  }
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `parallel.steps` | array | Yes | Steps to execute concurrently |
| `parallel.wait_all` | boolean | No | If true (default), wait for all steps. If false, continue when first completes. |

### `foreach` -- Iteration (v1.1)

Iterates over a list and executes sub-steps for each item.

```json
{
  "id": "process_recipients",
  "type": "foreach",
  "foreach": {
    "items": "payload.recipients",
    "item_var": "recipient",
    "index_var": "i",
    "max_concurrency": 5,
    "steps": [
      {
        "id": "send_to_recipient",
        "type": "call",
        "call": {
          "action": "email.send",
          "input": { "to": "{{ recipient.email }}", "subject": "Hello {{ recipient.name }}" }
        }
      }
    ]
  }
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `foreach.items` | string | Yes | CEL expression evaluating to a list |
| `foreach.item_var` | string | No | Variable name for current item (default: `"item"`) |
| `foreach.index_var` | string | No | Variable name for current index (default: `"index"`) |
| `foreach.max_concurrency` | int | No | Max parallel iterations (0 = sequential) |
| `foreach.steps` | array | Yes | Steps to execute per item |

### `signal_wait` -- Wait for External Signal (v1.2)

Pauses the workflow until an external signal is received. Used for human approvals and external callbacks.

```json
{
  "id": "wait_for_approval",
  "type": "signal_wait",
  "signal_wait": {
    "signal_name": "manager_approval",
    "timeout": "48h",
    "output_var": "approval"
  }
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `signal_wait.signal_name` | string | Yes | Signal channel name |
| `signal_wait.timeout` | string | No | Maximum wait time before timeout |
| `signal_wait.output_var` | string | No | Variable to store signal payload |

### `signal_send` -- Send Signal to Workflow (v1.2)

Sends a signal to another running workflow.

```json
{
  "id": "notify_parent",
  "type": "signal_send",
  "signal_send": {
    "target_workflow_id": "{{ vars.parent_workflow_id }}",
    "signal_name": "child_completed",
    "payload": { "result": "{{ vars.processing_result }}" }
  }
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `signal_send.target_workflow_id` | string | Yes | Target workflow ID (supports templates) |
| `signal_send.signal_name` | string | Yes | Signal channel name |
| `signal_send.payload` | object | No | Signal payload (supports templates) |

---

## CEL Expression Reference

CEL (Common Expression Language) is used in `if` conditions, `foreach` items, `delay` until, and trigger filter expressions.

### Available Variables

| Variable | Type | Description |
|----------|------|-------------|
| `payload` | map | Event payload data |
| `metadata` | map | Event metadata |
| `vars` | map | Accumulated step output variables |
| `env` | map | Workflow config values |
| `now` | timestamp | Current time |
| `item` | any | Current foreach item (only in foreach body) |
| `index` | int | Current foreach index (only in foreach body) |

### Operators

| Category | Operators |
|----------|----------|
| Comparison | `==`, `!=`, `<`, `<=`, `>`, `>=` |
| Logical | `&&`, `\|\|`, `!` |
| Arithmetic | `+`, `-`, `*`, `/`, `%` |
| Ternary | `condition ? true_value : false_value` |
| Membership | `value in list` |
| Field access | `object.field`, `map["key"]` |

### Built-in Functions

| Function | Example | Description |
|----------|---------|-------------|
| `size()` | `payload.items.size()` | Length of list or string |
| `contains()` | `payload.name.contains("test")` | Substring check |
| `startsWith()` | `payload.email.startsWith("admin")` | Prefix check |
| `endsWith()` | `payload.email.endsWith(".com")` | Suffix check |
| `matches()` | `payload.phone.matches("^\\+[0-9]+$")` | Regex match |
| `int()` | `int(payload.count)` | Type conversion |
| `string()` | `string(payload.id)` | Type conversion |
| `timestamp()` | `timestamp("2026-01-01T00:00:00Z")` | Parse timestamp |
| `duration()` | `duration("24h")` | Parse duration |

### Custom Functions

| Function | Example | Description |
|----------|---------|-------------|
| `has()` | `has(payload, "email")` | Check if map has key |
| `default()` | `default(payload.name, "Unknown")` | Fallback for missing values |
| `len()` | `len(payload.items)` | List or string length |
| `format_time()` | `format_time(now, "2006-01-02")` | Format timestamp |

### Expression Examples

```
// Simple condition
payload.amount > 100

// Compound condition
payload.region == "EU" && payload.amount > 1000

// List membership
payload.country in ["DE", "FR", "IT", "ES"]

// Field existence check
has(payload, "company_size") && payload.company_size > 50

// String matching
payload.email.endsWith("@company.com")

// List filtering
payload.items.filter(item, item.price > 10).size() >= 3

// Ternary
payload.priority == "high" ? "urgent-queue" : "standard-queue"

// Temporal operations
now - metadata.created_at > duration("24h")

// Default values
default(payload.category, "general")
```

---

## Complete Example

A lead capture workflow that sends a welcome email, waits 3 days, checks if the email was opened, and sends a follow-up or reminder accordingly.

```json
{
  "version": "1.0",
  "name": "Lead Capture Workflow",
  "description": "Welcome email sequence with 3-day follow-up",
  "input": {
    "email": "string",
    "name": "string",
    "company": "string"
  },
  "timeout": "30d",
  "steps": [
    {
      "id": "send_welcome",
      "type": "call",
      "name": "Send Welcome Email",
      "call": {
        "action": "email.send",
        "input": {
          "to": "{{ payload.email }}",
          "subject": "Welcome, {{ payload.name }}!",
          "template": "welcome_v2",
          "data": {
            "name": "{{ payload.name }}",
            "company": "{{ payload.company }}"
          }
        },
        "output_var": "welcome_result"
      },
      "retry": {
        "max_attempts": 3,
        "initial_interval": "5s",
        "backoff_coefficient": 2.0
      }
    },
    {
      "id": "notify_sales_team",
      "type": "call",
      "name": "Notify Sales on Slack",
      "call": {
        "action": "webhook.call",
        "input": {
          "url": "{{ env.slack_webhook_url }}",
          "method": "POST",
          "body": {
            "text": "New lead: {{ payload.name }} ({{ payload.company }}) - {{ payload.email }}"
          }
        }
      },
      "on_error": { "strategy": "continue" }
    },
    {
      "id": "wait_3_days",
      "type": "delay",
      "name": "Wait 3 Days",
      "delay": { "duration": "72h" }
    },
    {
      "id": "check_engagement",
      "type": "if",
      "name": "Check Email Engagement",
      "if": {
        "expr": "has(vars, 'welcome_result') && vars.welcome_result.opened == true",
        "then": [
          {
            "id": "send_engaged_followup",
            "type": "call",
            "name": "Send Engaged Follow-up",
            "call": {
              "action": "email.send",
              "input": {
                "to": "{{ payload.email }}",
                "subject": "Great to see you, {{ payload.name }}!",
                "template": "followup_engaged"
              }
            }
          }
        ],
        "else": [
          {
            "id": "send_reminder",
            "type": "call",
            "name": "Send Reminder",
            "call": {
              "action": "email.send",
              "input": {
                "to": "{{ payload.email }}",
                "subject": "Did you miss our welcome email?",
                "template": "followup_reminder"
              }
            }
          }
        ]
      }
    }
  ]
}
```

---

## Validation Rules

Before a workflow definition is stored, the validator checks:

1. **Schema compliance**: Required fields present, valid step types, valid duration strings
2. **ID uniqueness**: No duplicate step IDs across the entire step tree
3. **Reference integrity**: `depends_on` references existing step IDs, `call.action` references registered connectors
4. **Cycle detection**: No circular dependencies in step graph
5. **Reachability**: All steps are reachable from the root
6. **Expression validity**: All CEL expressions compile without error
7. **Template validity**: All `{{ }}` references resolve against declared input and known output_vars
8. **Timeout consistency**: Step timeouts do not exceed workflow timeout
9. **Routing**: `triggers` and `signals` follow the rules in [Triggers and Signal Routes](#triggers-and-signal-routes)

Validation errors are returned with the step ID and a human-readable message.

---

## Version Compatibility

| Version | Step Types | Status |
|---------|-----------|--------|
| `1.0` | call, delay, if, sequence | Active |
| `1.1` | + parallel, foreach | Active |
| `1.2` | + signal_wait, signal_send | Active |
| `1.3` | + sub_workflow, switch | Planned |
| `2.0` | Reserved for breaking changes | Future |

Minor versions are backward compatible. A v1.0 document executes unchanged on a system supporting v1.2. Unknown fields in known versions are ignored. Unknown step types cause validation failure.
