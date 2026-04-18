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
	"strings"

	"github.com/pitabwire/frame/datastore"
	"github.com/pitabwire/frame/datastore/pool"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
)

// WorkflowDefinitionRepository manages workflow definition persistence.
type WorkflowDefinitionRepository interface {
	Create(ctx context.Context, def *models.WorkflowDefinition) error
	GetByID(ctx context.Context, id string) (*models.WorkflowDefinition, error)
	GetByNameAndVersion(ctx context.Context, name string, version int) (*models.WorkflowDefinition, error)
	ListActiveByName(ctx context.Context, name string, limit int) ([]*models.WorkflowDefinition, error)
	ListPage(ctx context.Context, filter WorkflowDefinitionListFilter) (*WorkflowDefinitionPage, error)
	Update(ctx context.Context, def *models.WorkflowDefinition) error
}

type WorkflowDefinitionListFilter struct {
	Name    string
	Query   string
	IDQuery string
	Cursor  string
	Limit   int
}

type WorkflowDefinitionPage struct {
	Items      []*models.WorkflowDefinition
	NextCursor string
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
	name string,
	version int,
) (*models.WorkflowDefinition, error) {
	db := r.BaseRepository.Pool().DB(ctx, true)

	var def models.WorkflowDefinition

	result := db.Where(
		"name = ? AND workflow_version = ? AND deleted_at IS NULL",
		name, version,
	).First(&def)

	if result.Error != nil {
		return nil, fmt.Errorf("get workflow by name and version: %w", result.Error)
	}

	return &def, nil
}

func (r *workflowDefinitionRepository) ListActiveByName(
	ctx context.Context,
	name string,
	limit int,
) ([]*models.WorkflowDefinition, error) {
	page, err := r.ListPage(ctx, WorkflowDefinitionListFilter{
		Name:  name,
		Limit: limit,
	})
	if err != nil {
		return nil, err
	}

	return page.Items, nil
}

func (r *workflowDefinitionRepository) ListPage(
	ctx context.Context,
	filter WorkflowDefinitionListFilter,
) (*WorkflowDefinitionPage, error) {
	db := r.BaseRepository.Pool().DB(ctx, true)

	limit := normalizeListLimit(filter.Limit)

	var defs []*models.WorkflowDefinition

	query := db.Where("status = ? AND deleted_at IS NULL", models.WorkflowStatusActive)
	if filter.Name != "" {
		query = query.Where("name = ?", filter.Name)
	}
	if q := strings.TrimSpace(filter.IDQuery); q != "" {
		query = query.Where("id ILIKE ?", "%"+q+"%")
	}
	if q := strings.TrimSpace(filter.Query); q != "" {
		like := "%" + q + "%"
		query = query.Where("(id ILIKE ? OR name ILIKE ?)", like, like)
	}

	var err error
	query, err = applyDescendingCreatedAtCursor(query, filter.Cursor)
	if err != nil {
		return nil, fmt.Errorf("list workflow definitions: %w", err)
	}

	result := query.Order("created_at DESC, id DESC").Limit(limit + 1).Find(&defs)

	if result.Error != nil {
		return nil, fmt.Errorf("list active workflows: %w", result.Error)
	}

	nextCursor := ""
	if len(defs) > limit {
		last := defs[limit-1]
		nextCursor = encodeListCursor(last.CreatedAt, last.ID)
		defs = defs[:limit]
	}

	return &WorkflowDefinitionPage{
		Items:      defs,
		NextCursor: nextCursor,
	}, nil
}

func (r *workflowDefinitionRepository) Update(ctx context.Context, def *models.WorkflowDefinition) error {
	_, err := r.BaseRepository.Update(ctx, def)
	return err
}
