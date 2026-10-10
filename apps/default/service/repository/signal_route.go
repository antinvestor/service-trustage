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

	"github.com/pitabwire/frame/v2/datastore"
	"github.com/pitabwire/frame/v2/datastore/pool"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
)

// SignalRouteRepository manages signal route persistence.
type SignalRouteRepository interface {
	// ReplaceForWorkflowVersion makes routes the active signal routes of the
	// target version: they are upserted active and the version's routes not
	// in the set are deactivated. Other versions' routes are untouched —
	// they stay in force for that version's running instances. Idempotent.
	ReplaceForWorkflowVersion(ctx context.Context, target WorkflowRoutingTarget, routes []*models.SignalRoute) error
	// FindActiveByEventType returns up to limit active routes for the event
	// type (<= 0 is clamped to 200). Tenancy-scoped.
	FindActiveByEventType(ctx context.Context, eventType string, limit int) ([]*models.SignalRoute, error)
}

type signalRouteRepository struct {
	datastore.BaseRepository[*models.SignalRoute]
}

// NewSignalRouteRepository creates a new SignalRouteRepository.
func NewSignalRouteRepository(dbPool pool.Pool) SignalRouteRepository {
	ctx := context.Background()

	return &signalRouteRepository{
		BaseRepository: datastore.NewBaseRepository[*models.SignalRoute](
			ctx,
			dbPool,
			nil,
			func() *models.SignalRoute { return &models.SignalRoute{} },
		),
	}
}

func (r *signalRouteRepository) ReplaceForWorkflowVersion(
	ctx context.Context,
	target WorkflowRoutingTarget,
	routes []*models.SignalRoute,
) error {
	now := time.Now().UTC()
	ids := make([]string, 0, len(routes))

	for i, route := range routes {
		stampDeterministicRow(&route.BaseModel, routingRowID("sr_", "signal", target, i), target, now)
		route.WorkflowName = target.WorkflowName
		route.WorkflowVersion = target.WorkflowVersion
		route.Active = true
		ids = append(ids, route.ID)
	}

	db := r.BaseRepository.Pool().DB(ctx, false)

	txErr := db.Transaction(func(tx *gorm.DB) error {
		if len(routes) > 0 {
			upsert := tx.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "id"}},
				DoUpdates: clause.AssignmentColumns([]string{
					"event_type", "signal_name", "filter", "key_expr", "instance_expr",
					"active", "modified_at", "deleted_at",
				}),
			}).Create(&routes)
			if upsert.Error != nil {
				return fmt.Errorf("upsert signal routes: %w", upsert.Error)
			}
		}

		stale := tx.Model(&models.SignalRoute{}).
			Where(
				"tenant_id = ? AND partition_id = ? AND workflow_name = ? AND workflow_version = ? AND active = true AND deleted_at IS NULL",
				target.TenantID, target.PartitionID, target.WorkflowName, target.WorkflowVersion,
			)
		if len(ids) > 0 {
			stale = stale.Where("id NOT IN ?", ids)
		}

		if staleErr := stale.UpdateColumns(map[string]any{"active": false, "modified_at": now}).Error; staleErr != nil {
			return fmt.Errorf("deactivate stale signal routes: %w", staleErr)
		}

		return nil
	})
	if txErr != nil {
		return fmt.Errorf("replace signal routes for %s v%d: %w", target.WorkflowName, target.WorkflowVersion, txErr)
	}

	return nil
}

func (r *signalRouteRepository) FindActiveByEventType(
	ctx context.Context,
	eventType string,
	limit int,
) ([]*models.SignalRoute, error) {
	if limit <= 0 {
		limit = defaultFindByEventTypeLimit
	}

	db := r.BaseRepository.Pool().DB(ctx, true)

	var routes []*models.SignalRoute

	result := db.Where("event_type = ? AND active = true AND deleted_at IS NULL", eventType).
		Order("id ASC").
		Limit(limit).
		Find(&routes)
	if result.Error != nil {
		return nil, fmt.Errorf("find signal routes: %w", result.Error)
	}

	return routes, nil
}
