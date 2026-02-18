package repository

import (
	"context"
	"fmt"

	"github.com/pitabwire/frame/datastore"
	"github.com/pitabwire/frame/datastore/pool"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
)

// TriggerBindingRepository manages trigger binding persistence.
type TriggerBindingRepository interface {
	Create(ctx context.Context, binding *models.TriggerBinding) error
	FindByEventType(ctx context.Context, tenantID, eventType string) ([]*models.TriggerBinding, error)
}

type triggerBindingRepository struct {
	datastore.BaseRepository[*models.TriggerBinding]
}

// NewTriggerBindingRepository creates a new TriggerBindingRepository.
func NewTriggerBindingRepository(dbPool pool.Pool) TriggerBindingRepository {
	ctx := context.Background()

	return &triggerBindingRepository{
		BaseRepository: datastore.NewBaseRepository[*models.TriggerBinding](
			ctx,
			dbPool,
			nil,
			func() *models.TriggerBinding { return &models.TriggerBinding{} },
		),
	}
}

func (r *triggerBindingRepository) Create(ctx context.Context, binding *models.TriggerBinding) error {
	return r.BaseRepository.Create(ctx, binding)
}

func (r *triggerBindingRepository) FindByEventType(
	ctx context.Context,
	tenantID, eventType string,
) ([]*models.TriggerBinding, error) {
	db := r.BaseRepository.Pool().DB(ctx, true)

	var bindings []*models.TriggerBinding

	result := db.Where(
		"tenant_id = ? AND event_type = ? AND active = true AND deleted_at IS NULL",
		tenantID, eventType,
	).Find(&bindings)

	if result.Error != nil {
		return nil, fmt.Errorf("find trigger bindings: %w", result.Error)
	}

	return bindings, nil
}
