# ADR-005: Connector Adapter Pattern with Typed Schemas

## Status

Accepted

## Context

Workflows interact with external systems through connectors: sending emails, calling webhooks, querying databases, posting to Slack, updating CRMs, generating AI content, and more. The connector system must be:

1. **Extensible**: New connectors added without changes to the DSL, interpreter, or engine
2. **Discoverable**: Both AI (for workflow generation) and the visual builder (for step configuration) must enumerate available connectors and their input/output schemas at runtime
3. **Safe**: Authentication, retries, rate limiting, and error handling encapsulated within each connector
4. **Testable**: Connectors can be unit-tested in isolation with mocked credentials
5. **Tenant-scoped**: Configuration and credentials are isolated per tenant
6. **Versionable**: Connector behavior can evolve without breaking existing workflow definitions

## Decision

Define a **connector.Adapter** Go interface that all connectors implement, registered in a **connector.Registry** at startup.

### Adapter Interface

```go
// Adapter is the interface that all connectors implement.
type Adapter interface {
    // Identity
    Type() string                    // e.g., "email.send"
    DisplayName() string             // e.g., "Send Email"
    Description() string             // e.g., "Send an email via configured SMTP or provider"
    Category() string                // e.g., "communication"

    // Schemas (JSON Schema)
    InputSchema() json.RawMessage    // JSON Schema for the input fields
    ConfigSchema() json.RawMessage   // JSON Schema for tenant-scoped configuration
    OutputSchema() json.RawMessage   // JSON Schema for the output/result

    // Execution
    Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResponse, error)

    // Validation
    Validate(input map[string]any) error
}
```

### Execute Request and Response

```go
// ExecuteRequest contains everything a connector needs to execute.
type ExecuteRequest struct {
    Input      map[string]any  // Resolved input from DSL (templates interpolated)
    Config     map[string]any  // Tenant-scoped connector configuration
    Credential map[string]any  // Decrypted credential (API keys, OAuth tokens, etc.)
    TenantID   string
    WorkflowID string
    StepID     string
}

// ExecuteResponse contains the result of a connector execution.
type ExecuteResponse struct {
    Result  map[string]any   // Output data, stored in vars[output_var]
    Metrics *ExecuteMetrics  // Execution metrics for observability
}

// ExecuteMetrics captures connector execution telemetry.
type ExecuteMetrics struct {
    Duration    time.Duration
    BytesSent   int64
    BytesRecv   int64
    StatusCode  int            // HTTP status code if applicable
    RetryCount  int
}
```

### Registry

```go
// Registry holds all registered connector adapters.
type Registry struct {
    adapters map[string]Adapter
}

func (r *Registry) Register(a Adapter) error { ... }
func (r *Registry) Get(actionType string) (Adapter, error) { ... }
func (r *Registry) List() []Adapter { ... }
func (r *Registry) ListByCategory(category string) []Adapter { ... }
```

Adapters are registered at startup during service initialization. The registry is immutable after startup (no dynamic registration).

### Connector Naming Convention

Connectors follow a `{category}.{action}` naming pattern:

| Name | Category | Description |
|------|----------|-------------|
| `webhook.call` | integration | Call an external webhook URL |
| `email.send` | communication | Send an email via configured provider |
| `http.request` | integration | Make an arbitrary HTTP request |
| `slack.post` | communication | Post a message to a Slack channel |
| `crm.upsert` | crm | Create or update a CRM contact/deal |
| `db.query` | database | Execute a read-only database query |
| `storage.upload` | storage | Upload a file to cloud storage |
| `transform.jq` | transform | Apply a jq transformation to data |
| `ai.generate` | ai | Generate text via LLM API |

### DSL Integration

A `call` step in the DSL references a connector by its action name:

```json
{
  "id": "send-email",
  "type": "call",
  "action": "email.send",
  "input": {
    "to": "{{ payload.email }}",
    "subject": "Welcome, {{ payload.name }}!",
    "template": "welcome-v1"
  },
  "output_var": "email_result"
}
```

The DSL interpreter resolves templates in `input`, looks up the adapter in the registry, and executes it via the state engine's worker dispatch.

## Alternatives Considered

| Option | Pros | Cons | Verdict |
|--------|------|------|---------|
| **Go interface + registry** | Simple to implement. Type-safe. Full control over execution. JSON Schema enables AI/UI discovery. Tenant credential isolation. Engine manages retries/timeouts. | New connectors require code change + deployment. Interface is hard to change after adoption. Must build credential infrastructure. | **Chosen** |
| **Go plugins** | Dynamic loading without recompilation. | Fragile across Go versions. Limited platform support (no Windows, no cross-compilation). Plugin interface changes break all plugins. No ecosystem adoption. | Rejected |
| **WASM plugins** | Language-agnostic. Sandboxed execution. Dynamic loading. | Performance overhead (serialization boundary). Immature Go-WASM ecosystem. Complex debugging. Limited I/O capabilities within sandbox. | Rejected |
| **HTTP-based connectors** | Language-agnostic. Simple protocol. Easy to deploy independently. | Loses type safety. Additional network hop per step. Must build service discovery. Authentication between services. Latency overhead. | Rejected (for now; see Plugin Evolution) |
| **OpenAPI-driven** | Auto-generate connectors from API specs. Large spec library. | Not all APIs have OpenAPI specs. Specs do not capture authentication flows, rate limits, or pagination. Generated code is often incomplete. Cannot handle non-HTTP connectors. | Rejected |

## Rationale

1. **Go interface is the right abstraction level.** It is simple enough that implementing a new connector takes under an hour, yet powerful enough to encapsulate complex authentication flows, rate limiting, pagination, and error handling. The interface boundary is clear: the framework handles credentials, retries, and observability; the adapter handles the API call.

2. **JSON Schema enables AI generation, visual builder forms, and validation.** Each adapter publishes `InputSchema()`, `ConfigSchema()`, and `OutputSchema()` as JSON Schema. The AI uses these to generate valid `call` step inputs. The visual builder renders form fields from the schema. The validator checks inputs against the schema before execution.

3. **The registry enables runtime introspection.** A `ListConnectorTypes` RPC endpoint enumerates all registered connectors with their schemas, categories, and descriptions. This powers both the visual builder's connector picker and the AI's tool inventory.

4. **Tenant-scoped configuration separates logic from data.** The adapter receives `Config` (tenant-specific settings like SMTP server, API base URL) and `Credential` (tenant-specific secrets like API keys, OAuth tokens) as separate fields. This keeps the adapter implementation stateless and tenant-agnostic.

5. **Engine-dispatched executions are the natural boundary for connector calls.** Each connector call is a state execution dispatched by the engine, which gives us automatic retries, timeouts, and execution recording -- without the connector needing to implement any of it.

## Consequences

**Positive:**

- New connectors are added by implementing one interface and registering at startup
- AI and visual builder auto-discover connectors via registry introspection and JSON Schema
- Connector authentication, retry, and rate limiting logic is encapsulated per adapter
- The state engine manages retries, timeouts, and execution recording for all connectors
- Tenant credential isolation is enforced at the framework level, not per connector
- Connectors are independently testable with mocked credentials and configurations

**Negative:**

- Adding a new connector requires a code change and redeployment of the service
- The Adapter interface is hard to change after connectors are built against it (adding methods is a breaking change)
- Must build credential storage, encryption, and injection infrastructure (see ADR-007)
- Go-only connector authoring limits contributions from non-Go developers (addressed in Plugin Evolution)

## Implementation Notes

### Connector Lifecycle in a Workflow Step

The full execution path from DSL to result:

```
DSL call state
  -> Engine dispatch: create execution record, validate input
     -> Resolve {{ templates }} in input fields
     -> Build ConnectorInput from resolved values
     -> Worker receives execution command via NATS
        -> registry.Get(action) to find adapter
        -> Load tenant Config from database/cache
        -> Decrypt tenant Credential from credential store
        -> adapter.Execute(ctx, &ExecuteRequest{...})
        -> Return ExecuteResponse
     -> Worker commits result via Commit API
     -> Engine validates output, advances state
```

### Example Adapter Implementation

```go
type WebhookCallAdapter struct{}

func (a *WebhookCallAdapter) Type() string        { return "webhook.call" }
func (a *WebhookCallAdapter) DisplayName() string  { return "Call Webhook" }
func (a *WebhookCallAdapter) Description() string  { return "Send an HTTP POST to a webhook URL" }
func (a *WebhookCallAdapter) Category() string     { return "integration" }

func (a *WebhookCallAdapter) InputSchema() json.RawMessage {
    return json.RawMessage(`{
        "type": "object",
        "properties": {
            "url":     { "type": "string", "format": "uri", "description": "Webhook URL" },
            "method":  { "type": "string", "enum": ["POST", "PUT", "PATCH"], "default": "POST" },
            "headers": { "type": "object", "additionalProperties": { "type": "string" } },
            "body":    { "type": "object", "description": "JSON request body" }
        },
        "required": ["url"]
    }`)
}

func (a *WebhookCallAdapter) Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResponse, error) {
    // Build HTTP request from req.Input
    // Apply authentication from req.Credential
    // Send request with timeout from context
    // Return response body as Result
}
```

### Registration at Startup

```go
func RegisterConnectors(reg *connector.Registry) {
    reg.Register(&webhook.CallAdapter{})
    reg.Register(&email.SendAdapter{})
    reg.Register(&http.RequestAdapter{})
    reg.Register(&slack.PostAdapter{})
    reg.Register(&transform.JQAdapter{})
    reg.Register(&ai.GenerateAdapter{})
}
```

### Future Connector Categories

| Category | Connectors | Description |
|----------|------------|-------------|
| **Communication** | `email.send`, `sms.send`, `slack.post`, `teams.post`, `push.send` | Send messages via various channels |
| **Integration** | `webhook.call`, `http.request`, `graphql.query` | Generic HTTP/API integration |
| **CRM** | `crm.upsert`, `crm.search`, `crm.delete` | Customer relationship management |
| **Payment** | `payment.charge`, `payment.refund`, `payment.subscription` | Payment processing |
| **Storage** | `storage.upload`, `storage.download`, `storage.delete` | File and object storage |
| **Database** | `db.query`, `db.insert`, `db.update` | Database operations |
| **AI** | `ai.generate`, `ai.classify`, `ai.extract`, `ai.summarize` | AI/ML model invocations |
| **Transform** | `transform.jq`, `transform.map`, `transform.filter` | Data transformation |
| **Internal** | `event.emit`, `workflow.start`, `cache.set` | Internal platform operations |
| **Custom** | `custom.function` | User-defined functions (future) |

### Plugin Evolution

| Phase | Approach | Description |
|-------|----------|-------------|
| **Phase 1** | Go interface (in-process) | All connectors compiled into the binary. Fastest execution. Full type safety. |
| **Phase 2** | Go interface (modular packages) | Connectors in separate Go packages. Imported selectively. Still compiled in. |
| **Phase 3** | Go interface (generated scaffolding) | CLI tool generates connector boilerplate from schema. Faster development. |
| **Phase 4** | HTTP connector protocol | External connectors communicate via HTTP: `POST /execute`, `GET /schema`, `GET /health`. Language-agnostic. Network boundary adds latency but enables third-party connectors. |
| **Phase 5** | Connector marketplace | Published connector catalog. Versioned. Reviewed. Community contributions. Connectors installable via configuration. |
