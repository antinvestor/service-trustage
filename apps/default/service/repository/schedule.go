package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/pitabwire/frame/datastore"
	"github.com/pitabwire/frame/datastore/pool"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
)

// ScheduleRepository manages schedule definition persistence.
type ScheduleRepository interface {
	Create(ctx context.Context, schedule *models.ScheduleDefinition) error
	FindDue(ctx context.Context, now time.Time, limit int) ([]*models.ScheduleDefinition, error)
	UpdateFireTimes(ctx context.Context, id string, lastFired time.Time, nextFire *time.Time) error
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

// FindDue returns schedules that are due to fire (next_fire_at <= now, active, not deleted).
// Uses FOR UPDATE SKIP LOCKED for safe multi-node operation.
func (r *scheduleRepository) FindDue(
	ctx context.Context,
	now time.Time,
	limit int,
) ([]*models.ScheduleDefinition, error) {
	db := r.BaseRepository.Pool().DB(ctx, false)

	var schedules []*models.ScheduleDefinition

	result := db.Raw(`
		SELECT * FROM schedule_definitions
		WHERE active = true
		  AND deleted_at IS NULL
		  AND next_fire_at IS NOT NULL
		  AND next_fire_at <= ?
		ORDER BY next_fire_at ASC
		LIMIT ?
		FOR UPDATE SKIP LOCKED
	`, now, limit).Scan(&schedules)

	if result.Error != nil {
		return nil, fmt.Errorf("find due schedules: %w", result.Error)
	}

	return schedules, nil
}

func (r *scheduleRepository) UpdateFireTimes(
	ctx context.Context,
	id string,
	lastFired time.Time,
	nextFire *time.Time,
) error {
	db := r.BaseRepository.Pool().DB(ctx, false)

	result := db.Exec(`
		UPDATE schedule_definitions
		SET last_fired_at = ?, next_fire_at = ?, modified_at = NOW()
		WHERE id = ?
	`, lastFired, nextFire, id)

	if result.Error != nil {
		return fmt.Errorf("update fire times: %w", result.Error)
	}

	return nil
}
