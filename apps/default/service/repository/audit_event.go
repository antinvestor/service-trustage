package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/pitabwire/frame/datastore"
	"github.com/pitabwire/frame/datastore/pool"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
)

// AuditEventRepository manages append-only audit event persistence.
type AuditEventRepository interface {
	Append(ctx context.Context, event *models.WorkflowAuditEvent) error
	ListByInstance(ctx context.Context, instanceID string) ([]*models.WorkflowAuditEvent, error)
	ListByInstanceWithLimit(ctx context.Context, instanceID string, limit int) ([]*models.WorkflowAuditEvent, error)
	// DeleteBefore soft-deletes audit events older than the given time.
	DeleteBefore(ctx context.Context, before time.Time, limit int) (int64, error)
}

type auditEventRepository struct {
	datastore.BaseRepository[*models.WorkflowAuditEvent]
}

// NewAuditEventRepository creates a new AuditEventRepository.
func NewAuditEventRepository(dbPool pool.Pool) AuditEventRepository {
	ctx := context.Background()
	return &auditEventRepository{
		BaseRepository: datastore.NewBaseRepository[*models.WorkflowAuditEvent](
			ctx,
			dbPool,
			nil,
			func() *models.WorkflowAuditEvent { return &models.WorkflowAuditEvent{} },
		),
	}
}

// Append inserts a single audit event (insert-only, never update).
func (r *auditEventRepository) Append(ctx context.Context, event *models.WorkflowAuditEvent) error {
	return r.BaseRepository.Create(ctx, event)
}

// DeleteBefore soft-deletes audit events older than the given time.
func (r *auditEventRepository) DeleteBefore(
	ctx context.Context,
	before time.Time,
	limit int,
) (int64, error) {
	db := r.BaseRepository.Pool().DB(ctx, false)
	if limit <= 0 {
		limit = 100
	}

	var ids []string
	selectResult := db.Model(&models.WorkflowAuditEvent{}).
		Where("created_at < ? AND deleted_at IS NULL", before).
		Limit(limit).
		Pluck("id", &ids)
	if selectResult.Error != nil {
		return 0, fmt.Errorf("select old audit events: %w", selectResult.Error)
	}
	if len(ids) == 0 {
		return 0, nil
	}

	if err := r.BaseRepository.DeleteBatch(ctx, ids); err != nil {
		return 0, fmt.Errorf("delete old audit events: %w", err)
	}

	return int64(len(ids)), nil
}

func (r *auditEventRepository) ListByInstance(
	ctx context.Context,
	instanceID string,
) ([]*models.WorkflowAuditEvent, error) {
	return r.ListByInstanceWithLimit(ctx, instanceID, maxListLimit)
}

func (r *auditEventRepository) ListByInstanceWithLimit(
	ctx context.Context,
	instanceID string,
	limit int,
) ([]*models.WorkflowAuditEvent, error) {
	db := r.BaseRepository.Pool().DB(ctx, true)

	if limit <= 0 {
		limit = 100
	}
	if limit > maxListLimit {
		limit = maxListLimit
	}

	var events []*models.WorkflowAuditEvent

	result := db.Where("instance_id = ? AND deleted_at IS NULL", instanceID).
		Order("created_at ASC").
		Limit(limit).
		Find(&events)

	if result.Error != nil {
		return nil, fmt.Errorf("list audit events: %w", result.Error)
	}

	return events, nil
}
