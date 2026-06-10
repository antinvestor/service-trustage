//nolint:testpackage // white-box tests exercise the unexported workflow client seam intentionally.
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
	name := req.Msg.GetName()
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
