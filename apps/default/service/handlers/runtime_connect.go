// Copyright 2023-2026 Ant Investor Ltd
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"connectrpc.com/connect"

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

	runtimev1connect.UnimplementedRuntimeServiceHandler
}

const defaultRuntimePageLimit = 50

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
	}
}

func (s *RuntimeConnectServer) ListInstances(
	ctx context.Context,
	req *connect.Request[runtimev1.ListInstancesRequest],
) (*connect.Response[runtimev1.ListInstancesResponse], error) {
	if err := requireConnectAuth(ctx); err != nil {
		return nil, err
	}

	pageLimit := searchLimit(req.Msg.GetSearch(), defaultRuntimePageLimit)
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

	newExec, err := createRetryExecution(ctx, s.runtimeRepo, s.auditRepo, exec, instance)
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

	pageLimit := searchLimit(req.Msg.GetSearch(), defaultRuntimePageLimit)
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

	newExec, err := createRetryExecution(ctx, s.runtimeRepo, s.auditRepo, exec, instance)
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

	if req.Msg.GetExecutionId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("execution_id is required"))
	}

	exec, err := s.execRepo.GetByID(ctx, req.Msg.GetExecutionId())
	if err != nil {
		return nil, connectLookupError(err, "execution not found")
	}

	switch exec.Status {
	case models.ExecStatusWaiting:
		payloadBytes, payloadErr := rawJSONFromStruct(req.Msg.GetPayload())
		if payloadErr != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, payloadErr)
		}
		if payloadBytes == nil {
			payloadBytes = json.RawMessage(`{}`)
		}

		if resumeErr := s.engine.ResumeWaitingExecution(ctx, exec.ID, payloadBytes); resumeErr != nil {
			return nil, connectErrorForBusiness(resumeErr)
		}

		updatedExec, getErr := s.execRepo.GetByID(ctx, exec.ID)
		if getErr != nil {
			return nil, connectLookupError(getErr, "execution not found")
		}

		return connect.NewResponse(&runtimev1.ResumeExecutionResponse{
			Execution: workflowExecutionToProto(updatedExec, "", false),
			Action:    "resumed_waiting_execution",
		}), nil
	case models.ExecStatusPending,
		models.ExecStatusDispatched,
		models.ExecStatusRunning,
		models.ExecStatusCompleted,
		models.ExecStatusFailed,
		models.ExecStatusFatal,
		models.ExecStatusTimedOut,
		models.ExecStatusInvalidInputContract,
		models.ExecStatusInvalidOutputContract,
		models.ExecStatusStale,
		models.ExecStatusRetryScheduled:
		instance, getErr := s.instanceRepo.GetByID(ctx, exec.InstanceID)
		if getErr != nil {
			return nil, connectLookupError(getErr, "instance not found")
		}

		newExec, retryErr := createRetryExecution(ctx, s.runtimeRepo, s.auditRepo, exec, instance)
		if retryErr != nil {
			return nil, connect.NewError(connect.CodeFailedPrecondition, retryErr)
		}

		return connect.NewResponse(&runtimev1.ResumeExecutionResponse{
			Execution: workflowExecutionToProto(newExec, "", false),
			Action:    "created_retry_execution",
		}), nil
	default:
		return nil, connect.NewError(
			connect.CodeFailedPrecondition,
			fmt.Errorf("unsupported execution status %s", exec.Status),
		)
	}
}

func (s *RuntimeConnectServer) GetInstanceRun(
	ctx context.Context,
	req *connect.Request[runtimev1.GetInstanceRunRequest],
) (*connect.Response[runtimev1.GetInstanceRunResponse], error) {
	if err := s.authorizeInstanceView(ctx); err != nil {
		return nil, err
	}
	if req.Msg.GetInstanceId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("instance_id is required"))
	}

	runData, err := s.loadInstanceRunData(ctx, req.Msg)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&runtimev1.GetInstanceRunResponse{
		Instance:        workflowInstanceToProto(runData.instance),
		LatestExecution: workflowExecutionToProto(runData.latestExec, "", false),
		TraceId:         runData.latestExec.TraceID,
		ResumeStrategy:  resumeStrategyForExecution(runData.latestExec),
		Executions:      runData.executionProtos(req.Msg.GetIncludePayloads()),
		Timeline:        runData.timelineProtos(req.Msg.GetIncludePayloads()),
		Outputs:         runData.outputProtos(req.Msg.GetIncludePayloads()),
		ScopeRuns:       runData.scopeProtos(req.Msg.GetIncludePayloads()),
		SignalWaits:     runData.waitProtos(),
		SignalMessages:  runData.messageProtos(req.Msg.GetIncludePayloads()),
	}), nil
}

func (s *RuntimeConnectServer) authorizeInstanceView(ctx context.Context) error {
	if err := requireConnectAuth(ctx); err != nil {
		return err
	}

	return nil
}

type instanceRunData struct {
	instance       *models.WorkflowInstance
	latestExec     *models.WorkflowStateExecution
	executions     []*models.WorkflowStateExecution
	timeline       []*models.WorkflowAuditEvent
	outputs        []*models.WorkflowStateOutput
	scopeRuns      []*models.WorkflowScopeRun
	signalWaits    []*models.WorkflowSignalWait
	signalMessages []*models.WorkflowSignalMessage
}

func (s *RuntimeConnectServer) loadInstanceRunData(
	ctx context.Context,
	msg *runtimev1.GetInstanceRunRequest,
) (*instanceRunData, error) {
	instance, err := s.instanceRepo.GetByID(ctx, msg.GetInstanceId())
	if err != nil {
		return nil, connectLookupError(err, "instance not found")
	}
	latestExec, err := s.execRepo.GetLatestByInstance(ctx, msg.GetInstanceId())
	if err != nil {
		return nil, connectLookupError(err, "latest execution not found")
	}

	executionLimit := int(msg.GetExecutionLimit())
	executions, err := s.execRepo.List(ctx, "", msg.GetInstanceId(), executionLimit)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("trustage: %w", err))
	}
	timeline, err := s.auditRepo.ListByInstanceWithLimit(ctx, msg.GetInstanceId(), int(msg.GetTimelineLimit()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("trustage: %w", err))
	}
	outputs, err := s.outputRepo.ListByInstance(ctx, msg.GetInstanceId(), executionLimit)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("trustage: %w", err))
	}
	scopeRuns, err := s.scopeRepo.ListByInstance(ctx, msg.GetInstanceId(), executionLimit)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("trustage: %w", err))
	}
	signalWaits, err := s.signalWaitRepo.ListByInstance(ctx, msg.GetInstanceId(), executionLimit)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("trustage: %w", err))
	}
	signalMessages, err := s.signalMsgRepo.ListByInstance(ctx, msg.GetInstanceId(), executionLimit)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("trustage: %w", err))
	}

	return &instanceRunData{
		instance:       instance,
		latestExec:     latestExec,
		executions:     executions,
		timeline:       timeline,
		outputs:        outputs,
		scopeRuns:      scopeRuns,
		signalWaits:    signalWaits,
		signalMessages: signalMessages,
	}, nil
}

func (d *instanceRunData) executionProtos(includePayloads bool) []*runtimev1.WorkflowExecution {
	items := make([]*runtimev1.WorkflowExecution, 0, len(d.executions))
	for _, item := range d.executions {
		items = append(items, workflowExecutionToProto(item, "", includePayloads))
	}

	return items
}

func (d *instanceRunData) timelineProtos(includePayloads bool) []*runtimev1.RunTimelineEntry {
	items := make([]*runtimev1.RunTimelineEntry, 0, len(d.timeline))
	for _, item := range d.timeline {
		items = append(items, runtimeTimelineEntryToProto(item, includePayloads))
	}

	return items
}

func (d *instanceRunData) outputProtos(includePayloads bool) []*runtimev1.StateOutput {
	items := make([]*runtimev1.StateOutput, 0, len(d.outputs))
	for _, item := range d.outputs {
		items = append(items, stateOutputToProto(item, includePayloads))
	}

	return items
}

func (d *instanceRunData) scopeProtos(includePayloads bool) []*runtimev1.ScopeRun {
	items := make([]*runtimev1.ScopeRun, 0, len(d.scopeRuns))
	for _, item := range d.scopeRuns {
		items = append(items, scopeRunToProto(item, includePayloads))
	}

	return items
}

func (d *instanceRunData) waitProtos() []*runtimev1.SignalWait {
	items := make([]*runtimev1.SignalWait, 0, len(d.signalWaits))
	for _, item := range d.signalWaits {
		items = append(items, signalWaitToProto(item))
	}

	return items
}

func (d *instanceRunData) messageProtos(includePayloads bool) []*runtimev1.SignalMessage {
	items := make([]*runtimev1.SignalMessage, 0, len(d.signalMessages))
	for _, item := range d.signalMessages {
		items = append(items, signalMessageToProto(item, includePayloads))
	}

	return items
}

func resumeStrategyForExecution(exec *models.WorkflowStateExecution) string {
	if exec == nil {
		return ""
	}

	switch exec.Status {
	case models.ExecStatusWaiting:
		return "resume_waiting_execution"
	case models.ExecStatusFailed,
		models.ExecStatusFatal,
		models.ExecStatusTimedOut,
		models.ExecStatusInvalidInputContract,
		models.ExecStatusInvalidOutputContract,
		models.ExecStatusRetryScheduled:
		return "retry_execution"
	case models.ExecStatusPending,
		models.ExecStatusDispatched,
		models.ExecStatusRunning,
		models.ExecStatusCompleted,
		models.ExecStatusStale:
		return "none"
	default:
		return "none"
	}
}
