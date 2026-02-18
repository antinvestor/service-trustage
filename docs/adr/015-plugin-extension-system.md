# ADR-015: Plugin Extension System (Future Architecture)

## Status

Proposed (Phase 4+)

## Context

The Orchestrator currently supports a fixed set of connector types (webhook, email, CRM, Slack) implemented as Go interfaces compiled into the main binary. Adding a new connector requires writing Go code, adding it to the connector registry in `main.go`, running the test suite, and deploying a new version of the Orchestrator. This process works well for the core team during the MVP and growth phases, but it does not scale as the platform matures.

As the platform gains adoption, three categories of extension demand will emerge. First, enterprise customers will need connectors for internal systems (proprietary APIs, legacy SOAP services, on-premise databases) that the core team cannot build or maintain. Second, technology partners will want to offer first-class integrations with their products (e.g., a monitoring vendor building a native Stawi.dev connector). Third, the community will want to contribute connectors for the long tail of SaaS APIs that are individually niche but collectively essential for platform completeness.

The current compile-time extension model cannot serve any of these audiences. Enterprise customers cannot modify the Orchestrator source code. Partners do not want to submit Go pull requests and wait for release cycles. Community contributors may not know Go at all. The platform needs a plugin architecture that progressively opens up extensibility without sacrificing reliability, security, or operational simplicity. This ADR proposes a three-stage progression from the current model to a full connector marketplace.

## Decision

Adopt a progressive extension model that evolves in three stages, each building on the foundation of the previous one. The core principle is: **start closed and open incrementally, ensuring each stage is production-hardened before advancing to the next.**

### Stage 1: Compile-Time Go Interfaces (Current, Phase 1-3)

All connectors implement the `ConnectorAdapter` Go interface and are registered at startup:

```go
type ConnectorAdapter interface {
    Type() string
    Execute(ctx context.Context, input ConnectorInput) (ConnectorOutput, error)
    Validate(config json.RawMessage) error
    InputSchema() json.RawMessage
    OutputSchema() json.RawMessage
}

// Registration in main.go
registry.Register("webhook", &WebhookAdapter{})
registry.Register("email", &EmailAdapter{})
registry.Register("slack", &SlackAdapter{})
registry.Register("crm", &CRMAdapter{})
```

This model provides maximum reliability (type-safe, compiled, tested), but requires a code change and deployment for every new connector.

### Stage 2: HTTP Connector Protocol (Phase 4)

External connectors communicate with the Orchestrator via a standard HTTP protocol. Any service that implements this protocol can be registered as a connector, regardless of implementation language.

**Protocol endpoints:**

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/execute` | POST | Execute the connector action |
| `/schema` | GET | Return input, config, and output JSON schemas |
| `/health` | GET | Health check for availability monitoring |

**Execute request:**

```json
{
  "input": {
    "to": "user@example.com",
    "subject": "Order Confirmation",
    "body": "Your order #1234 has been confirmed."
  },
  "config": {
    "api_key_ref": "sendgrid_key",
    "from_address": "noreply@example.com"
  },
  "context": {
    "workflow_id": "wf-tenant_abc-wfdef_order-evt_01HQ",
    "step_id": "send_confirmation",
    "tenant_id": "tenant_abc",
    "idempotency_key": "wf-tenant_abc-wfdef_order-evt_01HQ-send_confirmation"
  }
}
```

**Execute response:**

```json
{
  "result": {
    "message_id": "msg_abc123",
    "status": "sent"
  },
  "metadata": {
    "duration_ms": 245,
    "external_id": "sg_msg_abc123"
  }
}
```

**Schema response:**

```json
{
  "input_schema": {
    "type": "object",
    "properties": {
      "to": { "type": "string", "format": "email" },
      "subject": { "type": "string", "maxLength": 200 },
      "body": { "type": "string" }
    },
    "required": ["to", "subject", "body"]
  },
  "config_schema": {
    "type": "object",
    "properties": {
      "api_key_ref": { "type": "string" },
      "from_address": { "type": "string", "format": "email" }
    },
    "required": ["api_key_ref"]
  },
  "output_schema": {
    "type": "object",
    "properties": {
      "message_id": { "type": "string" },
      "status": { "type": "string", "enum": ["sent", "queued", "failed"] }
    }
  }
}
```

**Registration:**

External connectors are registered via `connector_configs` with `type = "external"`:

```json
{
  "type": "external",
  "name": "Custom Email Sender",
  "config": {
    "endpoint": "https://connectors.example.com/email",
    "timeout_ms": 5000,
    "retry_policy": {
      "max_attempts": 3,
      "backoff_coefficient": 2.0
    },
    "auth": {
      "type": "bearer",
      "token_ref": "connector_auth_token"
    }
  }
}
```

The Orchestrator calls external connectors via a generic `ExternalConnectorWorker` that makes HTTP requests according to the protocol. The worker handles timeouts, circuit breaking, and idempotency key forwarding. The engine manages retries via its retry scheduler.

### Stage 3: Connector Marketplace (Phase 5+)

A curated catalog of connectors available for one-click installation per tenant:

- **Catalog**: Searchable directory of connectors with descriptions, schemas, ratings, and usage statistics.
- **Installation**: Tenant admin installs a connector, which creates the `connector_config` entry and configures credentials.
- **Verification**: Connectors undergo a review process before appearing in the catalog. Verified connectors display a trust badge.
- **Shared infrastructure**: Popular connectors (e.g., Slack, Stripe) run on Stawi.dev-managed infrastructure. Niche connectors run on the author's infrastructure.
- **Revenue sharing**: Paid connectors split revenue between the author and Stawi.dev.

### Security Model

| Concern | Stage 1 | Stage 2 | Stage 3 |
|---------|---------|---------|---------|
| Code trust | Compiled into binary | External HTTP, sandboxed via network policy | Verified by review process |
| Credential access | Direct access to credential store | Receives resolved credential values (never refs) | Same as Stage 2, plus credential scoping per connector |
| Network isolation | Same process | Separate service, network policy restricted | Isolated per connector, egress allowlist |
| Resource limits | Go runtime limits | HTTP timeout + circuit breaker | Per-connector CPU/memory quotas |
| Audit | Application logs | Request/response logging | Full audit trail with tenant attribution |

## Alternatives Considered

| Option | Pros | Cons |
|--------|------|------|
| **Progressive stages (chosen)** | Start simple, add complexity only when justified. Each stage is production-hardened before advancing. Go interfaces keep MVP reliable. HTTP protocol enables polyglot connectors. Marketplace is natural evolution. | Slow to reach full extensibility (Phase 4+). HTTP protocol adds latency vs. in-process. Three different extension models to maintain simultaneously. |
| **gRPC plugin protocol (HashiCorp-style)** | Type-safe contract. Streaming support. Mature pattern (Terraform, Vault). Proto-defined interface. | Heavier than HTTP for simple request-response. Requires gRPC tooling for connector authors. Over-engineered for stateless connectors. |
| **WebAssembly (Wasm) plugins** | In-process execution (low latency). Language-agnostic. Strong sandboxing. | Immature ecosystem for Go hosts. Limited I/O capabilities in Wasm. Debugging is difficult. Small community of Wasm plugin authors. |
| **Shared library plugins (Go plugin package)** | In-process execution. Full Go interface. No serialization overhead. | Only works with Go. Fragile (must be compiled with exact same Go version). No Windows support. Cannot update plugins without restarting process. |
| **Open from day one (marketplace immediately)** | Maximum extensibility from the start. Faster ecosystem growth. | Premature complexity. Security model not yet proven. No established patterns to build marketplace on. Risk of low-quality connectors damaging platform reputation. |

## Rationale

1. **Go interfaces keep the MVP and growth phases simple and reliable.** Compile-time registration, type safety, and in-process execution eliminate an entire class of failure modes (network errors, serialization bugs, version mismatches). This is the right trade-off when the core team is the only connector author.

2. **HTTP protocol enables polyglot connectors without infrastructure complexity.** A connector author can implement the three-endpoint protocol in any language, deploy it anywhere, and register it with the Orchestrator. No gRPC tooling, no Wasm compilation, no shared library headaches.

3. **The marketplace is a natural evolution once critical mass exists.** A marketplace requires a catalog, a review process, credential management, billing integration, and a community of connector authors. None of these make sense until the platform has enough tenants to justify the investment. Building the marketplace on top of a proven HTTP protocol (Stage 2) reduces risk.

4. **Security must be progressive, not all-or-nothing.** Stage 1 connectors are fully trusted (they are compiled into the binary). Stage 2 connectors are partially trusted (HTTP isolation, credential scoping, network policy). Stage 3 connectors are minimally trusted (review process, resource quotas, full audit). Each stage adds security controls proportional to the trust level.

5. **Maintaining all three stages simultaneously is acceptable.** Core connectors (webhook, email, Slack) will always remain as Go interfaces for performance and reliability. The HTTP protocol and marketplace serve different audiences. The maintenance cost of three models is justified by the flexibility they provide.

## Consequences

**Positive:**

- MVP remains simple: Go interfaces, compile-time registration, in-process execution
- HTTP protocol opens the platform to any language (Python, TypeScript, Java, Rust)
- Enterprise customers can build private connectors without forking the Orchestrator
- Partners can build and maintain their own connectors independently
- Marketplace creates a network effect: more connectors attract more tenants
- Progressive security model matches trust level to extension model
- Each stage can be adopted independently based on customer demand

**Negative:**

- HTTP connector calls add network latency compared to in-process Go interfaces
- Three extension models create cognitive overhead for the team
- Connector quality varies: Go interfaces are battle-tested, marketplace connectors may not be
- HTTP protocol must be versioned and maintained as a public contract
- Marketplace requires significant investment in catalog, review process, billing, and trust infrastructure
- Security model for external connectors (credential handling, network isolation) adds operational complexity

## Implementation Notes

### ExternalConnectorWorker

The generic worker for calling HTTP connectors:

```go
type ExternalConnectorWorker struct {
    httpClient  *http.Client
    callLog     *ConnectorCallLog
    circuitBreaker *CircuitBreaker
}

func (w *ExternalConnectorWorker) Execute(ctx context.Context, input ExternalConnectorInput) (ConnectorOutput, error) {
    // Check circuit breaker
    if !w.circuitBreaker.Allow(input.Endpoint) {
        return ConnectorOutput{}, fmt.Errorf("circuit breaker open for %s", input.Endpoint)
    }

    // Check connector_call_log for idempotent retry
    if cached, err := w.callLog.Get(ctx, input.IdempotencyKey); err == nil {
        return cached, nil
    }

    // Build request
    reqBody := ExternalConnectorRequest{
        Input:   input.Input,
        Config:  input.ResolvedConfig,
        Context: ExternalConnectorContext{
            WorkflowID:     input.WorkflowID,
            StepID:         input.StepID,
            TenantID:       input.TenantID,
            IdempotencyKey: input.IdempotencyKey,
        },
    }

    // Call external connector
    resp, err := w.httpClient.Post(input.Endpoint+"/execute", reqBody)
    if err != nil {
        w.circuitBreaker.RecordFailure(input.Endpoint)
        return ConnectorOutput{}, fmt.Errorf("external connector call: %w", err)
    }

    w.circuitBreaker.RecordSuccess(input.Endpoint)
    w.callLog.Put(ctx, input.IdempotencyKey, resp)

    return resp, nil
}
```

### Circuit Breaker Configuration

External connectors are protected by per-endpoint circuit breakers:

| Parameter | Default | Description |
|-----------|---------|-------------|
| `failure_threshold` | 5 | Consecutive failures before opening circuit |
| `success_threshold` | 3 | Consecutive successes before closing circuit |
| `timeout` | 30s | Time in open state before allowing a probe request |
| `max_concurrent` | 10 | Maximum concurrent requests to a single endpoint |

### Extensibility Beyond Connectors

The HTTP protocol pattern can be extended to other extension types in the future:

| Extension Type | Protocol | Timeline |
|----------------|----------|----------|
| Custom connectors | HTTP (Stage 2) | Phase 4 |
| Custom step types | HTTP (same pattern as connectors) | Phase 5 |
| Custom event sources | Webhook normalization layer | Phase 5 |
| Custom CEL functions | CEL function registry (must be pure and deterministic) | Phase 4 |

Custom CEL functions require special treatment: they must be pure (no side effects) and deterministic (same input always produces same output) to ensure consistent evaluation across retries and state transitions. They are registered via a configuration file rather than an HTTP protocol:

```yaml
cel_extensions:
  - name: "normalize_phone"
    implementation: "go"          # Only Go for determinism guarantee
    package: "github.com/tenant/cel-funcs"
    function: "NormalizePhone"
    input_types: ["string"]
    output_type: "string"
```
