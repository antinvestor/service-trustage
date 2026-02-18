package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/pitabwire/frame/datastore"
	"github.com/pitabwire/frame/datastore/pool"
	"gorm.io/gorm"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
)

// EventLogRepository manages event log persistence for the outbox pattern.
type EventLogRepository interface {
	Create(ctx context.Context, event *models.EventLog) error
	FindByIdempotencyKey(ctx context.Context, tenantID, key string) (*models.EventLog, error)
	FindUnpublished(ctx context.Context, limit int) ([]*models.EventLog, error)
	MarkPublished(ctx context.Context, id string) error
	// FindAndProcessUnpublished atomically finds unpublished events and processes each one
	// within a single transaction, ensuring the FOR UPDATE SKIP LOCKED lock is held.
	FindAndProcessUnpublished(ctx context.Context, limit int, fn func(event *models.EventLog) error) (int, error)
	// DeletePublishedBefore hard-deletes published events older than the given time.
	DeletePublishedBefore(ctx context.Context, before time.Time, limit int) (int64, error)
}

type eventLogRepository struct {
	datastore.BaseRepository[*models.EventLog]
	pool pool.Pool
}

// NewEventLogRepository creates a new EventLogRepository.
func NewEventLogRepository(dbPool pool.Pool) EventLogRepository {
	ctx := context.Background()

	return &eventLogRepository{
		BaseRepository: datastore.NewBaseRepository[*models.EventLog](
			ctx,
			dbPool,
			nil,
			func() *models.EventLog { return &models.EventLog{} },
		),
		pool: dbPool,
	}
}

func (r *eventLogRepository) Create(ctx context.Context, event *models.EventLog) error {
	return r.BaseRepository.Create(ctx, event)
}

// FindByIdempotencyKey returns an existing event with the given idempotency key, or nil if not found.
func (r *eventLogRepository) FindByIdempotencyKey(
	ctx context.Context,
	tenantID, key string,
) (*models.EventLog, error) {
	db := r.pool.DB(ctx, true)

	var event models.EventLog

	result := db.Where("tenant_id = ? AND idempotency_key = ?", tenantID, key).First(&event)
	if result.Error != nil {
		return nil, fmt.Errorf("find by idempotency key: %w", result.Error)
	}

	return &event, nil
}

// FindUnpublished finds unpublished events using FOR UPDATE SKIP LOCKED.
func (r *eventLogRepository) FindUnpublished(ctx context.Context, limit int) ([]*models.EventLog, error) {
	db := r.pool.DB(ctx, false)

	var events []*models.EventLog

	result := db.Raw(
		`SELECT * FROM event_log
		 WHERE published = false AND deleted_at IS NULL
		 ORDER BY created_at
		 FOR UPDATE SKIP LOCKED
		 LIMIT ?`, limit,
	).Scan(&events)

	if result.Error != nil {
		return nil, fmt.Errorf("find unpublished: %w", result.Error)
	}

	return events, nil
}

func (r *eventLogRepository) MarkPublished(ctx context.Context, id string) error {
	db := r.pool.DB(ctx, false)

	now := time.Now()

	result := db.Model(&models.EventLog{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"published":    true,
			"published_at": now,
		})

	if result.Error != nil {
		return fmt.Errorf("mark published: %w", result.Error)
	}

	return nil
}

// DeletePublishedBefore hard-deletes published events older than the given time.
func (r *eventLogRepository) DeletePublishedBefore(
	ctx context.Context,
	before time.Time,
	limit int,
) (int64, error) {
	db := r.pool.DB(ctx, false)

	result := db.Exec(
		`DELETE FROM event_log WHERE id IN (
			SELECT id FROM event_log
			WHERE published = true AND published_at < ?
			LIMIT ?
		)`, before, limit,
	)
	if result.Error != nil {
		return 0, fmt.Errorf("delete published events: %w", result.Error)
	}

	return result.RowsAffected, nil
}

// FindAndProcessUnpublished runs a transaction that locks unpublished events with FOR UPDATE SKIP LOCKED,
// then calls fn for each event. If fn succeeds, the event is marked as published within the same transaction.
func (r *eventLogRepository) FindAndProcessUnpublished(
	ctx context.Context,
	limit int,
	fn func(event *models.EventLog) error,
) (int, error) {
	db := r.pool.DB(ctx, false)
	published := 0

	txErr := db.Transaction(func(tx *gorm.DB) error {
		var eventList []*models.EventLog

		result := tx.Raw(
			`SELECT * FROM event_log
			 WHERE published = false AND deleted_at IS NULL
			 ORDER BY created_at
			 FOR UPDATE SKIP LOCKED
			 LIMIT ?`, limit,
		).Scan(&eventList)
		if result.Error != nil {
			return fmt.Errorf("find unpublished: %w", result.Error)
		}

		now := time.Now()

		for _, event := range eventList {
			if processErr := fn(event); processErr != nil {
				// Skip this event but continue processing others.
				continue
			}

			markResult := tx.Model(&models.EventLog{}).
				Where("id = ?", event.ID).
				Updates(map[string]any{
					"published":    true,
					"published_at": now,
				})
			if markResult.Error != nil {
				continue
			}

			published++
		}

		return nil
	})

	if txErr != nil {
		return 0, fmt.Errorf("outbox transaction: %w", txErr)
	}

	return published, nil
}
