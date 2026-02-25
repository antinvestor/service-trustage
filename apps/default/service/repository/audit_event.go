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

	result := db.Exec(
		`UPDATE workflow_audit_events
		 SET deleted_at = NOW()
		 WHERE id IN (
		 	SELECT id FROM workflow_audit_events
		 	WHERE created_at < ? AND deleted_at IS NULL
		 	LIMIT ?
		 )`,
		before, limit,
	)
	if result.Error != nil {
		return 0, fmt.Errorf("delete old audit events: %w", result.Error)
	}

	return result.RowsAffected, nil
}

func (r *auditEventRepository) ListByInstance(
	ctx context.Context,
	instanceID string,
) ([]*models.WorkflowAuditEvent, error) {
	db := r.BaseRepository.Pool().DB(ctx, true)

	var events []*models.WorkflowAuditEvent

	result := db.Where("instance_id = ? AND deleted_at IS NULL", instanceID).
		Order("created_at ASC").
		Find(&events)

	if result.Error != nil {
		return nil, fmt.Errorf("list audit events: %w", result.Error)
	}

	return events, nil
}
