//nolint:testpackage // tests need access to unexported types
package queues

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

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
	commitErr        error
	parkErr          error
	signalWaitErr    error
	signalSendErr    error
	branchScopeErr   error
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
	return s.commitErr
}

func (s *stubEngine) ParkExecutionUntil(
	_ context.Context,
	_, _ string,
	fireAt time.Time,
) error {
	s.parkCalls = append(s.parkCalls, fireAt)
	return s.parkErr
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
	return s.signalWaitErr
}

func (s *stubEngine) SendSignal(
	_ context.Context,
	_, _ string,
	_ json.RawMessage,
) (bool, error) {
	s.signalSendCalls++
	return true, s.signalSendErr
}

func (s *stubEngine) StartBranchScope(
	_ context.Context,
	_ *business.ExecutionCommand,
	_ *dsl.StepSpec,
) error {
	s.branchScopeCalls++
	return s.branchScopeErr
}

func (s *stubEngine) ReconcileBranchScope(_ context.Context, _ string) error {
	return nil
}

type stubDefRepo struct {
	dsl string
	err error
}

func (s *stubDefRepo) GetByNameAndVersion(_ context.Context, _ string, _ int) (*models.WorkflowDefinition, error) {
	if s.err != nil {
		return nil, s.err
	}
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

type failingAdapter struct{}

func (a failingAdapter) Type() string                  { return "failing.adapter" }
func (a failingAdapter) DisplayName() string           { return "Failing Adapter" }
func (a failingAdapter) InputSchema() json.RawMessage  { return json.RawMessage(`{"type":"object"}`) }
func (a failingAdapter) ConfigSchema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }
func (a failingAdapter) OutputSchema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }
func (a failingAdapter) Validate(_ *connector.ExecuteRequest) error {
	return nil
}
func (a failingAdapter) Execute(
	_ context.Context,
	_ *connector.ExecuteRequest,
) (*connector.ExecuteResponse, *connector.ExecutionError) {
	return nil, &connector.ExecutionError{
		Class:   connector.ErrorRetryable,
		Code:    "upstream_failure",
		Message: "connector failed",
	}
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

func TestExecutionWorker_HandleCoreErrorPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		defRepo     *stubDefRepo
		spec        *dsl.WorkflowSpec
		registry    func() *connector.Registry
		message     []byte
		wantErr     string
		wantCode    string
		wantCommits int
	}{
		{
			name:        "invalid message returns error",
			defRepo:     &stubDefRepo{},
			registry:    connector.NewRegistry,
			message:     []byte("{"),
			wantErr:     "unmarshal execution command",
			wantCommits: 0,
		},
		{
			name:        "definition load failure commits fatal error",
			defRepo:     &stubDefRepo{err: errors.New("not found")},
			registry:    connector.NewRegistry,
			message:     mustExecutionCommandJSON(t, "step_a"),
			wantCode:    "definition_not_found",
			wantCommits: 1,
		},
		{
			name:        "dsl parse failure commits fatal error",
			defRepo:     &stubDefRepo{dsl: "{"},
			registry:    connector.NewRegistry,
			message:     mustExecutionCommandJSON(t, "step_a"),
			wantCode:    "dsl_parse_error",
			wantCommits: 1,
		},
		{
			name: "step not found commits fatal error",
			defRepo: &stubDefRepo{dsl: mustWorkflowJSON(t, &dsl.WorkflowSpec{
				Version: "1.0",
				Name:    "wf",
				Steps: []*dsl.StepSpec{{
					ID:   "other_step",
					Type: dsl.StepTypeCall,
					Call: &dsl.CallSpec{Action: "test.adapter"},
				}},
			})},
			registry:    connector.NewRegistry,
			message:     mustExecutionCommandJSON(t, "missing_step"),
			wantCode:    "step_not_found",
			wantCommits: 1,
		},
		{
			name: "connector execution error commits business failure",
			defRepo: &stubDefRepo{dsl: mustWorkflowJSON(t, &dsl.WorkflowSpec{
				Version: "1.0",
				Name:    "wf",
				Steps: []*dsl.StepSpec{{
					ID:   "step_a",
					Type: dsl.StepTypeCall,
					Call: &dsl.CallSpec{Action: "failing.adapter"},
				}},
			})},
			registry: func() *connector.Registry {
				reg := connector.NewRegistry()
				require.NoError(t, reg.Register(failingAdapter{}))
				return reg
			},
			message:     mustExecutionCommandJSON(t, "step_a"),
			wantCode:    "upstream_failure",
			wantCommits: 1,
		},
	}

	for _, tc := range tests {

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			engine := &stubEngine{}
			worker := NewExecutionWorker(engine, tc.defRepo, tc.registry()).(*ExecutionWorker)

			err := worker.Handle(context.Background(), nil, tc.message)
			if tc.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErr)
			} else {
				require.NoError(t, err)
			}

			require.Len(t, engine.commits, tc.wantCommits)
			if tc.wantCode != "" {
				require.NotNil(t, engine.commits[0].Error)
				require.Equal(t, tc.wantCode, engine.commits[0].Error.Code)
			}
		})
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

func TestResolveDelayFireAt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		step         *dsl.StepSpec
		inputPayload json.RawMessage
		wantErr      string
	}{
		{
			name: "duration adds from now",
			step: &dsl.StepSpec{
				ID:    "wait",
				Type:  dsl.StepTypeDelay,
				Delay: &dsl.DelaySpec{Duration: dsl.Duration{Duration: time.Minute}},
			},
		},
		{
			name: "until expression returns timestamp string",
			step: &dsl.StepSpec{
				ID:   "wait",
				Type: dsl.StepTypeDelay,
				Delay: &dsl.DelaySpec{
					Until: "payload.fire_at",
				},
			},
			inputPayload: json.RawMessage(`{"fire_at":"2030-01-01T00:00:00Z"}`),
		},
		{
			name: "missing delay config fails",
			step: &dsl.StepSpec{
				ID:   "wait",
				Type: dsl.StepTypeDelay,
			},
			wantErr: "missing delay configuration",
		},
		{
			name: "invalid payload fails",
			step: &dsl.StepSpec{
				ID:   "wait",
				Type: dsl.StepTypeDelay,
				Delay: &dsl.DelaySpec{
					Until: "payload.fire_at",
				},
			},
			inputPayload: json.RawMessage(`{"fire_at":`),
			wantErr:      "unmarshal delay input payload",
		},
		{
			name: "non timestamp result fails",
			step: &dsl.StepSpec{
				ID:   "wait",
				Type: dsl.StepTypeDelay,
				Delay: &dsl.DelaySpec{
					Until: "1 + 2",
				},
			},
			wantErr: "must evaluate to RFC3339 string or timestamp",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fireAt, err := resolveDelayFireAt(tc.step, tc.inputPayload)
			if tc.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErr)
				return
			}

			require.NoError(t, err)
			require.False(t, fireAt.IsZero())
		})
	}
}

func TestResolveSignalSend(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		spec         *dsl.SignalSendSpec
		inputPayload json.RawMessage
		wantTarget   string
		wantPayload  string
		wantErr      string
	}{
		{
			name: "template target and payload resolve",
			spec: &dsl.SignalSendSpec{
				TargetWorkflowID: "{{ parent_instance_id }}",
				SignalName:       "done",
				Payload:          map[string]any{"approved": "{{ item.approved }}"},
			},
			inputPayload: json.RawMessage(`{"parent_instance_id":"inst-1","item":{"approved":true}}`),
			wantTarget:   "inst-1",
			wantPayload:  `{"approved":"true"}`,
		},
		{
			name: "nil payload becomes empty object",
			spec: &dsl.SignalSendSpec{
				TargetWorkflowID: "{{ parent_instance_id }}",
				SignalName:       "done",
			},
			inputPayload: json.RawMessage(`{"parent_instance_id":"inst-2"}`),
			wantTarget:   "inst-2",
			wantPayload:  `{}`,
		},
		{
			name: "invalid input payload fails",
			spec: &dsl.SignalSendSpec{
				TargetWorkflowID: "{{ parent_instance_id }}",
				SignalName:       "done",
			},
			inputPayload: json.RawMessage(`{"parent_instance_id":`),
			wantErr:      "unmarshal input payload",
		},
		{
			name: "missing template variable fails",
			spec: &dsl.SignalSendSpec{
				TargetWorkflowID: "{{ parent_instance_id }}",
				SignalName:       "done",
			},
			inputPayload: json.RawMessage(`{"other":"value"}`),
			wantErr:      "resolve signal target",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			target, payload, err := resolveSignalSend(tc.spec, tc.inputPayload)
			if tc.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.wantTarget, target)
			require.JSONEq(t, tc.wantPayload, string(payload))
		})
	}
}

func TestExecutionWorker_LocalValidationErrorsCommitFailure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		spec         *dsl.WorkflowSpec
		inputPayload json.RawMessage
		wantCode     string
	}{
		{
			name: "invalid if expression commits fatal error",
			spec: &dsl.WorkflowSpec{
				Version: "1.0",
				Name:    "wf",
				Steps: []*dsl.StepSpec{{
					ID:   "check",
					Type: dsl.StepTypeIf,
					If:   &dsl.IfSpec{Expr: "payload.amount >", Then: []*dsl.StepSpec{}, Else: []*dsl.StepSpec{}},
				}},
			},
			inputPayload: json.RawMessage(`{"amount":2}`),
			wantCode:     "invalid_if_expression",
		},
		{
			name: "missing delay config commits fatal error",
			spec: &dsl.WorkflowSpec{
				Version: "1.0",
				Name:    "wf",
				Steps: []*dsl.StepSpec{{
					ID:   "wait",
					Type: dsl.StepTypeDelay,
				}},
			},
			wantCode: "delay_resolution_failed",
		},
		{
			name: "missing signal send config commits fatal error",
			spec: &dsl.WorkflowSpec{
				Version: "1.0",
				Name:    "wf",
				Steps: []*dsl.StepSpec{{
					ID:   "send",
					Type: dsl.StepTypeSignalSend,
				}},
			},
			wantCode: "invalid_signal_send_step",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			blob, err := json.Marshal(tc.spec)
			require.NoError(t, err)

			engine := &stubEngine{}
			worker := NewExecutionWorker(engine, &stubDefRepo{dsl: string(blob)}, connector.NewRegistry()).(*ExecutionWorker)

			cmd := business.ExecutionCommand{
				ExecutionID:     "exec-1",
				InstanceID:      "inst-1",
				Workflow:        "wf",
				WorkflowVersion: 1,
				State:           tc.spec.Steps[0].ID,
				Attempt:         1,
				ExecutionToken:  "token",
				InputPayload:    tc.inputPayload,
			}
			message, err := json.Marshal(cmd)
			require.NoError(t, err)

			require.NoError(t, worker.Handle(context.Background(), nil, message))
			require.Len(t, engine.commits, 1)
			require.NotNil(t, engine.commits[0].Error)
			require.Equal(t, tc.wantCode, engine.commits[0].Error.Code)
		})
	}
}

func TestExecutionWorker_EngineConflictAndFailurePaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		spec        *dsl.WorkflowSpec
		engine      *stubEngine
		wantErr     string
		wantCommits int
	}{
		{
			name: "stale commit is swallowed",
			spec: &dsl.WorkflowSpec{
				Version: "1.0",
				Name:    "wf",
				Steps: []*dsl.StepSpec{{
					ID:   "step_a",
					Type: dsl.StepTypeCall,
					Call: &dsl.CallSpec{Action: "test.adapter"},
				}},
			},
			engine:      &stubEngine{commitErr: business.ErrStaleExecution},
			wantCommits: 1,
		},
		{
			name: "signal wait stale is swallowed",
			spec: &dsl.WorkflowSpec{
				Version: "1.0",
				Name:    "wf",
				Steps: []*dsl.StepSpec{{
					ID:         "wait_signal",
					Type:       dsl.StepTypeSignalWait,
					SignalWait: &dsl.SignalWaitSpec{SignalName: "approved"},
				}},
			},
			engine:      &stubEngine{signalWaitErr: business.ErrInvalidToken},
			wantCommits: 0,
		},
		{
			name: "branch scope stale is swallowed",
			spec: &dsl.WorkflowSpec{
				Version: "1.0",
				Name:    "wf",
				Steps: []*dsl.StepSpec{{
					ID:   "fanout",
					Type: dsl.StepTypeParallel,
					Parallel: &dsl.ParallelSpec{
						Steps: []*dsl.StepSpec{
							{ID: "child", Type: dsl.StepTypeCall, Call: &dsl.CallSpec{Action: "test.adapter"}},
						},
					},
				}},
			},
			engine:      &stubEngine{branchScopeErr: business.ErrStaleExecution},
			wantCommits: 0,
		},
		{
			name: "signal send hard failure bubbles up",
			spec: &dsl.WorkflowSpec{
				Version: "1.0",
				Name:    "wf",
				Steps: []*dsl.StepSpec{{
					ID:   "send",
					Type: dsl.StepTypeSignalSend,
					SignalSend: &dsl.SignalSendSpec{
						TargetWorkflowID: "{{ parent_instance_id }}",
						SignalName:       "approved",
						Payload:          map[string]any{"ok": true},
					},
				}},
			},
			engine:      &stubEngine{signalSendErr: errors.New("queue down")},
			wantErr:     "send signal: queue down",
			wantCommits: 0,
		},
		{
			name: "delay park hard failure bubbles up",
			spec: &dsl.WorkflowSpec{
				Version: "1.0",
				Name:    "wf",
				Steps: []*dsl.StepSpec{{
					ID:    "wait",
					Type:  dsl.StepTypeDelay,
					Delay: &dsl.DelaySpec{Duration: dsl.Duration{Duration: time.Minute}},
				}},
			},
			engine:      &stubEngine{parkErr: errors.New("db unavailable")},
			wantErr:     "park execution until",
			wantCommits: 0,
		},
	}

	for _, tc := range tests {

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			blob, err := json.Marshal(tc.spec)
			require.NoError(t, err)

			registry := connector.NewRegistry()
			require.NoError(t, registry.Register(stubAdapter{}))
			worker := NewExecutionWorker(tc.engine, &stubDefRepo{dsl: string(blob)}, registry).(*ExecutionWorker)

			cmd := business.ExecutionCommand{
				ExecutionID:     "exec-1",
				InstanceID:      "inst-1",
				Workflow:        "wf",
				WorkflowVersion: 1,
				State:           tc.spec.Steps[0].ID,
				Attempt:         1,
				ExecutionToken:  "token",
				InputPayload:    json.RawMessage(`{"parent_instance_id":"root-1"}`),
			}
			message, err := json.Marshal(cmd)
			require.NoError(t, err)

			err = worker.Handle(context.Background(), nil, message)
			if tc.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErr)
			} else {
				require.NoError(t, err)
			}
			require.Len(t, tc.engine.commits, tc.wantCommits)
		})
	}
}

func mustWorkflowJSON(t *testing.T, spec *dsl.WorkflowSpec) string {
	t.Helper()

	blob, err := json.Marshal(spec)
	require.NoError(t, err)

	return string(blob)
}

func mustExecutionCommandJSON(t *testing.T, state string) []byte {
	t.Helper()

	payload, err := json.Marshal(business.ExecutionCommand{
		ExecutionID:     "exec-1",
		InstanceID:      "inst-1",
		Workflow:        "wf",
		WorkflowVersion: 1,
		State:           state,
		Attempt:         1,
		ExecutionToken:  "token",
		InputPayload:    json.RawMessage(`{"parent_instance_id":"root-1"}`),
	})
	require.NoError(t, err)

	return payload
}
