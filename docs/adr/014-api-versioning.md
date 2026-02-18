# ADR-014: API Versioning and Stability Strategy

## Status

Accepted

## Context

The Orchestrator exposes ConnectRPC APIs consumed by multiple clients: the Stawi.dev React frontend, other internal Stawi.dev services (e.g., the form builder, analytics), and in the future, external integrations and CLI tools. Each of these consumers has different tolerance for API changes and different upgrade cadences. The frontend is deployed alongside the backend and can be updated in lockstep. External integrations, however, are controlled by customers and may not be updated for months after an API change.

Protobuf and ConnectRPC provide a strong foundation for API evolution. Protobuf's wire format is designed for forward and backward compatibility: unknown fields are preserved, new fields can be added without breaking existing clients, and field numbers provide stable identity independent of field names. However, these compatibility guarantees only hold when evolution rules are followed consistently. A single breaking change — renaming a field, changing a field type, altering RPC semantics — can break every client that depends on the old behavior.

Without a clear versioning strategy, the team faces a recurring tension: move fast and break things (frustrating customers) or freeze the API (blocking product development). The strategy must define what constitutes a breaking change, how breaking changes are introduced when necessary, how deprecated APIs are retired, and which APIs carry stability guarantees versus which are internal and can evolve freely.

## Decision

Adopt protobuf package-level versioning with explicit evolution rules, a deprecation process, and a distinction between internal and external API surfaces.

### Package Structure

Each service domain has its own versioned protobuf package:

```
proto/
  stawi/orchestrator/
    form/v1/form.proto
    workflow/v1/workflow.proto
    connector/v1/connector.proto
    event/v1/event.proto
    trigger/v1/trigger.proto
```

The version is part of the package name (e.g., `stawi.orchestrator.form.v1`), the directory structure, and the generated Go package path.

### Evolution Rules

**Non-breaking changes (no version bump required):**

| Change | Why It Is Safe |
|--------|----------------|
| Add new RPC method | Existing clients never call it |
| Add optional field to request message | Existing clients do not send it; server uses default value |
| Add field to response message | Existing clients ignore unknown fields (protobuf wire format) |
| Add new enum value | Safe if clients handle `UNSPECIFIED` / unknown values gracefully |
| Add new message type | No existing code references it |
| Change field documentation | No wire format impact |

**Breaking changes (require new package version):**

| Change | Why It Breaks |
|--------|---------------|
| Remove or rename a field | Existing clients send/expect the old field name/number |
| Change a field's type | Wire format incompatibility |
| Change RPC request/response type | Existing clients send/expect old types |
| Change RPC semantics (same signature, different behavior) | Clients depend on old behavior |
| Remove an RPC method | Existing clients call it |
| Rename an RPC method | ConnectRPC routes by method name |
| Change a field from optional to required | Existing clients may not send it |

When a breaking change is necessary, a new package version is created (e.g., `form.v2`) alongside the existing version. Both versions are served simultaneously during the migration period.

### Deprecation Process

1. **Mark deprecated**: Add `[deprecated = true]` to the field, method, or message in the proto file.
2. **Log on use**: Server logs a deprecation warning when the deprecated field or method is used, including the caller identity.
3. **Maintain minimum 6 months**: The deprecated API continues to function for at least 6 months after the deprecation is announced.
4. **Communicate removal**: At least 30 days before removal, notify affected consumers via changelog, migration guide, and direct communication for known external integrators.
5. **Remove**: Delete the deprecated field, method, or package version.

### Internal vs. External API Surface

| Classification | Stability Guarantee | Consumers |
|---------------|---------------------|-----------|
| **Internal** (default) | Best-effort backward compatibility. May break with notice in release notes. | Stawi.dev frontend, internal services |
| **External** | Full backward compatibility. Breaking changes only via new version. 6-month deprecation minimum. | Customer integrations, CLI tools, partner APIs |

All RPCs are internal by default. External-facing RPCs are explicitly marked with a `(stawi.api_stability) = STABLE` option and documented in the public API reference.

### Field Number Reservation

When fields are removed, their field numbers are reserved to prevent accidental reuse:

```protobuf
message WorkflowDefinition {
  reserved 7, 12;           // Removed: old_field_name, another_old_field
  reserved "old_field_name", "another_old_field";
}
```

## Alternatives Considered

| Option | Pros | Cons |
|--------|------|------|
| **Protobuf package versioning (chosen)** | Standard protobuf/gRPC convention. Both versions can be served simultaneously. Clean separation in code. Generated types do not conflict. Compatible with ConnectRPC ecosystem. | Directory and import duplication when new version is created. Must maintain two implementations during migration. Verbose package paths. |
| **URL path versioning (/v1/..., /v2/...)** | Familiar to REST developers. Easy to route at load balancer level. | Not idiomatic for protobuf/ConnectRPC. Duplicates routing logic. Does not leverage protobuf's built-in compatibility. |
| **Header-based versioning (Accept-Version)** | Single endpoint. No URL duplication. | Harder to route. Not supported natively by ConnectRPC. Requires custom middleware. Easy to forget the header. |
| **No explicit versioning (rely on protobuf compatibility)** | Simplest. No version numbers to manage. Protobuf handles most evolution. | No escape hatch for truly breaking changes. Cannot serve old and new simultaneously. No clear communication about stability. |
| **Semantic versioning on the API** | Familiar convention. Clear meaning of major/minor/patch. | Over-engineered for protobuf APIs. Semver is designed for libraries, not wire protocols. Protobuf package versioning is the industry standard. |

## Rationale

1. **Protobuf package versioning is the established convention in the gRPC and ConnectRPC ecosystem.** Google APIs, Buf Schema Registry, and major cloud providers all use this pattern. Following convention reduces cognitive load and leverages tooling support.

2. **Non-breaking changes cover the vast majority of API evolution.** Adding fields, methods, and enum values accounts for 90%+ of API changes. These require no version bump, no client changes, and no migration period.

3. **Breaking changes are rare but must have a clear path.** When breaking changes are necessary (field type change, semantic change), the new package version pattern provides a clean migration path: both versions coexist, clients migrate at their own pace, the old version is retired after the deprecation period.

4. **Internal/external distinction prevents premature stability commitments.** Most RPCs are consumed only by the Stawi.dev frontend, which is deployed in lockstep. Committing to full backward compatibility for all RPCs would slow development unnecessarily. Only RPCs explicitly marked as external carry the full stability guarantee.

5. **Consistent with Foundry's proto versioning approach.** The Orchestrator is part of the broader Stawi.dev platform. Using the same versioning conventions as Foundry ensures consistency across the platform and reduces context-switching for developers working on both.

## Consequences

**Positive:**

- Clear rules for what constitutes a breaking change, eliminating ambiguity
- Non-breaking changes require no coordination with consumers
- Breaking changes have a defined migration path with coexisting versions
- Deprecation process gives consumers time to migrate
- Internal/external distinction allows fast iteration on internal APIs
- Field number reservation prevents subtle wire format bugs from field number reuse
- Tooling support from Buf, ConnectRPC, and protobuf ecosystem

**Negative:**

- Creating a new package version (e.g., `form.v2`) requires duplicating proto files and maintaining two implementations
- Deprecation period (6 months) means old code paths must be maintained longer
- Internal/external classification requires ongoing judgment calls
- Team must be disciplined about following evolution rules (code review enforcement)
- Proto linting alone cannot catch semantic breaking changes (same signature, different behavior)

## Implementation Notes

### Buf Lint Configuration

Buf is configured to enforce API evolution rules:

```yaml
# buf.yaml
version: v2
lint:
  use:
    - DEFAULT
    - PACKAGE_VERSION_SUFFIX
  except:
    - PACKAGE_NO_IMPORT_CYCLE
breaking:
  use:
    - PACKAGE
    - WIRE_JSON
```

### Breaking Change Detection in CI

The CI pipeline runs `buf breaking` against the main branch to detect breaking changes in pull requests:

```bash
buf breaking --against '.git#branch=main'
```

Breaking changes detected by `buf breaking` block the PR unless the author explicitly acknowledges the break by adding a `breaking-change` label and updating the migration guide.

### Deprecation Logging Middleware

A ConnectRPC interceptor logs deprecation warnings when deprecated RPCs are called:

```go
func DeprecationInterceptor() connect.UnaryInterceptorFunc {
    return func(next connect.UnaryFunc) connect.UnaryFunc {
        return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
            if isDeprecated(req.Spec().Procedure) {
                slog.Warn("deprecated RPC called",
                    "procedure", req.Spec().Procedure,
                    "peer", req.Peer().Addr,
                )
                // Set response header to inform client
                resp, err := next(ctx, req)
                if resp != nil {
                    resp.Header().Set("Deprecation", "true")
                    resp.Header().Set("Sunset", sunsetDate(req.Spec().Procedure))
                }
                return resp, err
            }
            return next(ctx, req)
        }
    }
}
```

### Extensibility

| Capability | Timeline | Description |
|------------|----------|-------------|
| Public API reference | Phase 3 | Auto-generated API docs from proto files with stability annotations |
| API changelog | Phase 3 | Automated changelog generation from proto diffs between releases |
| Client SDK generation | Phase 4 | Generated TypeScript and Python SDKs with versioned packages |
| API usage analytics | Phase 4 | Track which RPCs are used by which consumers to inform deprecation decisions |
