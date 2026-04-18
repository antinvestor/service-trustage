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

// ScheduleRepository manages schedule_definitions persistence.
//
// The v1 surface is intentionally narrow: schedules are declared in workflow specs
// and materialised at CreateWorkflow; there is no schedule-level mutation RPC.
// Callers:
//   - business layer (Create, ListByWorkflow, SetActiveByWorkflow) — workflow lifecycle.
//   - CronScheduler (ClaimAndFireBatch) — the fire hot path.
type ScheduleRepository interface {
	Create(ctx context.Context, schedule *models.ScheduleDefinition) error

	ListByWorkflow(ctx context.Context, workflowName string, workflowVersion int) ([]*models.ScheduleDefinition, error)

	// SetActiveByWorkflow flips active on all non-deleted schedules for the given
	// (workflowName, workflowVersion) tuple. When activating (active=true), it also
	// seeds next_fire_at using the provided baseline; when deactivating, it clears
	// next_fire_at (avoids stale due rows lingering in the partial index).
	//
	// Must be called inside tx so the flip is atomic with the workflow status update.
	SetActiveByWorkflow(
		ctx context.Context,
		tx *gorm.DB,
		workflowName string,
		workflowVersion int,
		active bool,
		seedNextFireAt *time.Time,
		seedJitterSeconds int,
	) error

	// ClaimAndFireBatch scans for due schedules under one tx, invokes fireFn for each,
	// and commits atomically. fireFn receives the schedule and a DB handle bound to
	// the same tx so event_log inserts and next_fire_at updates participate in the
	// same transaction as the FOR UPDATE SKIP LOCKED row lock.
	//
	// fireFn returns the new next_fire_at and jitter_seconds. The repository persists
	// those onto the row before committing.
	//
	// Returns the number of schedules for which fireFn returned nil error.
	ClaimAndFireBatch(
		ctx context.Context,
		now time.Time,
		limit int,
		fireFn func(ctx context.Context, tx *gorm.DB, sched *models.ScheduleDefinition) (nextFire *time.Time, jitterSeconds int, err error),
	) (int, error)

	// Pool exposes the underlying pool for callers that need to drive tx boundaries
	// spanning multiple repositories.
	Pool() pool.Pool
}

type scheduleRepository struct {
	datastore.BaseRepository[*models.ScheduleDefinition]
}

// NewScheduleRepository creates a new ScheduleRepository.
func NewScheduleRepository(dbPool pool.Pool) ScheduleRepository {
	ctx := context.Background()

	return &scheduleRepository{
		BaseRepository: datastore.NewBaseRepository[*models.ScheduleDefinition](
			ctx,
			dbPool,
			nil,
			func() *models.ScheduleDefinition { return &models.ScheduleDefinition{} },
		),
	}
}

func (r *scheduleRepository) Create(ctx context.Context, schedule *models.ScheduleDefinition) error {
	return r.BaseRepository.Create(ctx, schedule)
}

func (r *scheduleRepository) Pool() pool.Pool {
	return r.BaseRepository.Pool()
}

func (r *scheduleRepository) ListByWorkflow(
	ctx context.Context,
	workflowName string,
	workflowVersion int,
) ([]*models.ScheduleDefinition, error) {
	db := r.BaseRepository.Pool().DB(ctx, false)

	var out []*models.ScheduleDefinition
	result := db.Where(
		"workflow_name = ? AND workflow_version = ? AND deleted_at IS NULL",
		workflowName, workflowVersion,
	).Order("name ASC").Find(&out)

	if result.Error != nil {
		return nil, fmt.Errorf("list schedules by workflow: %w", result.Error)
	}

	return out, nil
}

func (r *scheduleRepository) SetActiveByWorkflow(
	ctx context.Context,
	tx *gorm.DB,
	workflowName string,
	workflowVersion int,
	active bool,
	seedNextFireAt *time.Time,
	seedJitterSeconds int,
) error {
	if tx == nil {
		return fmt.Errorf("SetActiveByWorkflow requires a non-nil tx")
	}

	updates := map[string]any{
		"active":      active,
		"modified_at": time.Now().UTC(),
	}
	if active {
		updates["next_fire_at"] = seedNextFireAt
		updates["jitter_seconds"] = seedJitterSeconds
	} else {
		updates["next_fire_at"] = nil
	}

	query := tx.Model(&models.ScheduleDefinition{}).
		Where("workflow_name = ? AND deleted_at IS NULL", workflowName)
	if workflowVersion >= 0 {
		query = query.Where("workflow_version = ?", workflowVersion)
	}

	result := query.Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("set active by workflow: %w", result.Error)
	}

	return nil
}

func (r *scheduleRepository) ClaimAndFireBatch(
	ctx context.Context,
	now time.Time,
	limit int,
	fireFn func(ctx context.Context, tx *gorm.DB, sched *models.ScheduleDefinition) (*time.Time, int, error),
) (int, error) {
	db := r.BaseRepository.Pool().DB(ctx, false)

	fired := 0

	for i := 0; i < limit; i++ {
		// Each schedule gets its own transaction so a single fireFn failure
		// does not abort the entire batch and so SKIP LOCKED is applied per-row.
		var sched *models.ScheduleDefinition

		txErr := db.Transaction(func(tx *gorm.DB) error {
			// Claim exactly one due schedule under SKIP LOCKED.
			var due []*models.ScheduleDefinition
			selectErr := tx.
				Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
				Where("active = ? AND deleted_at IS NULL AND next_fire_at IS NOT NULL AND next_fire_at <= ?", true, now).
				Order("next_fire_at ASC").
				Limit(1).
				Find(&due).Error
			if selectErr != nil {
				return fmt.Errorf("claim due schedule: %w", selectErr)
			}
			if len(due) == 0 {
				return nil
			}
			sched = due[0]

			nextFire, jitterSeconds, fireErr := fireFn(ctx, tx, sched)
			if fireErr != nil {
				return fmt.Errorf("fire schedule %s: %w", sched.ID, fireErr)
			}

			updateErr := tx.Exec(
				"UPDATE schedule_definitions SET last_fired_at = ?, next_fire_at = ?, jitter_seconds = ?, modified_at = ? WHERE id = ?",
				now, nextFire, jitterSeconds, now, sched.ID,
			).Error
			if updateErr != nil {
				return fmt.Errorf("update fire times for %s: %w", sched.ID, updateErr)
			}

			return nil
		})

		if txErr != nil {
			return fired, txErr
		}

		if sched == nil {
			// No due schedules remain.
			break
		}

		fired++
	}

	return fired, nil
}
