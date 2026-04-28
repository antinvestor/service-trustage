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

package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
	ListPage(ctx context.Context, filter WorkflowExecutionListFilter) (*WorkflowExecutionPage, error)
	GetLatestByInstance(ctx context.Context, instanceID string) (*models.WorkflowStateExecution, error)
	FindPending(ctx context.Context, limit int) ([]*models.WorkflowStateExecution, error)
	FindRetryDue(ctx context.Context, limit int) ([]*models.WorkflowStateExecution, error)
	FindTimedOut(ctx context.Context, timeoutSeconds int, limit int) ([]*models.WorkflowStateExecution, error)
	VerifyAndConsumeToken(ctx context.Context, executionID, tokenHash string) (*models.WorkflowStateExecution, error)
	VerifyAndConsumeTokenTx(tx *gorm.DB, executionID, tokenHash string) (*models.WorkflowStateExecution, error)
	UpdateStatus(ctx context.Context, executionID string, status models.ExecutionStatus, fields map[string]any) error
	// MarkTimedOutAndCreateRetry atomically marks oldID as timed_out and inserts
	// the replacement retry execution in a single transaction. This prevents a
	// pod crash between the two steps from leaving a stuck dispatched execution.
	MarkTimedOutAndCreateRetry(ctx context.Context, oldID string, retry *models.WorkflowStateExecution) error
	MarkStale(ctx context.Context, executionID string) error
	Pool() pool.Pool
	// DeleteCompletedBefore batch-deletes terminal-state executions whose
	// finished_at is older than cutoff. Returns the number of rows deleted.
	DeleteCompletedBefore(ctx context.Context, cutoff time.Time, limit int) (int64, error)
}

type WorkflowExecutionListFilter struct {
	Status     string
	InstanceID string
	Query      string
	IDQuery    string
	Cursor     string
	Limit      int
}

type WorkflowExecutionPage struct {
	Items      []*models.WorkflowStateExecution
	NextCursor string
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
	page, err := r.ListPage(ctx, WorkflowExecutionListFilter{
		Status:     status,
		InstanceID: instanceID,
		Limit:      limit,
	})
	if err != nil {
		return nil, err
	}

	return page.Items, nil
}

func (r *workflowExecutionRepository) ListPage(
	ctx context.Context,
	filter WorkflowExecutionListFilter,
) (*WorkflowExecutionPage, error) {
	db := r.BaseRepository.Pool().DB(ctx, true)

	limit := normalizeListLimit(filter.Limit)

	query := db.Where("deleted_at IS NULL")
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.InstanceID != "" {
		query = query.Where("instance_id = ?", filter.InstanceID)
	}
	if q := strings.TrimSpace(filter.IDQuery); q != "" {
		query = query.Where("id ILIKE ?", "%"+q+"%")
	}
	if q := strings.TrimSpace(filter.Query); q != "" {
		like := "%" + q + "%"
		query = query.Where(
			"(id ILIKE ? OR instance_id ILIKE ? OR state ILIKE ? OR status ILIKE ? OR trace_id ILIKE ? OR error_class ILIKE ? OR error_message ILIKE ?)",
			like,
			like,
			like,
			like,
			like,
			like,
			like,
		)
	}

	var err error
	query, err = applyDescendingCreatedAtCursor(query, filter.Cursor)
	if err != nil {
		return nil, fmt.Errorf("list executions: %w", err)
	}

	var execs []*models.WorkflowStateExecution
	result := query.Order("created_at DESC, id DESC").Limit(limit + 1).Find(&execs)
	if result.Error != nil {
		return nil, fmt.Errorf("list executions: %w", result.Error)
	}

	nextCursor := ""
	if len(execs) > limit {
		last := execs[limit-1]
		nextCursor = encodeListCursor(last.CreatedAt, last.ID)
		execs = execs[:limit]
	}

	return &WorkflowExecutionPage{
		Items:      execs,
		NextCursor: nextCursor,
	}, nil
}

func (r *workflowExecutionRepository) GetLatestByInstance(
	ctx context.Context,
	instanceID string,
) (*models.WorkflowStateExecution, error) {
	db := r.BaseRepository.Pool().DB(ctx, true)

	var exec models.WorkflowStateExecution
	// Order by id DESC as well: BaseModel.CreatedAt is derived from the
	// xid timestamp (second-resolution), so multiple executions created
	// in the same second tie. Tie-breaking by id (also xid-derived, with
	// a monotonic counter) gives a deterministic "latest".
	result := db.Where("instance_id = ? AND deleted_at IS NULL", instanceID).
		Order("created_at DESC, id DESC").
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

// MarkTimedOutAndCreateRetry atomically marks oldID as timed_out and inserts
// the new retry execution. Using a single transaction prevents a pod crash
// between the two statements from leaving a stuck dispatched execution.
func (r *workflowExecutionRepository) MarkTimedOutAndCreateRetry(
	ctx context.Context,
	oldID string,
	retry *models.WorkflowStateExecution,
) error {
	db := r.BaseRepository.Pool().DB(ctx, false)

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.WorkflowStateExecution{}).
			Where("id = ? AND deleted_at IS NULL", oldID).
			UpdateColumns(map[string]any{
				"status":        string(models.ExecStatusTimedOut),
				"error_class":   "retryable",
				"error_message": "execution timed out",
				"finished_at":   time.Now().UTC(),
			}).Error; err != nil {
			return fmt.Errorf("mark timed out: %w", err)
		}

		if err := tx.Create(retry).Error; err != nil {
			return fmt.Errorf("create retry execution: %w", err)
		}

		return nil
	})
}

// DeleteCompletedBefore batch-deletes terminal-state executions whose
// finished_at is older than cutoff. Returns the number of rows deleted.
func (r *workflowExecutionRepository) DeleteCompletedBefore(
	ctx context.Context,
	cutoff time.Time,
	limit int,
) (int64, error) {
	db := r.BaseRepository.Pool().DB(ctx, false)
	if limit <= 0 {
		limit = 100
	}

	terminalStatuses := []string{
		string(models.ExecStatusCompleted),
		string(models.ExecStatusFatal),
		string(models.ExecStatusFailed),
		string(models.ExecStatusTimedOut),
		string(models.ExecStatusInvalidInputContract),
		string(models.ExecStatusInvalidOutputContract),
		string(models.ExecStatusStale),
	}

	var ids []string
	selectResult := db.Model(&models.WorkflowStateExecution{}).
		Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where(
			"status IN ? AND finished_at IS NOT NULL AND finished_at < ? AND deleted_at IS NULL",
			terminalStatuses, cutoff,
		).
		Limit(limit).
		Pluck("id", &ids)
	if selectResult.Error != nil {
		return 0, fmt.Errorf("select completed executions: %w", selectResult.Error)
	}
	if len(ids) == 0 {
		return 0, nil
	}

	res := db.Where("id IN ? AND deleted_at IS NULL", ids).Delete(&models.WorkflowStateExecution{})
	if res.Error != nil {
		return 0, fmt.Errorf("delete completed executions: %w", res.Error)
	}

	return res.RowsAffected, nil
}
