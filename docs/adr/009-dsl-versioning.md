# ADR-009: DSL Versioning and Backward Compatibility

## Status
Accepted

## Context
The Orchestrator DSL (Domain-Specific Language) defines workflow structure, step types, expressions, conditions, and connector configurations. As the platform evolves, the DSL must evolve with it: new step types are added, expression functions are introduced, existing fields gain new options, and occasionally breaking changes are unavoidable. This evolution creates a fundamental tension between innovation and stability.

Running workflows may execute for days or weeks. A workflow that started under DSL version 1.2 must continue executing correctly even after the platform upgrades to DSL version 1.5. Similarly, triggers bound to specific workflow definitions must not silently change behavior when the underlying definition is updated. The system must guarantee that a workflow behaves identically regardless of when it was started relative to DSL updates.

Additional complexity arises from AI-generated workflows. The AI generation service must know which DSL version to target, and the validation layer must understand version-specific rules. Trigger bindings, workflow definitions, and the interpreter all need version awareness. Without a deliberate versioning strategy, any DSL change risks breaking running workflows, invalidating stored definitions, or producing AI-generated DSL that fails validation.

## Decision
We adopt explicit DSL versioning with a version envelope, semantic version rules, trigger version pinning, and full definition version history.

### Version Envelope

Every DSL document includes a top-level `version` field:

```json
{
  "version": "1.2",
  "name": "onboard-new-customer",
  "trigger": { ... },
  "steps": [ ... ]
}
```

The version field is required and validated on every DSL ingestion path: user creation, AI generation, API import, and trigger evaluation.

### Semantic Versioning Rules

**Minor version increment (1.0 -> 1.1):**
- Adds new step types (existing step types unchanged)
- Adds optional fields to existing step types
- Adds new expression functions
- Adds new trigger condition operators
- Fully backward compatible: a valid 1.0 DSL is also valid under 1.1

**Major version increment (1.x -> 2.0):**
- Renames or removes existing fields
- Changes step type semantics
- Alters expression evaluation rules
- Requires a migration tool to convert from previous major version
- Not backward compatible

### Version Handling in the Interpreter

```
DSL Input
    │
    ▼
Parse version field
    │
    ▼
Select version-specific parser/validator
    │
    ▼
Validate against version schema
    │
    ├─ Unknown fields in known version → Ignored (forward compatibility)
    ├─ Unknown step types → Validation failure
    └─ Missing required fields → Validation failure
    │
    ▼
Execute with version-appropriate logic
```

The interpreter selects parsing and execution logic based on the `version` field. This allows the same interpreter binary to handle workflows across multiple DSL versions simultaneously.

### Trigger Version Pinning

The `trigger_bindings` table includes a `workflow_version` column that pins to a specific `workflow_definitions.dsl_version`:

| Column | Purpose |
|--------|---------|
| `trigger_bindings.workflow_version` | Pins to specific `workflow_definitions.dsl_version` |
| `workflow_version = N` | Trigger always uses definition version N |
| `workflow_version = 0` | Special value: always use the latest version |

**Behavior on workflow definition update:**
- Existing triggers with a pinned version continue using the old definition version
- Triggers with `workflow_version = 0` automatically use the new definition
- Users explicitly update trigger bindings to point to new versions

### Workflow Definition Versioning

The `workflow_definitions` table maintains full version history:

| Column | Purpose |
|--------|---------|
| `dsl_version` | Monotonically incrementing version, incremented on every update |
| `dsl_content` | Full DSL JSON for this version |
| `dsl_schema_version` | The DSL language version (e.g., "1.2") used by this definition |
| `is_current` | Boolean indicating the latest version |

All historical versions are retained. Running workflow instances record the `dsl_version` they started with, ensuring they execute against that exact definition for their entire lifetime.

## Alternatives Considered

| Option | Pros | Cons |
|--------|------|------|
| Explicit version envelope (chosen) | Clear contract; version-specific validation; forward compatibility via ignored fields; AI generation targets specific version | Version field required on every document; interpreter must handle multiple versions |
| No versioning (always latest) | Simple implementation; no version management | Running workflows break on DSL changes; triggers change behavior silently; no backward compatibility |
| Content-hash versioning | Immutable definitions; precise reproducibility | No semantic meaning; cannot determine compatibility between versions; migration tooling impossible |
| Embedded migration scripts | Self-contained version transitions | Complex execution model; migration scripts are code that must be tested; storage overhead |

## Rationale
1. The version envelope prevents silent breaking changes by making the DSL contract explicit at the document level.
2. Trigger version pinning prevents surprise workflow behavior changes in production, giving operators explicit control over when to adopt new definition versions.
3. Forward compatibility (ignoring unknown fields) allows gradual rollout of new DSL features without invalidating existing documents.
4. Full definition version history ensures running workflows always have access to the exact DSL they started with, regardless of subsequent updates.
5. Semantic versioning rules give clear guidance on what constitutes a compatible vs. breaking change, enabling automated compatibility checking.

## Consequences

**Positive:**
- Running workflows are immune to DSL updates that happen after they start
- Triggers behave predictably with explicit version pinning
- AI generation can target specific DSL versions for maximum compatibility
- Gradual DSL evolution without forced migration of existing workflows
- Full audit trail of every definition version

**Negative:**
- Interpreter complexity increases with each supported DSL version
- Storage grows with full version history (mitigated by JSON compression)
- Users must understand version pinning to manage triggers effectively
- Major version migrations require tooling and user communication

## Implementation Notes

### DSL Version Validation

```go
type DSLDocument struct {
    Version string          `json:"version"`
    Name    string          `json:"name"`
    Trigger json.RawMessage `json:"trigger"`
    Steps   json.RawMessage `json:"steps"`
}

func ParseDSL(raw []byte) (*WorkflowDefinition, error) {
    var doc DSLDocument
    if err := json.Unmarshal(raw, &doc); err != nil {
        return nil, fmt.Errorf("parse DSL: %w", err)
    }

    if doc.Version == "" {
        return nil, fmt.Errorf("DSL version field is required")
    }

    parser, ok := versionParsers[doc.Version]
    if !ok {
        return nil, fmt.Errorf("unsupported DSL version: %s", doc.Version)
    }

    return parser.Parse(doc)
}
```

### Trigger Version Resolution

```go
func (r *TriggerResolver) ResolveDefinition(
    ctx context.Context,
    binding TriggerBinding,
) (*WorkflowDefinition, error) {
    if binding.WorkflowVersion == 0 {
        // Always-latest: fetch current version
        return r.repo.GetCurrentDefinition(ctx, binding.TenantID, binding.WorkflowID)
    }

    // Pinned: fetch specific version
    return r.repo.GetDefinitionVersion(ctx, binding.TenantID, binding.WorkflowID, binding.WorkflowVersion)
}
```

### Future Extensibility
- **DSL migration tool:** CLI command to migrate definitions from one major version to the next, with dry-run and diff output.
- **Per-tenant version limits:** Enterprise tenants may opt into beta DSL versions before general availability.
- **DSL feature flags:** Certain step types or expression functions gated by plan level, enforced during validation.
- **Deprecation warnings:** Validation emits warnings for deprecated fields or step types, giving users advance notice before removal in the next major version.
