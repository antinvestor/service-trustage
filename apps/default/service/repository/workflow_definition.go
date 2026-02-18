package repository

import (
	"context"
	"fmt"

	"github.com/pitabwire/frame/datastore"
	"github.com/pitabwire/frame/datastore/pool"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
)

// WorkflowDefinitionRepository manages workflow definition persistence.
type WorkflowDefinitionRepository interface {
	Create(ctx context.Context, def *models.WorkflowDefinition) error
	GetByID(ctx context.Context, id string) (*models.WorkflowDefinition, error)
	GetByNameAndVersion(ctx context.Context, tenantID, name string, version int) (*models.WorkflowDefinition, error)
	ListActiveByName(ctx context.Context, tenantID, name string) ([]*models.WorkflowDefinition, error)
	Update(ctx context.Context, def *models.WorkflowDefinition) error
}

type workflowDefinitionRepository struct {
	datastore.BaseRepository[*models.WorkflowDefinition]
}

// NewWorkflowDefinitionRepository creates a new WorkflowDefinitionRepository.
func NewWorkflowDefinitionRepository(dbPool pool.Pool) WorkflowDefinitionRepository {
	ctx := context.Background()

	return &workflowDefinitionRepository{
		BaseRepository: datastore.NewBaseRepository[*models.WorkflowDefinition](
			ctx,
			dbPool,
			nil,
			func() *models.WorkflowDefinition { return &models.WorkflowDefinition{} },
		),
	}
}

func (r *workflowDefinitionRepository) Create(ctx context.Context, def *models.WorkflowDefinition) error {
	return r.BaseRepository.Create(ctx, def)
}

func (r *workflowDefinitionRepository) GetByID(ctx context.Context, id string) (*models.WorkflowDefinition, error) {
	return r.BaseRepository.GetByID(ctx, id)
}

func (r *workflowDefinitionRepository) GetByNameAndVersion(
	ctx context.Context,
	tenantID, name string,
	version int,
) (*models.WorkflowDefinition, error) {
	db := r.BaseRepository.Pool().DB(ctx, true)

	var def models.WorkflowDefinition

	result := db.Where(
		"tenant_id = ? AND name = ? AND workflow_version = ? AND deleted_at IS NULL",
		tenantID, name, version,
	).First(&def)

	if result.Error != nil {
		return nil, fmt.Errorf("get workflow by name and version: %w", result.Error)
	}

	return &def, nil
}

func (r *workflowDefinitionRepository) ListActiveByName(
	ctx context.Context,
	tenantID, name string,
) ([]*models.WorkflowDefinition, error) {
	db := r.BaseRepository.Pool().DB(ctx, true)

	var defs []*models.WorkflowDefinition

	result := db.Where(
		"tenant_id = ? AND name = ? AND status = ? AND deleted_at IS NULL",
		tenantID, name, models.WorkflowStatusActive,
	).Order("workflow_version DESC").Find(&defs)

	if result.Error != nil {
		return nil, fmt.Errorf("list active workflows: %w", result.Error)
	}

	return defs, nil
}

func (r *workflowDefinitionRepository) Update(ctx context.Context, def *models.WorkflowDefinition) error {
	_, err := r.BaseRepository.Update(ctx, def)
	return err
}
