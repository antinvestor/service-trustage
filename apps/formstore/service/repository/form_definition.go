package repository

import (
	"context"
	"fmt"

	"github.com/pitabwire/frame/datastore"
	"github.com/pitabwire/frame/datastore/pool"

	"github.com/antinvestor/service-trustage/apps/formstore/service/models"
)

// maxListLimit caps the maximum number of items returned by list queries.
const maxListLimit = 200

// FormDefinitionRepository manages form definition persistence.
type FormDefinitionRepository interface {
	Create(ctx context.Context, def *models.FormDefinition) error
	GetByID(ctx context.Context, id string) (*models.FormDefinition, error)
	GetByFormID(ctx context.Context, formID string) (*models.FormDefinition, error)
	List(ctx context.Context, activeOnly bool, limit, offset int) ([]*models.FormDefinition, error)
	Update(ctx context.Context, def *models.FormDefinition) error
	SoftDelete(ctx context.Context, def *models.FormDefinition) error
}

type formDefinitionRepository struct {
	datastore.BaseRepository[*models.FormDefinition]
}

// NewFormDefinitionRepository creates a new FormDefinitionRepository.
func NewFormDefinitionRepository(dbPool pool.Pool) FormDefinitionRepository {
	ctx := context.Background()

	return &formDefinitionRepository{
		BaseRepository: datastore.NewBaseRepository[*models.FormDefinition](
			ctx,
			dbPool,
			nil,
			func() *models.FormDefinition { return &models.FormDefinition{} },
		),
	}
}

func (r *formDefinitionRepository) Create(ctx context.Context, def *models.FormDefinition) error {
	return r.BaseRepository.Create(ctx, def)
}

func (r *formDefinitionRepository) GetByID(ctx context.Context, id string) (*models.FormDefinition, error) {
	return r.BaseRepository.GetByID(ctx, id)
}

func (r *formDefinitionRepository) GetByFormID(
	ctx context.Context,
	formID string,
) (*models.FormDefinition, error) {
	db := r.BaseRepository.Pool().DB(ctx, true)

	var def models.FormDefinition

	result := db.Where(
		"form_id = ? AND deleted_at IS NULL",
		formID,
	).First(&def)

	if result.Error != nil {
		return nil, fmt.Errorf("get form definition by form_id: %w", result.Error)
	}

	return &def, nil
}

func (r *formDefinitionRepository) List(
	ctx context.Context,
	activeOnly bool,
	limit, offset int,
) ([]*models.FormDefinition, error) {
	db := r.BaseRepository.Pool().DB(ctx, true)

	query := db.Where("deleted_at IS NULL")

	if activeOnly {
		query = query.Where("active = ?", true)
	}

	if limit <= 0 {
		limit = 20
	}

	if limit > maxListLimit {
		limit = maxListLimit
	}

	if offset < 0 {
		offset = 0
	}

	var defs []*models.FormDefinition

	result := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&defs)
	if result.Error != nil {
		return nil, fmt.Errorf("list form definitions: %w", result.Error)
	}

	return defs, nil
}

func (r *formDefinitionRepository) Update(ctx context.Context, def *models.FormDefinition) error {
	_, err := r.BaseRepository.Update(ctx, def)
	return err
}

func (r *formDefinitionRepository) SoftDelete(ctx context.Context, def *models.FormDefinition) error {
	db := r.BaseRepository.Pool().DB(ctx, false)

	result := db.Delete(def)
	if result.Error != nil {
		return fmt.Errorf("soft delete form definition: %w", result.Error)
	}

	return nil
}
