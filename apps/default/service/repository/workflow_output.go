package repository

import (
	"context"
	"fmt"

	"github.com/pitabwire/frame/datastore/pool"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
)

// WorkflowOutputRepository manages workflow state output persistence.
type WorkflowOutputRepository interface {
	Store(ctx context.Context, output *models.WorkflowStateOutput) error
	GetByExecution(ctx context.Context, tenantID, executionID string) (*models.WorkflowStateOutput, error)
	GetByInstanceAndState(ctx context.Context, tenantID, instanceID, state string) (*models.WorkflowStateOutput, error)
}

type workflowOutputRepository struct {
	pool pool.Pool
}

// NewWorkflowOutputRepository creates a new WorkflowOutputRepository.
func NewWorkflowOutputRepository(dbPool pool.Pool) WorkflowOutputRepository {
	return &workflowOutputRepository{pool: dbPool}
}

func (r *workflowOutputRepository) Store(ctx context.Context, output *models.WorkflowStateOutput) error {
	db := r.pool.DB(ctx, false)

	result := db.Create(output)
	if result.Error != nil {
		return fmt.Errorf("store output: %w", result.Error)
	}

	return nil
}

func (r *workflowOutputRepository) GetByExecution(
	ctx context.Context,
	tenantID, executionID string,
) (*models.WorkflowStateOutput, error) {
	db := r.pool.DB(ctx, true)

	var output models.WorkflowStateOutput

	result := db.Where("tenant_id = ? AND execution_id = ?", tenantID, executionID).First(&output)
	if result.Error != nil {
		return nil, fmt.Errorf("get output by execution: %w", result.Error)
	}

	return &output, nil
}

func (r *workflowOutputRepository) GetByInstanceAndState(
	ctx context.Context,
	tenantID, instanceID, state string,
) (*models.WorkflowStateOutput, error) {
	db := r.pool.DB(ctx, true)

	var output models.WorkflowStateOutput

	result := db.Where("tenant_id = ? AND instance_id = ? AND state = ?", tenantID, instanceID, state).
		Order("created_at DESC").
		First(&output)

	if result.Error != nil {
		return nil, fmt.Errorf("get output by instance and state: %w", result.Error)
	}

	return &output, nil
}
