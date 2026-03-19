package repository

import (
	"context"
	"fmt"

	"github.com/pitabwire/frame/datastore"
	"github.com/pitabwire/frame/datastore/pool"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
)

// WorkflowOutputRepository manages workflow state output persistence.
type WorkflowOutputRepository interface {
	Store(ctx context.Context, output *models.WorkflowStateOutput) error
	GetByExecution(ctx context.Context, executionID string) (*models.WorkflowStateOutput, error)
	GetByInstanceAndState(ctx context.Context, instanceID, state string) (*models.WorkflowStateOutput, error)
	ListByInstance(ctx context.Context, instanceID string, limit int) ([]*models.WorkflowStateOutput, error)
}

type workflowOutputRepository struct {
	datastore.BaseRepository[*models.WorkflowStateOutput]
}

// NewWorkflowOutputRepository creates a new WorkflowOutputRepository.
func NewWorkflowOutputRepository(dbPool pool.Pool) WorkflowOutputRepository {
	ctx := context.Background()
	return &workflowOutputRepository{
		BaseRepository: datastore.NewBaseRepository[*models.WorkflowStateOutput](
			ctx,
			dbPool,
			nil,
			func() *models.WorkflowStateOutput { return &models.WorkflowStateOutput{} },
		),
	}
}

func (r *workflowOutputRepository) Store(ctx context.Context, output *models.WorkflowStateOutput) error {
	return r.BaseRepository.Create(ctx, output)
}

func (r *workflowOutputRepository) GetByExecution(
	ctx context.Context,
	executionID string,
) (*models.WorkflowStateOutput, error) {
	db := r.BaseRepository.Pool().DB(ctx, true)

	var output models.WorkflowStateOutput

	result := db.Where("execution_id = ? AND deleted_at IS NULL", executionID).First(&output)
	if result.Error != nil {
		return nil, fmt.Errorf("get output by execution: %w", result.Error)
	}

	return &output, nil
}

func (r *workflowOutputRepository) GetByInstanceAndState(
	ctx context.Context,
	instanceID, state string,
) (*models.WorkflowStateOutput, error) {
	db := r.BaseRepository.Pool().DB(ctx, true)

	var output models.WorkflowStateOutput

	result := db.Where("instance_id = ? AND state = ? AND deleted_at IS NULL", instanceID, state).
		Order("created_at DESC").
		First(&output)

	if result.Error != nil {
		return nil, fmt.Errorf("get output by instance and state: %w", result.Error)
	}

	return &output, nil
}

func (r *workflowOutputRepository) ListByInstance(
	ctx context.Context,
	instanceID string,
	limit int,
) ([]*models.WorkflowStateOutput, error) {
	db := r.BaseRepository.Pool().DB(ctx, true)

	if limit <= 0 {
		limit = 100
	}
	if limit > maxListLimit {
		limit = maxListLimit
	}

	var outputs []*models.WorkflowStateOutput
	result := db.Where("instance_id = ? AND deleted_at IS NULL", instanceID).
		Order("created_at ASC").
		Limit(limit).
		Find(&outputs)
	if result.Error != nil {
		return nil, fmt.Errorf("list outputs by instance: %w", result.Error)
	}

	return outputs, nil
}
