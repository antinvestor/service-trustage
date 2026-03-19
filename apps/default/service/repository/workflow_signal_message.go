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

// WorkflowSignalMessageRepository manages signal message persistence.
type WorkflowSignalMessageRepository interface {
	Create(ctx context.Context, message *models.WorkflowSignalMessage) error
	ListByInstance(ctx context.Context, instanceID string, limit int) ([]*models.WorkflowSignalMessage, error)
	ClaimOldestPendingForTarget(
		ctx context.Context,
		instanceID, signalName, owner string,
		leaseUntil time.Time,
	) (*models.WorkflowSignalMessage, error)
	MarkDeliveredByOwner(ctx context.Context, id, owner, waitID string, deliveredAt time.Time) error
	ReleaseClaim(ctx context.Context, id, owner string) error
}

type workflowSignalMessageRepository struct {
	datastore.BaseRepository[*models.WorkflowSignalMessage]
	pool pool.Pool
}

// NewWorkflowSignalMessageRepository creates a new WorkflowSignalMessageRepository.
func NewWorkflowSignalMessageRepository(dbPool pool.Pool) WorkflowSignalMessageRepository {
	ctx := context.Background()

	return &workflowSignalMessageRepository{
		BaseRepository: datastore.NewBaseRepository[*models.WorkflowSignalMessage](
			ctx,
			dbPool,
			nil,
			func() *models.WorkflowSignalMessage { return &models.WorkflowSignalMessage{} },
		),
		pool: dbPool,
	}
}

func (r *workflowSignalMessageRepository) Create(ctx context.Context, message *models.WorkflowSignalMessage) error {
	return r.BaseRepository.Create(ctx, message)
}

func (r *workflowSignalMessageRepository) ListByInstance(
	ctx context.Context,
	instanceID string,
	limit int,
) ([]*models.WorkflowSignalMessage, error) {
	db := r.pool.DB(ctx, true)

	if limit <= 0 {
		limit = 100
	}
	if limit > maxListLimit {
		limit = maxListLimit
	}

	var messages []*models.WorkflowSignalMessage
	result := db.Where("target_instance_id = ? AND deleted_at IS NULL", instanceID).
		Order("created_at ASC").
		Limit(limit).
		Find(&messages)
	if result.Error != nil {
		return nil, fmt.Errorf("list signal messages by instance: %w", result.Error)
	}

	return messages, nil
}

func (r *workflowSignalMessageRepository) ClaimOldestPendingForTarget(
	ctx context.Context,
	instanceID, signalName, owner string,
	leaseUntil time.Time,
) (*models.WorkflowSignalMessage, error) {
	db := r.pool.DB(ctx, false)

	var message models.WorkflowSignalMessage
	txErr := db.Transaction(func(tx *gorm.DB) error {
		result := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where(
				"target_instance_id = ? AND signal_name = ? AND status = ? AND deleted_at IS NULL AND (claim_until IS NULL OR claim_until < ?)",
				instanceID,
				signalName,
				"pending",
				time.Now(),
			).
			Order("created_at ASC").
			First(&message)
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil
		}
		if result.Error != nil {
			return fmt.Errorf("claim signal message: %w", result.Error)
		}

		updateResult := tx.Model(&models.WorkflowSignalMessage{}).
			Where("id = ? AND deleted_at IS NULL", message.ID).
			UpdateColumns(map[string]any{
				"claim_owner": owner,
				"claim_until": leaseUntil,
				"attempts":    gorm.Expr("attempts + 1"),
			})
		if updateResult.Error != nil {
			return fmt.Errorf("claim signal message: %w", updateResult.Error)
		}

		message.ClaimOwner = owner
		message.ClaimUntil = &leaseUntil
		message.Attempts++

		return nil
	})
	if txErr != nil {
		return nil, txErr
	}

	if message.ID == "" {
		return nil, nil
	}

	return &message, nil
}

func (r *workflowSignalMessageRepository) MarkDeliveredByOwner(
	ctx context.Context,
	id, owner, waitID string,
	deliveredAt time.Time,
) error {
	db := r.pool.DB(ctx, false)

	result := db.Model(&models.WorkflowSignalMessage{}).
		Where("id = ? AND claim_owner = ? AND status = ? AND deleted_at IS NULL", id, owner, "pending").
		UpdateColumns(map[string]any{
			"status":       "delivered",
			"delivered_at": deliveredAt,
			"wait_id":      waitID,
			"claim_owner":  "",
			"claim_until":  gorm.Expr("NULL"),
		})
	if result.Error != nil {
		return fmt.Errorf("mark signal message delivered: %w", result.Error)
	}

	return nil
}

func (r *workflowSignalMessageRepository) ReleaseClaim(ctx context.Context, id, owner string) error {
	db := r.pool.DB(ctx, false)

	result := db.Model(&models.WorkflowSignalMessage{}).
		Where("id = ? AND claim_owner = ? AND status = ? AND deleted_at IS NULL", id, owner, "pending").
		UpdateColumns(map[string]any{
			"claim_owner": "",
			"claim_until": gorm.Expr("NULL"),
		})
	if result.Error != nil {
		return fmt.Errorf("release signal message claim: %w", result.Error)
	}

	return nil
}
