# Workflow Loader SDK Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a client-side Go package in the Trustage repo that consumer apps import to idempotently register workflow definitions from a directory of JSON files during their migration Jobs.

**Architecture:** A `client/workflows` package exposes `SyncFromDir(ctx, connectClient, dir)`. It reads `*.json` files, calls Trustage's Connect RPC `ListWorkflows` / `CreateWorkflow` / `ActivateWorkflow` endpoints, and skips workflows that already exist at the same version. Consumer apps call it after DB migrations in their `DO_DATABASE_MIGRATE=true` code path.

**Tech Stack:** Go, Connect RPC (`connectrpc.com/connect`), Trustage protobuf types (`gen/go/workflow/v1`), `google.protobuf.Struct` for DSL payloads.

**Repos:**
- Primary: `/home/j/code/antinvestor/service-trustage` (the new package lives here)
- Consumer: `/home/j/code/stawi.opportunities` (wiring into crawler migration)
- Manifests: `/home/j/code/stawi.org/deployment.manifests` (ConfigMap + env vars)

---

### Task 1: DSL File Parser

**Files:**
- Create: `client/workflows/dsl.go`
- Test: `client/workflows/dsl_test.go`

- [ ] **Step 1: Write the failing test for parseDSLFile**

```go
// client/workflows/dsl_test.go
package workflows

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseDSLFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test-workflow.json")
	content := `{
		"version": "1.0",
		"name": "test.workflow",
		"schedule": {"cron": "30s", "active": true},
		"steps": [{"id": "step1", "type": "call", "name": "Do thing",
			"call": {"action": "http.request", "input": {"url": "http://example.com", "method": "POST"}}}]
	}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	dslStruct, name, err := parseDSLFile(path)
	if err != nil {
		t.Fatalf("parseDSLFile: %v", err)
	}
	if name != "test.workflow" {
		t.Errorf("name = %q, want %q", name, "test.workflow")
	}
	if dslStruct == nil {
		t.Fatal("dslStruct is nil")
	}
	// Verify the struct round-trips the name field.
	nameField := dslStruct.Fields["name"].GetStringValue()
	if nameField != "test.workflow" {
		t.Errorf("struct name = %q, want %q", nameField, "test.workflow")
	}
}

func TestParseDSLFile_MissingName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte(`{"version":"1.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	_, _, err := parseDSLFile(path)
	if err == nil {
		t.Fatal("expected error for missing name, got nil")
	}
}

func TestDSLHash_Deterministic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hash-test.json")
	content := `{"version":"1.0","name":"hash.test","steps":[]}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	s1, _, _ := parseDSLFile(path)
	s2, _, _ := parseDSLFile(path)
	h1 := dslHash(s1)
	h2 := dslHash(s2)
	if h1 != h2 {
		t.Errorf("hash not deterministic: %s != %s", h1, h2)
	}
	if h1 == "" {
		t.Error("hash is empty")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /home/j/code/antinvestor/service-trustage && go test ./client/workflows/ -run TestParseDSLFile -v`
Expected: FAIL — `package client/workflows is not in std`

- [ ] **Step 3: Write the implementation**

```go
// client/workflows/dsl.go
package workflows

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

// parseDSLFile reads a workflow JSON file and returns the protobuf
// Struct (ready for CreateWorkflowRequest.dsl) plus the extracted
// workflow name.
func parseDSLFile(path string) (*structpb.Struct, string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("read %s: %w", path, err)
	}

	s := &structpb.Struct{}
	if err := protojson.Unmarshal(raw, s); err != nil {
		return nil, "", fmt.Errorf("parse %s: %w", path, err)
	}

	nameVal, ok := s.Fields["name"]
	if !ok || nameVal.GetStringValue() == "" {
		return nil, "", fmt.Errorf("%s: missing or empty 'name' field", path)
	}

	return s, nameVal.GetStringValue(), nil
}

// dslHash returns a deterministic SHA-256 hex digest of the protobuf
// Struct. Used to detect whether a definition has changed between
// releases without comparing the full DSL.
func dslHash(s *structpb.Struct) string {
	b, err := protojson.MarshalOptions{Indent: "", Multiline: false}.Marshal(s)
	if err != nil {
		return ""
	}
	// Re-marshal through encoding/json to get sorted keys.
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	sorted, _ := json.Marshal(m)
	h := sha256.Sum256(sorted)
	return hex.EncodeToString(h[:])
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /home/j/code/antinvestor/service-trustage && go test ./client/workflows/ -v`
Expected: PASS (all three tests)

- [ ] **Step 5: Commit**

```bash
cd /home/j/code/antinvestor/service-trustage
git add client/workflows/dsl.go client/workflows/dsl_test.go
git commit -m "feat(client/workflows): add DSL file parser and hash helper"
```

---

### Task 2: Workflow Sync Loader

**Files:**
- Create: `client/workflows/loader.go`
- Test: `client/workflows/loader_test.go`

- [ ] **Step 1: Write the failing test with a mock client**

```go
// client/workflows/loader_test.go
package workflows

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	workflowv1 "github.com/antinvestor/service-trustage/gen/go/workflow/v1"
)

type mockWorkflowClient struct {
	listed    int
	created   int
	activated int
	existing  map[string]*workflowv1.WorkflowDefinition
}

func (m *mockWorkflowClient) ListWorkflows(
	_ context.Context,
	req *connect.Request[workflowv1.ListWorkflowsRequest],
) (*connect.Response[workflowv1.ListWorkflowsResponse], error) {
	m.listed++
	name := req.Msg.Name
	if def, ok := m.existing[name]; ok {
		return connect.NewResponse(&workflowv1.ListWorkflowsResponse{
			Items: []*workflowv1.WorkflowDefinition{def},
		}), nil
	}
	return connect.NewResponse(&workflowv1.ListWorkflowsResponse{}), nil
}

func (m *mockWorkflowClient) CreateWorkflow(
	_ context.Context,
	_ *connect.Request[workflowv1.CreateWorkflowRequest],
) (*connect.Response[workflowv1.CreateWorkflowResponse], error) {
	m.created++
	return connect.NewResponse(&workflowv1.CreateWorkflowResponse{
		Workflow: &workflowv1.WorkflowDefinition{
			Id:     "new-id",
			Name:   "test",
			Status: workflowv1.WorkflowStatus_WORKFLOW_STATUS_DRAFT,
		},
	}), nil
}

func (m *mockWorkflowClient) ActivateWorkflow(
	_ context.Context,
	_ *connect.Request[workflowv1.ActivateWorkflowRequest],
) (*connect.Response[workflowv1.ActivateWorkflowResponse], error) {
	m.activated++
	return connect.NewResponse(&workflowv1.ActivateWorkflowResponse{}), nil
}

func writeDSL(t *testing.T, dir, filename, name string) {
	t.Helper()
	content := `{"version":"1.0","name":"` + name + `","schedule":{"cron":"30s","active":true},"steps":[{"id":"s1","type":"call","name":"test","call":{"action":"http.request","input":{"url":"http://localhost","method":"POST"}}}]}`
	if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSyncFromDir_CreatesNewWorkflows(t *testing.T) {
	dir := t.TempDir()
	writeDSL(t, dir, "wf1.json", "test.wf1")
	writeDSL(t, dir, "wf2.json", "test.wf2")

	mock := &mockWorkflowClient{existing: map[string]*workflowv1.WorkflowDefinition{}}
	err := SyncFromDir(context.Background(), mock, dir)
	if err != nil {
		t.Fatalf("SyncFromDir: %v", err)
	}
	if mock.listed != 2 {
		t.Errorf("listed = %d, want 2", mock.listed)
	}
	if mock.created != 2 {
		t.Errorf("created = %d, want 2", mock.created)
	}
	if mock.activated != 2 {
		t.Errorf("activated = %d, want 2", mock.activated)
	}
}

func TestSyncFromDir_SkipsExisting(t *testing.T) {
	dir := t.TempDir()
	writeDSL(t, dir, "existing.json", "test.existing")

	// Parse the file to get the hash the loader will compute.
	s, _, _ := parseDSLFile(filepath.Join(dir, "existing.json"))
	hash := dslHash(s)

	mock := &mockWorkflowClient{
		existing: map[string]*workflowv1.WorkflowDefinition{
			"test.existing": {
				Id:              "existing-id",
				Name:            "test.existing",
				Version:         1,
				Status:          workflowv1.WorkflowStatus_WORKFLOW_STATUS_ACTIVE,
				InputSchemaHash: hash,
			},
		},
	}
	err := SyncFromDir(context.Background(), mock, dir)
	if err != nil {
		t.Fatalf("SyncFromDir: %v", err)
	}
	if mock.created != 0 {
		t.Errorf("created = %d, want 0 (should skip)", mock.created)
	}
	if mock.activated != 0 {
		t.Errorf("activated = %d, want 0 (should skip)", mock.activated)
	}
}

func TestSyncFromDir_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	mock := &mockWorkflowClient{existing: map[string]*workflowv1.WorkflowDefinition{}}
	err := SyncFromDir(context.Background(), mock, dir)
	if err != nil {
		t.Fatalf("SyncFromDir: %v", err)
	}
	if mock.listed != 0 {
		t.Errorf("listed = %d, want 0", mock.listed)
	}
}

func TestSyncFromDir_MissingDir(t *testing.T) {
	mock := &mockWorkflowClient{existing: map[string]*workflowv1.WorkflowDefinition{}}
	err := SyncFromDir(context.Background(), mock, "/nonexistent/path")
	if err != nil {
		t.Fatalf("SyncFromDir should be no-op for missing dir, got: %v", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /home/j/code/antinvestor/service-trustage && go test ./client/workflows/ -run TestSyncFromDir -v`
Expected: FAIL — `SyncFromDir` not defined

- [ ] **Step 3: Write the loader implementation**

```go
// client/workflows/loader.go
package workflows

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"
	"github.com/pitabwire/util"

	workflowv1 "github.com/antinvestor/service-trustage/gen/go/workflow/v1"
)

// WorkflowClient is the subset of the generated Connect client the
// loader needs. Satisfied by workflowv1connect.WorkflowServiceClient.
type WorkflowClient interface {
	ListWorkflows(context.Context, *connect.Request[workflowv1.ListWorkflowsRequest]) (*connect.Response[workflowv1.ListWorkflowsResponse], error)
	CreateWorkflow(context.Context, *connect.Request[workflowv1.CreateWorkflowRequest]) (*connect.Response[workflowv1.CreateWorkflowResponse], error)
	ActivateWorkflow(context.Context, *connect.Request[workflowv1.ActivateWorkflowRequest]) (*connect.Response[workflowv1.ActivateWorkflowResponse], error)
}

// SyncFromDir reads every *.json file in dir and ensures each workflow
// exists in Trustage. Creates missing workflows, activates them, and
// skips ones that already exist with the same DSL hash. Returns nil
// on full success, nil for empty/missing directories, or the first
// hard error (network, auth, malformed JSON).
func SyncFromDir(ctx context.Context, client WorkflowClient, dir string) error {
	log := util.Log(ctx)

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		log.WithField("dir", dir).Debug("workflows: directory does not exist, skipping")
		return nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("workflows: read dir %s: %w", dir, err)
	}

	var synced, skipped, created int
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, entry.Name())

		action, syncErr := syncOne(ctx, client, path)
		if syncErr != nil {
			return fmt.Errorf("workflows: sync %s: %w", entry.Name(), syncErr)
		}

		switch action {
		case "created":
			created++
			log.WithField("file", entry.Name()).Info("workflows: created and activated")
		case "skipped":
			skipped++
			log.WithField("file", entry.Name()).Debug("workflows: already exists, skipped")
		}
		synced++
	}

	log.WithField("synced", synced).
		WithField("created", created).
		WithField("skipped", skipped).
		Info("workflows: sync complete")
	return nil
}

func syncOne(ctx context.Context, client WorkflowClient, path string) (string, error) {
	dslStruct, name, err := parseDSLFile(path)
	if err != nil {
		return "", err
	}

	hash := dslHash(dslStruct)

	// Check if workflow already exists.
	listResp, err := client.ListWorkflows(ctx, connect.NewRequest(&workflowv1.ListWorkflowsRequest{
		Name: name,
	}))
	if err != nil {
		return "", fmt.Errorf("list workflows: %w", err)
	}

	for _, existing := range listResp.Msg.Items {
		if existing.InputSchemaHash == hash &&
			existing.Status == workflowv1.WorkflowStatus_WORKFLOW_STATUS_ACTIVE {
			return "skipped", nil
		}
	}

	// Create the workflow.
	createResp, err := client.CreateWorkflow(ctx, connect.NewRequest(&workflowv1.CreateWorkflowRequest{
		Dsl: dslStruct,
	}))
	if err != nil {
		// Tolerate "already exists" for idempotency.
		if isAlreadyExists(err) {
			return "skipped", nil
		}
		return "", fmt.Errorf("create workflow %s: %w", name, err)
	}

	wfID := createResp.Msg.Workflow.GetId()

	// Activate it.
	_, err = client.ActivateWorkflow(ctx, connect.NewRequest(&workflowv1.ActivateWorkflowRequest{
		Id: wfID,
	}))
	if err != nil {
		return "", fmt.Errorf("activate workflow %s (id=%s): %w", name, wfID, err)
	}

	return "created", nil
}

func isAlreadyExists(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "already exists") ||
		strings.Contains(msg, "duplicate") ||
		strings.Contains(msg, "unique constraint")
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /home/j/code/antinvestor/service-trustage && go test ./client/workflows/ -v`
Expected: PASS (all tests from both files)

- [ ] **Step 5: Commit**

```bash
cd /home/j/code/antinvestor/service-trustage
git add client/workflows/loader.go client/workflows/loader_test.go
git commit -m "feat(client/workflows): add SyncFromDir workflow loader"
```

---

### Task 3: Tag and Release service-trustage

**Files:**
- None (git tag only)

- [ ] **Step 1: Push changes and tag a release**

```bash
cd /home/j/code/antinvestor/service-trustage
git push
git tag v0.XX.0  # bump to next minor version
git push origin v0.XX.0
```

Check the current latest tag first:
```bash
git tag --sort=-v:refname | head -1
```

- [ ] **Step 2: Verify the module is fetchable**

```bash
cd /tmp && go mod init test && go get github.com/antinvestor/service-trustage/client/workflows@v0.XX.0
```

- [ ] **Step 3: Commit (no-op, already pushed)**

---

### Task 4: Wire Loader into Opportunities Crawler Migration

**Files:**
- Modify: `/home/j/code/stawi.opportunities/apps/crawler/cmd/main.go` (migration block)
- Modify: `/home/j/code/stawi.opportunities/apps/crawler/config/config.go` (add TrustageURL + WorkflowsDir)
- Modify: `/home/j/code/stawi.opportunities/go.mod` (add service-trustage dependency)

- [ ] **Step 1: Add config fields**

In `apps/crawler/config/config.go`, add to the Config struct:

```go
	// Trustage workflow loader — syncs definitions from a mounted
	// directory into Trustage during the migration Job.
	TrustageURL         string `env:"TRUSTAGE_URL" envDefault:""`
	TrustageWorkflowsDir string `env:"TRUSTAGE_WORKFLOWS_DIR" envDefault:""`
```

- [ ] **Step 2: Wire the loader into the migration block**

In `apps/crawler/cmd/main.go`, after the existing migration logic and before the `return` statement in the `if cfg.DoDatabaseMigrate()` block, add:

```go
	// Sync Trustage workflow definitions from the mounted ConfigMap.
	if cfg.TrustageURL != "" && cfg.TrustageWorkflowsDir != "" {
		trustageCli, cliErr := services.NewTrustageWorkflowClient(ctx, &cfg, cfg.TrustageURL)
		if cliErr != nil {
			util.Log(ctx).WithError(cliErr).Fatal("trustage workflow client init failed")
		}
		if syncErr := workflows.SyncFromDir(ctx, trustageCli, cfg.TrustageWorkflowsDir); syncErr != nil {
			util.Log(ctx).WithError(syncErr).Fatal("trustage workflow sync failed")
		}
	}
```

Import: `"github.com/antinvestor/service-trustage/client/workflows"`

- [ ] **Step 3: Add Trustage client constructor to services package**

Create or add to `pkg/services/clients.go`:

```go
func NewTrustageWorkflowClient(
	ctx context.Context,
	cfg any,
	trustageURL string,
) (workflowv1connect.WorkflowServiceClient, error) {
	return connection.NewServiceClient(ctx, cfg, apis.ServiceTarget{
		Endpoint:  trustageURL,
		Audiences: []string{"service_trustage"},
	}, workflowv1connect.NewWorkflowServiceClient)
}
```

Import: `"github.com/antinvestor/service-trustage/gen/go/workflow/v1/workflowv1connect"`

- [ ] **Step 4: Update go.mod**

```bash
cd /home/j/code/stawi.opportunities
go get github.com/antinvestor/service-trustage@v0.XX.0
go mod tidy
```

- [ ] **Step 5: Verify it builds**

```bash
go build ./apps/crawler/...
```

- [ ] **Step 6: Commit**

```bash
git add apps/crawler/cmd/main.go apps/crawler/config/config.go pkg/services/clients.go go.mod go.sum
git commit -m "feat(crawler): wire Trustage workflow loader into migration Job"
```

---

### Task 5: Deployment Manifests — ConfigMap and Env Vars

**Files:**
- Create: `/home/j/code/stawi.org/deployment.manifests/namespaces/product-opportunities/common/trustage-workflows-cm.yaml`
- Modify: `/home/j/code/stawi.org/deployment.manifests/namespaces/product-opportunities/crawler/opportunities-crawler.yaml` (migration env + volume mount)
- Modify: `/home/j/code/stawi.org/deployment.manifests/namespaces/product-opportunities/common/kustomization.yaml` (add the ConfigMap resource)

- [ ] **Step 1: Create ConfigMap from workflow definition files**

```yaml
# namespaces/product-opportunities/common/trustage-workflows-cm.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: trustage-workflows
data:
  # Each key is a workflow JSON file from definitions/trustage/ in the
  # opportunities repo. The migration Job mounts this at
  # /etc/trustage-workflows and the workflow loader reads *.json.
  scheduler-tick.json: |
    <contents of definitions/trustage/scheduler-tick.json>
  retention-expire.json: |
    <contents of definitions/trustage/retention-expire.json>
  # ... one entry per workflow definition file
```

Generate this from the repo files:

```bash
kubectl create configmap trustage-workflows \
  --from-file=definitions/trustage/ \
  --dry-run=client -o yaml > configmap.yaml
```

- [ ] **Step 2: Add env vars and volume mount to crawler migration Job**

In the crawler HelmRelease's `migration.env` section, add:

```yaml
- name: TRUSTAGE_URL
  value: "http://trustage.operations.svc"
- name: TRUSTAGE_WORKFLOWS_DIR
  value: "/etc/trustage-workflows"
```

In the migration volumes/volumeMounts:

```yaml
volumes:
  - name: trustage-workflows
    configMap:
      name: trustage-workflows
volumeMounts:
  - name: trustage-workflows
    mountPath: /etc/trustage-workflows
    readOnly: true
```

- [ ] **Step 3: Add the ConfigMap to the common kustomization**

In the common `kustomization.yaml`, add `trustage-workflows-cm.yaml` to resources.

- [ ] **Step 4: Commit and push**

```bash
cd /home/j/code/stawi.org/deployment.manifests
git add namespaces/product-opportunities/common/trustage-workflows-cm.yaml \
        namespaces/product-opportunities/crawler/opportunities-crawler.yaml \
        namespaces/product-opportunities/common/kustomization.yaml
git commit -m "feat(opportunities): mount Trustage workflow definitions as ConfigMap for migration loader"
git push
```

---

### Task 6: Tag Release and Verify End-to-End

**Files:**
- None (release + cluster verification)

- [ ] **Step 1: Tag and push the opportunities release**

```bash
cd /home/j/code/stawi.opportunities
git tag v8.0.54
git push origin v8.0.54
```

Wait for the Release workflow to build Docker images.

- [ ] **Step 2: Update image tags in deployment manifests**

Update the crawler image tag in the HelmRelease to `v8.0.54`.

- [ ] **Step 3: Trigger the crawler migration Job**

```bash
kubectl delete job -n product-opportunities opportunities-crawler-migrate --ignore-not-found
# Flux will recreate it, or manually apply
```

- [ ] **Step 4: Verify Trustage has the workflow definitions**

```bash
kubectl exec -n operations operations-db-1 -c postgres -- \
  psql -U postgres -d trustage -c \
  "SELECT name, active, cron_expr FROM schedule_definitions ORDER BY name"
```

Expected: 14 rows with `active=true`.

- [ ] **Step 5: Verify the crawler starts crawling**

```bash
kubectl logs -n product-opportunities -l app.kubernetes.io/name=opportunities-crawler --tail=20 | grep -i "crawl\|source\|dispatch"
```

Expected: Log lines showing sources being dispatched for crawling.
