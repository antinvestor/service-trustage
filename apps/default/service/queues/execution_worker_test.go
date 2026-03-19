//nolint:testpackage // tests need access to unexported types
package queues

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/antinvestor/service-trustage/apps/default/service/business"
	"github.com/antinvestor/service-trustage/apps/default/service/models"
	"github.com/antinvestor/service-trustage/connector"
	"github.com/antinvestor/service-trustage/dsl"
)

type stubEngine struct {
	commits          []*business.CommitRequest
	parkCalls        []time.Time
	signalWaitCalls  int
	signalSendCalls  int
	branchScopeCalls int
}

var errNotUsed = errors.New("not used in this test")

func (s *stubEngine) CreateInitialExecution(
	_ context.Context,
	_ *models.WorkflowInstance,
	_ json.RawMessage,
) (*business.ExecutionCommand, error) {
	return nil, errNotUsed
}

func (s *stubEngine) Dispatch(_ context.Context, _ *models.WorkflowStateExecution) (*business.ExecutionCommand, error) {
	return nil, errNotUsed
}

func (s *stubEngine) Commit(_ context.Context, req *business.CommitRequest) error {
	s.commits = append(s.commits, req)
	return nil
}

func (s *stubEngine) ParkExecutionUntil(
	_ context.Context,
	_, _ string,
	fireAt time.Time,
) error {
	s.parkCalls = append(s.parkCalls, fireAt)
	return nil
}

func (s *stubEngine) ResumeWaitingExecution(_ context.Context, _ string, _ json.RawMessage) error {
	return nil
}

func (s *stubEngine) FailWaitingExecution(
	_ context.Context,
	_ string,
	_ models.ExecutionStatus,
	_ *business.CommitError,
) error {
	return nil
}

func (s *stubEngine) StartSignalWait(
	_ context.Context,
	_ *business.ExecutionCommand,
	_ *dsl.StepSpec,
) error {
	s.signalWaitCalls++
	return nil
}

func (s *stubEngine) SendSignal(
	_ context.Context,
	_, _ string,
	_ json.RawMessage,
) (bool, error) {
	s.signalSendCalls++
	return true, nil
}

func (s *stubEngine) StartBranchScope(
	_ context.Context,
	_ *business.ExecutionCommand,
	_ *dsl.StepSpec,
) error {
	s.branchScopeCalls++
	return nil
}

func (s *stubEngine) ReconcileBranchScope(_ context.Context, _ string) error {
	return nil
}

type stubDefRepo struct {
	dsl string
}

func (s *stubDefRepo) GetByNameAndVersion(_ context.Context, _ string, _ int) (*models.WorkflowDefinition, error) {
	return &models.WorkflowDefinition{DSLBlob: s.dsl}, nil
}

type stubAdapter struct{}

func (a stubAdapter) Type() string                               { return "test.adapter" }
func (a stubAdapter) DisplayName() string                        { return "Test Adapter" }
func (a stubAdapter) InputSchema() json.RawMessage               { return json.RawMessage(`{"type":"object"}`) }
func (a stubAdapter) ConfigSchema() json.RawMessage              { return json.RawMessage(`{"type":"object"}`) }
func (a stubAdapter) OutputSchema() json.RawMessage              { return json.RawMessage(`{"type":"object"}`) }
func (a stubAdapter) Validate(_ *connector.ExecuteRequest) error { return nil }

func (a stubAdapter) Execute(
	_ context.Context,
	_ *connector.ExecuteRequest,
) (*connector.ExecuteResponse, *connector.ExecutionError) {
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
		ExecutionID:     "exec-1",
		InstanceID:      "inst-1",
		Workflow:        "wf",
		WorkflowVersion: 1,
		State:           "step_a",
		Attempt:         1,
		ExecutionToken:  "token",
		InputPayload:    json.RawMessage(`{"hello":"world"}`),
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
		ExecutionID:     "exec-1",
		InstanceID:      "inst-1",
		Workflow:        "wf",
		WorkflowVersion: 1,
		State:           "step_a",
		Attempt:         1,
		ExecutionToken:  "token",
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

func TestExecutionWorker_SequenceCommitsSuccess(t *testing.T) {
	spec := &dsl.WorkflowSpec{
		Version: "1.0",
		Name:    "wf",
		Steps: []*dsl.StepSpec{
			{
				ID:   "sequence_root",
				Type: dsl.StepTypeSequence,
				Sequence: &dsl.SequenceSpec{
					Steps: []*dsl.StepSpec{
						{ID: "child_a", Type: dsl.StepTypeCall, Call: &dsl.CallSpec{Action: "test.adapter"}},
					},
				},
			},
		},
	}
	blob, _ := json.Marshal(spec)

	engine := &stubEngine{}
	defRepo := &stubDefRepo{dsl: string(blob)}
	registry := connector.NewRegistry()
	_ = registry.Register(stubAdapter{})

	worker := NewExecutionWorker(engine, defRepo, registry).(*ExecutionWorker)

	cmd := business.ExecutionCommand{
		ExecutionID:     "exec-seq",
		InstanceID:      "inst-seq",
		Workflow:        "wf",
		WorkflowVersion: 1,
		State:           "sequence_root",
		Attempt:         1,
		ExecutionToken:  "token",
	}
	message, _ := json.Marshal(cmd)

	if err := worker.Handle(context.Background(), nil, message); err != nil {
		t.Fatalf("handle failed: %v", err)
	}

	if len(engine.commits) != 1 {
		t.Fatalf("expected 1 commit, got %d", len(engine.commits))
	}
	if engine.commits[0].Error != nil {
		t.Fatalf("expected sequence entry to commit success, got %#v", engine.commits[0].Error)
	}
}

func TestExecutionWorker_ParallelStartsBranchScope(t *testing.T) {
	spec := &dsl.WorkflowSpec{
		Version: "1.0",
		Name:    "wf",
		Steps: []*dsl.StepSpec{
			{ID: "fanout", Type: dsl.StepTypeParallel, Parallel: &dsl.ParallelSpec{Steps: []*dsl.StepSpec{
				{ID: "child", Type: dsl.StepTypeCall, Call: &dsl.CallSpec{Action: "test.adapter"}},
			}}},
		},
	}
	blob, _ := json.Marshal(spec)

	engine := &stubEngine{}
	defRepo := &stubDefRepo{dsl: string(blob)}
	registry := connector.NewRegistry()

	worker := NewExecutionWorker(engine, defRepo, registry).(*ExecutionWorker)

	cmd := business.ExecutionCommand{
		ExecutionID:     "exec-parallel",
		InstanceID:      "inst-parallel",
		Workflow:        "wf",
		WorkflowVersion: 1,
		State:           "fanout",
		Attempt:         1,
		ExecutionToken:  "token",
	}
	message, _ := json.Marshal(cmd)

	if err := worker.Handle(context.Background(), nil, message); err != nil {
		t.Fatalf("handle failed: %v", err)
	}

	if engine.branchScopeCalls != 1 {
		t.Fatalf("expected 1 branch scope call, got %d", engine.branchScopeCalls)
	}
	if len(engine.commits) != 0 {
		t.Fatalf("expected no direct commit for parallel branch scope, got %d", len(engine.commits))
	}
}

func TestExecutionWorker_IfCommitsBranchSelection(t *testing.T) {
	spec := &dsl.WorkflowSpec{
		Version: "1.0",
		Name:    "wf",
		Steps: []*dsl.StepSpec{
			{
				ID:   "check",
				Type: dsl.StepTypeIf,
				If: &dsl.IfSpec{
					Expr: "payload.amount > 100",
					Then: []*dsl.StepSpec{
						{ID: "high", Type: dsl.StepTypeCall, Call: &dsl.CallSpec{Action: "test.adapter"}},
					},
					Else: []*dsl.StepSpec{
						{ID: "low", Type: dsl.StepTypeCall, Call: &dsl.CallSpec{Action: "test.adapter"}},
					},
				},
			},
		},
	}
	blob, _ := json.Marshal(spec)

	engine := &stubEngine{}
	defRepo := &stubDefRepo{dsl: string(blob)}
	registry := connector.NewRegistry()
	worker := NewExecutionWorker(engine, defRepo, registry).(*ExecutionWorker)

	cmd := business.ExecutionCommand{
		ExecutionID:     "exec-if",
		InstanceID:      "inst-if",
		Workflow:        "wf",
		WorkflowVersion: 1,
		State:           "check",
		Attempt:         1,
		ExecutionToken:  "token",
		InputPayload:    json.RawMessage(`{"amount":150}`),
	}
	message, _ := json.Marshal(cmd)

	if err := worker.Handle(context.Background(), nil, message); err != nil {
		t.Fatalf("handle failed: %v", err)
	}

	if len(engine.commits) != 1 {
		t.Fatalf("expected 1 commit, got %d", len(engine.commits))
	}

	var output map[string]any
	if err := json.Unmarshal(engine.commits[0].Output, &output); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if output["branch"] != "then" {
		t.Fatalf("expected then branch, got %#v", output["branch"])
	}
}

func TestExecutionWorker_DelayParksFutureExecution(t *testing.T) {
	spec := &dsl.WorkflowSpec{
		Version: "1.0",
		Name:    "wf",
		Steps: []*dsl.StepSpec{
			{
				ID:    "wait_step",
				Type:  dsl.StepTypeDelay,
				Delay: &dsl.DelaySpec{Duration: dsl.Duration{Duration: time.Minute}},
			},
		},
	}
	blob, _ := json.Marshal(spec)

	engine := &stubEngine{}
	defRepo := &stubDefRepo{dsl: string(blob)}
	registry := connector.NewRegistry()
	worker := NewExecutionWorker(engine, defRepo, registry).(*ExecutionWorker)

	cmd := business.ExecutionCommand{
		ExecutionID:     "exec-delay",
		InstanceID:      "inst-delay",
		Workflow:        "wf",
		WorkflowVersion: 1,
		State:           "wait_step",
		Attempt:         1,
		ExecutionToken:  "token",
	}
	message, _ := json.Marshal(cmd)

	if err := worker.Handle(context.Background(), nil, message); err != nil {
		t.Fatalf("handle failed: %v", err)
	}

	if len(engine.parkCalls) != 1 {
		t.Fatalf("expected 1 park call, got %d", len(engine.parkCalls))
	}
	if len(engine.commits) != 0 {
		t.Fatalf("expected no commit for future delay, got %d", len(engine.commits))
	}
}

func TestExecutionWorker_SignalWaitStartsWait(t *testing.T) {
	spec := &dsl.WorkflowSpec{
		Version: "1.0",
		Name:    "wf",
		Steps: []*dsl.StepSpec{
			{
				ID:         "approval_wait",
				Type:       dsl.StepTypeSignalWait,
				SignalWait: &dsl.SignalWaitSpec{SignalName: "approval_response"},
			},
		},
	}
	blob, _ := json.Marshal(spec)

	engine := &stubEngine{}
	defRepo := &stubDefRepo{dsl: string(blob)}
	worker := NewExecutionWorker(engine, defRepo, connector.NewRegistry()).(*ExecutionWorker)

	cmd := business.ExecutionCommand{
		ExecutionID:     "exec-signal-wait",
		InstanceID:      "inst-signal-wait",
		Workflow:        "wf",
		WorkflowVersion: 1,
		State:           "approval_wait",
		Attempt:         1,
		ExecutionToken:  "token",
	}
	message, _ := json.Marshal(cmd)

	if err := worker.Handle(context.Background(), nil, message); err != nil {
		t.Fatalf("handle failed: %v", err)
	}

	if engine.signalWaitCalls != 1 {
		t.Fatalf("expected 1 signal wait call, got %d", engine.signalWaitCalls)
	}
}

func TestExecutionWorker_SignalSendCallsEngine(t *testing.T) {
	spec := &dsl.WorkflowSpec{
		Version: "1.0",
		Name:    "wf",
		Steps: []*dsl.StepSpec{
			{
				ID:   "notify_parent",
				Type: dsl.StepTypeSignalSend,
				SignalSend: &dsl.SignalSendSpec{
					TargetWorkflowID: "{{ parent_instance_id }}",
					SignalName:       "child_completed",
					Payload:          map[string]any{"approved": "{{ item.approved }}"},
				},
			},
		},
	}
	blob, _ := json.Marshal(spec)

	engine := &stubEngine{}
	defRepo := &stubDefRepo{dsl: string(blob)}
	worker := NewExecutionWorker(engine, defRepo, connector.NewRegistry()).(*ExecutionWorker)

	cmd := business.ExecutionCommand{
		ExecutionID:     "exec-signal-send",
		InstanceID:      "inst-signal-send",
		Workflow:        "wf",
		WorkflowVersion: 1,
		State:           "notify_parent",
		Attempt:         1,
		ExecutionToken:  "token",
		InputPayload:    json.RawMessage(`{"parent_instance_id":"root-1","item":{"approved":true}}`),
	}
	message, _ := json.Marshal(cmd)

	if err := worker.Handle(context.Background(), nil, message); err != nil {
		t.Fatalf("handle failed: %v", err)
	}

	if engine.signalSendCalls != 1 {
		t.Fatalf("expected 1 signal send call, got %d", engine.signalSendCalls)
	}
	if len(engine.commits) != 1 || engine.commits[0].Error != nil {
		t.Fatalf("expected successful commit after signal send")
	}
}
