package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/pitabwire/frame/datastore"
	"github.com/pitabwire/frame/datastore/pool"
	"gorm.io/gorm/clause"

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
	result := db.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("active = ? AND deleted_at IS NULL AND next_fire_at IS NOT NULL AND next_fire_at <= ?", true, now).
		Order("next_fire_at ASC").
		Limit(limit).
		Find(&schedules)

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

	result := db.Model(&models.ScheduleDefinition{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]any{
			"last_fired_at": lastFired,
			"next_fire_at":  nextFire,
			"modified_at":   time.Now(),
		})

	if result.Error != nil {
		return fmt.Errorf("update fire times: %w", result.Error)
	}

	return nil
}
