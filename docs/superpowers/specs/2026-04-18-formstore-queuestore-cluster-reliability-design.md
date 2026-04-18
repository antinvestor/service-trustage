# formstore & queuestore cluster reliability

**Date:** 2026-04-18
**Status:** approved
**Scope:** restore `trustage-formstore` and `trustage-queuestore` to a reliably running state in the `antinvestor-cluster`, aligned with the lifecycle pattern used by `trustage-api`.

## Problem

Both services are broken in the cluster:

- `trustage-queuestore` pod crash-loops on startup with `no such user (SQLSTATE 08P01)` followed by a SIGSEGV in `repository.Migrate`.
- `trustage-formstore` HelmRelease churns through install/uninstall remediation because its Deployment never becomes Ready — same root cause.

Investigation found three distinct defects stacked on top of each other.

### 1. CNPG managed roles never defined

The CNPG `Cluster/hub` manifest (`deployments/manifests/namespaces/datastore/setup_datastore.yaml`) defines parent + blue/green roles for every service *except* `trustage-formstore` and `trustage-queuestore`. Only `trustage-trustage` is present in the trustage block.

Consequence: `Database/formstore` and `Database/queuestore` CRs sit in `APPLIED=false` with `role "trustage-formstore" does not exist (SQLSTATE 42704)` (and the queuestore equivalent). All downstream secrets (`db-credentials-{formstore,queuestore}` in the `trustage` namespace, Vault entries, blue/green ExternalSecrets) are correctly populated — PostgreSQL simply doesn't know the role.

### 2. `repository.Migrate` panics on DB init failure

Both `apps/formstore/service/repository/migrate.go` and `apps/queue/service/repository/migrate.go` do:

```go
dbPool := manager.GetPool(ctx, datastore.DefaultPoolName)
db := dbPool.DB(ctx, false)   // returns nil when pool init failed
err := db.AutoMigrate(...)    // nil deref → panic
```

When the DB user doesn't exist, Frame logs the connect error and returns a pool whose `.DB()` is nil. AutoMigrate then panics with SIGSEGV rather than returning a clean fatal. This is a bug independent of the role issue.

### 3. Migration runs in-process on every pod startup

`main.go` for both services calls `repository.Migrate()` unconditionally, and `migration.enabled: false` in Helm values — so there is no pre-install migration Job. Contrast with `trustage-api` (`apps/default/cmd/main.go:73-83`), which gates migration behind `cfg.DoDatabaseMigrate()` and pairs it with `migration.enabled: true` in its HelmRelease, producing a proper pre-install hook Job.

At current single-replica deployment this is latent, but `autoscaling.maxReplicas: 12` makes a concurrent-DDL race possible on scale-out.

### 4. `readinessProbe` uses `tcpSocket`

Both manifests configure `readinessProbe: { tcpSocket: { port: http } }`. A TCP listener being open is not the same as the service being ready to serve — the binaries already expose `/readyz` (which checks the DB pool) on `publicMux` *before* OIDC middleware is attached, so the HTTP probe is both safe and more meaningful.

## Goals

- Pods start, stay Running, and roll cleanly under normal operations.
- Migration is decoupled from serving-pod startup via the colony chart's pre-install hook.
- Readiness signals track actual service health (DB pool), not just listener presence.
- Pattern parity with `trustage-api` so the three trustage services behave identically w.r.t. lifecycle.

## Non-goals

- Reworking the Frame migration strategy itself (e.g., removing the serve-time `Migrate()` call). The api service still calls `Migrate()` on serving pods as a belt-and-braces idempotent step; this design intentionally preserves that pattern.
- Introducing a generic migration runner, schema-versioning, or out-of-band SQL migrations.
- Revisiting liveness/startup probes — only readiness changes.
- Changing HPA, PDB, topology spread, or resource limits.
- Adding new observability surfaces or dashboards.

## Change set

### Part A — `deployments` repo, CNPG roles

File: `manifests/namespaces/datastore/setup_datastore.yaml`

After the existing `trustage-trustage` block, append two stanzas mirroring it exactly — one for `trustage-formstore`, one for `trustage-queuestore`. Each stanza contains three managed roles:

1. Parent role (`trustage-formstore` / `trustage-queuestore`): `login: false`, `createdb: false`, `inherit: true`.
2. Blue login role: `login: true`, `passwordSecret.name: trustage-{formstore,queuestore}-blue`, `inRoles: [<parent>]`.
3. Green login role: same shape as blue with `-green` suffix.

Password secrets `trustage-{formstore,queuestore}-{blue,green}` already exist in the `datastore` namespace (populated by the existing ExternalSecret + password generator), so no secret-side work is needed.

### Part B — `service-trustage` repo, Go changes

#### B1. Make `Migrate()` fail cleanly instead of panicking

Files:

- `apps/formstore/service/repository/migrate.go`
- `apps/queue/service/repository/migrate.go`

At the top of `Migrate()`, after `dbPool := manager.GetPool(...)` and `db := dbPool.DB(...)`, add nil-guards that return wrapped errors. The caller in `main.go` already treats a non-nil error as `log.Fatal`, so this converts SIGSEGV into a readable fatal log.

#### B2. Gate migration behind `cfg.DoDatabaseMigrate()`

Files:

- `apps/formstore/cmd/main.go`
- `apps/queue/cmd/main.go`

Replace the single unconditional `Migrate()` call with the two-block pattern from `apps/default/cmd/main.go:73-83`:

```go
if cfg.DoDatabaseMigrate() {
    if migrateErr := repository.Migrate(ctx, dbManager); migrateErr != nil {
        log.WithError(migrateErr).Fatal("database migration failed")
    }
    log.Debug("database migration completed")
    return
}

if migrateErr := repository.Migrate(ctx, dbManager); migrateErr != nil {
    log.WithError(migrateErr).Fatal("database migration failed")
}
```

`DoDatabaseMigrate()` (Frame `ConfigurationDefault.DoDatabaseMigrate`) is true when `DO_MIGRATION=true` or `argv[0] == "migrate"`. The colony chart's migration Job sets both, so the Job exits after migrating; serving pods take the second branch and run AutoMigrate idempotently.

### Part C — `deployments` repo, Helm value changes

Files:

- `manifests/namespaces/trustage/formstore/trustage-formstore.yaml`
- `manifests/namespaces/trustage/queuestore/trustage-queuestore.yaml`

Two edits each.

#### C1. Enable the migration Job

Replace:

```yaml
migration:
  enabled: false
```

with a block mirroring `trustage-api.yaml:101-115`, providing DB creds + primary pooler URL for the database of that service:

```yaml
migration:
  enabled: true
  env:
    - name: DATABASE_USERNAME
      valueFrom:
        secretKeyRef:
          name: db-credentials-formstore   # or -queuestore
          key: username
    - name: DATABASE_PASSWORD
      valueFrom:
        secretKeyRef:
          name: db-credentials-formstore   # or -queuestore
          key: password
    - name: DATABASE_URL
      value: "postgresql://$(DATABASE_USERNAME):$(DATABASE_PASSWORD)@pooler-rw.datastore.svc:5432/formstore?sslmode=require"
```

The colony chart (`charts/colony@1.10.3/templates/migration-job.yaml`) already:

- Annotates the Job with `helm.toolkit.fluxcd.io/hook: pre-install,pre-upgrade`.
- Sets `args: ["migrate"]` and `DO_MIGRATION=true`.
- Sets `ttlSecondsAfterFinished: 300`, `backoffLimit: 3`, `activeDeadlineSeconds: 600`.

No chart changes needed.

#### C2. Switch readinessProbe to HTTP

Replace:

```yaml
readinessProbe:
  httpGet: null
  tcpSocket:
    port: http
  initialDelaySeconds: 5
  periodSeconds: 5
  timeoutSeconds: 3
  failureThreshold: 3
```

with:

```yaml
readinessProbe:
  httpGet:
    path: /readyz
    port: http
  initialDelaySeconds: 5
  periodSeconds: 5
  timeoutSeconds: 3
  failureThreshold: 3
```

`publicMux` serves `/readyz` in both services (`apps/formstore/cmd/main.go:52-60`, `apps/queue/cmd/main.go:125-133`) and registers it *before* the tenancy-access middleware is attached to `/`, so the probe does not hit OIDC. Leave `startupProbe` and `livenessProbe` as `tcpSocket`.

## Rollout sequence

The steps are strictly ordered — each unblocks the next.

1. **Land Part A (deployments).** Commit + push. Wait for Flux. Verify with `kubectl -n datastore get database formstore queuestore` showing `APPLIED=true` and `kubectl -n datastore exec hub-1 -c postgres -- psql -U postgres -c '\du trustage-formstore*'` listing three roles.

2. **Land Part B (service-trustage).** Run `make tests && make lint && go build ./apps/formstore/cmd/... ./apps/queue/cmd/...` locally. Tag a release; the existing release workflow builds both Dockerfiles via the matrix in `.github/workflows/release.yaml`. Confirm the new tag resolves in both `ImagePolicy` resources.

3. **Land Part C (deployments).** Commit + push. Flux reconciles. The colony chart runs the pre-install migration Job, then rolls the serving Deployment. Serving pod's second `Migrate()` call is a no-op.

## Verification

Cluster-side, after Step 3:

```bash
kubectl -n trustage get helmrelease trustage-formstore trustage-queuestore         # READY=True
kubectl -n trustage get jobs | grep migration                                      # Completions 1/1
kubectl -n trustage get pods                                                       # 1/1 Running, 0 restarts
kubectl -n trustage logs deploy/trustage-formstore | head                          # no panic
kubectl -n trustage exec deploy/trustage-formstore -- wget -qO- localhost:8081/readyz   # "ok"
```

Through the Gateway:

```bash
curl -sS -o /dev/null -w "%{http_code}\n" https://api.stawi.dev/formstore/healthz   # 200
curl -sS -o /dev/null -w "%{http_code}\n" https://api.stawi.dev/queuestore/healthz  # 200
```

Operational sanity:

```bash
kubectl -n trustage rollout restart deploy/trustage-formstore
# expect: new migration Job runs and completes → new pod starts → readiness hits /readyz → rollout completes clean
```

## Success criteria

- Both HelmReleases `READY=True` for ≥10 minutes with no CrashLoopBackOff.
- CNPG `Database/formstore` and `Database/queuestore` `APPLIED=true`.
- `/healthz` returns 200 through the Gateway on `api.stawi.{org,dev,im}` for both services.
- A manual rolling restart completes cleanly, with a fresh migration Job running and exiting 0 before the new pod becomes Ready.

## Rollback

- **After Step 1 only:** revert the `setup_datastore.yaml` commit. CNPG leaves existing roles in place (managed-role removal is not `ensure: absent`), so this is non-destructive.
- **After Step 2 (image built):** no rollback needed if Step 3 hasn't landed — the cluster still runs the old broken manifests.
- **After Step 3:** preferred path is "revert forward" (undo Part C, re-land with fix), because reverting to the prior state restores the original crashloop. Emergency rollback: `flux suspend helmrelease trustage-{formstore,queuestore} -n trustage` to stop the install loop while diagnosing.

## Risks & non-resolved items

1. **Migration Job timeout.** `activeDeadlineSeconds: 600` is chart default. AutoMigrate on empty/tiny schema is well under this, but worth revisiting if we accumulate large index builds.
2. **Concurrent scale-up AutoMigrate.** Serving pods still run `Migrate()` at startup for api parity. GORM AutoMigrate is idempotent when schemas match, so the realistic race is a no-op. Flagged here, not addressed in this design.
3. **No formal integration test against the real cluster.** Verification is manual `kubectl` + `curl`. A follow-up could add a smoke-test job, but out of scope here.

## References

- Parity target: `apps/default/cmd/main.go:73-83` (migration gating) and `deployments/manifests/namespaces/trustage/api/trustage-api.yaml:101-115` (migration Helm values).
- Colony chart migration template: `charts/charts/colony/templates/migration-job.yaml`.
- Frame migration flag: `github.com/pitabwire/frame/config.ConfigurationDefault.DoDatabaseMigrate` (`DO_MIGRATION` env or `migrate` argv).
