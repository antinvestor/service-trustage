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

	"github.com/antinvestor/service-trustage/apps/queue/service/models"
)

// maxListLimit caps the maximum number of items returned by list queries.
const maxListLimit = 200

// QueueItemRepository manages queue item persistence.
type QueueItemRepository interface {
	Pool() pool.Pool
	Create(ctx context.Context, item *models.QueueItem) error
	GetByID(ctx context.Context, id string) (*models.QueueItem, error)
	FindNextWaiting(ctx context.Context, queueID string, categories []string) (*models.QueueItem, error)
	ListWaiting(ctx context.Context, queueID string, limit, offset int) ([]*models.QueueItem, error)
	GetPosition(ctx context.Context, item *models.QueueItem) (int, error)
	Update(ctx context.Context, item *models.QueueItem) error
	CountByStatus(ctx context.Context, queueID string, status models.QueueItemStatus) (int64, error)
	CountWaitingForUpdate(ctx context.Context, queueID string) (int64, error)
	AvgWaitMinutes(ctx context.Context, queueID string, since time.Time) (float64, error)
	LongestWaitMinutes(ctx context.Context, queueID string) (float64, error)
	CountByStatusSince(
		ctx context.Context,
		queueID string,
		status models.QueueItemStatus,
		since time.Time,
	) (int64, error)
}

type queueItemRepository struct {
	datastore.BaseRepository[*models.QueueItem]
}

// NewQueueItemRepository creates a new QueueItemRepository.
func NewQueueItemRepository(dbPool pool.Pool) QueueItemRepository {
	ctx := context.Background()

	return &queueItemRepository{
		BaseRepository: datastore.NewBaseRepository[*models.QueueItem](
			ctx,
			dbPool,
			nil,
			func() *models.QueueItem { return &models.QueueItem{} },
		),
	}
}

func (r *queueItemRepository) Pool() pool.Pool {
	return r.BaseRepository.Pool()
}

func (r *queueItemRepository) Create(ctx context.Context, item *models.QueueItem) error {
	return r.BaseRepository.Create(ctx, item)
}

func (r *queueItemRepository) GetByID(ctx context.Context, id string) (*models.QueueItem, error) {
	return r.BaseRepository.GetByID(ctx, id)
}

// FindNextWaiting atomically finds and locks the next waiting item using FOR UPDATE SKIP LOCKED.
func (r *queueItemRepository) FindNextWaiting(
	ctx context.Context,
	queueID string,
	categories []string,
) (*models.QueueItem, error) {
	db := r.BaseRepository.Pool().DB(ctx, false)

	var item models.QueueItem
	query := db.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where(
			"queue_id = ? AND status = ? AND deleted_at IS NULL",
			queueID,
			models.ItemStatusWaiting,
		)
	if len(categories) > 0 {
		query = query.Where("category IN ?", categories)
	}

	result := query.Order("priority DESC, joined_at ASC").First(&item)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, errors.New("find next waiting item: no rows found")
	}
	if result.Error != nil {
		return nil, fmt.Errorf("find next waiting item: %w", result.Error)
	}

	return &item, nil
}

func (r *queueItemRepository) ListWaiting(
	ctx context.Context,
	queueID string,
	limit, offset int,
) ([]*models.QueueItem, error) {
	db := r.BaseRepository.Pool().DB(ctx, true)

	if limit <= 0 {
		limit = 50
	}

	if limit > maxListLimit {
		limit = maxListLimit
	}

	if offset < 0 {
		offset = 0
	}

	var items []*models.QueueItem

	result := db.Where(
		"queue_id = ? AND status = ? AND deleted_at IS NULL",
		queueID, models.ItemStatusWaiting,
	).Order("priority DESC, joined_at ASC").Limit(limit).Offset(offset).Find(&items)

	if result.Error != nil {
		return nil, fmt.Errorf("list waiting items: %w", result.Error)
	}

	return items, nil
}

// GetPosition returns the 1-based position of an item in its queue.
func (r *queueItemRepository) GetPosition(ctx context.Context, item *models.QueueItem) (int, error) {
	db := r.BaseRepository.Pool().DB(ctx, true)

	var count int64

	result := db.Model(&models.QueueItem{}).Where(
		"queue_id = ? AND status = ? AND deleted_at IS NULL AND (priority > ? OR (priority = ? AND joined_at < ?))",
		item.QueueID, models.ItemStatusWaiting, item.Priority, item.Priority, item.JoinedAt,
	).Count(&count)

	if result.Error != nil {
		return 0, fmt.Errorf("get queue position: %w", result.Error)
	}

	return int(count) + 1, nil
}

func (r *queueItemRepository) Update(ctx context.Context, item *models.QueueItem) error {
	_, err := r.BaseRepository.Update(ctx, item)
	return err
}

func (r *queueItemRepository) CountByStatus(
	ctx context.Context,
	queueID string,
	status models.QueueItemStatus,
) (int64, error) {
	db := r.BaseRepository.Pool().DB(ctx, true)

	var count int64

	result := db.Model(&models.QueueItem{}).Where(
		"queue_id = ? AND status = ? AND deleted_at IS NULL",
		queueID, status,
	).Count(&count)

	if result.Error != nil {
		return 0, fmt.Errorf("count items by status: %w", result.Error)
	}

	return count, nil
}

// CountWaitingForUpdate counts waiting items within a write transaction for atomic capacity checks.
func (r *queueItemRepository) CountWaitingForUpdate(ctx context.Context, queueID string) (int64, error) {
	db := r.BaseRepository.Pool().DB(ctx, false)

	var ids []string
	result := db.Model(&models.QueueItem{}).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where(
			"queue_id = ? AND status = ? AND deleted_at IS NULL",
			queueID,
			models.ItemStatusWaiting,
		).
		Pluck("id", &ids)
	if result.Error != nil {
		return 0, fmt.Errorf("count waiting for update: %w", result.Error)
	}

	return int64(len(ids)), nil
}

func (r *queueItemRepository) AvgWaitMinutes(
	ctx context.Context,
	queueID string,
	since time.Time,
) (float64, error) {
	db := r.BaseRepository.Pool().DB(ctx, true)

	var avg *float64

	result := db.Model(&models.QueueItem{}).
		Select("AVG(EXTRACT(EPOCH FROM (called_at - joined_at)) / 60)").
		Where(
			"queue_id = ? AND called_at IS NOT NULL AND called_at >= ? AND deleted_at IS NULL",
			queueID, since,
		).Scan(&avg)

	if result.Error != nil {
		return 0, fmt.Errorf("avg wait minutes: %w", result.Error)
	}

	if avg == nil {
		return 0, nil
	}

	return *avg, nil
}

func (r *queueItemRepository) LongestWaitMinutes(
	ctx context.Context,
	queueID string,
) (float64, error) {
	db := r.BaseRepository.Pool().DB(ctx, true)

	var maxWait *float64

	result := db.Model(&models.QueueItem{}).
		Select("MAX(EXTRACT(EPOCH FROM (NOW() - joined_at)) / 60)").
		Where(
			"queue_id = ? AND status = ? AND deleted_at IS NULL",
			queueID, models.ItemStatusWaiting,
		).Scan(&maxWait)

	if result.Error != nil {
		return 0, fmt.Errorf("longest wait minutes: %w", result.Error)
	}

	if maxWait == nil {
		return 0, nil
	}

	return *maxWait, nil
}

func (r *queueItemRepository) CountByStatusSince(
	ctx context.Context,
	queueID string,
	status models.QueueItemStatus,
	since time.Time,
) (int64, error) {
	db := r.BaseRepository.Pool().DB(ctx, true)

	var count int64

	result := db.Model(&models.QueueItem{}).Where(
		"queue_id = ? AND status = ? AND modified_at >= ? AND deleted_at IS NULL",
		queueID, status, since,
	).Count(&count)

	if result.Error != nil {
		return 0, fmt.Errorf("count items by status since: %w", result.Error)
	}

	return count, nil
}
