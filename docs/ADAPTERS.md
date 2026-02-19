# Connector Adapters Reference

Connector adapters are the integration layer between trustage workflows and external systems. Each adapter is a self-contained unit that implements a single operation with typed schemas for input, configuration, and output.

## Table of Contents

- [Architecture](#architecture)
- [Adapter Interface](#adapter-interface)
- [Error Classification](#error-classification)
- [HTTP Status Classification](#http-status-classification)
- [SSRF Protection](#ssrf-protection)
- [Shared API Post Helper](#shared-api-post-helper)
- [Adapters](#adapters)
  - [`webhook.call` — Send Webhook](#webhookcall--send-webhook)
  - [`http.request` — Generic HTTP Request](#httprequest--generic-http-request)
  - [`notification.send` — Send Notification](#notificationsend--send-notification)
  - [`notification.status` — Check Notification Status](#notificationstatus--check-notification-status)
  - [`payment.initiate` — Initiate Payment](#paymentinitiate--initiate-payment)
  - [`payment.verify` — Verify Payment](#paymentverify--verify-payment)
  - [`data.transform` — Transform Data](#datatransform--transform-data)
  - [`log.entry` — Audit Log Entry](#logentry--audit-log-entry)
  - [`form.validate` — Validate Form Data](#formvalidate--validate-form-data)
  - [`approval.request` — Request Human Approval](#approvalrequest--request-human-approval)
  - [`ai.chat` — AI Chat (LLM)](#aichat--ai-chat-llm)
- [Adding a New Adapter](#adding-a-new-adapter)

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Connector Registry                       │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐       │
│  │ webhook  │ │   http   │ │ notif.   │ │ payment  │  ...   │
│  │  .call   │ │ .request │ │  .send   │ │.initiate │       │
│  └────┬─────┘ └────┬─────┘ └────┬─────┘ └────┬─────┘       │
│       │             │            │             │             │
│       └─────────────┴────────────┴─────────────┘             │
│                          │                                   │
│              ┌───────────▼───────────┐                       │
│              │   Adapter Interface   │                       │
│              │  Type()               │                       │
│              │  DisplayName()        │                       │
│              │  InputSchema()        │                       │
│              │  ConfigSchema()       │                       │
│              │  OutputSchema()       │                       │
│              │  Execute(ctx, req)    │                       │
│              │  Validate(req)        │                       │
│              └───────────────────────┘                       │
└─────────────────────────────────────────────────────────────┘

Execution Flow:
  Worker receives execution_id → loads state from PostgreSQL
    → resolves adapter from Registry by type
    → builds ExecuteRequest (input, config, credentials, metadata)
    → calls adapter.Execute(ctx, req)
    → commits ExecuteResponse or ExecutionError back to engine
```

### Source Files

| File | Purpose |
|------|---------|
| `connector/adapter.go` | `Adapter` interface definition |
| `connector/types.go` | `ExecuteRequest`, `ExecuteResponse`, `ExecutionError`, `ErrorClass` |
| `connector/registry.go` | Thread-safe adapter registry (Register, Get, List) |
| `connector/adapters/webhook.go` | Webhook adapter + HTTP status classifier |
| `connector/adapters/http.go` | Generic HTTP request adapter |
| `connector/adapters/notification.go` | Send notification adapter |
| `connector/adapters/notification_status.go` | Check notification status adapter |
| `connector/adapters/payment.go` | Initiate payment adapter |
| `connector/adapters/payment_verify.go` | Verify payment adapter |
| `connector/adapters/transform.go` | CEL data transformation adapter |
| `connector/adapters/log.go` | Structured log entry adapter |
| `connector/adapters/form_validate.go` | Form field validation adapter |
| `connector/adapters/approval.go` | Human approval request adapter |
| `connector/adapters/apipost.go` | Shared helper for API POST calls |
| `connector/adapters/urlvalidation.go` | SSRF protection (URL + IP validation) |
| `connector/adapters/ai_chat.go` | AI Chat (LLM) adapter via BAML |

### Adapter Categories

| Category | Adapters | HTTP Required | Side Effects |
|----------|----------|---------------|--------------|
| **HTTP** | `webhook.call`, `http.request` | Yes | External HTTP calls |
| **Notification** | `notification.send`, `notification.status` | Yes | Sends/checks notifications via API |
| **Payment** | `payment.initiate`, `payment.verify` | Yes | Initiates/verifies payments via API |
| **Computation** | `data.transform`, `log.entry`, `form.validate` | No | Pure in-memory, no I/O |
| **Human-in-the-loop** | `approval.request` | Yes | Sends approval request via API |
| **AI** | `ai.chat` | No (BAML-managed) | Calls LLM provider APIs |

---

## Adapter Interface

Every adapter implements the `connector.Adapter` interface:

```go
type Adapter interface {
    Type() string                                                    // Unique identifier
    DisplayName() string                                             // Human-readable name
    InputSchema() json.RawMessage                                    // JSON Schema for input
    ConfigSchema() json.RawMessage                                   // JSON Schema for configuration
    OutputSchema() json.RawMessage                                   // JSON Schema for output
    Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResponse, *ExecutionError)
    Validate(req *ExecuteRequest) error                              // Validate without executing
}
```

### ExecuteRequest

The request passed to every adapter:

```go
type ExecuteRequest struct {
    Input          map[string]any    // Adapter-specific input fields
    Config         map[string]any    // Adapter-specific configuration (e.g. api_url)
    Credentials    map[string]string // Decrypted secrets (e.g. api_key) — NEVER logged
    Metadata       map[string]string // Execution context (execution_id, instance_id, etc.)
    IdempotencyKey string            // Unique key for safe retries
}
```

- **Input**: Populated from the workflow step's input mappings after template resolution.
- **Config**: Static per-step configuration (API URLs, service endpoints).
- **Credentials**: Decrypted at dispatch time, cached in Valkey with 5-minute TTL. Never persisted in messages, logs, or audit events.
- **Metadata**: Contextual information from the execution (execution_id, instance_id, tenant_id). Used by adapters like `approval.request` to enable callback signal routing.
- **IdempotencyKey**: Set by the engine for safe retries. HTTP-based adapters forward this as the `Idempotency-Key` header.

### ExecuteResponse

Returned on success:

```go
type ExecuteResponse struct {
    Output   map[string]any  // Adapter-specific output fields
    Metadata map[string]any  // Optional execution metadata
    RawBody  json.RawMessage // Raw HTTP response body (for HTTP adapters)
}
```

### ExecutionError

Returned on failure (implements `error` interface):

```go
type ExecutionError struct {
    Class   ErrorClass     // Error classification (see below)
    Code    string         // Machine-readable error code (e.g. "HTTP_429", "SSRF_BLOCKED")
    Message string         // Human-readable error description
    Details map[string]any // Optional structured details
}
```

---

## Error Classification

Every adapter failure must return an `ExecutionError` with exactly one `ErrorClass`. The state engine uses this to determine retry behavior:

| ErrorClass | Constant | When to Use | Engine Behavior |
|------------|----------|-------------|-----------------|
| `retryable` | `ErrorRetryable` | Transient failures: timeouts, rate limits (HTTP 429), response read errors | Retry with exponential backoff per retry policy |
| `fatal` | `ErrorFatal` | Permanent failures: validation errors, bad config, SSRF blocked, HTTP 4xx (except 429) | Mark execution as `failed`, stop workflow progression |
| `compensatable` | `ErrorCompensatable` | Partial success requiring rollback | Trigger compensation workflow if defined |
| `external_dependency` | `ErrorExternalDependency` | External service unreachable or returning 5xx | Retry with potentially different strategy, longer backoff |

---

## HTTP Status Classification

All HTTP-based adapters use a shared classifier (`classifyHTTPStatus`) that maps HTTP response codes to error classes:

| HTTP Status | ErrorClass | Code | Notes |
|-------------|------------|------|-------|
| 200–299 | *(success)* | — | No error returned |
| 400, 401, 403, 404, 405, 409, 422 | `fatal` | `HTTP_{status}` | Client errors — request is malformed, won't succeed on retry |
| 429 | `retryable` | `HTTP_429` | Rate limited — retry after backoff |
| 500+ | `external_dependency` | `HTTP_{status}` | Server errors — external system problem |
| All others | `retryable` | `HTTP_{status}` | Unknown status — conservatively retry |

Response bodies included in error messages are truncated to 512 characters to prevent data leakage.

---

## SSRF Protection

All HTTP-based adapters call `validateExternalURL()` before making requests. This prevents Server-Side Request Forgery attacks against internal infrastructure.

### Blocked Schemes
Only `http` and `https` are allowed. All other schemes (file, ftp, gopher, etc.) are rejected.

### Blocked Hostnames
| Pattern | Reason |
|---------|--------|
| `localhost` | Loopback |
| `metadata.google.internal` | Cloud metadata service |
| `*.internal` | Internal DNS convention |
| `*.local` | mDNS / local network |

### Blocked IP Ranges (after DNS resolution)

| CIDR | Description |
|------|-------------|
| `10.0.0.0/8` | RFC 1918 private (Class A) |
| `172.16.0.0/12` | RFC 1918 private (Class B) |
| `192.168.0.0/16` | RFC 1918 private (Class C) |
| `127.0.0.0/8` | Loopback |
| `169.254.0.0/16` | Link-local |
| `100.64.0.0/10` | Carrier-grade NAT (RFC 6598) |
| `0.0.0.0/8` | "This" network |
| `198.18.0.0/15` | Benchmark testing (RFC 2544) |
| `224.0.0.0/4` | Multicast |
| `240.0.0.0/4` | Reserved |
| `::1/128` | IPv6 loopback |
| `fc00::/7` | IPv6 unique local |
| `fe80::/10` | IPv6 link-local |
| `fd00::/8` | IPv6 unique local |

### DNS Resolution Behavior
If DNS resolution fails, the request is **allowed to proceed** — the HTTP client will produce a more descriptive error. This prevents DNS hiccups from blocking legitimate requests.

---

## Shared API Post Helper

Adapters that call external JSON APIs with Bearer authentication share a common execution path via `executeAPIPost()` (in `apipost.go`):

```
executeAPIPost(ctx, client, req, payload)
  → Validates api_url from req.Config (required)
  → SSRF check on api_url
  → JSON-marshals payload
  → POST with Content-Type: application/json
  → Sets Idempotency-Key header if present
  → Sets Authorization: Bearer {api_key} from req.Credentials
  → Reads response (max 1MB)
  → Classifies HTTP status
  → Returns parsed JSON body
```

Used by: `notification.send`, `payment.initiate`, `approval.request`

---

## Adapters

### `webhook.call` — Send Webhook

**Type:** `webhook.call`
**Display Name:** Webhook
**Category:** HTTP
**Requires HTTP Client:** Yes
**Source:** `connector/adapters/webhook.go`

Sends an HTTP POST/PUT/PATCH request to an external webhook URL. Designed for fire-and-forget or request/response webhook integrations.

#### Input Schema

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `url` | string (URI) | Yes | — | Target webhook URL |
| `method` | string | No | `"POST"` | HTTP method: `POST`, `PUT`, or `PATCH` |
| `headers` | object (string values) | No | — | Custom HTTP headers |
| `body` | object | No | — | JSON request body |

#### Config Schema

Empty object — no configuration needed.

#### Output Schema

| Field | Type | Description |
|-------|------|-------------|
| `status_code` | integer | HTTP response status code |
| `body` | object | Parsed JSON response body |

#### Behavior

1. Validates `url` is present
2. SSRF validation on the URL
3. Defaults method to `POST` if not specified
4. Marshals `body` to JSON
5. Sets `Content-Type: application/json`
6. Applies custom headers from `headers` input
7. Sets `Idempotency-Key` header if `IdempotencyKey` is provided
8. Executes HTTP request
9. Reads response body (max 1MB)
10. Classifies HTTP status code
11. Returns parsed response

#### Error Codes

| Code | Class | Cause |
|------|-------|-------|
| `SSRF_BLOCKED` | fatal | URL targets internal network |
| `MARSHAL_ERROR` | fatal | Failed to serialize request body |
| `REQUEST_ERROR` | fatal | Failed to create HTTP request |
| `HTTP_ERROR` | external_dependency | HTTP client error (connection, DNS, timeout) |
| `READ_ERROR` | retryable | Failed to read response body |
| `HTTP_{status}` | varies | HTTP response status (see classification table) |

#### Example Workflow Step

```yaml
steps:
  - id: notify_partner
    type: call
    adapter: webhook.call
    input:
      url: "{{ config.partner_webhook_url }}"
      method: POST
      headers:
        X-Webhook-Source: trustage
        X-Event-Type: "{{ payload.event_type }}"
      body:
        event_id: "{{ payload.id }}"
        timestamp: "{{ payload.timestamp }}"
        data: "{{ payload.data }}"
```

---

### `http.request` — Generic HTTP Request

**Type:** `http.request`
**Display Name:** HTTP Request
**Category:** HTTP
**Requires HTTP Client:** Yes
**Source:** `connector/adapters/http.go`

Full-featured HTTP client adapter supporting all standard methods, query parameters, authorization headers, and custom headers. More flexible than `webhook.call` — use this when you need GET requests, query parameters, or detailed response headers.

#### Input Schema

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `url` | string (URI) | Yes | — | Target URL |
| `method` | string | Yes | — | `GET`, `POST`, `PUT`, `PATCH`, `DELETE` |
| `headers` | object (string values) | No | — | Custom HTTP headers |
| `query` | object (string values) | No | — | URL query parameters |
| `body` | object | No | — | JSON request body |
| `auth_header` | string | No | — | Value for `Authorization` header |

#### Config Schema

Empty object — no configuration needed.

#### Output Schema

| Field | Type | Description |
|-------|------|-------------|
| `status_code` | integer | HTTP response status code |
| `headers` | object | Response headers (single values as strings, multiple as arrays) |
| `body` | object | Parsed JSON response body |

#### Behavior

1. Validates `url` and `method` are present
2. SSRF validation on the URL
3. Parses URL and appends query parameters from `query` input
4. Marshals `body` to JSON (sets `Content-Type: application/json` only when body present)
5. Applies custom headers
6. Sets `Authorization` header from `auth_header` if provided
7. Sets `Idempotency-Key` header if provided
8. Executes HTTP request
9. Reads response body (max 1MB)
10. Classifies HTTP status code
11. Returns status, response headers, and parsed body

#### Differences from `webhook.call`

| Feature | `webhook.call` | `http.request` |
|---------|---------------|----------------|
| Methods | POST, PUT, PATCH | GET, POST, PUT, PATCH, DELETE |
| Query params | No | Yes |
| Auth header | No | Yes (`auth_header` field) |
| Response headers | Not returned | Returned in output |
| Default method | POST | None (required) |

#### Error Codes

Same as `webhook.call` — see error codes table above.

#### Example Workflow Step

```yaml
steps:
  - id: fetch_user
    type: call
    adapter: http.request
    input:
      url: "https://api.example.com/users"
      method: GET
      query:
        email: "{{ payload.email }}"
      auth_header: "Bearer {{ credentials.api_token }}"
```

---

### `notification.send` — Send Notification

**Type:** `notification.send`
**Display Name:** Send Notification
**Category:** Notification
**Requires HTTP Client:** Yes
**Source:** `connector/adapters/notification.go`

Dispatches a single notification (SMS, email, or push) via an external notification service API. Uses the shared `executeAPIPost` helper for API communication.

#### Input Schema

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `recipient` | string | Yes | — | Recipient address: phone number, email, or device token |
| `channel` | string | Yes | — | Delivery channel: `sms`, `email`, or `push` |
| `subject` | string | Required for email | — | Notification subject line |
| `body` | string | Yes | — | Notification body text |
| `template_id` | string | No | — | Template identifier for the notification service |
| `template_vars` | object | No | — | Variables for server-side template rendering |

#### Config Schema

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `api_url` | string (URI) | Yes | Notification service API endpoint |

#### Output Schema

| Field | Type | Description |
|-------|------|-------------|
| `notification_id` | string | ID assigned by the notification service |
| `status` | string | Delivery status (default: `"sent"`) |
| `channel` | string | Channel used (echoed from input) |

#### Validation Rules

- `recipient` must be present
- `channel` must be one of: `sms`, `email`, `push`
- `body` must be present
- `subject` is required when `channel` is `email`

#### API Payload

The adapter POSTs the following JSON to `config.api_url`:

```json
{
  "recipient": "...",
  "channel": "sms|email|push",
  "body": "...",
  "subject": "...",           // optional
  "template_id": "...",      // optional
  "template_vars": { ... }   // optional
}
```

#### Credential Usage

If `credentials.api_key` is set, it is sent as `Authorization: Bearer {api_key}`.

#### Example Workflow Step

```yaml
steps:
  - id: send_otp
    type: call
    adapter: notification.send
    config:
      api_url: "https://notify.stawi.dev/api/v1/send"
    input:
      recipient: "{{ payload.phone }}"
      channel: sms
      body: "Your verification code is {{ vars.otp_code }}"
```

---

### `notification.status` — Check Notification Status

**Type:** `notification.status`
**Display Name:** Check Notification Status
**Category:** Notification
**Requires HTTP Client:** Yes
**Source:** `connector/adapters/notification_status.go`

Polls the notification service for the delivery status of a previously sent notification. Designed to be used in a polling loop or after a delay step following `notification.send`.

#### Input Schema

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `notification_id` | string | Yes | ID of the notification to check (from `notification.send` output) |

#### Config Schema

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `api_url` | string (URI) | Yes | Notification service API base URL |

#### Output Schema

| Field | Type | Description |
|-------|------|-------------|
| `notification_id` | string | ID of the checked notification |
| `status` | string | One of: `pending`, `sent`, `delivered`, `failed`, `bounced` |
| `delivered_at` | string (date-time) | Delivery timestamp (if delivered) |
| `error` | string | Error message (if failed/bounced) |

#### Behavior

1. Validates `notification_id` is present
2. Constructs status URL: `{api_url}/{notification_id}`
3. SSRF validation on the constructed URL
4. Sends GET request with `Accept: application/json`
5. Uses `credentials.api_key` as Bearer token if present
6. Returns parsed status fields from API response

#### Example Workflow Step

```yaml
steps:
  - id: check_delivery
    type: call
    adapter: notification.status
    config:
      api_url: "https://notify.stawi.dev/api/v1/notifications"
    input:
      notification_id: "{{ steps.send_otp.output.notification_id }}"
```

---

### `payment.initiate` — Initiate Payment

**Type:** `payment.initiate`
**Display Name:** Initiate Payment
**Category:** Payment
**Requires HTTP Client:** Yes
**Source:** `connector/adapters/payment.go`

Initiates a single payment transaction via an external payment service API. Supports mobile money, bank transfer, and card payments. Uses the shared `executeAPIPost` helper.

#### Input Schema

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `amount` | number | Yes | — | Payment amount (minimum: 0) |
| `currency` | string | Yes | — | ISO 4217 currency code (3 characters, e.g. `KES`, `USD`) |
| `recipient` | string | Yes | — | Recipient identifier (phone number, account number, etc.) |
| `reference` | string | Yes | — | Unique payment reference for idempotency/tracking |
| `description` | string | No | — | Human-readable payment description |
| `method` | string | No | — | Payment method: `mobile_money`, `bank_transfer`, or `card` |

#### Config Schema

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `api_url` | string (URI) | Yes | Payment service API endpoint |

#### Output Schema

| Field | Type | Description |
|-------|------|-------------|
| `payment_id` | string | ID assigned by the payment service |
| `status` | string | Payment status (default: `"initiated"`) |
| `reference` | string | Payment reference (echoed from input) |

#### API Payload

```json
{
  "amount": 1000,
  "currency": "KES",
  "recipient": "+254712345678",
  "reference": "PAY-20240101-001",
  "description": "Monthly subscription",  // optional
  "method": "mobile_money"                 // optional
}
```

#### Example Workflow Step

```yaml
steps:
  - id: pay_vendor
    type: call
    adapter: payment.initiate
    config:
      api_url: "https://payments.stawi.dev/api/v1/payments"
    input:
      amount: "{{ payload.amount }}"
      currency: "{{ payload.currency }}"
      recipient: "{{ payload.vendor_phone }}"
      reference: "{{ payload.invoice_id }}"
      method: mobile_money
      description: "Invoice {{ payload.invoice_id }} payment"
```

---

### `payment.verify` — Verify Payment

**Type:** `payment.verify`
**Display Name:** Verify Payment
**Category:** Payment
**Requires HTTP Client:** Yes
**Source:** `connector/adapters/payment_verify.go`

Checks the status of a previously initiated payment. Used after `payment.initiate` to confirm completion, typically with a delay or polling loop.

#### Input Schema

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `payment_id` | string | Yes | ID of the payment to verify (from `payment.initiate` output) |

#### Config Schema

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `api_url` | string (URI) | Yes | Payment service API base URL |

#### Output Schema

| Field | Type | Description |
|-------|------|-------------|
| `payment_id` | string | ID of the checked payment |
| `status` | string | One of: `pending`, `processing`, `completed`, `failed`, `reversed` |
| `amount` | number | Payment amount (from service response) |
| `currency` | string | Currency code (from service response) |
| `completed_at` | string (date-time) | Completion timestamp (if completed) |
| `error` | string | Error message (if failed) |

#### Behavior

1. Validates `payment_id` is present
2. Constructs verify URL: `{api_url}/{payment_id}`
3. SSRF validation on the constructed URL
4. Sends GET request with `Accept: application/json`
5. Uses `credentials.api_key` as Bearer token if present
6. Extracts: `status`, `amount`, `currency`, `completed_at`, `error` from response

#### Example: Payment with Verification Loop

```yaml
steps:
  - id: initiate
    type: call
    adapter: payment.initiate
    config:
      api_url: "https://payments.stawi.dev/api/v1/payments"
    input:
      amount: "{{ payload.amount }}"
      currency: KES
      recipient: "{{ payload.phone }}"
      reference: "{{ payload.ref }}"

  - id: wait_processing
    type: wait
    duration: 30s

  - id: verify
    type: call
    adapter: payment.verify
    config:
      api_url: "https://payments.stawi.dev/api/v1/payments"
    input:
      payment_id: "{{ steps.initiate.output.payment_id }}"

  - id: check_status
    type: condition
    expression: "steps.verify.output.status == 'completed'"
    on_true: payment_success
    on_false: payment_failed
```

---

### `data.transform` — Transform Data

**Type:** `data.transform`
**Display Name:** Transform Data
**Category:** Computation
**Requires HTTP Client:** No
**Source:** `connector/adapters/transform.go`

Reshapes data between workflow steps using CEL (Common Expression Language) expressions. Pure computation — no HTTP calls, no side effects. Useful for data mapping, aggregation, filtering, and field extraction.

#### Input Schema

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `source` | object | Yes | — | Source data to transform |
| `expression` | string | No* | — | Single CEL expression evaluated against source |
| `mappings` | object (string values) | No* | — | Map of `output_key` → CEL expression pairs |

*At least one of `expression` or `mappings` is required. Both can be provided simultaneously.

#### Config Schema

Empty object — no configuration needed.

#### Output Schema

| Field | Type | Description |
|-------|------|-------------|
| `result` | any | Result of `expression` evaluation (only present if `expression` was provided) |
| `data` | object | Results of `mappings` evaluations (only present if `mappings` was provided) |

#### CEL Environment

Expressions are evaluated in the trustage CEL environment (`dsl.NewExpressionEnv()`) with these variables:

| Variable | Value | Description |
|----------|-------|-------------|
| `payload` | `source` input | Alias for the source data |
| `vars` | `source` input | Alias for the source data |
| `metadata` | `{}` | Empty metadata map |
| `env` | `{}` | Empty environment map |

#### Error Codes

| Code | Class | Cause |
|------|-------|-------|
| `CEL_ENV_ERROR` | fatal | Failed to create CEL environment |
| `EXPRESSION_ERROR` | fatal | Expression compilation or evaluation failed |
| `MAPPING_ERROR` | fatal | Mapping expression is not a string, or compilation/evaluation failed |

#### Example: Single Expression

```yaml
steps:
  - id: calc_total
    type: call
    adapter: data.transform
    input:
      source: "{{ payload }}"
      expression: "payload.items.map(i, i.price * i.quantity).reduce(a, b, a + b)"
```

Output: `{ "result": 1250.00 }`

#### Example: Multiple Mappings

```yaml
steps:
  - id: reshape
    type: call
    adapter: data.transform
    input:
      source:
        first_name: "{{ payload.first }}"
        last_name: "{{ payload.last }}"
        scores: "{{ payload.exam_scores }}"
      mappings:
        full_name: "payload.first_name + ' ' + payload.last_name"
        avg_score: "math.mean(payload.scores)"
        passed: "math.mean(payload.scores) >= 50.0"
        grade: "math.mean(payload.scores) >= 80.0 ? 'A' : math.mean(payload.scores) >= 60.0 ? 'B' : 'C'"
```

Output:
```json
{
  "data": {
    "full_name": "John Doe",
    "avg_score": 85.0,
    "passed": true,
    "grade": "A"
  }
}
```

---

### `log.entry` — Audit Log Entry

**Type:** `log.entry`
**Display Name:** Log Entry
**Category:** Computation
**Requires HTTP Client:** No
**Source:** `connector/adapters/log.go`

Records a structured audit log entry into the workflow execution output. Pure computation — no HTTP calls, always succeeds. Useful for debugging workflows, recording milestones, or creating audit trails within workflow execution history.

#### Input Schema

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `level` | string | Yes | — | Severity: `info`, `warn`, `error`, `debug` |
| `message` | string | Yes | — | Log message |
| `data` | object | No | — | Structured key-value data to include |

#### Config Schema

Empty object — no configuration needed.

#### Output Schema

| Field | Type | Description |
|-------|------|-------------|
| `logged` | boolean | Always `true` |
| `timestamp` | string (date-time) | UTC timestamp in RFC 3339 format |
| `level` | string | Echoed severity level |
| `message` | string | Echoed message |
| `data` | object | Echoed structured data (if provided) |

#### Behavior

This adapter never fails. It captures the log entry data into the execution output, which is persisted as part of the workflow execution record. The output can be viewed in instance timelines.

#### Example Workflow Step

```yaml
steps:
  - id: log_start
    type: call
    adapter: log.entry
    input:
      level: info
      message: "Payment workflow started"
      data:
        customer_id: "{{ payload.customer_id }}"
        amount: "{{ payload.amount }}"
        currency: "{{ payload.currency }}"
```

---

### `form.validate` — Validate Form Data

**Type:** `form.validate`
**Display Name:** Validate Form Data
**Category:** Computation
**Requires HTTP Client:** No
**Source:** `connector/adapters/form_validate.go`

Validates form submission fields against required field and type rules. Pure computation — no HTTP calls. Returns a fatal error with validation details if validation fails, allowing the workflow to branch on validation results.

#### Input Schema

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `fields` | object | Yes | — | Form fields to validate |
| `required_fields` | array of strings | Yes | — | Field names that must be present and non-empty |
| `field_types` | object (string values) | No | — | Expected type per field: `string`, `number`, `boolean`, `array`, `object` |

#### Config Schema

Empty object — no configuration needed.

#### Output Schema (on success)

| Field | Type | Description |
|-------|------|-------------|
| `valid` | boolean | `true` |
| `errors` | array of strings | Empty array |
| `fields` | object | Pass-through of validated fields |

#### Error Behavior (on failure)

Returns `ExecutionError` with:
- **Class:** `fatal`
- **Code:** `VALIDATION_FAILED`
- **Message:** `"form validation failed: N errors"`
- **Details:** `{ "valid": false, "errors": [...], "fields": {...} }`

#### Validation Rules

**Required fields check:**
- Field must exist in `fields` object
- Field must not be an empty string
- Error: `"missing required field \"name\""` or `"field \"name\" must not be empty"`

**Type check (only for present fields):**
- `string` → Go `string`
- `number` → Go `float64` (JSON number)
- `boolean` → Go `bool`
- `array` → Go `[]any`
- `object` → Go `map[string]any`
- Error: `"field \"age\": expected type number"`

#### Example Workflow Step

```yaml
steps:
  - id: validate_signup
    type: call
    adapter: form.validate
    input:
      fields: "{{ payload.form_data }}"
      required_fields:
        - name
        - email
        - age
        - accepted_terms
      field_types:
        name: string
        email: string
        age: number
        accepted_terms: boolean
```

---

### `approval.request` — Request Human Approval

**Type:** `approval.request`
**Display Name:** Request Approval
**Category:** Human-in-the-loop
**Requires HTTP Client:** Yes
**Source:** `connector/adapters/approval.go`

Sends a human approval request via an external notification/approval service. Designed to be paired with a `signal_wait` step that pauses the workflow until the approver responds. The adapter includes `execution_id` and `instance_id` in the API payload to enable the approval service to signal back to the correct workflow instance.

#### Input Schema

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `approver` | string | Yes | — | Approver identifier (email, user ID, phone) |
| `title` | string | Yes | — | Approval request title |
| `description` | string | No | — | Detailed description of what needs approval |
| `options` | array of strings | No | `["approve", "reject"]` | Available response options |
| `callback_url` | string (URI) | No | — | URL where the approver can respond |
| `expires_in` | string | No | — | Expiration duration (e.g. `24h`, `7d`) |

#### Config Schema

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `api_url` | string (URI) | Yes | Approval/notification service API endpoint |

#### Output Schema

| Field | Type | Description |
|-------|------|-------------|
| `request_id` | string | ID assigned by the approval service |
| `status` | string | One of: `pending`, `sent`, `failed` |
| `approver` | string | Approver identifier (echoed from input) |

#### API Payload

The adapter POSTs:

```json
{
  "type": "approval",
  "approver": "manager@example.com",
  "title": "Expense Approval Required",
  "description": "...",
  "options": ["approve", "reject"],
  "callback_url": "...",
  "expires_in": "24h",
  "execution_id": "exec_abc123",
  "instance_id": "inst_xyz789"
}
```

The `execution_id` and `instance_id` are extracted from `req.Metadata` and included automatically. This allows the approval service to construct a signal-back webhook that resumes the correct workflow instance.

#### Example: Approval with Signal Wait

```yaml
steps:
  - id: request_approval
    type: call
    adapter: approval.request
    config:
      api_url: "https://approvals.stawi.dev/api/v1/requests"
    input:
      approver: "{{ payload.manager_email }}"
      title: "Approve expense: {{ payload.description }}"
      description: "Amount: {{ payload.currency }} {{ payload.amount }}"
      options:
        - approve
        - reject
        - escalate
      expires_in: "48h"

  - id: wait_for_decision
    type: signal_wait
    signal: approval_response
    timeout: 48h

  - id: check_decision
    type: condition
    expression: "steps.wait_for_decision.output.decision == 'approve'"
    on_true: process_payment
    on_false: notify_requester_rejected
```

---

### `ai.chat` — AI Chat (LLM)

**Type:** `ai.chat`
**Display Name:** AI Chat
**Category:** AI
**Requires HTTP Client:** No (BAML manages its own HTTP transport)
**Source:** `connector/adapters/ai_chat.go`

Sends conversation messages to an LLM provider and returns the generated response. Provider-agnostic — switching between OpenAI, Anthropic, Google, Azure, Bedrock, Vertex, or any OpenAI-compatible API requires only configuration changes. Uses [BAML](https://docs.boundaryml.com/) for provider abstraction and structured output handling.

#### Input Schema

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `messages` | array of objects | Yes | — | Conversation message history |
| `messages[].role` | string | Yes | — | Message role: `user` or `assistant` |
| `messages[].content` | string | Yes | — | Message content |
| `system` | string | No | `""` | System prompt for the LLM |

#### Config Schema

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `provider` | string | Yes | — | BAML provider: `openai`, `anthropic`, `google-ai`, `vertex-ai`, `aws-bedrock`, `azure-openai`, `openai-generic` |
| `model` | string | Yes | — | Model identifier (e.g. `gpt-4o`, `claude-sonnet-4-20250514`, `gemini-2.0-flash`) |
| `base_url` | string (URI) | No | — | Override provider API base URL |
| `temperature` | number (0–2) | No | — | Sampling temperature |
| `max_tokens` | integer | No | — | Maximum response tokens |

#### Credentials

| Key | Description |
|-----|-------------|
| `api_key` | Provider API key (required) |

#### Output Schema

| Field | Type | Description |
|-------|------|-------------|
| `content` | string | Generated response text |
| `model` | string | Model that produced the response |

#### Error Codes

| Code | Class | Cause |
|------|-------|-------|
| `CONFIG_ERROR` | fatal | Missing provider or model in config |
| `CREDENTIALS_ERROR` | fatal | Missing api_key credential |
| `INPUT_ERROR` | fatal | Invalid message format or role |
| `AUTH_ERROR` | fatal | Provider rejected API key (401) |
| `REQUEST_ERROR` | fatal | Invalid request (400, 422, unknown model) |
| `RATE_LIMITED` | retryable | Provider rate limit (429) |
| `TIMEOUT` | retryable | Request timed out |
| `CANCELLED` | retryable | Request context cancelled |
| `CONNECTION_ERROR` | external_dependency | Provider unreachable (DNS, connection refused) |
| `PROVIDER_ERROR` | external_dependency | Provider server error (5xx) |
| `LLM_ERROR` | external_dependency | Unclassified provider error |

#### Example: Intent Classification with Anthropic

```yaml
steps:
  - id: classify_intent
    type: call
    adapter: ai.chat
    config:
      provider: anthropic
      model: claude-sonnet-4-20250514
      temperature: 0
      max_tokens: 100
    input:
      system: "Classify the user message intent. Respond with exactly one of: support, billing, sales, spam"
      messages:
        - role: user
          content: "{{ payload.message }}"
```

#### Example: Content Generation with OpenAI

```yaml
steps:
  - id: generate_summary
    type: call
    adapter: ai.chat
    config:
      provider: openai
      model: gpt-4o
      temperature: 0.3
      max_tokens: 500
    input:
      system: "Summarize the following document in 2-3 sentences."
      messages:
        - role: user
          content: "{{ payload.document_text }}"
```

#### Example: Multi-turn Conversation

```yaml
steps:
  - id: chat_response
    type: call
    adapter: ai.chat
    config:
      provider: openai-generic
      model: llama-3.1-70b
      base_url: "https://api.together.xyz/v1"
    input:
      system: "You are a helpful customer support agent for Stawi."
      messages: "{{ payload.conversation_history }}"
```

#### Example: Google AI

```yaml
steps:
  - id: extract_entities
    type: call
    adapter: ai.chat
    config:
      provider: google-ai
      model: gemini-2.0-flash
      temperature: 0
    input:
      system: "Extract all person names, organizations, and locations from the text. Return as JSON."
      messages:
        - role: user
          content: "{{ payload.text }}"
```

---

## Adding a New Adapter

### 1. Create the adapter file

Create `connector/adapters/your_adapter.go`:

```go
package adapters

import (
    "context"
    "encoding/json"

    "github.com/antinvestor/service-trustage/connector"
)

const (
    yourType        = "your.type"
    yourDisplayName = "Your Adapter"
)

type YourAdapter struct {
    // Include *http.Client if adapter makes HTTP calls
}

func NewYourAdapter() *YourAdapter {
    return &YourAdapter{}
}

func (a *YourAdapter) Type() string        { return yourType }
func (a *YourAdapter) DisplayName() string { return yourDisplayName }
func (a *YourAdapter) InputSchema() json.RawMessage { return json.RawMessage(`{...}`) }
func (a *YourAdapter) ConfigSchema() json.RawMessage { return json.RawMessage(`{...}`) }
func (a *YourAdapter) OutputSchema() json.RawMessage { return json.RawMessage(`{...}`) }

func (a *YourAdapter) Validate(req *connector.ExecuteRequest) error {
    // Validate required fields
    return nil
}

func (a *YourAdapter) Execute(
    ctx context.Context,
    req *connector.ExecuteRequest,
) (*connector.ExecuteResponse, *connector.ExecutionError) {
    // Implementation
    return &connector.ExecuteResponse{Output: map[string]any{...}}, nil
}
```

### 2. Register in main.go

Add to `apps/default/cmd/main.go`:

```go
if regErr := registry.Register(adapters.NewYourAdapter()); regErr != nil {
    log.WithError(regErr).Fatal("failed to register your adapter")
}
```

### 3. Conventions

- **One adapter per file** — keep each adapter self-contained
- **Use `executeAPIPost`** for adapters that POST JSON to external APIs with Bearer auth
- **Always call `validateExternalURL`** before any HTTP request to external URLs
- **Always call `classifyHTTPStatus`** to convert HTTP status codes to `ExecutionError`
- **Limit response body reads** to `maxResponseBody` (1MB)
- **Set `Idempotency-Key` header** from `req.IdempotencyKey` for all HTTP requests
- **Use `credentials.api_key`** for Bearer auth — never hard-code credentials
- **Classify every error** — return `ExecutionError` with the correct `ErrorClass`, never return a bare Go error from `Execute`
- **Truncate error details** — use `truncateBody()` for response bodies in error messages to prevent data leakage
- **No infrastructure imports** — adapters should not import Frame, NATS, or database packages directly
