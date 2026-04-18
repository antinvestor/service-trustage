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

	"github.com/antinvestor/service-trustage/apps/queue/service/models"
)

// QueueDefinitionRepository manages queue definition persistence.
type QueueDefinitionRepository interface {
	Create(ctx context.Context, def *models.QueueDefinition) error
	GetByID(ctx context.Context, id string) (*models.QueueDefinition, error)
	GetByName(ctx context.Context, name string) (*models.QueueDefinition, error)
	List(ctx context.Context, activeOnly bool) ([]*models.QueueDefinition, error)
	Update(ctx context.Context, def *models.QueueDefinition) error
	SoftDelete(ctx context.Context, def *models.QueueDefinition) error
}

type queueDefinitionRepository struct {
	datastore.BaseRepository[*models.QueueDefinition]
}

// NewQueueDefinitionRepository creates a new QueueDefinitionRepository.
func NewQueueDefinitionRepository(dbPool pool.Pool) QueueDefinitionRepository {
	ctx := context.Background()

	return &queueDefinitionRepository{
		BaseRepository: datastore.NewBaseRepository[*models.QueueDefinition](
			ctx,
			dbPool,
			nil,
			func() *models.QueueDefinition { return &models.QueueDefinition{} },
		),
	}
}

func (r *queueDefinitionRepository) Create(ctx context.Context, def *models.QueueDefinition) error {
	return r.BaseRepository.Create(ctx, def)
}

func (r *queueDefinitionRepository) GetByID(ctx context.Context, id string) (*models.QueueDefinition, error) {
	return r.BaseRepository.GetByID(ctx, id)
}

func (r *queueDefinitionRepository) GetByName(
	ctx context.Context,
	name string,
) (*models.QueueDefinition, error) {
	db := r.BaseRepository.Pool().DB(ctx, true)

	var def models.QueueDefinition

	result := db.Where(
		"name = ? AND deleted_at IS NULL",
		name,
	).First(&def)

	if result.Error != nil {
		return nil, fmt.Errorf("get queue by name: %w", result.Error)
	}

	return &def, nil
}

func (r *queueDefinitionRepository) List(
	ctx context.Context,
	activeOnly bool,
) ([]*models.QueueDefinition, error) {
	db := r.BaseRepository.Pool().DB(ctx, true)

	query := db.Where("deleted_at IS NULL")

	if activeOnly {
		query = query.Where("active = ?", true)
	}

	var defs []*models.QueueDefinition

	result := query.Order("name ASC").Limit(maxListLimit).Find(&defs)
	if result.Error != nil {
		return nil, fmt.Errorf("list queue definitions: %w", result.Error)
	}

	return defs, nil
}

func (r *queueDefinitionRepository) Update(ctx context.Context, def *models.QueueDefinition) error {
	_, err := r.BaseRepository.Update(ctx, def)
	return err
}

func (r *queueDefinitionRepository) SoftDelete(ctx context.Context, def *models.QueueDefinition) error {
	db := r.BaseRepository.Pool().DB(ctx, false)

	result := db.Delete(def)
	if result.Error != nil {
		return fmt.Errorf("soft delete queue definition: %w", result.Error)
	}

	return nil
}
