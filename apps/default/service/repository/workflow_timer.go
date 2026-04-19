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

// WorkflowTimerRepository manages durable execution wakeup timers.
type WorkflowTimerRepository interface {
	Create(ctx context.Context, timer *models.WorkflowTimer) error
	ClaimDue(
		ctx context.Context,
		now time.Time,
		limit int,
		owner string,
		leaseUntil time.Time,
	) ([]*models.WorkflowTimer, error)
	MarkFiredByOwner(ctx context.Context, id string, owner string, firedAt time.Time) error
	ReleaseClaim(ctx context.Context, id string, owner string) error
	GetByExecutionID(ctx context.Context, executionID string) (*models.WorkflowTimer, error)
	// DeleteCompletedBefore batch-deletes fired timers whose fired_at is older
	// than cutoff. Returns the number of rows deleted.
	DeleteCompletedBefore(ctx context.Context, cutoff time.Time, limit int) (int64, error)
}

type workflowTimerRepository struct {
	datastore.BaseRepository[*models.WorkflowTimer]
	pool pool.Pool
}

// NewWorkflowTimerRepository creates a new WorkflowTimerRepository.
func NewWorkflowTimerRepository(dbPool pool.Pool) WorkflowTimerRepository {
	ctx := context.Background()

	return &workflowTimerRepository{
		BaseRepository: datastore.NewBaseRepository[*models.WorkflowTimer](
			ctx,
			dbPool,
			nil,
			func() *models.WorkflowTimer { return &models.WorkflowTimer{} },
		),
		pool: dbPool,
	}
}

func (r *workflowTimerRepository) Create(ctx context.Context, timer *models.WorkflowTimer) error {
	return r.BaseRepository.Create(ctx, timer)
}

func (r *workflowTimerRepository) ClaimDue(
	ctx context.Context,
	now time.Time,
	limit int,
	owner string,
	leaseUntil time.Time,
) ([]*models.WorkflowTimer, error) {
	db := r.pool.DB(ctx, false)

	var timers []*models.WorkflowTimer

	txErr := db.Transaction(func(tx *gorm.DB) error {
		result := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where(
				"fired_at IS NULL AND deleted_at IS NULL AND fires_at <= ? AND (claim_until IS NULL OR claim_until < ?)",
				now,
				now,
			).
			Order("fires_at ASC").
			Limit(limit).
			Find(&timers)
		if result.Error != nil {
			return fmt.Errorf("claim due timers: %w", result.Error)
		}
		if len(timers) == 0 {
			return nil
		}

		ids := make([]string, 0, len(timers))
		for _, timer := range timers {
			ids = append(ids, timer.ID)
		}

		updateResult := tx.Model(&models.WorkflowTimer{}).
			Where("id IN ? AND deleted_at IS NULL", ids).
			UpdateColumns(map[string]any{
				"claim_owner": owner,
				"claim_until": leaseUntil,
				"attempts":    gorm.Expr("attempts + 1"),
			})
		if updateResult.Error != nil {
			return fmt.Errorf("claim due timers: %w", updateResult.Error)
		}

		for _, timer := range timers {
			timer.ClaimOwner = owner
			timer.ClaimUntil = &leaseUntil
			timer.Attempts++
		}

		return nil
	})
	if txErr != nil {
		return nil, txErr
	}

	return timers, nil
}

func (r *workflowTimerRepository) MarkFiredByOwner(
	ctx context.Context,
	id string,
	owner string,
	firedAt time.Time,
) error {
	db := r.pool.DB(ctx, false)

	result := db.Model(&models.WorkflowTimer{}).
		Where("id = ? AND claim_owner = ? AND fired_at IS NULL AND deleted_at IS NULL", id, owner).
		UpdateColumns(map[string]any{
			"fired_at":    firedAt,
			"claim_owner": "",
			"claim_until": gorm.Expr("NULL"),
		})
	if result.Error != nil {
		return fmt.Errorf("mark timer fired: %w", result.Error)
	}

	return nil
}

func (r *workflowTimerRepository) ReleaseClaim(ctx context.Context, id string, owner string) error {
	db := r.pool.DB(ctx, false)

	result := db.Model(&models.WorkflowTimer{}).
		Where("id = ? AND claim_owner = ? AND fired_at IS NULL AND deleted_at IS NULL", id, owner).
		UpdateColumns(map[string]any{
			"claim_owner": "",
			"claim_until": gorm.Expr("NULL"),
		})
	if result.Error != nil {
		return fmt.Errorf("release timer claim: %w", result.Error)
	}

	return nil
}

func (r *workflowTimerRepository) GetByExecutionID(
	ctx context.Context,
	executionID string,
) (*models.WorkflowTimer, error) {
	db := r.pool.DB(ctx, true)

	var timer models.WorkflowTimer
	result := db.Where("execution_id = ? AND deleted_at IS NULL", executionID).First(&timer)
	if result.Error != nil {
		return nil, fmt.Errorf("get timer by execution: %w", result.Error)
	}

	return &timer, nil
}

// DeleteCompletedBefore batch-deletes fired timers whose fired_at is older
// than cutoff. Returns the number of rows deleted.
func (r *workflowTimerRepository) DeleteCompletedBefore(
	ctx context.Context,
	cutoff time.Time,
	limit int,
) (int64, error) {
	db := r.pool.DB(ctx, false)
	if limit <= 0 {
		limit = 100
	}

	var ids []string
	selectResult := db.Model(&models.WorkflowTimer{}).
		Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("fired_at IS NOT NULL AND fired_at < ? AND deleted_at IS NULL", cutoff).
		Limit(limit).
		Pluck("id", &ids)
	if selectResult.Error != nil {
		return 0, fmt.Errorf("select fired timers: %w", selectResult.Error)
	}
	if len(ids) == 0 {
		return 0, nil
	}

	res := db.Where("id IN ? AND deleted_at IS NULL", ids).Delete(&models.WorkflowTimer{})
	if res.Error != nil {
		return 0, fmt.Errorf("delete fired timers: %w", res.Error)
	}

	return res.RowsAffected, nil
}
