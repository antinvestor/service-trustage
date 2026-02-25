package repository

import (
	"context"
	"fmt"

	"github.com/pitabwire/frame/datastore"
	"github.com/pitabwire/frame/datastore/pool"

	"github.com/antinvestor/service-trustage/apps/queue/service/models"
)

// QueueCounterRepository manages queue counter persistence.
type QueueCounterRepository interface {
	Create(ctx context.Context, counter *models.QueueCounter) error
	GetByID(ctx context.Context, id string) (*models.QueueCounter, error)
	ListByQueueID(ctx context.Context, queueID string) ([]*models.QueueCounter, error)
	Update(ctx context.Context, counter *models.QueueCounter) error
	CountOpen(ctx context.Context, queueID string) (int64, error)
	SoftDelete(ctx context.Context, counter *models.QueueCounter) error
}

type queueCounterRepository struct {
	datastore.BaseRepository[*models.QueueCounter]
}

// NewQueueCounterRepository creates a new QueueCounterRepository.
func NewQueueCounterRepository(dbPool pool.Pool) QueueCounterRepository {
	ctx := context.Background()

	return &queueCounterRepository{
		BaseRepository: datastore.NewBaseRepository[*models.QueueCounter](
			ctx,
			dbPool,
			nil,
			func() *models.QueueCounter { return &models.QueueCounter{} },
		),
	}
}

func (r *queueCounterRepository) Create(ctx context.Context, counter *models.QueueCounter) error {
	return r.BaseRepository.Create(ctx, counter)
}

func (r *queueCounterRepository) GetByID(ctx context.Context, id string) (*models.QueueCounter, error) {
	return r.BaseRepository.GetByID(ctx, id)
}

// maxCounterListLimit caps the maximum number of counters returned.
const maxCounterListLimit = 200

func (r *queueCounterRepository) ListByQueueID(
	ctx context.Context,
	queueID string,
) ([]*models.QueueCounter, error) {
	db := r.BaseRepository.Pool().DB(ctx, true)

	var counters []*models.QueueCounter

	result := db.Where(
		"queue_id = ? AND deleted_at IS NULL",
		queueID,
	).Order("name ASC").Limit(maxCounterListLimit).Find(&counters)

	if result.Error != nil {
		return nil, fmt.Errorf("list counters by queue: %w", result.Error)
	}

	return counters, nil
}

func (r *queueCounterRepository) Update(ctx context.Context, counter *models.QueueCounter) error {
	_, err := r.BaseRepository.Update(ctx, counter)
	return err
}

func (r *queueCounterRepository) CountOpen(ctx context.Context, queueID string) (int64, error) {
	db := r.BaseRepository.Pool().DB(ctx, true)

	var count int64

	result := db.Model(&models.QueueCounter{}).Where(
		"queue_id = ? AND status = ? AND deleted_at IS NULL",
		queueID, models.CounterStatusOpen,
	).Count(&count)

	if result.Error != nil {
		return 0, fmt.Errorf("count open counters: %w", result.Error)
	}

	return count, nil
}

func (r *queueCounterRepository) SoftDelete(ctx context.Context, counter *models.QueueCounter) error {
	db := r.BaseRepository.Pool().DB(ctx, false)

	result := db.Delete(counter)
	if result.Error != nil {
		return fmt.Errorf("soft delete counter: %w", result.Error)
	}

	return nil
}
