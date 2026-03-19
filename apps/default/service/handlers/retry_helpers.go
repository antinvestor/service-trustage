package handlers

import (
	"context"
	"errors"
	"fmt"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
	"github.com/antinvestor/service-trustage/apps/default/service/repository"
	"github.com/antinvestor/service-trustage/pkg/cryptoutil"
	"github.com/antinvestor/service-trustage/pkg/events"
)

//nolint:gochecknoglobals // execution status map is static and shared across handlers
var retryableStatuses = map[models.ExecutionStatus]bool{
	models.ExecStatusPending:               false,
	models.ExecStatusDispatched:            false,
	models.ExecStatusRunning:               false,
	models.ExecStatusCompleted:             false,
	models.ExecStatusFailed:                true,
	models.ExecStatusFatal:                 true,
	models.ExecStatusTimedOut:              true,
	models.ExecStatusInvalidInputContract:  true,
	models.ExecStatusInvalidOutputContract: true,
	models.ExecStatusStale:                 false,
	models.ExecStatusRetryScheduled:        true,
	models.ExecStatusWaiting:               false,
}

func createRetryExecution(
	ctx context.Context,
	execRepo repository.WorkflowExecutionRepository,
	runtimeRepo repository.WorkflowRuntimeRepository,
	auditRepo repository.AuditEventRepository,
	exec *models.WorkflowStateExecution,
	instance *models.WorkflowInstance,
) (*models.WorkflowStateExecution, error) {
	if !retryableStatuses[exec.Status] {
		return nil, fmt.Errorf("execution not retryable in status %s", exec.Status)
	}

	rawToken, tokenErr := cryptoutil.GenerateToken()
	if tokenErr != nil {
		return nil, fmt.Errorf("generate token: %w", tokenErr)
	}

	newExec := &models.WorkflowStateExecution{
		InstanceID:      exec.InstanceID,
		State:           exec.State,
		StateVersion:    exec.StateVersion,
		Attempt:         exec.Attempt + 1,
		Status:          models.ExecStatusPending,
		ExecutionToken:  cryptoutil.HashToken(rawToken),
		InputSchemaHash: exec.InputSchemaHash,
		InputPayload:    exec.InputPayload,
		TraceID:         exec.TraceID,
	}

	if _, err := runtimeRepo.CreateRetryExecution(ctx, &repository.CreateRetryExecutionRequest{
		Execution:    exec,
		Instance:     instance,
		NewExecution: newExec,
	}); err != nil {
		if errors.Is(err, repository.ErrStaleMutation) {
			return nil, fmt.Errorf("reset instance for retry: %w", err)
		}
		return nil, err
	}
	if auditRepo != nil {
		_ = auditRepo.Append(ctx, &models.WorkflowAuditEvent{
			InstanceID:  exec.InstanceID,
			ExecutionID: newExec.ID,
			EventType:   events.EventStateRetried,
			State:       exec.State,
			TraceID:     exec.TraceID,
		})
	}

	return newExec, nil
}
