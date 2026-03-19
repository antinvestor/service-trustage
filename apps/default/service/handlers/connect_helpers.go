package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/pitabwire/frame/security"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"

	"github.com/antinvestor/service-trustage/apps/default/service/business"
	"github.com/antinvestor/service-trustage/apps/default/service/models"
	eventv1 "github.com/antinvestor/service-trustage/gen/go/event/v1"
	runtimev1 "github.com/antinvestor/service-trustage/gen/go/runtime/v1"
	workflowv1 "github.com/antinvestor/service-trustage/gen/go/workflow/v1"
)

var errDuplicateRecord = errors.New("duplicate record")

func requireConnectAuth(ctx context.Context) error {
	if security.ClaimsFromContext(ctx) == nil {
		return connect.NewError(connect.CodeUnauthenticated, ErrMissingAuth)
	}

	return nil
}

func connectErrorForBusiness(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, business.ErrWorkflowNotFound),
		errors.Is(err, business.ErrInstanceNotFound),
		errors.Is(err, business.ErrExecutionNotFound),
		errors.Is(err, business.ErrSchemaNotFound),
		errors.Is(err, business.ErrTriggerNotFound),
		errors.Is(err, gorm.ErrRecordNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, business.ErrInputContractViolation),
		errors.Is(err, business.ErrOutputContractViolation),
		errors.Is(err, business.ErrDSLValidationFailed),
		errors.Is(err, business.ErrInvalidWorkflowStatus):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, business.ErrStaleExecution),
		errors.Is(err, business.ErrInvalidToken):
		return connect.NewError(connect.CodeAborted, err)
	case errors.Is(err, business.ErrWorkflowAlreadyActive):
		return connect.NewError(connect.CodeAlreadyExists, err)
	default:
		return connect.NewError(connect.CodeInternal, fmt.Errorf("trustage: %w", err))
	}
}

func connectLookupError(err error, notFoundMessage string) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, gorm.ErrRecordNotFound):
		return connect.NewError(connect.CodeNotFound, errors.New(notFoundMessage))
	default:
		return connect.NewError(connect.CodeInternal, fmt.Errorf("trustage: %w", err))
	}
}

func isDuplicateRecordError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, gorm.ErrDuplicatedKey) || errors.Is(err, errDuplicateRecord) {
		return true
	}

	return strings.Contains(strings.ToLower(err.Error()), "duplicate")
}

func rawJSONFromStruct(value *structpb.Struct) (json.RawMessage, error) {
	if value == nil {
		return nil, nil
	}

	raw, err := protojson.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal struct payload: %w", err)
	}

	return raw, nil
}

func structFromJSONString(raw string) (*structpb.Struct, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	out := &structpb.Struct{}
	if err := protojson.Unmarshal([]byte(raw), out); err != nil {
		return nil, err
	}

	return out, nil
}

func lossyStructFromJSONString(raw string) *structpb.Struct {
	value, err := structFromJSONString(raw)
	if err == nil {
		return value
	}

	fallback, fallbackErr := structpb.NewStruct(map[string]any{"raw_json": raw})
	if fallbackErr != nil {
		return nil
	}

	return fallback
}

func structFromMap(payload map[string]any) *structpb.Struct {
	if payload == nil {
		payload = map[string]any{}
	}

	value, err := structpb.NewStruct(payload)
	if err != nil {
		return nil
	}

	return value
}

func timestampFromPtr(value *time.Time) *timestamppb.Timestamp {
	if value == nil {
		return nil
	}

	return timestamppb.New(*value)
}

func timestampFromValue(value time.Time) *timestamppb.Timestamp {
	if value.IsZero() {
		return nil
	}

	return timestamppb.New(value)
}

func workflowStatusToProto(status models.WorkflowDefinitionStatus) workflowv1.WorkflowStatus {
	switch status {
	case models.WorkflowStatusDraft:
		return workflowv1.WorkflowStatus_WORKFLOW_STATUS_DRAFT
	case models.WorkflowStatusActive:
		return workflowv1.WorkflowStatus_WORKFLOW_STATUS_ACTIVE
	case models.WorkflowStatusArchived:
		return workflowv1.WorkflowStatus_WORKFLOW_STATUS_ARCHIVED
	default:
		return workflowv1.WorkflowStatus_WORKFLOW_STATUS_UNSPECIFIED
	}
}

func workflowStatusFilter(status workflowv1.WorkflowStatus) (string, error) {
	switch status {
	case workflowv1.WorkflowStatus_WORKFLOW_STATUS_UNSPECIFIED:
		return "", nil
	case workflowv1.WorkflowStatus_WORKFLOW_STATUS_ACTIVE:
		return string(models.WorkflowStatusActive), nil
	default:
		return "", connect.NewError(
			connect.CodeInvalidArgument,
			errors.New("only active workflow filtering is supported"),
		)
	}
}

func instanceStatusToProto(status models.WorkflowInstanceStatus) runtimev1.InstanceStatus {
	switch status {
	case models.InstanceStatusRunning:
		return runtimev1.InstanceStatus_INSTANCE_STATUS_RUNNING
	case models.InstanceStatusCompleted:
		return runtimev1.InstanceStatus_INSTANCE_STATUS_COMPLETED
	case models.InstanceStatusFailed:
		return runtimev1.InstanceStatus_INSTANCE_STATUS_FAILED
	case models.InstanceStatusCancelled:
		return runtimev1.InstanceStatus_INSTANCE_STATUS_CANCELLED
	case models.InstanceStatusSuspended:
		return runtimev1.InstanceStatus_INSTANCE_STATUS_SUSPENDED
	default:
		return runtimev1.InstanceStatus_INSTANCE_STATUS_UNSPECIFIED
	}
}

func instanceStatusFilter(status runtimev1.InstanceStatus) string {
	switch status {
	case runtimev1.InstanceStatus_INSTANCE_STATUS_RUNNING:
		return string(models.InstanceStatusRunning)
	case runtimev1.InstanceStatus_INSTANCE_STATUS_COMPLETED:
		return string(models.InstanceStatusCompleted)
	case runtimev1.InstanceStatus_INSTANCE_STATUS_FAILED:
		return string(models.InstanceStatusFailed)
	case runtimev1.InstanceStatus_INSTANCE_STATUS_CANCELLED:
		return string(models.InstanceStatusCancelled)
	case runtimev1.InstanceStatus_INSTANCE_STATUS_SUSPENDED:
		return string(models.InstanceStatusSuspended)
	default:
		return ""
	}
}

func executionStatusToProto(status models.ExecutionStatus) runtimev1.ExecutionStatus {
	switch status {
	case models.ExecStatusPending:
		return runtimev1.ExecutionStatus_EXECUTION_STATUS_PENDING
	case models.ExecStatusDispatched:
		return runtimev1.ExecutionStatus_EXECUTION_STATUS_DISPATCHED
	case models.ExecStatusRunning:
		return runtimev1.ExecutionStatus_EXECUTION_STATUS_RUNNING
	case models.ExecStatusCompleted:
		return runtimev1.ExecutionStatus_EXECUTION_STATUS_COMPLETED
	case models.ExecStatusFailed:
		return runtimev1.ExecutionStatus_EXECUTION_STATUS_FAILED
	case models.ExecStatusFatal:
		return runtimev1.ExecutionStatus_EXECUTION_STATUS_FATAL
	case models.ExecStatusTimedOut:
		return runtimev1.ExecutionStatus_EXECUTION_STATUS_TIMED_OUT
	case models.ExecStatusInvalidInputContract:
		return runtimev1.ExecutionStatus_EXECUTION_STATUS_INVALID_INPUT_CONTRACT
	case models.ExecStatusInvalidOutputContract:
		return runtimev1.ExecutionStatus_EXECUTION_STATUS_INVALID_OUTPUT_CONTRACT
	case models.ExecStatusStale:
		return runtimev1.ExecutionStatus_EXECUTION_STATUS_STALE
	case models.ExecStatusRetryScheduled:
		return runtimev1.ExecutionStatus_EXECUTION_STATUS_RETRY_SCHEDULED
	case models.ExecStatusWaiting:
		return runtimev1.ExecutionStatus_EXECUTION_STATUS_WAITING
	default:
		return runtimev1.ExecutionStatus_EXECUTION_STATUS_UNSPECIFIED
	}
}

func executionStatusFilter(status runtimev1.ExecutionStatus) string {
	switch status {
	case runtimev1.ExecutionStatus_EXECUTION_STATUS_PENDING:
		return string(models.ExecStatusPending)
	case runtimev1.ExecutionStatus_EXECUTION_STATUS_DISPATCHED:
		return string(models.ExecStatusDispatched)
	case runtimev1.ExecutionStatus_EXECUTION_STATUS_RUNNING:
		return string(models.ExecStatusRunning)
	case runtimev1.ExecutionStatus_EXECUTION_STATUS_COMPLETED:
		return string(models.ExecStatusCompleted)
	case runtimev1.ExecutionStatus_EXECUTION_STATUS_FAILED:
		return string(models.ExecStatusFailed)
	case runtimev1.ExecutionStatus_EXECUTION_STATUS_FATAL:
		return string(models.ExecStatusFatal)
	case runtimev1.ExecutionStatus_EXECUTION_STATUS_TIMED_OUT:
		return string(models.ExecStatusTimedOut)
	case runtimev1.ExecutionStatus_EXECUTION_STATUS_INVALID_INPUT_CONTRACT:
		return string(models.ExecStatusInvalidInputContract)
	case runtimev1.ExecutionStatus_EXECUTION_STATUS_INVALID_OUTPUT_CONTRACT:
		return string(models.ExecStatusInvalidOutputContract)
	case runtimev1.ExecutionStatus_EXECUTION_STATUS_STALE:
		return string(models.ExecStatusStale)
	case runtimev1.ExecutionStatus_EXECUTION_STATUS_RETRY_SCHEDULED:
		return string(models.ExecStatusRetryScheduled)
	case runtimev1.ExecutionStatus_EXECUTION_STATUS_WAITING:
		return string(models.ExecStatusWaiting)
	default:
		return ""
	}
}

func workflowDefinitionToProto(def *models.WorkflowDefinition) *workflowv1.WorkflowDefinition {
	if def == nil {
		return nil
	}

	return &workflowv1.WorkflowDefinition{
		Id:              def.ID,
		Name:            def.Name,
		Version:         int32(def.WorkflowVersion),
		Status:          workflowStatusToProto(def.Status),
		Dsl:             lossyStructFromJSONString(def.DSLBlob),
		InputSchemaHash: def.InputSchemaHash,
		TimeoutSeconds:  def.TimeoutSeconds,
		CreatedAt:       timestampFromValue(def.CreatedAt),
		UpdatedAt:       timestampFromValue(def.ModifiedAt),
	}
}

func eventRecordToProto(
	eventID, eventType, source, idempotencyKey string,
	payload map[string]any,
) *eventv1.EventRecord {
	return &eventv1.EventRecord{
		EventId:        eventID,
		EventType:      eventType,
		Source:         source,
		IdempotencyKey: idempotencyKey,
		Payload:        structFromMap(payload),
	}
}

func timelineEntryToProto(event *models.WorkflowAuditEvent) *eventv1.TimelineEntry {
	if event == nil {
		return nil
	}

	return &eventv1.TimelineEntry{
		EventType:   event.EventType,
		State:       event.State,
		FromState:   event.FromState,
		ToState:     event.ToState,
		ExecutionId: event.ExecutionID,
		TraceId:     event.TraceID,
		Payload:     lossyStructFromJSONString(event.Payload),
		CreatedAt:   timestampFromValue(event.CreatedAt),
	}
}

func workflowInstanceToProto(instance *models.WorkflowInstance) *runtimev1.WorkflowInstance {
	if instance == nil {
		return nil
	}

	return &runtimev1.WorkflowInstance{
		Id:                instance.ID,
		WorkflowName:      instance.WorkflowName,
		WorkflowVersion:   int32(instance.WorkflowVersion),
		CurrentState:      instance.CurrentState,
		Status:            instanceStatusToProto(instance.Status),
		Revision:          instance.Revision,
		TriggerEventId:    instance.TriggerEventID,
		Metadata:          lossyStructFromJSONString(instance.Metadata),
		StartedAt:         timestampFromPtr(instance.StartedAt),
		FinishedAt:        timestampFromPtr(instance.FinishedAt),
		CreatedAt:         timestampFromValue(instance.CreatedAt),
		UpdatedAt:         timestampFromValue(instance.ModifiedAt),
		ParentInstanceId:  instance.ParentInstanceID,
		ParentExecutionId: instance.ParentExecutionID,
		ScopeType:         instance.ScopeType,
		ScopeParentState:  instance.ScopeParentState,
		ScopeEntryState:   instance.ScopeEntryState,
		ScopeIndex:        int32(instance.ScopeIndex),
	}
}

func workflowExecutionToProto(
	exec *models.WorkflowStateExecution,
	outputPayload string,
	includeOutput bool,
) *runtimev1.WorkflowExecution {
	if exec == nil {
		return nil
	}

	item := &runtimev1.WorkflowExecution{
		Id:               exec.ID,
		InstanceId:       exec.InstanceID,
		State:            exec.State,
		StateVersion:     int32(exec.StateVersion),
		Attempt:          int32(exec.Attempt),
		Status:           executionStatusToProto(exec.Status),
		ErrorClass:       exec.ErrorClass,
		ErrorMessage:     exec.ErrorMessage,
		NextRetryAt:      timestampFromPtr(exec.NextRetryAt),
		StartedAt:        timestampFromPtr(exec.StartedAt),
		FinishedAt:       timestampFromPtr(exec.FinishedAt),
		CreatedAt:        timestampFromValue(exec.CreatedAt),
		UpdatedAt:        timestampFromValue(exec.ModifiedAt),
		TraceId:          exec.TraceID,
		InputSchemaHash:  exec.InputSchemaHash,
		OutputSchemaHash: exec.OutputSchemaHash,
		InputPayload:     lossyStructFromJSONString(exec.InputPayload),
	}

	if includeOutput {
		item.Output = lossyStructFromJSONString(outputPayload)
	}

	return item
}

func runtimeTimelineEntryToProto(event *models.WorkflowAuditEvent, includePayloads bool) *runtimev1.RunTimelineEntry {
	if event == nil {
		return nil
	}

	entry := &runtimev1.RunTimelineEntry{
		EventType:   event.EventType,
		State:       event.State,
		FromState:   event.FromState,
		ToState:     event.ToState,
		ExecutionId: event.ExecutionID,
		TraceId:     event.TraceID,
		CreatedAt:   timestampFromValue(event.CreatedAt),
	}
	if includePayloads {
		entry.Payload = lossyStructFromJSONString(event.Payload)
	}

	return entry
}

func stateOutputToProto(output *models.WorkflowStateOutput, includePayloads bool) *runtimev1.StateOutput {
	if output == nil {
		return nil
	}

	item := &runtimev1.StateOutput{
		ExecutionId: output.ExecutionID,
		State:       output.State,
		SchemaHash:  output.SchemaHash,
		CreatedAt:   timestampFromValue(output.CreatedAt),
	}
	if includePayloads {
		item.Payload = lossyStructFromJSONString(output.Payload)
	}

	return item
}

func scopeRunToProto(scope *models.WorkflowScopeRun, includePayloads bool) *runtimev1.ScopeRun {
	if scope == nil {
		return nil
	}

	item := &runtimev1.ScopeRun{
		Id:                scope.ID,
		ParentExecutionId: scope.ParentExecutionID,
		ParentState:       scope.ParentState,
		ScopeType:         scope.ScopeType,
		Status:            scope.Status,
		WaitAll:           scope.WaitAll,
		TotalChildren:     int32(scope.TotalChildren),
		CompletedChildren: int32(scope.CompletedChildren),
		FailedChildren:    int32(scope.FailedChildren),
		NextChildIndex:    int32(scope.NextChildIndex),
		MaxConcurrency:    int32(scope.MaxConcurrency),
		ItemVar:           scope.ItemVar,
		IndexVar:          scope.IndexVar,
		CreatedAt:         timestampFromValue(scope.CreatedAt),
		UpdatedAt:         timestampFromValue(scope.ModifiedAt),
	}
	if includePayloads {
		item.ItemsPayload = lossyStructFromJSONString(scope.ItemsPayload)
		item.ResultsPayload = lossyStructFromJSONString(scope.ResultsPayload)
	}

	return item
}

func signalWaitToProto(wait *models.WorkflowSignalWait) *runtimev1.SignalWait {
	if wait == nil {
		return nil
	}

	return &runtimev1.SignalWait{
		Id:          wait.ID,
		ExecutionId: wait.ExecutionID,
		State:       wait.State,
		SignalName:  wait.SignalName,
		OutputVar:   wait.OutputVar,
		Status:      wait.Status,
		TimeoutAt:   timestampFromPtr(wait.TimeoutAt),
		MatchedAt:   timestampFromPtr(wait.MatchedAt),
		TimedOutAt:  timestampFromPtr(wait.TimedOutAt),
		MessageId:   wait.MessageID,
		Attempts:    int32(wait.Attempts),
		CreatedAt:   timestampFromValue(wait.CreatedAt),
		UpdatedAt:   timestampFromValue(wait.ModifiedAt),
	}
}

func signalMessageToProto(message *models.WorkflowSignalMessage, includePayloads bool) *runtimev1.SignalMessage {
	if message == nil {
		return nil
	}

	item := &runtimev1.SignalMessage{
		Id:          message.ID,
		SignalName:  message.SignalName,
		Status:      message.Status,
		DeliveredAt: timestampFromPtr(message.DeliveredAt),
		WaitId:      message.WaitID,
		Attempts:    int32(message.Attempts),
		CreatedAt:   timestampFromValue(message.CreatedAt),
		UpdatedAt:   timestampFromValue(message.ModifiedAt),
	}
	if includePayloads {
		item.Payload = lossyStructFromJSONString(message.Payload)
	}

	return item
}
