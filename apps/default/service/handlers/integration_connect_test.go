//nolint:testpackage // package-local integration tests use unexported handler fixtures intentionally.
package handlers

import (
	"context"
	"encoding/json"
	"testing"

	"connectrpc.com/connect"
	commonv1 "github.com/antinvestor/common/v1"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
	eventv1 "github.com/antinvestor/service-trustage/gen/go/event/v1"
	runtimev1 "github.com/antinvestor/service-trustage/gen/go/runtime/v1"
	signalv1 "github.com/antinvestor/service-trustage/gen/go/signal/v1"
	workflowv1 "github.com/antinvestor/service-trustage/gen/go/workflow/v1"
)

func (s *HandlerSuite) TestWorkflowConnectServer_LifecycleAndAuth() {
	ctx := s.tenantCtx()
	server := NewWorkflowConnectServer(s.workflowBusiness())

	createResp, err := server.CreateWorkflow(
		ctx,
		connectReq(&workflowv1.CreateWorkflowRequest{Dsl: mustStructFromJSON(s.sampleDSL())}),
	)
	s.Require().NoError(err)
	workflowID := createResp.Msg.GetWorkflow().GetId()
	s.NotEmpty(workflowID)

	_, err = server.GetWorkflow(ctx, connectReq(&workflowv1.GetWorkflowRequest{Id: workflowID}))
	s.Require().NoError(err)

	activateResp, err := server.ActivateWorkflow(ctx, connectReq(&workflowv1.ActivateWorkflowRequest{Id: workflowID}))
	s.Require().NoError(err)
	s.Equal(workflowv1.WorkflowStatus_WORKFLOW_STATUS_ACTIVE, activateResp.Msg.GetWorkflow().GetStatus())

	_, err = server.ListWorkflows(ctx, connectReq(&workflowv1.ListWorkflowsRequest{
		Status: workflowv1.WorkflowStatus_WORKFLOW_STATUS_ACTIVE,
		Search: &commonv1.SearchRequest{
			Cursor: &commonv1.PageCursor{Limit: 10},
		},
	}))
	s.Require().NoError(err)

	_, err = server.ListWorkflows(context.Background(), connectReq(&workflowv1.ListWorkflowsRequest{}))
	s.Require().Error(err)
	s.Equal(connect.CodeUnauthenticated, connect.CodeOf(err))
}

func (s *HandlerSuite) TestEventConnectServer_IngestAndTimeline() {
	ctx := s.tenantCtx()
	server := NewEventConnectServer(s.eventRepo, s.auditRepo, s.metrics, nil)

	req := connectReq(&eventv1.IngestEventRequest{
		EventType:      "order.created",
		Source:         "shop-api",
		IdempotencyKey: "event-key-1",
		Payload:        mustStructFromMap(map[string]any{"order_id": "ord-1", "amount": 1200}),
	})

	firstResp, err := server.IngestEvent(ctx, req)
	s.Require().NoError(err)
	s.False(firstResp.Msg.GetIdempotent())
	s.NotEmpty(firstResp.Msg.GetEvent().GetEventId())

	secondResp, err := server.IngestEvent(ctx, req)
	s.Require().NoError(err)
	s.True(secondResp.Msg.GetIdempotent())

	s.Require().NoError(s.auditRepo.Append(ctx, &models.WorkflowAuditEvent{
		InstanceID: "inst-connect",
		EventType:  "state.started",
		State:      "step-a",
	}))

	timelineResp, err := server.GetInstanceTimeline(
		ctx,
		connectReq(&eventv1.GetInstanceTimelineRequest{InstanceId: "inst-connect"}),
	)
	s.Require().NoError(err)
	s.Len(timelineResp.Msg.GetItems(), 1)
}

func (s *HandlerSuite) TestRuntimeAndSignalConnectServer_Flows() {
	ctx := s.tenantCtx()
	engine := s.stateEngine()

	instance := &models.WorkflowInstance{
		WorkflowName:    "wf",
		WorkflowVersion: 1,
		CurrentState:    "wait",
		Status:          models.InstanceStatusRunning,
		Revision:        1,
		Metadata:        `{"tenant":"x"}`,
	}
	s.Require().NoError(s.instanceRepo.Create(ctx, instance))

	waitingExec := &models.WorkflowStateExecution{
		InstanceID:      instance.ID,
		State:           "wait",
		Attempt:         1,
		Status:          models.ExecStatusWaiting,
		ExecutionToken:  "token-a",
		InputSchemaHash: "hash-a",
		InputPayload:    `{"hello":"world"}`,
		TraceID:         "trace-1",
	}
	s.Require().NoError(s.execRepo.Create(ctx, waitingExec))

	failedExec := &models.WorkflowStateExecution{
		InstanceID:      instance.ID,
		State:           "notify",
		Attempt:         2,
		Status:          models.ExecStatusFailed,
		ExecutionToken:  "token-b",
		InputSchemaHash: "hash-b",
		InputPayload:    `{"bye":"world"}`,
		TraceID:         "trace-2",
		ErrorClass:      "fatal",
		ErrorMessage:    "boom",
	}
	s.Require().NoError(s.execRepo.Create(ctx, failedExec))

	s.Require().NoError(s.outputRepo.Store(ctx, &models.WorkflowStateOutput{
		ExecutionID: failedExec.ID,
		InstanceID:  instance.ID,
		State:       failedExec.State,
		SchemaHash:  "out",
		Payload:     `{"ok":false}`,
	}))
	s.Require().NoError(s.auditRepo.Append(ctx, &models.WorkflowAuditEvent{
		InstanceID:  instance.ID,
		ExecutionID: failedExec.ID,
		EventType:   "state.failed",
		State:       failedExec.State,
		Payload:     `{"reason":"boom"}`,
		TraceID:     failedExec.TraceID,
	}))
	s.Require().NoError(s.scopeRepo.Create(ctx, &models.WorkflowScopeRun{
		ParentExecutionID: failedExec.ID,
		ParentInstanceID:  instance.ID,
		ParentState:       "fanout",
		ScopeType:         "parallel",
		Status:            "running",
		WaitAll:           true,
		TotalChildren:     2,
		ResultsPayload:    `[null,null]`,
	}))
	s.Require().NoError(s.signalWaitRepo.Create(ctx, &models.WorkflowSignalWait{
		ExecutionID: waitingExec.ID,
		InstanceID:  instance.ID,
		State:       waitingExec.State,
		SignalName:  "approved",
		OutputVar:   "approval",
		Status:      "waiting",
	}))
	s.Require().NoError(s.signalMsgRepo.Create(ctx, &models.WorkflowSignalMessage{
		TargetInstanceID: instance.ID,
		SignalName:       "approved",
		Payload:          `{"ok":true}`,
		Status:           "pending",
	}))

	runtimeServer := NewRuntimeConnectServer(
		s.instanceRepo,
		s.execRepo,
		s.outputRepo,
		s.auditRepo,
		s.scopeRepo,
		s.signalWaitRepo,
		s.signalMsgRepo,
		engine,
	)

	_, err := runtimeServer.ListInstances(ctx, connectReq(&runtimev1.ListInstancesRequest{
		Search: &commonv1.SearchRequest{
			Cursor: &commonv1.PageCursor{Limit: 10},
		},
	}))
	s.Require().NoError(err)
	_, err = runtimeServer.ListExecutions(
		ctx,
		connectReq(&runtimev1.ListExecutionsRequest{
			InstanceId: instance.ID,
			Search: &commonv1.SearchRequest{
				Cursor: &commonv1.PageCursor{Limit: 10},
			},
		}),
	)
	s.Require().NoError(err)
	_, err = runtimeServer.GetExecution(ctx, connectReq(&runtimev1.GetExecutionRequest{
		ExecutionId:   failedExec.ID,
		IncludeOutput: true,
	}))
	s.Require().NoError(err)
	retryInstanceResp, err := runtimeServer.RetryInstance(ctx, connectReq(&runtimev1.RetryInstanceRequest{
		InstanceId: instance.ID,
	}))
	s.Require().NoError(err)
	s.Equal(int32(3), retryInstanceResp.Msg.GetExecution().GetAttempt())
	_, err = runtimeServer.GetInstanceRun(ctx, connectReq(&runtimev1.GetInstanceRunRequest{
		InstanceId:      instance.ID,
		ExecutionLimit:  10,
		TimelineLimit:   10,
		IncludePayloads: true,
	}))
	s.Require().NoError(err)
	_, err = runtimeServer.RetryExecution(ctx, connectReq(&runtimev1.RetryExecutionRequest{ExecutionId: failedExec.ID}))
	s.Require().NoError(err)

	waitingDSL := `{
  "version": "1.0",
  "name": "delay-workflow",
  "steps": [
    {
      "id": "wait",
      "type": "delay",
      "delay": { "duration": "1m" }
    },
    {
      "id": "after",
      "type": "call",
      "call": { "action": "log.entry", "input": { "message": "after" } }
    }
  ]
}`

	def := s.createWorkflow(ctx, waitingDSL)
	delayInstance := &models.WorkflowInstance{
		WorkflowName:    def.Name,
		WorkflowVersion: def.WorkflowVersion,
		CurrentState:    "wait",
		Status:          models.InstanceStatusRunning,
		Revision:        1,
		Metadata:        "{}",
	}
	s.Require().NoError(s.instanceRepo.Create(ctx, delayInstance))

	delayExec := &models.WorkflowStateExecution{
		InstanceID:      delayInstance.ID,
		State:           "wait",
		Attempt:         1,
		Status:          models.ExecStatusWaiting,
		InputSchemaHash: "hash",
		InputPayload:    `{"message":"preserved"}`,
	}
	s.Require().NoError(s.execRepo.Create(ctx, delayExec))

	resumeResp, err := runtimeServer.ResumeExecution(ctx, connectReq(&runtimev1.ResumeExecutionRequest{
		ExecutionId: delayExec.ID,
		Payload:     mustStructFromMap(map[string]any{}),
	}))
	s.Require().NoError(err)
	s.Equal("resumed_waiting_execution", resumeResp.Msg.GetAction())

	retryResumeResp, err := runtimeServer.ResumeExecution(ctx, connectReq(&runtimev1.ResumeExecutionRequest{
		ExecutionId: failedExec.ID,
	}))
	s.Require().NoError(err)
	s.Equal("created_retry_execution", retryResumeResp.Msg.GetAction())
	s.Equal(int32(3), retryResumeResp.Msg.GetExecution().GetAttempt())

	pendingSignalInstance := &models.WorkflowInstance{
		WorkflowName:    "pending-signal-wf",
		WorkflowVersion: 1,
		CurrentState:    "idle",
		Status:          models.InstanceStatusRunning,
		Revision:        1,
		Metadata:        "{}",
	}
	s.Require().NoError(s.instanceRepo.Create(ctx, pendingSignalInstance))

	signalServer := NewSignalConnectServer(engine)
	signalResp, err := signalServer.SendSignal(ctx, connectReq(&signalv1.SendSignalRequest{
		InstanceId: pendingSignalInstance.ID,
		SignalName: "approved",
		Payload:    mustStructFromMap(map[string]any{"ok": true}),
	}))
	s.Require().NoError(err)
	s.False(signalResp.Msg.GetDelivered())

	messages, err := s.signalMsgRepo.ListByInstance(ctx, pendingSignalInstance.ID, 10)
	s.Require().NoError(err)
	s.NotEmpty(messages)

	var payload map[string]any
	s.Require().NoError(json.Unmarshal([]byte(messages[0].Payload), &payload))
	s.Equal(true, payload["ok"])
}

func (s *HandlerSuite) TestConnectServers_ValidationAndFailurePaths() {
	ctx := s.tenantCtx()
	runtimeServer := NewRuntimeConnectServer(
		s.instanceRepo,
		s.execRepo,
		s.outputRepo,
		s.auditRepo,
		s.scopeRepo,
		s.signalWaitRepo,
		s.signalMsgRepo,
		s.stateEngine(),
	)
	eventServer := NewEventConnectServer(s.eventRepo, s.auditRepo, s.metrics, nil)
	signalServer := NewSignalConnectServer(s.stateEngine())
	workflowServer := NewWorkflowConnectServer(s.workflowBusiness())

	tests := []struct {
		name     string
		exec     func() error
		wantCode connect.Code
	}{
		{
			name: "event ingest requires event type",
			exec: func() error {
				_, err := eventServer.IngestEvent(ctx, connectReq(&eventv1.IngestEventRequest{
					Source:  "api",
					Payload: mustStructFromMap(map[string]any{}),
				}))
				return err
			},
			wantCode: connect.CodeInvalidArgument,
		},
		{
			name: "signal send requires instance id",
			exec: func() error {
				_, err := signalServer.SendSignal(ctx, connectReq(&signalv1.SendSignalRequest{
					SignalName: "approved",
					Payload:    mustStructFromMap(map[string]any{}),
				}))
				return err
			},
			wantCode: connect.CodeInvalidArgument,
		},
		{
			name: "runtime retry instance requires id",
			exec: func() error {
				_, err := runtimeServer.RetryInstance(ctx, connectReq(&runtimev1.RetryInstanceRequest{}))
				return err
			},
			wantCode: connect.CodeInvalidArgument,
		},
		{
			name: "runtime retry execution requires id",
			exec: func() error {
				_, err := runtimeServer.RetryExecution(ctx, connectReq(&runtimev1.RetryExecutionRequest{}))
				return err
			},
			wantCode: connect.CodeInvalidArgument,
		},
		{
			name: "runtime resume execution requires id",
			exec: func() error {
				_, err := runtimeServer.ResumeExecution(ctx, connectReq(&runtimev1.ResumeExecutionRequest{}))
				return err
			},
			wantCode: connect.CodeInvalidArgument,
		},
		{
			name: "workflow activate missing returns not found",
			exec: func() error {
				_, err := workflowServer.ActivateWorkflow(
					ctx,
					connectReq(&workflowv1.ActivateWorkflowRequest{Id: "missing"}),
				)
				return err
			},
			wantCode: connect.CodeNotFound,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			err := tc.exec()
			s.Require().Error(err)
			s.Equal(tc.wantCode, connect.CodeOf(err))
		})
	}
}

func TestResumeStrategyForExecution(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		exec   *models.WorkflowStateExecution
		expect string
	}{
		{name: "nil execution", exec: nil, expect: ""},
		{
			name:   "waiting execution",
			exec:   &models.WorkflowStateExecution{Status: models.ExecStatusWaiting},
			expect: "resume_waiting_execution",
		},
		{
			name:   "failed execution",
			exec:   &models.WorkflowStateExecution{Status: models.ExecStatusFailed},
			expect: "retry_execution",
		},
		{
			name:   "fatal execution",
			exec:   &models.WorkflowStateExecution{Status: models.ExecStatusFatal},
			expect: "retry_execution",
		},
		{
			name:   "timed out execution",
			exec:   &models.WorkflowStateExecution{Status: models.ExecStatusTimedOut},
			expect: "retry_execution",
		},
		{
			name:   "pending execution",
			exec:   &models.WorkflowStateExecution{Status: models.ExecStatusPending},
			expect: "none",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := resumeStrategyForExecution(tc.exec)
			if s != tc.expect {
				t.Fatalf("expected %q, got %q", tc.expect, s)
			}
		})
	}
}
