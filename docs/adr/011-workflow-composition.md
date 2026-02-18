# ADR-011: Workflow Composition and Sub-Workflows

## Status

Accepted

## Context

As workflows grow in complexity, users increasingly need to reuse common sequences of steps across multiple workflow definitions. A marketing team, for example, may have a standard "send email and wait for open" sequence used in a dozen different workflows. Today, that sequence must be copy-pasted into each definition, and any change to the shared logic requires updating every copy individually. This duplication is error-prone, difficult to maintain, and fundamentally limits how users think about building automation.

Beyond simple reuse, composition unlocks structural patterns that are impossible with flat step lists. Users need to trigger workflows from workflows, fan-out to a dynamic number of sub-workflows based on runtime data (e.g., process each line item in an order), and build hierarchical automations where a parent orchestrates multiple children. These patterns are essential for enterprise-grade workflow automation.

The `sub_workflow` step type, planned for Phase 3+, must address three concerns simultaneously: reuse (define once, invoke many times), composition (build complex flows from simple building blocks), and execution lifecycle management (what happens to children when parents fail or are cancelled). The design must integrate cleanly with the existing DSL interpreter, state engine, and multi-tenant isolation guarantees.

## Decision

Implement workflow composition via engine-managed child instances. A new step type `sub_workflow` is added to the DSL interpreter with the following fields:

- **workflow_id**: Reference to an existing workflow definition (by ID).
- **workflow_version**: Specific version of the target workflow to invoke.
- **input**: Object with template resolution, mapping parent context into child input.
- **output_var**: Variable name to store the child workflow result in the parent context.
- **execution**: Execution mode governing parent-child lifecycle coupling.
- **on_failure**: Failure propagation strategy.

### Execution Modes

| Mode | Behavior |
|------|----------|
| `sync` | Parent blocks until child completes. Child result is stored in `output_var`. Parent timeout includes child execution time. |
| `async` | Parent continues immediately after starting the child. No result is returned to parent. Child runs independently but is still tied to parent lifecycle. |
| `detached` | Child is fully independent of parent lifecycle. Parent cancellation does not cancel the child. Child has its own WorkflowInstance record with no parent dependency. |

### Composition Rules

- **Max nesting depth**: 5 levels (configurable via `workflow.max_nesting_depth`). Prevents runaway recursion and makes execution traces comprehensible.
- **Tenant isolation**: Child inherits parent `TenantID`, enforced at start time. A workflow in Tenant A cannot invoke a workflow definition belonging to Tenant B.
- **Instance tracking**: Each child has its own `WorkflowInstance` record in the database, linked to the parent via `parent_instance_id`.
- **Cycle detection**: The DSL validator statically prevents circular references (A -> B -> A) at definition save time by traversing the sub-workflow graph.

### DSL Example

```json
{
  "id": "process_each_order",
  "type": "sub_workflow",
  "sub_workflow": {
    "workflow_id": "order_processing_v2",
    "workflow_version": 3,
    "input": {
      "order_id": "{{ item.id }}",
      "customer_email": "{{ trigger.payload.email }}"
    },
    "output_var": "order_result",
    "execution": "sync",
    "on_failure": "fail_parent"
  }
}
```

### Fan-Out Pattern

Combining `foreach` with `sub_workflow` enables batch processing over dynamic collections:

```json
{
  "id": "process_all_orders",
  "type": "foreach",
  "foreach": {
    "items": "{{ trigger.payload.orders }}",
    "item_var": "item",
    "max_parallel": 10,
    "step": {
      "id": "process_order",
      "type": "sub_workflow",
      "sub_workflow": {
        "workflow_id": "order_processing_v2",
        "workflow_version": 3,
        "input": { "order_id": "{{ item.id }}" },
        "output_var": "order_result",
        "execution": "sync",
        "on_failure": "continue"
      }
    }
  }
}
```

## Alternatives Considered

| Option | Pros | Cons |
|--------|------|------|
| **Engine-managed child instances** | Each child has its own WorkflowInstance record. Independent retry policies per child. Full execution history. CAS-based lifecycle management. Queryable via SQL. | Nesting depth increases execution complexity. Each child is a full instance (resource overhead). Parent-child coupling requires lifecycle management logic. |
| **Inline step expansion (macro)** | Simpler execution model (single flat workflow). No child instance overhead. Easier debugging (one trace). | No independent lifecycle management. Cannot fan-out dynamically. Versioning of shared sequences is manual. No isolation between parent and child failures. |
| **External orchestration (API call)** | Maximum decoupling. Works across services. | Loses engine guarantees. Must build own tracking, retry, and lifecycle management. No transactional state management across the composition boundary. |

## Rationale

1. **Engine-managed child instances provide composition with full contract enforcement.** Each child workflow benefits from schema validation, retry policies, and its own execution history. The state engine enforces the same data contracts on child instances as on parent instances.

2. **Reusable workflow definitions reduce duplication and maintenance burden.** A shared "send and track email" workflow can be defined once and invoked from any parent workflow. Changes to the shared workflow propagate to all callers at the specified version.

3. **Fan-out via `foreach` + `sub_workflow` enables batch processing patterns.** Processing N orders, sending N notifications, or enriching N records are natural use cases that require dynamic parallelism. The engine manages child instance creation and lifecycle.

4. **Three execution modes cover the full spectrum of composition needs.** Synchronous for request-reply patterns, asynchronous for fire-and-forget notifications, and detached for workflows that must outlive their parent.

5. **Static cycle detection at save time prevents runtime infinite recursion.** The validator traverses the sub-workflow reference graph before persisting the definition, catching A -> B -> A cycles before they can execute.

## Consequences

**Positive:**

- Users can build complex automations from simple, tested building blocks
- Common sequences (email + wait, approval + escalation) are defined once and reused
- Fan-out patterns enable batch processing over dynamic collections
- Independent versioning allows parent and child workflows to evolve separately
- Each child workflow has its own WorkflowInstance record for tracking and debugging
- Engine provides cancellation propagation, timeout enforcement, and retry policies per child

**Negative:**

- Increased execution complexity: deeply nested workflows are harder to debug and trace
- Resource overhead: each child workflow is a full instance with its own execution records
- Nesting depth limit (5 levels) may be restrictive for some enterprise use cases
- Cycle detection adds validation complexity at definition save time
- Parent-child lifecycle semantics (especially around cancellation and failure) require careful documentation

## Implementation Notes

### Child Instance Creation

```go
childInstanceID := fmt.Sprintf("wf-%s-%s-%s-child-%s", tenantID, parentDefID, eventID, stepID)

childInstance := &WorkflowInstance{
    ID:               childInstanceID,
    TenantID:         tenantID,
    ParentInstanceID: &parentInstanceID,
    WorkflowName:     childDef.Name,
    WorkflowVersion:  childDef.Version,
    ExecutionMode:     step.Execution, // sync, async, detached
    Status:           "running",
    CurrentState:     childDef.InitialState,
    Revision:         0,
}
```

### Parent Close Policy Mapping

| Execution Mode | Parent Cancellation Behavior |
|----------------|------------------------------|
| `sync` | Cancel child instance (terminate all pending executions) |
| `async` | Request cancellation of child instance (complete current state, then stop) |
| `detached` | No action (child continues independently) |

On parent cancellation, the engine queries child instances with `parent_instance_id` and applies the appropriate policy based on execution mode.

### Nesting Depth Enforcement

The engine passes a `depth` counter through the instance metadata. Each `sub_workflow` state increments the counter before creating the child instance. If `depth >= max_nesting_depth`, the state fails immediately with a descriptive error.

### Extensibility

| Capability | Timeline | Description |
|------------|----------|-------------|
| Workflow marketplace | Phase 5+ | Shared workflow templates across tenants with one-click import |
| Cross-tenant composition | Phase 5+ | Invoking shared/public workflows owned by other tenants (with explicit permission grants) |
| Independent versioning | Phase 3 | Parent pins to child version; child can publish new versions without breaking parents |
| Composition analytics | Phase 4 | Dashboard showing which workflows are composed from which sub-workflows |
