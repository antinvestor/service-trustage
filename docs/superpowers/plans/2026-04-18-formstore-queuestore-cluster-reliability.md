# formstore & queuestore Cluster Reliability Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `trustage-formstore` and `trustage-queuestore` boot and stay healthy in the `antinvestor-cluster`, aligned with the lifecycle pattern used by `trustage-api` (migration-as-job + idempotent serve-time migrate).

**Architecture:** Three-part change across two repos. (A) Add missing CNPG managed roles to `deployments`. (B) Fix `repository.Migrate` nil-deref and gate migration behind `cfg.DoDatabaseMigrate()` in `service-trustage`. (C) Enable the colony chart's pre-install migration Job and switch `readinessProbe` from `tcpSocket` to `httpGet /readyz` in both HelmReleases.

**Tech Stack:** Go 1.26, Frame (`github.com/pitabwire/frame@v1.94.0`), GORM AutoMigrate, CloudNativePG, Flux, Helm (colony chart v1.10.3), Kustomize.

**Repos & working dirs:**
- `service-trustage` — `/home/j/code/antinvestor/service-trustage` (code changes in Part B).
- `deployments` — `/home/j/code/antinvestor/deployments` (YAML changes in Parts A + C).

**Spec:** `docs/superpowers/specs/2026-04-18-formstore-queuestore-cluster-reliability-design.md`.

**Strict ordering:** Task 1 must land and reconcile before Task 6 (release); Task 6 must produce an image before Tasks 7–8; verification (Task 9) only makes sense after 7–8.

---

## File Structure

**Part A (`deployments`) — modify:**
- `manifests/namespaces/datastore/setup_datastore.yaml` — append two managed-role stanzas to CNPG `Cluster/hub`.

**Part B (`service-trustage`) — modify:**
- `apps/formstore/service/repository/migrate.go` — add nil-guards at top of `Migrate()`.
- `apps/formstore/service/repository/repository_suite_test.go` — add `TestMigrate_ReturnsErrorWhenPoolMissing`.
- `apps/queue/service/repository/migrate.go` — add nil-guards at top of `Migrate()`.
- `apps/queue/service/repository/repository_suite_test.go` — add `TestMigrate_ReturnsErrorWhenPoolMissing`.
- `apps/formstore/cmd/main.go` — gate `Migrate` call behind `cfg.DoDatabaseMigrate()`.
- `apps/queue/cmd/main.go` — same.

**Part C (`deployments`) — modify:**
- `manifests/namespaces/trustage/formstore/trustage-formstore.yaml` — populate `migration` block, switch `readinessProbe` to `httpGet`.
- `manifests/namespaces/trustage/queuestore/trustage-queuestore.yaml` — same shape.

---

## Task 1: Add CNPG managed roles for formstore & queuestore

**Files:**
- Modify: `/home/j/code/antinvestor/deployments/manifests/namespaces/datastore/setup_datastore.yaml` (append after the `trustage-trustage` stanza ending around line 565)

- [ ] **Step 1: Open the file and locate the `trustage-trustage` block**

Run:
```bash
grep -n "trustage-trustage" /home/j/code/antinvestor/deployments/manifests/namespaces/datastore/setup_datastore.yaml
```
Expected: matches around lines 544–565 (one parent role, one blue, one green).

- [ ] **Step 2: Append the two new stanzas**

Insert the following block immediately after the existing `trustage-trustage-green` entry (after the line that reads `- trustage-trustage #comment: "Login slot green for trustage service"`), and before the next `# ──` comment block:

```yaml

      # ── trustage-formstore ──────────────────────────────────
      - name: trustage-formstore
        ensure: present
        login: false
        createdb: false
        inherit: true
        comment: "Parent role for trustage-formstore service"
      - name: trustage-formstore-blue
        ensure: present
        login: true
        inherit: true
        passwordSecret:
          name: trustage-formstore-blue
        inRoles:
          - trustage-formstore #comment: "Login slot blue for trustage-formstore service"
      - name: trustage-formstore-green
        ensure: present
        login: true
        inherit: true
        passwordSecret:
          name: trustage-formstore-green
        inRoles:
          - trustage-formstore #comment: "Login slot green for trustage-formstore service"

      # ── trustage-queuestore ─────────────────────────────────
      - name: trustage-queuestore
        ensure: present
        login: false
        createdb: false
        inherit: true
        comment: "Parent role for trustage-queuestore service"
      - name: trustage-queuestore-blue
        ensure: present
        login: true
        inherit: true
        passwordSecret:
          name: trustage-queuestore-blue
        inRoles:
          - trustage-queuestore #comment: "Login slot blue for trustage-queuestore service"
      - name: trustage-queuestore-green
        ensure: present
        login: true
        inherit: true
        passwordSecret:
          name: trustage-queuestore-green
        inRoles:
          - trustage-queuestore #comment: "Login slot green for trustage-queuestore service"
```

- [ ] **Step 3: Sanity-check with kubeval or YAML parse**

Run:
```bash
cd /home/j/code/antinvestor/deployments
python3 -c "import yaml; list(yaml.safe_load_all(open('manifests/namespaces/datastore/setup_datastore.yaml')))" && echo OK
```
Expected: `OK` (valid multi-doc YAML).

- [ ] **Step 4: Commit**

Run:
```bash
cd /home/j/code/antinvestor/deployments
git add manifests/namespaces/datastore/setup_datastore.yaml
git commit -m "fix(datastore): add CNPG managed roles for trustage-formstore and trustage-queuestore

Database CRs for formstore and queuestore were failing with
'role does not exist' because parent + blue/green managed roles
were missing from the CNPG Cluster spec. Password secrets already
exist — this just teaches PG about the roles."
```
Expected: commit succeeds.

- [ ] **Step 5: Push and wait for Flux reconcile**

Run:
```bash
cd /home/j/code/antinvestor/deployments
git push
flux -n flux-system reconcile kustomization datastore-setup --with-source
```
Expected: `Kustomization reconciled successfully`. (Verified kustomization name: `datastore-setup`.)

- [ ] **Step 6: Verify CNPG roles are applied**

Run:
```bash
kubectl -n datastore get cluster hub -o yaml | grep -c "name: trustage-formstore"
kubectl -n datastore get cluster hub -o yaml | grep -c "name: trustage-queuestore"
```
Expected: `3` for each (parent + blue + green).

Run (may take up to 2 minutes for CNPG to reconcile managed roles into PG):
```bash
PRIMARY=$(kubectl -n datastore get pods -l cnpg.io/cluster=hub,role=primary -o jsonpath='{.items[0].metadata.name}')
kubectl -n datastore exec "$PRIMARY" -c postgres -- psql -U postgres -tAc "SELECT rolname FROM pg_roles WHERE rolname LIKE 'trustage-%' ORDER BY rolname;"
```
Expected output includes:
```
trustage-formstore
trustage-formstore-blue
trustage-formstore-green
trustage-queuestore
trustage-queuestore-blue
trustage-queuestore-green
trustage-trustage
trustage-trustage-blue
trustage-trustage-green
```

- [ ] **Step 7: Verify Database CRs flipped to APPLIED=true**

Run:
```bash
kubectl -n datastore get database formstore queuestore
```
Expected:
```
NAME         AGE   CLUSTER   PG NAME      APPLIED   MESSAGE
formstore    ...   hub       formstore    true
queuestore   ...   hub       queuestore   true
```

If still `false` after 2 minutes, check `kubectl -n datastore describe database formstore` — CNPG logs any bootstrap failures there. Most common remaining cause would be the CronJob `db-role-ownership-fix` not yet having run to set `SET role` on the new blue/green logins, but that is not required for the Database CR itself to apply.

---

## Task 2: TDD nil-pool guard in formstore Migrate

**Files:**
- Modify: `/home/j/code/antinvestor/service-trustage/apps/formstore/service/repository/repository_suite_test.go`
- Modify: `/home/j/code/antinvestor/service-trustage/apps/formstore/service/repository/migrate.go`

- [ ] **Step 1: Write the failing test**

Append this method to the `RepositorySuite` in `repository_suite_test.go` (after the existing `TestMigrate_CreatesTablesAndIndexes` method):

```go
func (s *RepositorySuite) TestMigrate_ReturnsErrorWhenPoolMissing() {
	ctx := context.Background()
	manager, err := datastoremanager.NewManager(ctx)
	s.Require().NoError(err)

	// No pool added — Migrate must return an error, not panic.
	err = Migrate(ctx, manager)
	s.Require().Error(err)
	s.Contains(err.Error(), "pool")
}
```

- [ ] **Step 2: Run the test and verify it currently panics**

Run:
```bash
cd /home/j/code/antinvestor/service-trustage
go test ./apps/formstore/service/repository/ -run 'TestRepositorySuite/TestMigrate_ReturnsErrorWhenPoolMissing' -race -v
```
Expected: FAIL with a nil-pointer panic (SIGSEGV) or "invalid memory address" — this is what we're fixing. If the suite's `SetupSuite` hangs on testcontainers startup, wait for it; the failure happens in the new test method itself, not during setup.

- [ ] **Step 3: Apply the nil-guard fix**

Edit `apps/formstore/service/repository/migrate.go`. Replace lines 18-20 of the existing `Migrate` function body:

```go
	dbPool := manager.GetPool(ctx, datastore.DefaultPoolName)
	db := dbPool.DB(ctx, false)
```

with:

```go
	dbPool := manager.GetPool(ctx, datastore.DefaultPoolName)
	if dbPool == nil {
		return fmt.Errorf("datastore pool %q not available", datastore.DefaultPoolName)
	}
	db := dbPool.DB(ctx, false)
	if db == nil {
		return fmt.Errorf("datastore pool %q has no active connection", datastore.DefaultPoolName)
	}
```

- [ ] **Step 4: Run the test and verify it passes**

Run:
```bash
cd /home/j/code/antinvestor/service-trustage
go test ./apps/formstore/service/repository/ -run 'TestRepositorySuite/TestMigrate_ReturnsErrorWhenPoolMissing' -race -v
```
Expected: PASS.

- [ ] **Step 5: Run the full package test suite (regression check)**

Run:
```bash
cd /home/j/code/antinvestor/service-trustage
go test ./apps/formstore/service/repository/... -race
```
Expected: all tests pass (no regression in `TestMigrate_CreatesTablesAndIndexes`, which still uses a real pool).

- [ ] **Step 6: Commit**

Run:
```bash
cd /home/j/code/antinvestor/service-trustage
git add apps/formstore/service/repository/migrate.go apps/formstore/service/repository/repository_suite_test.go
git commit -m "fix(formstore): guard Migrate against nil pool and nil DB

Migrate used to SIGSEGV when the datastore pool failed to initialise
(e.g. missing PG role). Return a wrapped error instead so the caller
can Fatal with a readable message."
```

---

## Task 3: TDD nil-pool guard in queue Migrate

**Files:**
- Modify: `/home/j/code/antinvestor/service-trustage/apps/queue/service/repository/repository_suite_test.go`
- Modify: `/home/j/code/antinvestor/service-trustage/apps/queue/service/repository/migrate.go`

- [ ] **Step 1: Write the failing test**

Append this method to the `RepositorySuite` in `apps/queue/service/repository/repository_suite_test.go` (at the end of the file, after `TestQueueRepository_MigrateAndHelpers`):

```go
func (s *RepositorySuite) TestMigrate_ReturnsErrorWhenPoolMissing() {
	ctx := context.Background()
	manager, err := datastoremanager.NewManager(ctx)
	s.Require().NoError(err)

	// No pool added — Migrate must return an error, not panic.
	err = Migrate(ctx, manager)
	s.Require().Error(err)
	s.Contains(err.Error(), "pool")
}
```

(Imports `context` and `datastoremanager "github.com/pitabwire/frame/datastore/manager"` are already in the file — verified.)

- [ ] **Step 2: Run the test and verify it currently panics**

Run:
```bash
cd /home/j/code/antinvestor/service-trustage
go test ./apps/queue/service/repository/ -run 'TestRepositorySuite/TestMigrate_ReturnsErrorWhenPoolMissing' -race -v
```
Expected: FAIL with nil-pointer panic.

- [ ] **Step 3: Apply the nil-guard fix**

Edit `apps/queue/service/repository/migrate.go`. Replace lines 18-20:

```go
	dbPool := manager.GetPool(ctx, datastore.DefaultPoolName)
	db := dbPool.DB(ctx, false)
```

with:

```go
	dbPool := manager.GetPool(ctx, datastore.DefaultPoolName)
	if dbPool == nil {
		return fmt.Errorf("datastore pool %q not available", datastore.DefaultPoolName)
	}
	db := dbPool.DB(ctx, false)
	if db == nil {
		return fmt.Errorf("datastore pool %q has no active connection", datastore.DefaultPoolName)
	}
```

- [ ] **Step 4: Run the test and verify it passes**

Run:
```bash
cd /home/j/code/antinvestor/service-trustage
go test ./apps/queue/service/repository/ -run 'TestRepositorySuite/TestMigrate_ReturnsErrorWhenPoolMissing' -race -v
```
Expected: PASS.

- [ ] **Step 5: Run the full package test suite**

Run:
```bash
cd /home/j/code/antinvestor/service-trustage
go test ./apps/queue/service/repository/... -race
```
Expected: all tests pass.

- [ ] **Step 6: Commit**

Run:
```bash
cd /home/j/code/antinvestor/service-trustage
git add apps/queue/service/repository/migrate.go apps/queue/service/repository/repository_suite_test.go
git commit -m "fix(queue): guard Migrate against nil pool and nil DB

Mirrors the formstore fix. Migrate now returns a clean error instead
of panicking when the datastore pool failed to initialise."
```

---

## Task 4: Gate migration in formstore main.go behind DoDatabaseMigrate()

**Files:**
- Modify: `/home/j/code/antinvestor/service-trustage/apps/formstore/cmd/main.go` (around lines 88-92)

- [ ] **Step 1: Read the current block**

Run:
```bash
sed -n '86,96p' /home/j/code/antinvestor/service-trustage/apps/formstore/cmd/main.go
```
Expected current content (approximately):
```go
	// Database setup.
	dbManager := svc.DatastoreManager()

	if migrateErr := repository.Migrate(ctx, dbManager); migrateErr != nil {
		log.WithError(migrateErr).Fatal("database migration failed")
	}

	dbPool := dbManager.GetPool(ctx, datastore.DefaultPoolName)
```

- [ ] **Step 2: Replace the unconditional migrate with the gated pattern**

Replace the three-line `if migrateErr := repository.Migrate(...)` block with this two-block form (mirrors `apps/default/cmd/main.go:73-83`):

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

- [ ] **Step 3: Build to confirm it compiles**

Run:
```bash
cd /home/j/code/antinvestor/service-trustage
go build ./apps/formstore/cmd/...
```
Expected: no output (success).

- [ ] **Step 4: Run lint**

Run:
```bash
cd /home/j/code/antinvestor/service-trustage
make lint
```
Expected: no new findings in `apps/formstore/cmd/main.go`. If `make lint` complains about pre-existing issues unrelated to this change, note them but do not fix in this task.

- [ ] **Step 5: Commit**

Run:
```bash
cd /home/j/code/antinvestor/service-trustage
git add apps/formstore/cmd/main.go
git commit -m "feat(formstore): gate Migrate behind cfg.DoDatabaseMigrate()

Mirror the trustage-api pattern so the colony chart's pre-install
migration Job (DO_MIGRATION=true / argv[0]=migrate) migrates and
exits, while serving pods re-run AutoMigrate idempotently."
```

---

## Task 5: Gate migration in queue main.go behind DoDatabaseMigrate()

**Files:**
- Modify: `/home/j/code/antinvestor/service-trustage/apps/queue/cmd/main.go` (around lines 46-50)

- [ ] **Step 1: Read the current block**

Run:
```bash
sed -n '44,56p' /home/j/code/antinvestor/service-trustage/apps/queue/cmd/main.go
```
Expected current content:
```go
	// Database setup.
	dbManager := svc.DatastoreManager()

	if migrateErr := repository.Migrate(ctx, dbManager); migrateErr != nil {
		log.WithError(migrateErr).Fatal("database migration failed")
	}

	dbPool := dbManager.GetPool(ctx, datastore.DefaultPoolName)
```

- [ ] **Step 2: Replace with the gated pattern**

Replace the unconditional migrate block with:

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

- [ ] **Step 3: Build**

Run:
```bash
cd /home/j/code/antinvestor/service-trustage
go build ./apps/queue/cmd/...
```
Expected: no output.

- [ ] **Step 4: Run lint**

Run:
```bash
cd /home/j/code/antinvestor/service-trustage
make lint
```
Expected: no new findings.

- [ ] **Step 5: Commit**

Run:
```bash
cd /home/j/code/antinvestor/service-trustage
git add apps/queue/cmd/main.go
git commit -m "feat(queue): gate Migrate behind cfg.DoDatabaseMigrate()

Same pattern as the formstore change: enables the colony chart's
pre-install migration Job hook to work correctly."
```

---

## Task 6: Release new image tag

**Files:** no file edits — this task only runs the release pipeline.

- [ ] **Step 1: Run the full test suite**

Run:
```bash
cd /home/j/code/antinvestor/service-trustage
make tests
```
Expected: all packages pass. If this fails anywhere unrelated to our changes, stop and investigate before tagging.

- [ ] **Step 2: Push all accumulated commits on `main`**

Run:
```bash
cd /home/j/code/antinvestor/service-trustage
git log --oneline origin/main..HEAD
```
Expected: four commits from Tasks 2–5. If fewer, something was missed.

```bash
git push origin main
```

- [ ] **Step 3: Determine the next release tag**

Run:
```bash
cd /home/j/code/antinvestor/service-trustage
git tag --sort=-v:refname | head -5
```
Expected: latest existing tag is `v0.3.32`. Next tag will be `v0.3.33`.

- [ ] **Step 4: Tag and push the release**

Run:
```bash
cd /home/j/code/antinvestor/service-trustage
git tag -a v0.3.33 -m "Release v0.3.33: formstore/queue migration gating + nil-pool guard"
git push origin v0.3.33
```

- [ ] **Step 5: Watch the release workflow finish**

Run:
```bash
cd /home/j/code/antinvestor/service-trustage
gh run watch
```
Expected: the `Release` workflow completes green. It builds both `ghcr.io/antinvestor/service-trustage-formstore:v0.3.33` and `ghcr.io/antinvestor/service-trustage-queue:v0.3.33` via the matrix in `antinvestor/common/.github/workflows/docker-release.yml`.

- [ ] **Step 6: Verify the new images resolve in cluster ImagePolicies**

Wait up to 15 minutes for Flux image-automation to pick them up (interval on the HelmRelease ImageRepository is `15m`). Force it if you don't want to wait:

```bash
kubectl -n trustage annotate imagerepository trustage-formstore reconcile.fluxcd.io/requestedAt="$(date -Iseconds)" --overwrite
kubectl -n trustage annotate imagerepository trustage-queuestore reconcile.fluxcd.io/requestedAt="$(date -Iseconds)" --overwrite
sleep 30
kubectl -n trustage get imagepolicy trustage-formstore trustage-queuestore -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.status.latestImage}{"\n"}{end}'
```
Expected: both show `...:v0.3.33`.

Note: with Task 1 (DB roles) already landed, the new v0.3.33 pods will likely start serving successfully once Flux rolls the new image — serving pods still call `Migrate()` idempotently in the ungated branch. Tasks 7–8 are what achieve full parity (pre-install migration Job + `httpGet /readyz`), so do NOT treat "pods running" here as "done" — continue with the remaining tasks.

---

## Task 7: Enable migration Job + fix readinessProbe for formstore

**Files:**
- Modify: `/home/j/code/antinvestor/deployments/manifests/namespaces/trustage/formstore/trustage-formstore.yaml`

- [ ] **Step 1: Replace the `migration` block**

Find the block at lines 165-166 (roughly):
```yaml
    migration:
      enabled: false
```

Replace with:
```yaml
    migration:
      enabled: true
      env:
        - name: DATABASE_USERNAME
          valueFrom:
            secretKeyRef:
              name: db-credentials-formstore
              key: username
        - name: DATABASE_PASSWORD
          valueFrom:
            secretKeyRef:
              name: db-credentials-formstore
              key: password
        - name: DATABASE_URL
          value: "postgresql://$(DATABASE_USERNAME):$(DATABASE_PASSWORD)@pooler-rw.datastore.svc:5432/formstore?sslmode=require"
```

- [ ] **Step 2: Replace the `readinessProbe` block**

Find the current readinessProbe (around lines 157-164):
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

Replace with:
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

Leave `startupProbe` and `livenessProbe` unchanged.

- [ ] **Step 3: YAML parse check**

Run:
```bash
python3 -c "import yaml; yaml.safe_load(open('/home/j/code/antinvestor/deployments/manifests/namespaces/trustage/formstore/trustage-formstore.yaml'))" && echo OK
```
Expected: `OK`.

- [ ] **Step 4: Commit**

Run:
```bash
cd /home/j/code/antinvestor/deployments
git add manifests/namespaces/trustage/formstore/trustage-formstore.yaml
git commit -m "feat(trustage-formstore): enable migration Job, use /readyz for readiness

The colony chart's pre-install hook now runs AutoMigrate with
DO_MIGRATION=true, matching trustage-api. readinessProbe hits
/readyz so Service endpoints track the real DB-pool state."
```

---

## Task 8: Enable migration Job + fix readinessProbe for queuestore

**Files:**
- Modify: `/home/j/code/antinvestor/deployments/manifests/namespaces/trustage/queuestore/trustage-queuestore.yaml`

- [ ] **Step 1: Replace the `migration` block**

Find the block (around line 165-166):
```yaml
    migration:
      enabled: false
```

Replace with:
```yaml
    migration:
      enabled: true
      env:
        - name: DATABASE_USERNAME
          valueFrom:
            secretKeyRef:
              name: db-credentials-queuestore
              key: username
        - name: DATABASE_PASSWORD
          valueFrom:
            secretKeyRef:
              name: db-credentials-queuestore
              key: password
        - name: DATABASE_URL
          value: "postgresql://$(DATABASE_USERNAME):$(DATABASE_PASSWORD)@pooler-rw.datastore.svc:5432/queuestore?sslmode=require"
```

- [ ] **Step 2: Replace the `readinessProbe` block**

Find (around lines 157-164):
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

Replace with:
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

- [ ] **Step 3: YAML parse check**

Run:
```bash
python3 -c "import yaml; yaml.safe_load(open('/home/j/code/antinvestor/deployments/manifests/namespaces/trustage/queuestore/trustage-queuestore.yaml'))" && echo OK
```
Expected: `OK`.

- [ ] **Step 4: Commit and push both deployment commits**

Run:
```bash
cd /home/j/code/antinvestor/deployments
git add manifests/namespaces/trustage/queuestore/trustage-queuestore.yaml
git commit -m "feat(trustage-queuestore): enable migration Job, use /readyz for readiness

Mirrors the formstore change."
git push origin main
```

- [ ] **Step 5: Reconcile Flux**

Run:
```bash
flux -n flux-system reconcile kustomization trustage-setup --with-source
```
Expected: `Kustomization reconciled successfully`. (Verified kustomization name: `trustage-setup`.)

---

## Task 9: End-to-end verification

**Files:** none.

- [ ] **Step 1: Confirm migration Jobs ran successfully**

Run:
```bash
kubectl -n trustage get jobs | grep migration
```
Expected: one completed Job for each service, `Completions 1/1`. Example:
```
trustage-formstore-migration-1      Complete   1/1   ...
trustage-queuestore-migration-1     Complete   1/1   ...
```

Check Job logs:
```bash
kubectl -n trustage logs -l app.kubernetes.io/component=migration --tail=20
```
Expected: clean exit with `database migration completed` (at debug level — may not appear if LOG_LEVEL excludes debug) and no panic.

- [ ] **Step 2: Confirm HelmReleases are READY=True**

Run:
```bash
kubectl -n trustage get helmrelease trustage-formstore trustage-queuestore
```
Expected: `READY=True` for both, with `STATUS` containing `install succeeded` (or `upgrade succeeded`).

- [ ] **Step 3: Confirm serving pods are Running**

Run:
```bash
kubectl -n trustage get pods -l 'app.kubernetes.io/name in (trustage-formstore,trustage-queuestore)'
```
Expected: `1/1 Running` for each, `RESTARTS = 0`.

- [ ] **Step 4: Confirm serving pod logs show clean startup**

Run:
```bash
kubectl -n trustage logs deploy/trustage-formstore --tail=30
kubectl -n trustage logs deploy/trustage-queuestore --tail=30
```
Expected: no panic, one line each like `starting formstore service` / `starting queue service` with a port number.

- [ ] **Step 5: Probe health endpoints from inside the cluster**

Run:
```bash
kubectl -n trustage exec deploy/trustage-formstore -- wget -qO- http://localhost:8081/healthz
kubectl -n trustage exec deploy/trustage-formstore -- wget -qO- http://localhost:8081/readyz
kubectl -n trustage exec deploy/trustage-queuestore -- wget -qO- http://localhost:8082/healthz
kubectl -n trustage exec deploy/trustage-queuestore -- wget -qO- http://localhost:8082/readyz
```
Expected: `ok` from each endpoint.

- [ ] **Step 6: Probe through the Gateway**

Run:
```bash
for host in api.stawi.dev api.stawi.org api.stawi.im; do
  echo "--- $host"
  curl -sS -o /dev/null -w "formstore %{http_code}\n"  "https://$host/formstore/healthz"
  curl -sS -o /dev/null -w "queuestore %{http_code}\n" "https://$host/queuestore/healthz"
done
```
Expected: `200` for every line.

- [ ] **Step 7: Exercise a rolling restart (migration-hook sanity)**

Run:
```bash
kubectl -n trustage rollout restart deploy/trustage-formstore
kubectl -n trustage rollout status deploy/trustage-formstore --timeout=300s
kubectl -n trustage rollout restart deploy/trustage-queuestore
kubectl -n trustage rollout status deploy/trustage-queuestore --timeout=300s
```
Expected: rollouts complete cleanly. A new migration Job should appear and finish between the restart and the new pod becoming Ready — verify:
```bash
kubectl -n trustage get jobs -l app.kubernetes.io/component=migration --sort-by=.metadata.creationTimestamp
```
Expected: more than one migration Job per service over time (the chart's `hook-delete-policy: before-hook-creation,hook-succeeded,hook-failed` keeps only the current one, but you'll see the fresh timestamps).

- [ ] **Step 8: Sustained health check (10 minutes)**

Run in a separate terminal:
```bash
for i in $(seq 1 20); do
  date
  kubectl -n trustage get pods -l 'app.kubernetes.io/name in (trustage-formstore,trustage-queuestore)' --no-headers
  sleep 30
done
```
Expected: continuous `1/1 Running`, `RESTARTS = 0`, no flaps.

- [ ] **Step 9: Declare done**

All of the above green → services are reliably running. If anything failed, capture:
- `kubectl -n trustage describe helmrelease <name>`
- `kubectl -n trustage get jobs,pods`
- Failing pod logs

and return to the relevant task before declaring complete.

---

## Rollback

Per spec §Rollback:

- After Task 1 only: revert the `setup_datastore.yaml` commit. CNPG leaves existing roles in place (removal is not `ensure: absent`) — non-destructive.
- After Task 6 (image built, nothing deployed yet): no rollback needed; cluster still runs old manifests.
- After Task 8: prefer "revert forward" (fix + re-land), because reverting the YAML restores the original crashloop. Emergency stop: `flux -n trustage suspend helmrelease trustage-formstore trustage-queuestore`.
