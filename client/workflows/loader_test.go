//nolint:testpackage // white-box tests exercise the unexported workflow client seam intentionally.
package workflows

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"connectrpc.com/connect/v2"

	workflowv1 "github.com/antinvestor/service-trustage/gen/go/workflow/v1"
	"github.com/antinvestor/service-trustage/gen/go/workflow/v1/workflowv1connect"
)

// The generated Connect client must keep satisfying the loader's seam.
var _ WorkflowClient = (workflowv1connect.WorkflowServiceClient)(nil)

type mockWorkflowClient struct {
	listed    int
	created   []string
	activated []string
	// active names make CreateWorkflow behave as the server does for an
	// identical ACTIVE version: return it (re-asserting its routing).
	active    map[string]bool
	createErr error
}

func (m *mockWorkflowClient) ListWorkflows(
	_ context.Context,
	_ *workflowv1.ListWorkflowsRequest,
) (*workflowv1.ListWorkflowsResponse, error) {
	m.listed++
	return &workflowv1.ListWorkflowsResponse{}, nil
}

func (m *mockWorkflowClient) CreateWorkflow(
	_ context.Context,
	req *workflowv1.CreateWorkflowRequest,
) (*workflowv1.CreateWorkflowResponse, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}

	name := req.GetDsl().GetFields()["name"].GetStringValue()
	m.created = append(m.created, name)

	status := workflowv1.WorkflowStatus_WORKFLOW_STATUS_DRAFT
	if m.active[name] {
		status = workflowv1.WorkflowStatus_WORKFLOW_STATUS_ACTIVE
	}

	return &workflowv1.CreateWorkflowResponse{
		Workflow: &workflowv1.WorkflowDefinition{Id: "id-" + name, Name: name, Status: status},
	}, nil
}

func (m *mockWorkflowClient) ActivateWorkflow(
	_ context.Context,
	req *workflowv1.ActivateWorkflowRequest,
) (*workflowv1.ActivateWorkflowResponse, error) {
	m.activated = append(m.activated, req.GetId())
	return &workflowv1.ActivateWorkflowResponse{}, nil
}

func dslDoc(name string) string {
	return `{"version":"1.0","name":"` + name + `",` +
		`"triggers":[{"event_type":"proposal.created","key":"payload.proposal_id"}],` +
		`"steps":[{"id":"s1","type":"call","name":"test","call":{"action":"http.request",` +
		`"input":{"url":"http://localhost","method":"POST"}}}]}`
}

func writeDSL(t *testing.T, dir, filename, name string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, filename), []byte(dslDoc(name)), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestSyncFromDir_CreatesAndActivatesNewWorkflows(t *testing.T) {
	dir := t.TempDir()
	writeDSL(t, dir, "wf1.json", "test.wf1")
	writeDSL(t, dir, "wf2.json", "test.wf2")

	mock := &mockWorkflowClient{}
	if err := SyncFromDir(context.Background(), mock, dir); err != nil {
		t.Fatalf("SyncFromDir: %v", err)
	}

	if len(mock.created) != 2 {
		t.Errorf("created = %v, want 2 creates", mock.created)
	}
	if len(mock.activated) != 2 || mock.activated[0] != "id-test.wf1" || mock.activated[1] != "id-test.wf2" {
		t.Errorf("activated = %v, want both new versions in file order", mock.activated)
	}
}

// An identical ACTIVE version must still go through CreateWorkflow: the
// server re-asserts its trigger bindings and signal routes there, which
// repairs workflows registered before routing was persisted.
func TestSyncFromDir_IdenticalActiveVersionStillCallsCreate(t *testing.T) {
	dir := t.TempDir()
	writeDSL(t, dir, "existing.json", "test.existing")

	mock := &mockWorkflowClient{active: map[string]bool{"test.existing": true}}
	if err := SyncFromDir(context.Background(), mock, dir); err != nil {
		t.Fatalf("SyncFromDir: %v", err)
	}

	if len(mock.created) != 1 {
		t.Errorf("created = %v, want CreateWorkflow called once", mock.created)
	}
	if len(mock.activated) != 0 {
		t.Errorf("activated = %v, want none (already ACTIVE)", mock.activated)
	}
}

func TestSyncFromFS_SyncsEmbeddedDocuments(t *testing.T) {
	fsys := fstest.MapFS{
		"b.json":         {Data: []byte(dslDoc("test.b"))},
		"a.json":         {Data: []byte(dslDoc("test.a"))},
		"README.md":      {Data: []byte("not a workflow")},
		"nested/c.json":  {Data: []byte(dslDoc("test.c"))},
		"active.json":    {Data: []byte(dslDoc("test.active"))},
		"notes.json.bak": {Data: []byte("{}")},
	}

	mock := &mockWorkflowClient{active: map[string]bool{"test.active": true}}
	if err := SyncFromFS(context.Background(), mock, fsys); err != nil {
		t.Fatalf("SyncFromFS: %v", err)
	}

	want := []string{"test.a", "test.active", "test.b"}
	if len(mock.created) != len(want) {
		t.Fatalf("created = %v, want %v", mock.created, want)
	}
	for i := range want {
		if mock.created[i] != want[i] {
			t.Fatalf("created = %v, want %v (sorted, top level only)", mock.created, want)
		}
	}
	if len(mock.activated) != 2 {
		t.Errorf("activated = %v, want the two DRAFT versions", mock.activated)
	}
}

func TestSyncFromFS_ReportsFailingFile(t *testing.T) {
	fsys := fstest.MapFS{"bad.json": {Data: []byte(`{"version":"1.0"}`)}}

	err := SyncFromFS(context.Background(), &mockWorkflowClient{}, fsys)
	if err == nil {
		t.Fatal("expected an error for a document without a name")
	}

	fsys = fstest.MapFS{"ok.json": {Data: []byte(dslDoc("test.ok"))}}
	err = SyncFromFS(context.Background(), &mockWorkflowClient{createErr: errors.New("boom")}, fsys)
	if err == nil {
		t.Fatal("expected the create error to propagate")
	}
}

func TestSyncFromFS_AlreadyExistsIsSkipped(t *testing.T) {
	fsys := fstest.MapFS{"ok.json": {Data: []byte(dslDoc("test.ok"))}}

	for name, createErr := range map[string]error{
		"connect code":     connect.NewError(connect.CodeAlreadyExists, "conflict"),
		"message fallback": errors.New("unique constraint violated"),
	} {
		t.Run(name, func(t *testing.T) {
			mock := &mockWorkflowClient{createErr: createErr}
			if err := SyncFromFS(context.Background(), mock, fsys); err != nil {
				t.Fatalf("SyncFromFS: %v", err)
			}
			if len(mock.activated) != 0 {
				t.Errorf("activated = %v, want none", mock.activated)
			}
		})
	}
}

func TestSyncFromDir_EmptyDir(t *testing.T) {
	mock := &mockWorkflowClient{}
	if err := SyncFromDir(context.Background(), mock, t.TempDir()); err != nil {
		t.Fatalf("SyncFromDir: %v", err)
	}
	if len(mock.created) != 0 {
		t.Errorf("created = %v, want none", mock.created)
	}
}

func TestSyncFromDir_MissingDir(t *testing.T) {
	mock := &mockWorkflowClient{}
	if err := SyncFromDir(context.Background(), mock, "/nonexistent/path"); err != nil {
		t.Fatalf("SyncFromDir should be no-op for missing dir, got: %v", err)
	}
}
