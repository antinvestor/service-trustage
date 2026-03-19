package tests_test

import (
	"context"
	"encoding/json"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/antinvestor/service-trustage/apps/default/service/handlers"
	"github.com/antinvestor/service-trustage/apps/default/service/models"
	eventv1 "github.com/antinvestor/service-trustage/gen/go/event/v1"
	runtimev1 "github.com/antinvestor/service-trustage/gen/go/runtime/v1"
	workflowv1 "github.com/antinvestor/service-trustage/gen/go/workflow/v1"
	"github.com/antinvestor/service-trustage/pkg/telemetry"
)

func mustStructFromJSON(raw string) *structpb.Struct {
	var payload map[string]any
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		panic(err)
	}

	value, err := structpb.NewStruct(payload)
	if err != nil {
		panic(err)
	}

	return value
}

func mustStructFromMap(payload map[string]any) *structpb.Struct {
	value, err := structpb.NewStruct(payload)
	if err != nil {
		panic(err)
	}

	return value
}

func (s *DefaultServiceSuite) TestWorkflowConnectServer_Lifecycle() {
	ctx := s.tenantCtx()
	server := handlers.NewWorkflowConnectServer(s.workflowBusiness(), allowAllAuthz{})

	createResp, err := server.CreateWorkflow(
		ctx,
		connect.NewRequest(&workflowv1.CreateWorkflowRequest{Dsl: mustStructFromJSON(s.sampleDSL())}),
	)
	s.Require().NoError(err)
	s.Require().NotNil(createResp.Msg.GetWorkflow())

	workflowID := createResp.Msg.GetWorkflow().GetId()
	s.NotEmpty(workflowID)

	getResp, err := server.GetWorkflow(
		ctx,
		connect.NewRequest(&workflowv1.GetWorkflowRequest{Id: workflowID}),
	)
	s.Require().NoError(err)
	s.Equal(workflowID, getResp.Msg.GetWorkflow().GetId())

	activateResp, err := server.ActivateWorkflow(
		ctx,
		connect.NewRequest(&workflowv1.ActivateWorkflowRequest{Id: workflowID}),
	)
	s.Require().NoError(err)
	s.Equal(
		workflowv1.WorkflowStatus_WORKFLOW_STATUS_ACTIVE,
		activateResp.Msg.GetWorkflow().GetStatus(),
	)

	listResp, err := server.ListWorkflows(
		ctx,
		connect.NewRequest(&workflowv1.ListWorkflowsRequest{
			Status: workflowv1.WorkflowStatus_WORKFLOW_STATUS_ACTIVE,
			Limit:  10,
		}),
	)
	s.Require().NoError(err)
	s.NotEmpty(listResp.Msg.GetItems())
}

func (s *DefaultServiceSuite) TestWorkflowConnectServer_AcceptsParallelRuntimeSteps() {
	ctx := s.tenantCtx()
	server := handlers.NewWorkflowConnectServer(s.workflowBusiness(), allowAllAuthz{})

	unsupportedDSL := `{
  "version": "1.0",
  "name": "unsupported-parallel",
  "steps": [
    {
      "id": "fanout",
      "type": "parallel",
      "parallel": {
        "steps": [
          {
            "id": "child",
            "type": "call",
            "call": {
              "action": "log.entry",
              "input": {"message": "hi"}
            }
          }
        ]
      }
    }
  ]
}`

	resp, err := server.CreateWorkflow(
		ctx,
		connect.NewRequest(&workflowv1.CreateWorkflowRequest{Dsl: mustStructFromJSON(unsupportedDSL)}),
	)
	s.Require().NoError(err)
	s.NotEmpty(resp.Msg.GetWorkflow().GetId())
}

func (s *DefaultServiceSuite) TestWorkflowConnectServer_RequiresAuth() {
	server := handlers.NewWorkflowConnectServer(s.workflowBusiness(), allowAllAuthz{})

	_, err := server.ListWorkflows(
		context.Background(),
		connect.NewRequest(&workflowv1.ListWorkflowsRequest{}),
	)
	s.Require().Error(err)
	s.Equal(connect.CodeUnauthenticated, connect.CodeOf(err))
}

func (s *DefaultServiceSuite) TestEventConnectServer_IngestAndIdempotency() {
	ctx := s.tenantCtx()
	server := handlers.NewEventConnectServer(
		s.eventRepo,
		s.auditRepo,
		allowAllAuthz{},
		telemetry.NewMetrics(),
		nil,
	)

	req := connect.NewRequest(&eventv1.IngestEventRequest{
		EventType:      "order.created",
		Source:         "shop-api",
		IdempotencyKey: "event-key-1",
		Payload: mustStructFromMap(map[string]any{
			"order_id": "ord-1",
			"amount":   1200,
		}),
	})

	firstResp, err := server.IngestEvent(ctx, req)
	s.Require().NoError(err)
	s.False(firstResp.Msg.GetIdempotent())
	s.NotEmpty(firstResp.Msg.GetEvent().GetEventId())

	secondResp, err := server.IngestEvent(ctx, req)
	s.Require().NoError(err)
	s.True(secondResp.Msg.GetIdempotent())
	s.Equal(firstResp.Msg.GetEvent().GetEventId(), secondResp.Msg.GetEvent().GetEventId())
}

func (s *DefaultServiceSuite) TestEventConnectServer_Timeline() {
	ctx := s.tenantCtx()
	server := handlers.NewEventConnectServer(
		s.eventRepo,
		s.auditRepo,
		allowAllAuthz{},
		telemetry.NewMetrics(),
		nil,
	)

	s.Require().NoError(s.auditRepo.Append(ctx, &models.WorkflowAuditEvent{
		InstanceID: "inst-connect",
		EventType:  "state.started",
		State:      "step-a",
	}))

	resp, err := server.GetInstanceTimeline(
		ctx,
		connect.NewRequest(&eventv1.GetInstanceTimelineRequest{InstanceId: "inst-connect"}),
	)
	s.Require().NoError(err)
	s.Len(resp.Msg.GetItems(), 1)
	s.Equal("state.started", resp.Msg.GetItems()[0].GetEventType())
}

func (s *DefaultServiceSuite) TestRuntimeConnectServer_Lifecycle() {
	ctx := s.tenantCtx()
	instance := &models.WorkflowInstance{
		WorkflowName:    "wf",
		WorkflowVersion: 1,
		CurrentState:    "step-a",
		Status:          models.InstanceStatusFailed,
		Revision:        1,
	}
	s.Require().NoError(s.instanceRepo.Create(ctx, instance))

	exec := &models.WorkflowStateExecution{
		InstanceID:     instance.ID,
		State:          "step-a",
		Attempt:        1,
		Status:         models.ExecStatusFailed,
		InputPayload:   `{"hello":"world"}`,
		ExecutionToken: "token",
	}
	s.Require().NoError(s.execRepo.Create(ctx, exec))

	s.Require().NoError(s.outputRepo.Store(ctx, &models.WorkflowStateOutput{
		ExecutionID: exec.ID,
		InstanceID:  instance.ID,
		State:       "step-a",
		SchemaHash:  "hash",
		Payload:     `{"result":"ok"}`,
	}))

	server := handlers.NewRuntimeConnectServer(
		s.instanceRepo,
		s.execRepo,
		s.outputRepo,
		s.auditRepo,
		s.scopeRepo,
		s.signalWaitRepo,
		s.signalMsgRepo,
		s.stateEngine(),
		allowAllAuthz{},
	)

	listInstancesResp, err := server.ListInstances(
		ctx,
		connect.NewRequest(&runtimev1.ListInstancesRequest{
			Status: runtimev1.InstanceStatus_INSTANCE_STATUS_FAILED,
			Limit:  10,
		}),
	)
	s.Require().NoError(err)
	s.NotEmpty(listInstancesResp.Msg.GetItems())

	getExecutionResp, err := server.GetExecution(
		ctx,
		connect.NewRequest(&runtimev1.GetExecutionRequest{
			ExecutionId:   exec.ID,
			IncludeOutput: true,
		}),
	)
	s.Require().NoError(err)
	s.Equal(exec.ID, getExecutionResp.Msg.GetExecution().GetId())
	s.NotNil(getExecutionResp.Msg.GetExecution().GetOutput())

	retryInstanceResp, err := server.RetryInstance(
		ctx,
		connect.NewRequest(&runtimev1.RetryInstanceRequest{InstanceId: instance.ID}),
	)
	s.Require().NoError(err)
	s.NotEmpty(retryInstanceResp.Msg.GetExecution().GetId())

	retryExecutionResp, err := server.RetryExecution(
		ctx,
		connect.NewRequest(&runtimev1.RetryExecutionRequest{ExecutionId: exec.ID}),
	)
	s.Require().NoError(err)
	s.NotEmpty(retryExecutionResp.Msg.GetExecution().GetId())
}
