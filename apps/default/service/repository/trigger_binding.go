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
	"time"

	"github.com/pitabwire/frame/v3/datastore"
	"github.com/pitabwire/frame/v3/datastore/pool"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
)

// TriggerBindingRepository manages trigger binding persistence.
type TriggerBindingRepository interface {
	Create(ctx context.Context, binding *models.TriggerBinding) error
	// FindByEventType returns up to limit active, non-deleted trigger bindings
	// for the given event type. Tenancy-scoped via BaseRepository.
	//
	// The limit protects the event-router handler from unbounded fanout — a single
	// event type with thousands of bindings would otherwise lock a NATS message
	// handler long enough to trigger redelivery (consumer_ack_wait=10s), amplifying
	// the storm. Pass a positive integer; <= 0 is clamped to 200.
	FindByEventType(ctx context.Context, eventType string, limit int) ([]*models.TriggerBinding, error)
	// ReplaceForWorkflow makes bindings the only active bindings of the
	// workflow name: every binding of the workflow's other versions is
	// deactivated, the given bindings are upserted active for the target
	// version, and the target version's bindings not in the set are
	// deactivated. One transaction; idempotent.
	ReplaceForWorkflow(ctx context.Context, target WorkflowRoutingTarget, bindings []*models.TriggerBinding) error
	// DeactivateByWorkflowVersion deactivates every binding of the target version.
	DeactivateByWorkflowVersion(ctx context.Context, target WorkflowRoutingTarget) error
	// ListByWorkflow returns every binding (active or not) of a workflow name.
	ListByWorkflow(ctx context.Context, workflowName string) ([]*models.TriggerBinding, error)
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

const defaultFindByEventTypeLimit = 200

func (r *triggerBindingRepository) FindByEventType(
	ctx context.Context,
	eventType string,
	limit int,
) ([]*models.TriggerBinding, error) {
	if limit <= 0 {
		limit = defaultFindByEventTypeLimit
	}

	db := r.BaseRepository.Pool().DB(ctx, true)

	var bindings []*models.TriggerBinding

	result := db.Where(
		"event_type = ? AND active = true AND deleted_at IS NULL",
		eventType,
	).Order("id ASC").Limit(limit).Find(&bindings)

	if result.Error != nil {
		return nil, fmt.Errorf("find trigger bindings: %w", result.Error)
	}

	return bindings, nil
}

func (r *triggerBindingRepository) ReplaceForWorkflow(
	ctx context.Context,
	target WorkflowRoutingTarget,
	bindings []*models.TriggerBinding,
) error {
	now := time.Now().UTC()
	ids := make([]string, 0, len(bindings))

	for i, b := range bindings {
		stampDeterministicRow(&b.BaseModel, routingRowID("tb_", "trigger", target, i), target, now)
		b.WorkflowName = target.WorkflowName
		b.WorkflowVersion = target.WorkflowVersion
		b.Active = true
		if b.InputMapping == "" {
			b.InputMapping = "{}"
		}
		ids = append(ids, b.ID)
	}

	db := r.BaseRepository.Pool().DB(ctx, false)

	txErr := db.Transaction(func(tx *gorm.DB) error {
		otherVersions := tx.Model(&models.TriggerBinding{}).
			Where(
				"tenant_id = ? AND partition_id = ? AND workflow_name = ? AND workflow_version <> ? AND active = true AND deleted_at IS NULL",
				target.TenantID, target.PartitionID, target.WorkflowName, target.WorkflowVersion,
			).
			UpdateColumns(map[string]any{"active": false, "modified_at": now})
		if otherVersions.Error != nil {
			return fmt.Errorf("deactivate other versions: %w", otherVersions.Error)
		}

		if len(bindings) > 0 {
			upsert := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "id"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"event_type", "event_filter", "key_expr", "input_mapping",
					"active", "modified_at", "deleted_at",
				}),
			}).Create(&bindings)
			if upsert.Error != nil {
				return fmt.Errorf("upsert bindings: %w", upsert.Error)
			}
		}

		stale := tx.Model(&models.TriggerBinding{}).
			Where(
				"tenant_id = ? AND partition_id = ? AND workflow_name = ? AND workflow_version = ? AND active = true AND deleted_at IS NULL",
				target.TenantID, target.PartitionID, target.WorkflowName, target.WorkflowVersion,
			)
		if len(ids) > 0 {
			stale = stale.Where("id NOT IN ?", ids)
		}

		if staleErr := stale.UpdateColumns(map[string]any{"active": false, "modified_at": now}).Error; staleErr != nil {
			return fmt.Errorf("deactivate stale bindings: %w", staleErr)
		}

		return nil
	})
	if txErr != nil {
		return fmt.Errorf("replace trigger bindings for %s v%d: %w", target.WorkflowName, target.WorkflowVersion, txErr)
	}

	return nil
}

func (r *triggerBindingRepository) DeactivateByWorkflowVersion(
	ctx context.Context,
	target WorkflowRoutingTarget,
) error {
	db := r.BaseRepository.Pool().DB(ctx, false)

	result := db.Model(&models.TriggerBinding{}).
		Where(
			"tenant_id = ? AND partition_id = ? AND workflow_name = ? AND workflow_version = ? AND active = true AND deleted_at IS NULL",
			target.TenantID, target.PartitionID, target.WorkflowName, target.WorkflowVersion,
		).
		UpdateColumns(map[string]any{"active": false, "modified_at": time.Now().UTC()})
	if result.Error != nil {
		return fmt.Errorf("deactivate trigger bindings: %w", result.Error)
	}

	return nil
}

func (r *triggerBindingRepository) ListByWorkflow(
	ctx context.Context,
	workflowName string,
) ([]*models.TriggerBinding, error) {
	db := r.BaseRepository.Pool().DB(ctx, false)

	var bindings []*models.TriggerBinding

	result := db.Where("workflow_name = ? AND deleted_at IS NULL", workflowName).
		Order("workflow_version ASC, id ASC").
		Find(&bindings)
	if result.Error != nil {
		return nil, fmt.Errorf("list trigger bindings: %w", result.Error)
	}

	return bindings, nil
}
