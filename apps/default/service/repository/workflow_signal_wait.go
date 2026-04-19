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
	"fmt"
	"time"

	"github.com/pitabwire/frame/datastore"
	"github.com/pitabwire/frame/datastore/pool"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
)

// WorkflowSignalWaitRepository manages signal wait persistence.
type WorkflowSignalWaitRepository interface {
	Create(ctx context.Context, wait *models.WorkflowSignalWait) error
	GetByExecutionID(ctx context.Context, executionID string) (*models.WorkflowSignalWait, error)
	FindActiveByInstanceAndSignal(
		ctx context.Context,
		instanceID, signalName string,
	) (*models.WorkflowSignalWait, error)
	ListByInstance(ctx context.Context, instanceID string, limit int) ([]*models.WorkflowSignalWait, error)
	ClaimTimedOut(
		ctx context.Context,
		now time.Time,
		limit int,
		owner string,
		leaseUntil time.Time,
	) ([]*models.WorkflowSignalWait, error)
	MarkCompletedByOwner(ctx context.Context, id, owner, messageID string, matchedAt time.Time) error
	MarkTimedOutByOwner(ctx context.Context, id, owner string, timedOutAt time.Time) error
	ReleaseClaim(ctx context.Context, id, owner string) error
	// DeleteCompletedBefore batch-deletes terminal signal waits (matched or
	// timed_out) whose modified_at is older than cutoff. Returns rows deleted.
	DeleteCompletedBefore(ctx context.Context, cutoff time.Time, limit int) (int64, error)
}

type workflowSignalWaitRepository struct {
	datastore.BaseRepository[*models.WorkflowSignalWait]
	pool pool.Pool
}

// NewWorkflowSignalWaitRepository creates a new WorkflowSignalWaitRepository.
func NewWorkflowSignalWaitRepository(dbPool pool.Pool) WorkflowSignalWaitRepository {
	ctx := context.Background()

	return &workflowSignalWaitRepository{
		BaseRepository: datastore.NewBaseRepository[*models.WorkflowSignalWait](
			ctx,
			dbPool,
			nil,
			func() *models.WorkflowSignalWait { return &models.WorkflowSignalWait{} },
		),
		pool: dbPool,
	}
}

func (r *workflowSignalWaitRepository) Create(ctx context.Context, wait *models.WorkflowSignalWait) error {
	return r.BaseRepository.Create(ctx, wait)
}

func (r *workflowSignalWaitRepository) GetByExecutionID(
	ctx context.Context,
	executionID string,
) (*models.WorkflowSignalWait, error) {
	db := r.pool.DB(ctx, true)

	var wait models.WorkflowSignalWait
	result := db.Where("execution_id = ? AND deleted_at IS NULL", executionID).First(&wait)
	if result.Error != nil {
		return nil, fmt.Errorf("get signal wait by execution: %w", result.Error)
	}

	return &wait, nil
}

func (r *workflowSignalWaitRepository) FindActiveByInstanceAndSignal(
	ctx context.Context,
	instanceID, signalName string,
) (*models.WorkflowSignalWait, error) {
	db := r.pool.DB(ctx, true)

	var wait models.WorkflowSignalWait
	result := db.Where(
		"instance_id = ? AND signal_name = ? AND status = 'waiting' AND deleted_at IS NULL",
		instanceID, signalName,
	).Order("created_at").First(&wait)
	if result.Error != nil {
		return nil, fmt.Errorf("find active signal wait: %w", result.Error)
	}

	return &wait, nil
}

func (r *workflowSignalWaitRepository) ListByInstance(
	ctx context.Context,
	instanceID string,
	limit int,
) ([]*models.WorkflowSignalWait, error) {
	db := r.pool.DB(ctx, true)

	if limit <= 0 {
		limit = 100
	}
	if limit > maxListLimit {
		limit = maxListLimit
	}

	var waits []*models.WorkflowSignalWait
	result := db.Where("instance_id = ? AND deleted_at IS NULL", instanceID).
		Order("created_at ASC").
		Limit(limit).
		Find(&waits)
	if result.Error != nil {
		return nil, fmt.Errorf("list signal waits by instance: %w", result.Error)
	}

	return waits, nil
}

func (r *workflowSignalWaitRepository) ClaimTimedOut(
	ctx context.Context,
	now time.Time,
	limit int,
	owner string,
	leaseUntil time.Time,
) ([]*models.WorkflowSignalWait, error) {
	db := r.pool.DB(ctx, false)

	var waits []*models.WorkflowSignalWait
	txErr := db.Transaction(func(tx *gorm.DB) error {
		result := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where(
				"status = ? AND timeout_at IS NOT NULL AND timeout_at <= ? AND deleted_at IS NULL AND (claim_until IS NULL OR claim_until < ?)",
				"waiting",
				now,
				now,
			).
			Order("timeout_at ASC").
			Limit(limit).
			Find(&waits)
		if result.Error != nil {
			return fmt.Errorf("claim timed out signal waits: %w", result.Error)
		}
		if len(waits) == 0 {
			return nil
		}

		ids := make([]string, 0, len(waits))
		for _, wait := range waits {
			ids = append(ids, wait.ID)
		}

		updateResult := tx.Model(&models.WorkflowSignalWait{}).
			Where("id IN ? AND deleted_at IS NULL", ids).
			UpdateColumns(map[string]any{
				"claim_owner": owner,
				"claim_until": leaseUntil,
				"attempts":    gorm.Expr("attempts + 1"),
			})
		if updateResult.Error != nil {
			return fmt.Errorf("claim timed out signal waits: %w", updateResult.Error)
		}

		for _, wait := range waits {
			wait.ClaimOwner = owner
			wait.ClaimUntil = &leaseUntil
			wait.Attempts++
		}

		return nil
	})
	if txErr != nil {
		return nil, txErr
	}

	return waits, nil
}

func (r *workflowSignalWaitRepository) MarkCompletedByOwner(
	ctx context.Context,
	id, owner, messageID string,
	matchedAt time.Time,
) error {
	db := r.pool.DB(ctx, false)

	result := db.Model(&models.WorkflowSignalWait{}).
		Where("id = ? AND claim_owner = ? AND status = ? AND deleted_at IS NULL", id, owner, "waiting").
		UpdateColumns(map[string]any{
			"status":      "matched",
			"matched_at":  matchedAt,
			"message_id":  messageID,
			"claim_owner": "",
			"claim_until": gorm.Expr("NULL"),
		})
	if result.Error != nil {
		return fmt.Errorf("mark signal wait completed: %w", result.Error)
	}

	return nil
}

func (r *workflowSignalWaitRepository) MarkTimedOutByOwner(
	ctx context.Context,
	id, owner string,
	timedOutAt time.Time,
) error {
	db := r.pool.DB(ctx, false)

	result := db.Model(&models.WorkflowSignalWait{}).
		Where("id = ? AND claim_owner = ? AND status = ? AND deleted_at IS NULL", id, owner, "waiting").
		UpdateColumns(map[string]any{
			"status":       "timed_out",
			"timed_out_at": timedOutAt,
			"claim_owner":  "",
			"claim_until":  gorm.Expr("NULL"),
		})
	if result.Error != nil {
		return fmt.Errorf("mark signal wait timed out: %w", result.Error)
	}

	return nil
}

func (r *workflowSignalWaitRepository) ReleaseClaim(ctx context.Context, id, owner string) error {
	db := r.pool.DB(ctx, false)

	result := db.Model(&models.WorkflowSignalWait{}).
		Where("id = ? AND claim_owner = ? AND status = ? AND deleted_at IS NULL", id, owner, "waiting").
		UpdateColumns(map[string]any{
			"claim_owner": "",
			"claim_until": gorm.Expr("NULL"),
		})
	if result.Error != nil {
		return fmt.Errorf("release signal wait claim: %w", result.Error)
	}

	return nil
}

// DeleteCompletedBefore batch-deletes terminal signal waits (matched or
// timed_out) whose modified_at is older than cutoff. Returns rows deleted.
func (r *workflowSignalWaitRepository) DeleteCompletedBefore(
	ctx context.Context,
	cutoff time.Time,
	limit int,
) (int64, error) {
	db := r.pool.DB(ctx, false)
	if limit <= 0 {
		limit = 100
	}

	terminalStatuses := []string{"matched", "timed_out"}

	var ids []string
	selectResult := db.Model(&models.WorkflowSignalWait{}).
		Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("status IN ? AND modified_at < ? AND deleted_at IS NULL", terminalStatuses, cutoff).
		Limit(limit).
		Pluck("id", &ids)
	if selectResult.Error != nil {
		return 0, fmt.Errorf("select terminal signal waits: %w", selectResult.Error)
	}
	if len(ids) == 0 {
		return 0, nil
	}

	res := db.Where("id IN ? AND deleted_at IS NULL", ids).Delete(&models.WorkflowSignalWait{})
	if res.Error != nil {
		return 0, fmt.Errorf("delete terminal signal waits: %w", res.Error)
	}

	return res.RowsAffected, nil
}
