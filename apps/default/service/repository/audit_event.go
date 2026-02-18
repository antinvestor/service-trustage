package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/pitabwire/frame/datastore/pool"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
)

// AuditEventRepository manages append-only audit event persistence.
type AuditEventRepository interface {
	Append(ctx context.Context, event *models.WorkflowAuditEvent) error
	ListByInstance(ctx context.Context, tenantID, instanceID string) ([]*models.WorkflowAuditEvent, error)
	// DeleteBefore hard-deletes audit events older than the given time.
	DeleteBefore(ctx context.Context, before time.Time, limit int) (int64, error)
}

type auditEventRepository struct {
	pool pool.Pool
}

// NewAuditEventRepository creates a new AuditEventRepository.
func NewAuditEventRepository(dbPool pool.Pool) AuditEventRepository {
	return &auditEventRepository{pool: dbPool}
}

// Append inserts a single audit event (insert-only, never update).
func (r *auditEventRepository) Append(ctx context.Context, event *models.WorkflowAuditEvent) error {
	db := r.pool.DB(ctx, false)

	result := db.Create(event)
	if result.Error != nil {
		return fmt.Errorf("append audit event: %w", result.Error)
	}

	return nil
}

// DeleteBefore hard-deletes audit events older than the given time.
func (r *auditEventRepository) DeleteBefore(
	ctx context.Context,
	before time.Time,
	limit int,
) (int64, error) {
	db := r.pool.DB(ctx, false)

	result := db.Exec(
		`DELETE FROM workflow_audit_events WHERE id IN (
			SELECT id FROM workflow_audit_events
			WHERE created_at < ?
			LIMIT ?
		)`, before, limit,
	)
	if result.Error != nil {
		return 0, fmt.Errorf("delete old audit events: %w", result.Error)
	}

	return result.RowsAffected, nil
}

func (r *auditEventRepository) ListByInstance(
	ctx context.Context,
	tenantID, instanceID string,
) ([]*models.WorkflowAuditEvent, error) {
	db := r.pool.DB(ctx, true)

	var events []*models.WorkflowAuditEvent

	result := db.Where("tenant_id = ? AND instance_id = ?", tenantID, instanceID).
		Order("created_at ASC").
		Find(&events)

	if result.Error != nil {
		return nil, fmt.Errorf("list audit events: %w", result.Error)
	}

	return events, nil
}
