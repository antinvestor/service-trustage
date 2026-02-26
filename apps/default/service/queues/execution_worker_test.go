package queues

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/antinvestor/service-trustage/apps/default/service/business"
	"github.com/antinvestor/service-trustage/apps/default/service/models"
	"github.com/antinvestor/service-trustage/connector"
	"github.com/antinvestor/service-trustage/dsl"
)

type stubEngine struct {
	commits []*business.CommitRequest
}

func (s *stubEngine) CreateInitialExecution(ctx context.Context, instance *models.WorkflowInstance, inputPayload json.RawMessage) (*business.ExecutionCommand, error) {
	return nil, nil
}

func (s *stubEngine) Dispatch(ctx context.Context, execution *models.WorkflowStateExecution) (*business.ExecutionCommand, error) {
	return nil, nil
}

func (s *stubEngine) Commit(ctx context.Context, req *business.CommitRequest) error {
	s.commits = append(s.commits, req)
	return nil
}

type stubDefRepo struct {
	dsl string
}

func (s *stubDefRepo) GetByNameAndVersion(ctx context.Context, name string, version int) (*models.WorkflowDefinition, error) {
	return &models.WorkflowDefinition{DSLBlob: s.dsl}, nil
}

type stubAdapter struct{}

func (a stubAdapter) Type() string { return "test.adapter" }
func (a stubAdapter) DisplayName() string { return "Test Adapter" }
func (a stubAdapter) InputSchema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }
func (a stubAdapter) ConfigSchema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }
func (a stubAdapter) OutputSchema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }
func (a stubAdapter) Validate(req *connector.ExecuteRequest) error { return nil }
func (a stubAdapter) Execute(ctx context.Context, req *connector.ExecuteRequest) (*connector.ExecuteResponse, *connector.ExecutionError) {
	return &connector.ExecuteResponse{Output: map[string]any{"ok": true}}, nil
}

func TestExecutionWorker_Success(t *testing.T) {
	spec := &dsl.WorkflowSpec{
		Version: "1.0",
		Name:    "wf",
		Steps: []*dsl.StepSpec{
			{ID: "step_a", Type: dsl.StepTypeCall, Call: &dsl.CallSpec{Action: "test.adapter"}},
		},
	}
	blob, _ := json.Marshal(spec)

	engine := &stubEngine{}
	defRepo := &stubDefRepo{dsl: string(blob)}
	registry := connector.NewRegistry()
	_ = registry.Register(stubAdapter{})

	worker := NewExecutionWorker(engine, defRepo, registry).(*ExecutionWorker)

	cmd := business.ExecutionCommand{
		ExecutionID:    "exec-1",
		InstanceID:     "inst-1",
		Workflow:       "wf",
		WorkflowVersion: 1,
		State:          "step_a",
		Attempt:        1,
		ExecutionToken: "token",
		InputPayload:   json.RawMessage(`{"hello":"world"}`),
	}
	message, _ := json.Marshal(cmd)

	err := worker.Handle(context.Background(), nil, message)
	if err != nil {
		t.Fatalf("handle failed: %v", err)
	}

	if len(engine.commits) != 1 {
		t.Fatalf("expected 1 commit, got %d", len(engine.commits))
	}
	if engine.commits[0].Error != nil {
		t.Fatalf("expected success commit, got error")
	}
}

func TestExecutionWorker_MissingAdapterCommitsError(t *testing.T) {
	spec := &dsl.WorkflowSpec{
		Version: "1.0",
		Name:    "wf",
		Steps: []*dsl.StepSpec{
			{ID: "step_a", Type: dsl.StepTypeCall, Call: &dsl.CallSpec{Action: "missing.adapter"}},
		},
	}
	blob, _ := json.Marshal(spec)

	engine := &stubEngine{}
	defRepo := &stubDefRepo{dsl: string(blob)}
	registry := connector.NewRegistry()

	worker := NewExecutionWorker(engine, defRepo, registry).(*ExecutionWorker)

	cmd := business.ExecutionCommand{
		ExecutionID:    "exec-1",
		InstanceID:     "inst-1",
		Workflow:       "wf",
		WorkflowVersion: 1,
		State:          "step_a",
		Attempt:        1,
		ExecutionToken: "token",
	}
	message, _ := json.Marshal(cmd)

	err := worker.Handle(context.Background(), nil, message)
	if err != nil {
		t.Fatalf("handle failed: %v", err)
	}

	if len(engine.commits) != 1 {
		t.Fatalf("expected 1 commit, got %d", len(engine.commits))
	}
	if engine.commits[0].Error == nil {
		t.Fatalf("expected error commit")
	}
}
