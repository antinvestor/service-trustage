package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/pitabwire/frame/datastore"
	"github.com/pitabwire/frame/datastore/pool"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
)

// WorkflowInstanceRepository manages workflow instance persistence.
type WorkflowInstanceRepository interface {
	Create(ctx context.Context, inst *models.WorkflowInstance) error
	GetByID(ctx context.Context, id string) (*models.WorkflowInstance, error)
	List(ctx context.Context, status, workflowName string, limit int) ([]*models.WorkflowInstance, error)
	CASTransition(
		ctx context.Context,
		instanceID, expectedState string,
		expectedRevision int64,
		newState string,
	) error
	UpdateStatus(ctx context.Context, instanceID string, status models.WorkflowInstanceStatus) error
}

type workflowInstanceRepository struct {
	datastore.BaseRepository[*models.WorkflowInstance]
}

// NewWorkflowInstanceRepository creates a new WorkflowInstanceRepository.
func NewWorkflowInstanceRepository(dbPool pool.Pool) WorkflowInstanceRepository {
	ctx := context.Background()

	return &workflowInstanceRepository{
		BaseRepository: datastore.NewBaseRepository[*models.WorkflowInstance](
			ctx,
			dbPool,
			nil,
			func() *models.WorkflowInstance { return &models.WorkflowInstance{} },
		),
	}
}

func (r *workflowInstanceRepository) Create(ctx context.Context, inst *models.WorkflowInstance) error {
	return r.BaseRepository.Create(ctx, inst)
}

func (r *workflowInstanceRepository) GetByID(ctx context.Context, id string) (*models.WorkflowInstance, error) {
	return r.BaseRepository.GetByID(ctx, id)
}

func (r *workflowInstanceRepository) List(
	ctx context.Context,
	status, workflowName string,
	limit int,
) ([]*models.WorkflowInstance, error) {
	db := r.BaseRepository.Pool().DB(ctx, true)

	if limit <= 0 {
		limit = 50
	}

	if limit > maxListLimit {
		limit = maxListLimit
	}

	query := db.Where("deleted_at IS NULL")
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if workflowName != "" {
		query = query.Where("workflow_name = ?", workflowName)
	}

	var instances []*models.WorkflowInstance
	result := query.Order("created_at DESC").Limit(limit).Find(&instances)
	if result.Error != nil {
		return nil, fmt.Errorf("list instances: %w", result.Error)
	}

	return instances, nil
}

// CASTransition performs a Compare-And-Swap state transition.
// Returns nil on success, error if zero rows affected (stale) or DB error.
func (r *workflowInstanceRepository) CASTransition(
	ctx context.Context,
	instanceID, expectedState string,
	expectedRevision int64,
	newState string,
) error {
	db := r.BaseRepository.Pool().DB(ctx, false)

	result := db.Exec(
		`UPDATE workflow_instances
		 SET current_state = ?, revision = revision + 1, modified_at = ?
		 WHERE id = ? AND current_state = ? AND revision = ? AND status = 'running'`,
		newState, time.Now(), instanceID, expectedState, expectedRevision,
	)

	if result.Error != nil {
		return fmt.Errorf("CAS transition: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.New("CAS transition failed: stale revision or unexpected state")
	}

	return nil
}

func (r *workflowInstanceRepository) UpdateStatus(
	ctx context.Context,
	instanceID string,
	status models.WorkflowInstanceStatus,
) error {
	db := r.BaseRepository.Pool().DB(ctx, false)

	now := time.Now()
	updates := map[string]any{
		"status":      string(status),
		"modified_at": now,
	}

	if status == models.InstanceStatusCompleted || status == models.InstanceStatusFailed ||
		status == models.InstanceStatusCancelled {
		updates["finished_at"] = now
	}

	result := db.Model(&models.WorkflowInstance{}).
		Where("id = ? AND deleted_at IS NULL", instanceID).
		Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("update instance status: %w", result.Error)
	}

	return nil
}
