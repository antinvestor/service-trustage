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

const outboxLeaseTTL = 30 * time.Second

// EventLogRepository manages event log persistence for the outbox pattern.
type EventLogRepository interface {
	Create(ctx context.Context, event *models.EventLog) error
	FindByIdempotencyKey(ctx context.Context, key string) (*models.EventLog, error)
	FindUnpublished(ctx context.Context, limit int) ([]*models.EventLog, error)
	MarkPublished(ctx context.Context, id string) error
	ClaimUnpublished(ctx context.Context, limit int, owner string, leaseUntil time.Time) ([]*models.EventLog, error)
	MarkPublishedByOwner(ctx context.Context, id string, owner string, publishedAt time.Time) error
	ReleaseClaim(ctx context.Context, id string, owner string) error
	FindAndProcessUnpublished(ctx context.Context, limit int, fn func(event *models.EventLog) error) (int, error)
	// DeletePublishedBefore soft-deletes published events older than the given time.
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
	key string,
) (*models.EventLog, error) {
	db := r.pool.DB(ctx, true)

	var event models.EventLog

	result := db.Where("idempotency_key = ? AND deleted_at IS NULL", key).First(&event)
	if result.Error != nil {
		return nil, fmt.Errorf("find by idempotency key: %w", result.Error)
	}

	return &event, nil
}

// FindUnpublished finds unpublished events using FOR UPDATE SKIP LOCKED.
func (r *eventLogRepository) FindUnpublished(ctx context.Context, limit int) ([]*models.EventLog, error) {
	db := r.pool.DB(ctx, false)
	if limit <= 0 {
		limit = 50
	}

	var events []*models.EventLog
	result := db.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("published = ? AND deleted_at IS NULL", false).
		Order("created_at ASC").
		Limit(limit).
		Find(&events)

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

func (r *eventLogRepository) ClaimUnpublished(
	ctx context.Context,
	limit int,
	owner string,
	leaseUntil time.Time,
) ([]*models.EventLog, error) {
	db := r.pool.DB(ctx, false)

	claimedAt := time.Now()
	var events []*models.EventLog

	txErr := db.Transaction(func(tx *gorm.DB) error {
		if limit <= 0 {
			limit = 50
		}

		result := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where(
				"published = ? AND deleted_at IS NULL AND (publish_claim_until IS NULL OR publish_claim_until < ?)",
				false,
				claimedAt,
			).
			Order("created_at ASC").
			Limit(limit).
			Find(&events)
		if result.Error != nil {
			return fmt.Errorf("claim unpublished: %w", result.Error)
		}
		if len(events) == 0 {
			return nil
		}

		ids := make([]string, 0, len(events))
		for _, event := range events {
			ids = append(ids, event.ID)
		}

		updateResult := tx.Model(&models.EventLog{}).
			Where("id IN ? AND deleted_at IS NULL", ids).
			UpdateColumns(map[string]any{
				"publish_claim_owner": owner,
				"publish_claim_until": leaseUntil,
				"publish_attempts":    gorm.Expr("publish_attempts + 1"),
			})
		if updateResult.Error != nil {
			return fmt.Errorf("claim unpublished: %w", updateResult.Error)
		}
		if updateResult.RowsAffected == 0 {
			return errors.New("claim unpublished: no rows updated")
		}

		for _, event := range events {
			event.PublishClaimOwner = owner
			event.PublishClaimUntil = &leaseUntil
			event.PublishAttempts++
		}

		return nil
	})
	if txErr != nil {
		return nil, txErr
	}

	return events, nil
}

func (r *eventLogRepository) MarkPublishedByOwner(
	ctx context.Context,
	id string,
	owner string,
	publishedAt time.Time,
) error {
	db := r.pool.DB(ctx, false)

	result := db.Model(&models.EventLog{}).
		Where("id = ? AND publish_claim_owner = ? AND deleted_at IS NULL", id, owner).
		UpdateColumns(map[string]any{
			"published":           true,
			"published_at":        publishedAt,
			"publish_claim_owner": "",
			"publish_claim_until": gorm.Expr("NULL"),
		})
	if result.Error != nil {
		return fmt.Errorf("mark published by owner: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("mark published by owner: no rows updated")
	}

	return nil
}

func (r *eventLogRepository) ReleaseClaim(ctx context.Context, id string, owner string) error {
	db := r.pool.DB(ctx, false)

	result := db.Model(&models.EventLog{}).
		Where("id = ? AND publish_claim_owner = ? AND published = false AND deleted_at IS NULL", id, owner).
		UpdateColumns(map[string]any{
			"publish_claim_owner": "",
			"publish_claim_until": gorm.Expr("NULL"),
		})
	if result.Error != nil {
		return fmt.Errorf("release event claim: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("release event claim: no rows updated")
	}

	return nil
}

// FindAndProcessUnpublished exists for compatibility with older tests/callers.
// New code should use ClaimUnpublished plus explicit publish/ack flow.
func (r *eventLogRepository) FindAndProcessUnpublished(
	ctx context.Context,
	limit int,
	fn func(event *models.EventLog) error,
) (int, error) {
	owner := "compat-processor"
	claimed, err := r.ClaimUnpublished(ctx, limit, owner, time.Now().Add(outboxLeaseTTL))
	if err != nil {
		return 0, fmt.Errorf("claim unpublished for processing: %w", err)
	}

	processed := 0
	now := time.Now()

	for _, event := range claimed {
		if processErr := fn(event); processErr != nil {
			_ = r.ReleaseClaim(ctx, event.ID, owner)
			continue
		}

		if markErr := r.MarkPublishedByOwner(ctx, event.ID, owner, now); markErr != nil {
			_ = r.ReleaseClaim(ctx, event.ID, owner)
			continue
		}

		processed++
	}

	return processed, nil
}

// DeletePublishedBefore soft-deletes published events older than the given time.
func (r *eventLogRepository) DeletePublishedBefore(
	ctx context.Context,
	before time.Time,
	limit int,
) (int64, error) {
	db := r.pool.DB(ctx, false)
	if limit <= 0 {
		limit = 100
	}

	var ids []string
	selectResult := db.Model(&models.EventLog{}).
		Where("published = ? AND published_at < ? AND deleted_at IS NULL", true, before).
		Limit(limit).
		Pluck("id", &ids)
	if selectResult.Error != nil {
		return 0, fmt.Errorf("select published events for delete: %w", selectResult.Error)
	}
	if len(ids) == 0 {
		return 0, nil
	}

	if err := r.BaseRepository.DeleteBatch(ctx, ids); err != nil {
		return 0, fmt.Errorf("delete published events: %w", err)
	}

	return int64(len(ids)), nil
}
