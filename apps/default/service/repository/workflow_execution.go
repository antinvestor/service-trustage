package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/pitabwire/frame/datastore"
	"github.com/pitabwire/frame/datastore/pool"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
)

// WorkflowExecutionRepository manages workflow state execution persistence.
type WorkflowExecutionRepository interface {
	Create(ctx context.Context, exec *models.WorkflowStateExecution) error
	GetByID(ctx context.Context, executionID string) (*models.WorkflowStateExecution, error)
	List(ctx context.Context, status, instanceID string, limit int) ([]*models.WorkflowStateExecution, error)
	GetLatestByInstance(ctx context.Context, instanceID string) (*models.WorkflowStateExecution, error)
	FindPending(ctx context.Context, limit int) ([]*models.WorkflowStateExecution, error)
	FindRetryDue(ctx context.Context, limit int) ([]*models.WorkflowStateExecution, error)
	FindTimedOut(ctx context.Context, timeoutSeconds int, limit int) ([]*models.WorkflowStateExecution, error)
	VerifyAndConsumeToken(ctx context.Context, executionID, tokenHash string) (*models.WorkflowStateExecution, error)
	VerifyAndConsumeTokenTx(tx *gorm.DB, executionID, tokenHash string) (*models.WorkflowStateExecution, error)
	UpdateStatus(ctx context.Context, executionID string, status models.ExecutionStatus, fields map[string]any) error
	MarkStale(ctx context.Context, executionID string) error
	Pool() pool.Pool
}

type workflowExecutionRepository struct {
	datastore.BaseRepository[*models.WorkflowStateExecution]
}

// NewWorkflowExecutionRepository creates a new repository for executions.
func NewWorkflowExecutionRepository(dbPool pool.Pool) WorkflowExecutionRepository {
	ctx := context.Background()
	return &workflowExecutionRepository{
		BaseRepository: datastore.NewBaseRepository[*models.WorkflowStateExecution](
			ctx,
			dbPool,
			nil,
			func() *models.WorkflowStateExecution { return &models.WorkflowStateExecution{} },
		),
	}
}

// Pool returns the underlying database pool for transaction support.
func (r *workflowExecutionRepository) Pool() pool.Pool {
	return r.BaseRepository.Pool()
}

func (r *workflowExecutionRepository) Create(ctx context.Context, exec *models.WorkflowStateExecution) error {
	return r.BaseRepository.Create(ctx, exec)
}

func (r *workflowExecutionRepository) GetByID(
	ctx context.Context,
	executionID string,
) (*models.WorkflowStateExecution, error) {
	return r.BaseRepository.GetByID(ctx, executionID)
}

func (r *workflowExecutionRepository) List(
	ctx context.Context,
	status, instanceID string,
	limit int,
) ([]*models.WorkflowStateExecution, error) {
	db := r.BaseRepository.Pool().DB(ctx, true)

	if limit <= 0 {
		limit = 50
	}
	if limit > maxListLimit {
		limit = maxListLimit
	}

	query := db.Where("deleted_at IS NULL")
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if instanceID != "" {
		query = query.Where("instance_id = ?", instanceID)
	}

	var execs []*models.WorkflowStateExecution
	result := query.Order("created_at DESC").Limit(limit).Find(&execs)
	if result.Error != nil {
		return nil, fmt.Errorf("list executions: %w", result.Error)
	}

	return execs, nil
}

func (r *workflowExecutionRepository) GetLatestByInstance(
	ctx context.Context,
	instanceID string,
) (*models.WorkflowStateExecution, error) {
	db := r.BaseRepository.Pool().DB(ctx, true)

	var exec models.WorkflowStateExecution
	result := db.Where("instance_id = ? AND deleted_at IS NULL", instanceID).
		Order("created_at DESC").
		First(&exec)
	if result.Error != nil {
		return nil, fmt.Errorf("get latest execution: %w", result.Error)
	}

	return &exec, nil
}

// FindPending finds pending executions using FOR UPDATE SKIP LOCKED for safe multi-node operation.
func (r *workflowExecutionRepository) FindPending(
	ctx context.Context,
	limit int,
) ([]*models.WorkflowStateExecution, error) {
	db := r.BaseRepository.Pool().DB(ctx, false)
	if limit <= 0 {
		limit = 50
	}

	var execs []*models.WorkflowStateExecution
	result := db.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("status = ? AND deleted_at IS NULL", models.ExecStatusPending).
		Order("created_at ASC").
		Limit(limit).
		Find(&execs)

	if result.Error != nil {
		return nil, fmt.Errorf("find pending: %w", result.Error)
	}

	return execs, nil
}

// FindRetryDue finds executions scheduled for retry that are past their next_retry_at.
func (r *workflowExecutionRepository) FindRetryDue(
	ctx context.Context,
	limit int,
) ([]*models.WorkflowStateExecution, error) {
	db := r.BaseRepository.Pool().DB(ctx, false)
	if limit <= 0 {
		limit = 50
	}

	var execs []*models.WorkflowStateExecution
	result := db.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("status = ? AND next_retry_at <= ? AND deleted_at IS NULL", models.ExecStatusRetryScheduled, time.Now()).
		Order("next_retry_at ASC").
		Limit(limit).
		Find(&execs)

	if result.Error != nil {
		return nil, fmt.Errorf("find retry due: %w", result.Error)
	}

	return execs, nil
}

// FindTimedOut finds dispatched executions that have exceeded their timeout.
func (r *workflowExecutionRepository) FindTimedOut(
	ctx context.Context,
	timeoutSeconds int,
	limit int,
) ([]*models.WorkflowStateExecution, error) {
	db := r.BaseRepository.Pool().DB(ctx, false)
	if limit <= 0 {
		limit = 50
	}

	deadline := time.Now().Add(-time.Duration(timeoutSeconds) * time.Second)
	var execs []*models.WorkflowStateExecution
	result := db.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where(
			"status = ? AND started_at IS NOT NULL AND started_at < ? AND deleted_at IS NULL",
			models.ExecStatusDispatched,
			deadline,
		).
		Order("started_at ASC").
		Limit(limit).
		Find(&execs)

	if result.Error != nil {
		return nil, fmt.Errorf("find timed out: %w", result.Error)
	}

	return execs, nil
}

// VerifyAndConsumeToken verifies the execution token and atomically clears it to prevent replay.
func (r *workflowExecutionRepository) VerifyAndConsumeToken(
	ctx context.Context,
	executionID, tokenHash string,
) (*models.WorkflowStateExecution, error) {
	db := r.BaseRepository.Pool().DB(ctx, false)
	var exec *models.WorkflowStateExecution
	txErr := db.Transaction(func(tx *gorm.DB) error {
		lockedExec, err := r.VerifyAndConsumeTokenTx(tx, executionID, tokenHash)
		if err != nil {
			return err
		}
		exec = lockedExec
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}

	return exec, nil
}

// VerifyAndConsumeTokenTx is the same as VerifyAndConsumeToken but runs within an existing transaction.
func (r *workflowExecutionRepository) VerifyAndConsumeTokenTx(
	tx *gorm.DB,
	executionID, tokenHash string,
) (*models.WorkflowStateExecution, error) {
	var exec models.WorkflowStateExecution
	result := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where(
			"id = ? AND execution_token = ? AND status = ? AND deleted_at IS NULL",
			executionID,
			tokenHash,
			models.ExecStatusDispatched,
		).
		First(&exec)

	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("verify token: %w", result.Error)
	}

	if errors.Is(result.Error, gorm.ErrRecordNotFound) || result.RowsAffected == 0 {
		return nil, errors.New("invalid execution token or execution not in dispatched state")
	}

	consumeResult := tx.Model(&models.WorkflowStateExecution{}).
		Where("id = ? AND deleted_at IS NULL", executionID).
		UpdateColumn("execution_token", "")
	if consumeResult.Error != nil {
		return nil, fmt.Errorf("consume token: %w", consumeResult.Error)
	}

	return &exec, nil
}

func (r *workflowExecutionRepository) UpdateStatus(
	ctx context.Context,
	executionID string,
	status models.ExecutionStatus,
	fields map[string]any,
) error {
	db := r.BaseRepository.Pool().DB(ctx, false)

	updates := map[string]any{
		"status": string(status),
	}

	for k, v := range fields {
		updates[k] = v
	}

	if status == models.ExecStatusCompleted || status == models.ExecStatusFailed ||
		status == models.ExecStatusFatal || status == models.ExecStatusTimedOut {
		updates["finished_at"] = time.Now()
	}

	result := db.Model(&models.WorkflowStateExecution{}).
		Where("id = ? AND deleted_at IS NULL", executionID).
		UpdateColumns(updates)

	if result.Error != nil {
		return fmt.Errorf("update execution status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("update execution status: no rows updated")
	}

	return nil
}

func (r *workflowExecutionRepository) MarkStale(ctx context.Context, executionID string) error {
	return r.UpdateStatus(ctx, executionID, models.ExecStatusStale, nil)
}
