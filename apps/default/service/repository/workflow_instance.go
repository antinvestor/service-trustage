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
	"time"

	"github.com/pitabwire/frame/datastore"
	"github.com/pitabwire/frame/datastore/pool"
	"gorm.io/gorm"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
)

// WorkflowInstanceRepository manages workflow instance persistence.
type WorkflowInstanceRepository interface {
	Create(ctx context.Context, inst *models.WorkflowInstance) error
	GetByID(ctx context.Context, id string) (*models.WorkflowInstance, error)
	ListByParentExecutionID(ctx context.Context, parentExecutionID string) ([]*models.WorkflowInstance, error)
	FindByTriggerEvent(
		ctx context.Context,
		workflowName string,
		workflowVersion int,
		triggerEventID string,
	) (*models.WorkflowInstance, error)
	List(ctx context.Context, status, workflowName string, limit int) ([]*models.WorkflowInstance, error)
	ListPage(ctx context.Context, filter WorkflowInstanceListFilter) (*WorkflowInstancePage, error)
	CASTransition(
		ctx context.Context,
		instanceID, expectedState string,
		expectedRevision int64,
		newState string,
	) error
	UpdateStatus(ctx context.Context, instanceID string, status models.WorkflowInstanceStatus) error
}

type WorkflowInstanceListFilter struct {
	Status            string
	WorkflowName      string
	Query             string
	IDQuery           string
	ParentInstanceID  string
	ParentExecutionID string
	Cursor            string
	Limit             int
}

type WorkflowInstancePage struct {
	Items      []*models.WorkflowInstance
	NextCursor string
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

func (r *workflowInstanceRepository) ListByParentExecutionID(
	ctx context.Context,
	parentExecutionID string,
) ([]*models.WorkflowInstance, error) {
	db := r.BaseRepository.Pool().DB(ctx, true)

	var children []*models.WorkflowInstance
	result := db.Model(&models.WorkflowInstance{}).
		Where("parent_execution_id = ? AND deleted_at IS NULL", parentExecutionID).
		Order("scope_index ASC").
		Find(&children)
	if result.Error != nil {
		return nil, fmt.Errorf("list instances by parent execution: %w", result.Error)
	}

	return children, nil
}

func (r *workflowInstanceRepository) FindByTriggerEvent(
	ctx context.Context,
	workflowName string,
	workflowVersion int,
	triggerEventID string,
) (*models.WorkflowInstance, error) {
	db := r.BaseRepository.Pool().DB(ctx, true)

	var instance models.WorkflowInstance

	result := db.Where(
		"workflow_name = ? AND workflow_version = ? AND trigger_event_id = ? AND deleted_at IS NULL",
		workflowName, workflowVersion, triggerEventID,
	).First(&instance)
	if result.Error != nil {
		return nil, fmt.Errorf("find instance by trigger event: %w", result.Error)
	}

	return &instance, nil
}

func (r *workflowInstanceRepository) List(
	ctx context.Context,
	status, workflowName string,
	limit int,
) ([]*models.WorkflowInstance, error) {
	page, err := r.ListPage(ctx, WorkflowInstanceListFilter{
		Status:       status,
		WorkflowName: workflowName,
		Limit:        limit,
	})
	if err != nil {
		return nil, err
	}

	return page.Items, nil
}

func (r *workflowInstanceRepository) ListPage(
	ctx context.Context,
	filter WorkflowInstanceListFilter,
) (*WorkflowInstancePage, error) {
	db := r.BaseRepository.Pool().DB(ctx, true)

	limit := normalizeListLimit(filter.Limit)

	query := db.Where("deleted_at IS NULL")
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.WorkflowName != "" {
		query = query.Where("workflow_name = ?", filter.WorkflowName)
	}
	if filter.ParentInstanceID != "" {
		query = query.Where("parent_instance_id = ?", filter.ParentInstanceID)
	}
	if filter.ParentExecutionID != "" {
		query = query.Where("parent_execution_id = ?", filter.ParentExecutionID)
	}
	if q := strings.TrimSpace(filter.IDQuery); q != "" {
		query = query.Where("id ILIKE ?", "%"+q+"%")
	}
	if q := strings.TrimSpace(filter.Query); q != "" {
		like := "%" + q + "%"
		query = query.Where(
			"(id ILIKE ? OR workflow_name ILIKE ? OR current_state ILIKE ? OR status ILIKE ? OR trigger_event_id ILIKE ? OR parent_instance_id ILIKE ? OR parent_execution_id ILIKE ?)",
			like,
			like,
			like,
			like,
			like,
			like,
			like,
		)
	}

	var err error
	query, err = applyDescendingCreatedAtCursor(query, filter.Cursor)
	if err != nil {
		return nil, fmt.Errorf("list instances: %w", err)
	}

	var instances []*models.WorkflowInstance
	result := query.Order("created_at DESC, id DESC").Limit(limit + 1).Find(&instances)
	if result.Error != nil {
		return nil, fmt.Errorf("list instances: %w", result.Error)
	}

	nextCursor := ""
	if len(instances) > limit {
		last := instances[limit-1]
		nextCursor = encodeListCursor(last.CreatedAt, last.ID)
		instances = instances[:limit]
	}

	return &WorkflowInstancePage{
		Items:      instances,
		NextCursor: nextCursor,
	}, nil
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
	result := db.Model(&models.WorkflowInstance{}).
		Where(
			"id = ? AND current_state = ? AND revision = ? AND status = ? AND deleted_at IS NULL",
			instanceID,
			expectedState,
			expectedRevision,
			models.InstanceStatusRunning,
		).
		UpdateColumns(map[string]any{
			"current_state": newState,
			"revision":      gorm.Expr("revision + 1"),
			"modified_at":   time.Now(),
		})

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
		UpdateColumns(updates)
	if result.Error != nil {
		return fmt.Errorf("update instance status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("update instance status: no rows updated")
	}

	return nil
}
