# Trustage Workflow Loader SDK

**Date:** 2026-05-27
**Status:** Approved

## Problem

Consumer apps (e.g. opportunities-crawler, opportunities-writer) define
Trustage workflow JSON files in their repos, but there's no mechanism to
load them into Trustage automatically. Operators must manually POST each
definition via the API, which is error-prone and not reproducible across
cluster rebuilds.

## Solution

A Go client package at `service-trustage/client/workflows` that consumer
apps import. During their migration Job (`DO_DATABASE_MIGRATE=true`),
after DB schema migrations complete, the app calls:

```go
workflows.SyncFromDir(ctx, trustageClient, "/etc/trustage-workflows")
```

This reads every `*.json` file from the directory, calls Trustage's
Connect RPC API (`CreateWorkflow` + `ActivateWorkflow`) for each, and
skips any that already exist at the same version.

## Components

### 1. `client/workflows/loader.go`

Public API:

```go
// SyncFromDir reads every *.json file in dir and ensures each workflow
// exists in Trustage at the declared version. Creates missing workflows,
// activates them, and skips ones that already exist at the same version.
// Returns nil on full success or the first hard error (network, auth).
func SyncFromDir(ctx context.Context, client WorkflowClient, dir string) error
```

`WorkflowClient` is an interface wrapping the three Connect RPC calls
the loader needs:

```go
type WorkflowClient interface {
    ListWorkflows(ctx, *ListWorkflowsRequest) (*ListWorkflowsResponse, error)
    CreateWorkflow(ctx, *CreateWorkflowRequest) (*CreateWorkflowResponse, error)
    ActivateWorkflow(ctx, *ActivateWorkflowRequest) (*ActivateWorkflowResponse, error)
}
```

This interface is satisfied by the generated Connect client
(`workflowv1connect.WorkflowServiceClient`). Tests can mock it.

Processing per file:

1. Read JSON, unmarshal into `google.protobuf.Struct` (the `dsl` field
   the API expects).
2. Extract `name` from the DSL for the existence check.
3. Call `ListWorkflows(name=X)` to check if it exists.
4. If missing: `CreateWorkflow(dsl)` → `ActivateWorkflow(id)`.
5. If exists at the same version with the same DSL hash: skip.
6. If DSL changed: `CreateWorkflow(dsl)` (creates a new version) →
   `ActivateWorkflow(id)`.
7. Log each action: created, activated, skipped, error.

### 2. `client/workflows/dsl.go`

Thin helpers:

- `parseDSLFile(path) (*structpb.Struct, string, error)` — reads a JSON
  file, returns the Struct + the extracted workflow name.
- `dslHash(s *structpb.Struct) string` — deterministic SHA-256 of the
  JSON for same-version change detection.

### 3. Consumer-side wiring

In each consumer app's `main.go`, after the existing migration block:

```go
if cfg.DoDatabaseMigrate() {
    repository.Migrate(ctx, dbManager)

    // Sync Trustage workflows from the mounted ConfigMap.
    trustageClient := buildTrustageClient(cfg)
    if err := workflows.SyncFromDir(ctx, trustageClient, cfg.TrustageWorkflowsDir); err != nil {
        log.WithError(err).Fatal("trustage workflow sync failed")
    }
    return
}
```

The workflows directory is mounted as a Kubernetes ConfigMap (same
pattern as `opportunity-kinds`). The Trustage client is constructed
with the app's existing OAuth2 credentials (`OAUTH2_SERVICE_*` env
vars that every Frame service already has).

### 4. Deployment manifests

For each consumer app that has Trustage workflows:

- A ConfigMap created from the `definitions/trustage/workflows/*.json`
  files in the consumer's repo.
- A volume mount at `/etc/trustage-workflows` in the migration Job
  container.
- `TRUSTAGE_WORKFLOWS_DIR=/etc/trustage-workflows` env var.
- `TRUSTAGE_URL=http://trustage.operations.svc` env var.

## Idempotency Contract

| Scenario | Action |
|---|---|
| Workflow doesn't exist | Create (draft) + Activate |
| Workflow exists, same version, same DSL | Skip |
| Workflow exists, DSL changed | Create new version + Activate |
| Trustage unreachable | Return error → migration Job fails → Flux retries |
| Empty or missing directory | Return nil (no-op) |
| Malformed JSON file | Return error with filename |

## Error Handling

- Network/auth errors are fatal — the migration Job fails and Flux
  retries with backoff. This is intentional: workflows are required
  for the app to function (e.g. the crawler needs scheduler-tick).
- "Already exists at same version" is a skip, not an error.
- Malformed JSON is fatal — a broken definition file is a code bug
  that should block the release.

## Testing

- Unit tests for `parseDSLFile` and `dslHash` (pure functions).
- Integration test for `SyncFromDir` using a mock `WorkflowClient`
  that asserts the correct sequence of List → Create → Activate calls.
- No need for end-to-end Trustage test — the Connect RPC client is
  generated and the server-side logic is already tested in
  service-trustage.

## Migration Path

1. Implement the `client/workflows` package in service-trustage.
2. Tag a service-trustage release so consumer apps can import it.
3. In stawi.opportunities: wire the loader into the crawler's
   migration Job, move `definitions/trustage/*.json` to a ConfigMap,
   add the volume mount + env vars.
4. On next deploy: migration Job runs, workflows are seeded, Trustage
   starts firing schedules, crawler starts crawling.

## What Changes

| Repo | Change |
|---|---|
| service-trustage | New `client/workflows/` package (loader, dsl helpers, tests) |
| stawi.opportunities | Wire loader in crawler migration, ConfigMap for workflow JSONs |
| deployment.manifests | ConfigMap + volume mount + env vars in crawler migration Job |

## What Stays the Same

- Trustage's CreateWorkflow business logic, DSL parser, schedule
  materializer — all unchanged.
- The JSON definition format — unchanged.
- The `schedule_definitions` row is created by Trustage as a
  side-effect of CreateWorkflow (existing behavior).
- Trustage's CronScheduler that fires due schedules — unchanged.
