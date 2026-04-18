// Copyright 2023-2026 Ant Investor Ltd
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	FindByEventType(ctx context.Context, eventType string) ([]*models.TriggerBinding, error)
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
	eventType string,
) ([]*models.TriggerBinding, error) {
	db := r.BaseRepository.Pool().DB(ctx, true)

	var bindings []*models.TriggerBinding

	result := db.Where(
		"event_type = ? AND active = true AND deleted_at IS NULL",
		eventType,
	).Find(&bindings)

	if result.Error != nil {
		return nil, fmt.Errorf("find trigger bindings: %w", result.Error)
	}

	return bindings, nil
}
