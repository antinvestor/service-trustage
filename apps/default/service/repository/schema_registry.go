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
	"errors"
	"fmt"
	"strings"

	"github.com/pitabwire/frame/datastore"
	"github.com/pitabwire/frame/datastore/pool"
	"gorm.io/gorm"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
)

// SchemaRegistryRepository manages immutable schema storage.
type SchemaRegistryRepository interface {
	Store(ctx context.Context, schema *models.WorkflowStateSchema) error
	Lookup(
		ctx context.Context,
		workflowName string,
		version int,
		state string,
		schemaType models.SchemaType,
	) (*models.WorkflowStateSchema, error)
	LookupByHash(ctx context.Context, hash string) (*models.WorkflowStateSchema, error)
}

type schemaRegistryRepository struct {
	datastore.BaseRepository[*models.WorkflowStateSchema]
}

// NewSchemaRegistryRepository creates a new SchemaRegistryRepository.
func NewSchemaRegistryRepository(dbPool pool.Pool) SchemaRegistryRepository {
	ctx := context.Background()
	return &schemaRegistryRepository{
		BaseRepository: datastore.NewBaseRepository[*models.WorkflowStateSchema](
			ctx,
			dbPool,
			nil,
			func() *models.WorkflowStateSchema { return &models.WorkflowStateSchema{} },
		),
	}
}

// Store upserts a schema (immutable by hash — if the schema already exists, it's a no-op).
func (r *schemaRegistryRepository) Store(ctx context.Context, schema *models.WorkflowStateSchema) error {
	db := r.BaseRepository.Pool().DB(ctx, false)

	if err := db.Create(schema).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(err.Error(), "duplicate") {
			return nil
		}
		return fmt.Errorf("store schema: %w", err)
	}

	return nil
}

func (r *schemaRegistryRepository) Lookup(
	ctx context.Context,
	workflowName string,
	version int,
	state string,
	schemaType models.SchemaType,
) (*models.WorkflowStateSchema, error) {
	db := r.BaseRepository.Pool().DB(ctx, true)

	var schema models.WorkflowStateSchema

	result := db.Where(
		"workflow_name = ? AND workflow_version = ? AND state = ? AND schema_type = ?",
		workflowName, version, state, schemaType,
	).First(&schema)

	if result.Error != nil {
		return nil, fmt.Errorf("lookup schema: %w", result.Error)
	}

	return &schema, nil
}

func (r *schemaRegistryRepository) LookupByHash(
	ctx context.Context,
	hash string,
) (*models.WorkflowStateSchema, error) {
	db := r.BaseRepository.Pool().DB(ctx, true)

	var schema models.WorkflowStateSchema

	result := db.Where("schema_hash = ?", hash).First(&schema)
	if result.Error != nil {
		return nil, fmt.Errorf("lookup schema by hash: %w", result.Error)
	}

	return &schema, nil
}
