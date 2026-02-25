package handlers

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
	"github.com/antinvestor/service-trustage/apps/default/service/repository"
	"github.com/antinvestor/service-trustage/pkg/cryptoutil"
	"github.com/antinvestor/service-trustage/pkg/events"
)

var retryableStatuses = map[models.ExecutionStatus]bool{
	models.ExecStatusFailed:                true,
	models.ExecStatusFatal:                 true,
	models.ExecStatusTimedOut:              true,
	models.ExecStatusInvalidInputContract:  true,
	models.ExecStatusInvalidOutputContract: true,
	models.ExecStatusRetryScheduled:        true,
}

func createRetryExecution(
	ctx context.Context,
	execRepo repository.WorkflowExecutionRepository,
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

	db := execRepo.Pool().DB(ctx, false)

	txErr := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(newExec).Error; err != nil {
			return fmt.Errorf("create retry execution: %w", err)
		}

		now := time.Now()
		updateErr := tx.Exec(
			`UPDATE workflow_instances
			 SET status = ?, current_state = ?, modified_at = ?, revision = revision + 1
			 WHERE id = ? AND deleted_at IS NULL`,
			string(models.InstanceStatusRunning), exec.State, now, instance.ID,
		).Error
		if updateErr != nil {
			return fmt.Errorf("update instance state: %w", updateErr)
		}

		return nil
	})
	if txErr != nil {
		return nil, txErr
	}

	if auditRepo != nil {
		_ = auditRepo.Append(ctx, &models.WorkflowAuditEvent{
			InstanceID:  exec.InstanceID,
			ExecutionID: newExec.ID,
			EventType:   events.EventStateRetried,
			State:       exec.State,
		})
	}

	return newExec, nil
}
