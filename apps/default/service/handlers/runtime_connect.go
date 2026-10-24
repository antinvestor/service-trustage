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
	"fmt"

	"connectrpc.com/connect/v2"

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
	req *runtimev1.ListInstancesRequest,
) (*runtimev1.ListInstancesResponse, error) {
	if err := requireConnectAuth(ctx); err != nil {
		return nil, err
	}

	pageLimit := searchLimit(req.GetSearch(), defaultRuntimePageLimit)
	itemsPage, err := s.instanceRepo.ListPage(ctx, repository.WorkflowInstanceListFilter{
		Status:            instanceStatusFilter(req.GetStatus()),
		WorkflowName:      req.GetWorkflowName(),
		Query:             searchQuery(req.GetSearch()),
		IDQuery:           searchIDQuery(req.GetSearch()),
		ParentInstanceID:  searchExtraString(req.GetSearch(), "parent_instance_id"),
		ParentExecutionID: searchExtraString(req.GetSearch(), "parent_execution_id"),
		Cursor:            searchPage(req.GetSearch()),
		Limit:             pageLimit,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, "failed to list instances").WithCause(err)
	}

	respItems := make([]*runtimev1.WorkflowInstance, 0, len(itemsPage.Items))
	for _, item := range itemsPage.Items {
		respItems = append(respItems, workflowInstanceToProto(item))
	}

	return &runtimev1.ListInstancesResponse{
		Items:      respItems,
		NextCursor: nextCursorProto(itemsPage.NextCursor, pageLimit),
	}, nil
}

func (s *RuntimeConnectServer) RetryInstance(
	ctx context.Context,
	req *runtimev1.RetryInstanceRequest,
) (*runtimev1.RetryInstanceResponse, error) {
	if err := requireConnectAuth(ctx); err != nil {
		return nil, err
	}

	if req.GetInstanceId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, "instance_id is required")
	}

	instance, err := s.instanceRepo.GetByID(ctx, req.GetInstanceId())
	if err != nil {
		return nil, connectLookupError(err, "instance not found")
	}

	exec, err := s.execRepo.GetLatestByInstance(ctx, req.GetInstanceId())
	if err != nil {
		return nil, connectLookupError(err, "latest execution not found")
	}

	newExec, err := createRetryExecution(ctx, s.runtimeRepo, s.auditRepo, exec, instance)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err.Error()).WithCause(err)
	}

	return &runtimev1.RetryInstanceResponse{
		Execution: workflowExecutionToProto(newExec, "", false),
	}, nil
}

func (s *RuntimeConnectServer) ListExecutions(
	ctx context.Context,
	req *runtimev1.ListExecutionsRequest,
) (*runtimev1.ListExecutionsResponse, error) {
	if err := requireConnectAuth(ctx); err != nil {
		return nil, err
	}

	pageLimit := searchLimit(req.GetSearch(), defaultRuntimePageLimit)
	itemsPage, err := s.execRepo.ListPage(ctx, repository.WorkflowExecutionListFilter{
		Status:     executionStatusFilter(req.GetStatus()),
		InstanceID: req.GetInstanceId(),
		Query:      searchQuery(req.GetSearch()),
		IDQuery:    searchIDQuery(req.GetSearch()),
		Cursor:     searchPage(req.GetSearch()),
		Limit:      pageLimit,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err.Error()).WithCause(err)
	}

	respItems := make([]*runtimev1.WorkflowExecution, 0, len(itemsPage.Items))
	for _, item := range itemsPage.Items {
		respItems = append(respItems, workflowExecutionToProto(item, "", false))
	}

	return &runtimev1.ListExecutionsResponse{
		Items:      respItems,
		NextCursor: nextCursorProto(itemsPage.NextCursor, pageLimit),
	}, nil
}

func (s *RuntimeConnectServer) GetExecution(
	ctx context.Context,
	req *runtimev1.GetExecutionRequest,
) (*runtimev1.GetExecutionResponse, error) {
	if err := requireConnectAuth(ctx); err != nil {
		return nil, err
	}

	if req.GetExecutionId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, "execution_id is required")
	}

	exec, err := s.execRepo.GetByID(ctx, req.GetExecutionId())
	if err != nil {
		return nil, connectLookupError(err, "execution not found")
	}

	var outputPayload string
	if req.GetIncludeOutput() {
		if output, outputErr := s.outputRepo.GetByExecution(
			ctx,
			req.GetExecutionId(),
		); outputErr == nil &&
			output != nil {
			outputPayload = output.Payload
		}
	}

	return &runtimev1.GetExecutionResponse{
		Execution: workflowExecutionToProto(exec, outputPayload, req.GetIncludeOutput()),
	}, nil
}

func (s *RuntimeConnectServer) RetryExecution(
	ctx context.Context,
	req *runtimev1.RetryExecutionRequest,
) (*runtimev1.RetryExecutionResponse, error) {
	if err := requireConnectAuth(ctx); err != nil {
		return nil, err
	}

	if req.GetExecutionId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, "execution_id is required")
	}

	exec, err := s.execRepo.GetByID(ctx, req.GetExecutionId())
	if err != nil {
		return nil, connectLookupError(err, "execution not found")
	}

	instance, err := s.instanceRepo.GetByID(ctx, exec.InstanceID)
	if err != nil {
		return nil, connectLookupError(err, "instance not found")
	}

	newExec, err := createRetryExecution(ctx, s.runtimeRepo, s.auditRepo, exec, instance)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err.Error()).WithCause(err)
	}

	return &runtimev1.RetryExecutionResponse{
		Execution: workflowExecutionToProto(newExec, "", false),
	}, nil
}

func (s *RuntimeConnectServer) ResumeExecution(
	ctx context.Context,
	req *runtimev1.ResumeExecutionRequest,
) (*runtimev1.ResumeExecutionResponse, error) {
	if err := requireConnectAuth(ctx); err != nil {
		return nil, err
	}

	if req.GetExecutionId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, "execution_id is required")
	}

	exec, err := s.execRepo.GetByID(ctx, req.GetExecutionId())
	if err != nil {
		return nil, connectLookupError(err, "execution not found")
	}

	switch exec.Status {
	case models.ExecStatusWaiting:
		payloadBytes, payloadErr := rawJSONFromStruct(req.GetPayload())
		if payloadErr != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, payloadErr.Error()).WithCause(payloadErr)
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

		return &runtimev1.ResumeExecutionResponse{
			Execution: workflowExecutionToProto(updatedExec, "", false),
			Action:    "resumed_waiting_execution",
		}, nil
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
			return nil, connect.NewError(connect.CodeFailedPrecondition, retryErr.Error()).WithCause(retryErr)
		}

		return &runtimev1.ResumeExecutionResponse{
			Execution: workflowExecutionToProto(newExec, "", false),
			Action:    "created_retry_execution",
		}, nil
	default:
		return nil, connect.Errorf(
			connect.CodeFailedPrecondition,
			"unsupported execution status %s", exec.Status,
		)
	}
}

func (s *RuntimeConnectServer) GetInstanceRun(
	ctx context.Context,
	req *runtimev1.GetInstanceRunRequest,
) (*runtimev1.GetInstanceRunResponse, error) {
	if err := s.authorizeInstanceView(ctx); err != nil {
		return nil, err
	}
	if req.GetInstanceId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, "instance_id is required")
	}

	runData, err := s.loadInstanceRunData(ctx, req)
	if err != nil {
		return nil, err
	}

	return &runtimev1.GetInstanceRunResponse{
		Instance:        workflowInstanceToProto(runData.instance),
		LatestExecution: workflowExecutionToProto(runData.latestExec, "", false),
		TraceId:         runData.latestExec.TraceID,
		ResumeStrategy:  resumeStrategyForExecution(runData.latestExec),
		Executions:      runData.executionProtos(req.GetIncludePayloads()),
		Timeline:        runData.timelineProtos(req.GetIncludePayloads()),
		Outputs:         runData.outputProtos(req.GetIncludePayloads()),
		ScopeRuns:       runData.scopeProtos(req.GetIncludePayloads()),
		SignalWaits:     runData.waitProtos(),
		SignalMessages:  runData.messageProtos(req.GetIncludePayloads()),
	}, nil
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
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("trustage: %w", err).Error()).WithCause(err)
	}
	timeline, err := s.auditRepo.ListByInstanceWithLimit(ctx, msg.GetInstanceId(), int(msg.GetTimelineLimit()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("trustage: %w", err).Error()).WithCause(err)
	}
	outputs, err := s.outputRepo.ListByInstance(ctx, msg.GetInstanceId(), executionLimit)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("trustage: %w", err).Error()).WithCause(err)
	}
	scopeRuns, err := s.scopeRepo.ListByInstance(ctx, msg.GetInstanceId(), executionLimit)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("trustage: %w", err).Error()).WithCause(err)
	}
	signalWaits, err := s.signalWaitRepo.ListByInstance(ctx, msg.GetInstanceId(), executionLimit)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("trustage: %w", err).Error()).WithCause(err)
	}
	signalMessages, err := s.signalMsgRepo.ListByInstance(ctx, msg.GetInstanceId(), executionLimit)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("trustage: %w", err).Error()).WithCause(err)
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
