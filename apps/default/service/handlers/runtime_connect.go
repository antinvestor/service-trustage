package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/pitabwire/frame/security/authorizer"

	"github.com/antinvestor/service-trustage/apps/default/service/authz"
	"github.com/antinvestor/service-trustage/apps/default/service/business"
	"github.com/antinvestor/service-trustage/apps/default/service/models"
	"github.com/antinvestor/service-trustage/apps/default/service/repository"
	runtimev1 "github.com/antinvestor/service-trustage/gen/go/runtime/v1"
	"github.com/antinvestor/service-trustage/gen/go/runtime/v1/runtimev1connect"
)

// RuntimeConnectServer exposes runtime reads and retry operations over ConnectRPC.
type RuntimeConnectServer struct {
	instanceRepo   repository.WorkflowInstanceRepository
	execRepo       repository.WorkflowExecutionRepository
	runtimeRepo    repository.WorkflowRuntimeRepository
	outputRepo     repository.WorkflowOutputRepository
	auditRepo      repository.AuditEventRepository
	scopeRepo      repository.WorkflowScopeRunRepository
	signalWaitRepo repository.WorkflowSignalWaitRepository
	signalMsgRepo  repository.WorkflowSignalMessageRepository
	engine         business.StateEngine
	authz          authz.Middleware

	runtimev1connect.UnimplementedRuntimeServiceHandler
}

// NewRuntimeConnectServer creates a new Connect runtime server.
func NewRuntimeConnectServer(
	instanceRepo repository.WorkflowInstanceRepository,
	execRepo repository.WorkflowExecutionRepository,
	outputRepo repository.WorkflowOutputRepository,
	auditRepo repository.AuditEventRepository,
	scopeRepo repository.WorkflowScopeRunRepository,
	signalWaitRepo repository.WorkflowSignalWaitRepository,
	signalMsgRepo repository.WorkflowSignalMessageRepository,
	engine business.StateEngine,
	authzMiddleware authz.Middleware,
) *RuntimeConnectServer {
	return &RuntimeConnectServer{
		instanceRepo:   instanceRepo,
		execRepo:       execRepo,
		runtimeRepo:    repository.NewWorkflowRuntimeRepository(execRepo.Pool()),
		outputRepo:     outputRepo,
		auditRepo:      auditRepo,
		scopeRepo:      scopeRepo,
		signalWaitRepo: signalWaitRepo,
		signalMsgRepo:  signalMsgRepo,
		engine:         engine,
		authz:          authzMiddleware,
	}
}

func (s *RuntimeConnectServer) ListInstances(
	ctx context.Context,
	req *connect.Request[runtimev1.ListInstancesRequest],
) (*connect.Response[runtimev1.ListInstancesResponse], error) {
	if err := requireConnectAuth(ctx); err != nil {
		return nil, err
	}

	if s.authz != nil {
		if err := s.authz.CanInstanceView(ctx); err != nil {
			return nil, authorizer.ToConnectError(err)
		}
	}

	pageLimit := searchLimit(req.Msg.GetSearch(), 50)
	itemsPage, err := s.instanceRepo.ListPage(ctx, repository.WorkflowInstanceListFilter{
		Status:            instanceStatusFilter(req.Msg.GetStatus()),
		WorkflowName:      req.Msg.GetWorkflowName(),
		Query:             searchQuery(req.Msg.GetSearch()),
		IDQuery:           searchIDQuery(req.Msg.GetSearch()),
		ParentInstanceID:  searchExtraString(req.Msg.GetSearch(), "parent_instance_id"),
		ParentExecutionID: searchExtraString(req.Msg.GetSearch(), "parent_execution_id"),
		Cursor:            searchPage(req.Msg.GetSearch()),
		Limit:             pageLimit,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("failed to list instances"))
	}

	respItems := make([]*runtimev1.WorkflowInstance, 0, len(itemsPage.Items))
	for _, item := range itemsPage.Items {
		respItems = append(respItems, workflowInstanceToProto(item))
	}

	return connect.NewResponse(&runtimev1.ListInstancesResponse{
		Items:      respItems,
		NextCursor: nextCursorProto(itemsPage.NextCursor, pageLimit),
	}), nil
}

func (s *RuntimeConnectServer) RetryInstance(
	ctx context.Context,
	req *connect.Request[runtimev1.RetryInstanceRequest],
) (*connect.Response[runtimev1.RetryInstanceResponse], error) {
	if err := requireConnectAuth(ctx); err != nil {
		return nil, err
	}

	if s.authz != nil {
		if err := s.authz.CanInstanceRetry(ctx); err != nil {
			return nil, authorizer.ToConnectError(err)
		}
	}

	if req.Msg.GetInstanceId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("instance_id is required"))
	}

	instance, err := s.instanceRepo.GetByID(ctx, req.Msg.GetInstanceId())
	if err != nil {
		return nil, connectLookupError(err, "instance not found")
	}

	exec, err := s.execRepo.GetLatestByInstance(ctx, req.Msg.GetInstanceId())
	if err != nil {
		return nil, connectLookupError(err, "latest execution not found")
	}

	newExec, err := createRetryExecution(ctx, s.execRepo, s.runtimeRepo, s.auditRepo, exec, instance)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}

	return connect.NewResponse(&runtimev1.RetryInstanceResponse{
		Execution: workflowExecutionToProto(newExec, "", false),
	}), nil
}

func (s *RuntimeConnectServer) ListExecutions(
	ctx context.Context,
	req *connect.Request[runtimev1.ListExecutionsRequest],
) (*connect.Response[runtimev1.ListExecutionsResponse], error) {
	if err := requireConnectAuth(ctx); err != nil {
		return nil, err
	}

	if s.authz != nil {
		if err := s.authz.CanExecutionView(ctx); err != nil {
			return nil, authorizer.ToConnectError(err)
		}
	}

	pageLimit := searchLimit(req.Msg.GetSearch(), 50)
	itemsPage, err := s.execRepo.ListPage(ctx, repository.WorkflowExecutionListFilter{
		Status:     executionStatusFilter(req.Msg.GetStatus()),
		InstanceID: req.Msg.GetInstanceId(),
		Query:      searchQuery(req.Msg.GetSearch()),
		IDQuery:    searchIDQuery(req.Msg.GetSearch()),
		Cursor:     searchPage(req.Msg.GetSearch()),
		Limit:      pageLimit,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	respItems := make([]*runtimev1.WorkflowExecution, 0, len(itemsPage.Items))
	for _, item := range itemsPage.Items {
		respItems = append(respItems, workflowExecutionToProto(item, "", false))
	}

	return connect.NewResponse(&runtimev1.ListExecutionsResponse{
		Items:      respItems,
		NextCursor: nextCursorProto(itemsPage.NextCursor, pageLimit),
	}), nil
}

func (s *RuntimeConnectServer) GetExecution(
	ctx context.Context,
	req *connect.Request[runtimev1.GetExecutionRequest],
) (*connect.Response[runtimev1.GetExecutionResponse], error) {
	if err := requireConnectAuth(ctx); err != nil {
		return nil, err
	}

	if s.authz != nil {
		if err := s.authz.CanExecutionView(ctx); err != nil {
			return nil, authorizer.ToConnectError(err)
		}
	}

	if req.Msg.GetExecutionId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("execution_id is required"))
	}

	exec, err := s.execRepo.GetByID(ctx, req.Msg.GetExecutionId())
	if err != nil {
		return nil, connectLookupError(err, "execution not found")
	}

	var outputPayload string
	if req.Msg.GetIncludeOutput() {
		if output, outputErr := s.outputRepo.GetByExecution(
			ctx,
			req.Msg.GetExecutionId(),
		); outputErr == nil &&
			output != nil {
			outputPayload = output.Payload
		}
	}

	return connect.NewResponse(&runtimev1.GetExecutionResponse{
		Execution: workflowExecutionToProto(exec, outputPayload, req.Msg.GetIncludeOutput()),
	}), nil
}

func (s *RuntimeConnectServer) RetryExecution(
	ctx context.Context,
	req *connect.Request[runtimev1.RetryExecutionRequest],
) (*connect.Response[runtimev1.RetryExecutionResponse], error) {
	if err := requireConnectAuth(ctx); err != nil {
		return nil, err
	}

	if s.authz != nil {
		if err := s.authz.CanExecutionRetry(ctx); err != nil {
			return nil, authorizer.ToConnectError(err)
		}
	}

	if req.Msg.GetExecutionId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("execution_id is required"))
	}

	exec, err := s.execRepo.GetByID(ctx, req.Msg.GetExecutionId())
	if err != nil {
		return nil, connectLookupError(err, "execution not found")
	}

	instance, err := s.instanceRepo.GetByID(ctx, exec.InstanceID)
	if err != nil {
		return nil, connectLookupError(err, "instance not found")
	}

	newExec, err := createRetryExecution(ctx, s.execRepo, s.runtimeRepo, s.auditRepo, exec, instance)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}

	return connect.NewResponse(&runtimev1.RetryExecutionResponse{
		Execution: workflowExecutionToProto(newExec, "", false),
	}), nil
}

func (s *RuntimeConnectServer) ResumeExecution(
	ctx context.Context,
	req *connect.Request[runtimev1.ResumeExecutionRequest],
) (*connect.Response[runtimev1.ResumeExecutionResponse], error) {
	if err := requireConnectAuth(ctx); err != nil {
		return nil, err
	}

	if s.authz != nil {
		if err := s.authz.CanExecutionRetry(ctx); err != nil {
			return nil, authorizer.ToConnectError(err)
		}
	}

	if req.Msg.GetExecutionId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("execution_id is required"))
	}

	exec, err := s.execRepo.GetByID(ctx, req.Msg.GetExecutionId())
	if err != nil {
		return nil, connectLookupError(err, "execution not found")
	}

	switch exec.Status {
	case "waiting":
		payloadBytes, payloadErr := rawJSONFromStruct(req.Msg.GetPayload())
		if payloadErr != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, payloadErr)
		}
		if payloadBytes == nil {
			payloadBytes = json.RawMessage(`{}`)
		}

		if err := s.engine.ResumeWaitingExecution(ctx, exec.ID, payloadBytes); err != nil {
			return nil, connectErrorForBusiness(err)
		}

		updatedExec, err := s.execRepo.GetByID(ctx, exec.ID)
		if err != nil {
			return nil, connectLookupError(err, "execution not found")
		}

		return connect.NewResponse(&runtimev1.ResumeExecutionResponse{
			Execution: workflowExecutionToProto(updatedExec, "", false),
			Action:    "resumed_waiting_execution",
		}), nil
	default:
		instance, err := s.instanceRepo.GetByID(ctx, exec.InstanceID)
		if err != nil {
			return nil, connectLookupError(err, "instance not found")
		}

		newExec, retryErr := createRetryExecution(ctx, s.execRepo, s.runtimeRepo, s.auditRepo, exec, instance)
		if retryErr != nil {
			return nil, connect.NewError(connect.CodeFailedPrecondition, retryErr)
		}

		return connect.NewResponse(&runtimev1.ResumeExecutionResponse{
			Execution: workflowExecutionToProto(newExec, "", false),
			Action:    "created_retry_execution",
		}), nil
	}
}

func (s *RuntimeConnectServer) GetInstanceRun(
	ctx context.Context,
	req *connect.Request[runtimev1.GetInstanceRunRequest],
) (*connect.Response[runtimev1.GetInstanceRunResponse], error) {
	if err := requireConnectAuth(ctx); err != nil {
		return nil, err
	}

	if s.authz != nil {
		if err := s.authz.CanInstanceView(ctx); err != nil {
			return nil, authorizer.ToConnectError(err)
		}
	}

	if req.Msg.GetInstanceId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("instance_id is required"))
	}

	includePayloads := req.Msg.GetIncludePayloads()
	instance, err := s.instanceRepo.GetByID(ctx, req.Msg.GetInstanceId())
	if err != nil {
		return nil, connectLookupError(err, "instance not found")
	}

	latestExec, err := s.execRepo.GetLatestByInstance(ctx, req.Msg.GetInstanceId())
	if err != nil {
		return nil, connectLookupError(err, "latest execution not found")
	}

	execs, err := s.execRepo.List(ctx, "", req.Msg.GetInstanceId(), int(req.Msg.GetExecutionLimit()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("trustage: %w", err))
	}
	timeline, err := s.auditRepo.ListByInstanceWithLimit(ctx, req.Msg.GetInstanceId(), int(req.Msg.GetTimelineLimit()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("trustage: %w", err))
	}
	outputs, err := s.outputRepo.ListByInstance(ctx, req.Msg.GetInstanceId(), int(req.Msg.GetExecutionLimit()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("trustage: %w", err))
	}
	scopeRuns, err := s.scopeRepo.ListByInstance(ctx, req.Msg.GetInstanceId(), int(req.Msg.GetExecutionLimit()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("trustage: %w", err))
	}
	signalWaits, err := s.signalWaitRepo.ListByInstance(ctx, req.Msg.GetInstanceId(), int(req.Msg.GetExecutionLimit()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("trustage: %w", err))
	}
	signalMessages, err := s.signalMsgRepo.ListByInstance(
		ctx,
		req.Msg.GetInstanceId(),
		int(req.Msg.GetExecutionLimit()),
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("trustage: %w", err))
	}

	execItems := make([]*runtimev1.WorkflowExecution, 0, len(execs))
	for _, item := range execs {
		execItems = append(execItems, workflowExecutionToProto(item, "", false))
	}
	timelineItems := make([]*runtimev1.RunTimelineEntry, 0, len(timeline))
	for _, item := range timeline {
		timelineItems = append(timelineItems, runtimeTimelineEntryToProto(item, includePayloads))
	}
	outputItems := make([]*runtimev1.StateOutput, 0, len(outputs))
	for _, item := range outputs {
		outputItems = append(outputItems, stateOutputToProto(item, includePayloads))
	}
	scopeItems := make([]*runtimev1.ScopeRun, 0, len(scopeRuns))
	for _, item := range scopeRuns {
		scopeItems = append(scopeItems, scopeRunToProto(item, includePayloads))
	}
	waitItems := make([]*runtimev1.SignalWait, 0, len(signalWaits))
	for _, item := range signalWaits {
		waitItems = append(waitItems, signalWaitToProto(item))
	}
	messageItems := make([]*runtimev1.SignalMessage, 0, len(signalMessages))
	for _, item := range signalMessages {
		messageItems = append(messageItems, signalMessageToProto(item, includePayloads))
	}

	return connect.NewResponse(&runtimev1.GetInstanceRunResponse{
		Instance:        workflowInstanceToProto(instance),
		LatestExecution: workflowExecutionToProto(latestExec, "", false),
		TraceId:         latestExec.TraceID,
		ResumeStrategy:  resumeStrategyForExecution(latestExec),
		Executions:      execItems,
		Timeline:        timelineItems,
		Outputs:         outputItems,
		ScopeRuns:       scopeItems,
		SignalWaits:     waitItems,
		SignalMessages:  messageItems,
	}), nil
}

func resumeStrategyForExecution(exec *models.WorkflowStateExecution) string {
	if exec == nil {
		return ""
	}

	switch exec.Status {
	case "waiting":
		return "resume_waiting_execution"
	case "failed", "fatal", "timed_out", "invalid_input_contract", "invalid_output_contract", "retry_scheduled":
		return "retry_execution"
	default:
		return "none"
	}
}
