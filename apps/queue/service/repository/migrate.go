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

	"github.com/pitabwire/frame/datastore"
	"github.com/pitabwire/util"

	"github.com/antinvestor/service-trustage/apps/queue/service/models"
)

// Migrate runs GORM AutoMigrate for all models and creates partial indexes.
// Returns an error if migration fails — callers must treat this as fatal.
func Migrate(ctx context.Context, manager datastore.Manager) error {
	log := util.Log(ctx)

	dbPool := manager.GetPool(ctx, datastore.DefaultPoolName)
	if dbPool == nil {
		return fmt.Errorf("datastore pool %q not available", datastore.DefaultPoolName)
	}
	db := dbPool.DB(ctx, false)
	if db == nil {
		return fmt.Errorf("datastore pool %q has no active connection", datastore.DefaultPoolName)
	}

	err := db.AutoMigrate(
		&models.QueueDefinition{},
		&models.QueueItem{},
		&models.QueueCounter{},
	)
	if err != nil {
		return fmt.Errorf("auto-migrate database schema: %w", err)
	}

	for _, indexDef := range migrationIndexes() {
		for _, indexName := range indexDef.Names {
			if db.Migrator().HasIndex(indexDef.Model, indexName) {
				continue
			}

			if indexErr := db.Migrator().CreateIndex(indexDef.Model, indexName); indexErr != nil {
				return fmt.Errorf("create index %s on %T: %w", indexName, indexDef.Model, indexErr)
			}
		}
	}

	log.Debug("database auto-migration completed")

	return nil
}

type migrationIndex struct {
	Model any
	Names []string
}

func migrationIndexes() []migrationIndex {
	return []migrationIndex{
		{
			Model: &queueDefinitionIndexModel{},
			Names: []string{"idx_qd_name", "idx_qd_tenant"},
		},
		{
			Model: &queueItemIndexModel{},
			Names: []string{"idx_qi_waiting", "idx_qi_queue", "idx_qi_tenant", "idx_qi_counter", "idx_qi_ticket"},
		},
		{
			Model: &queueCounterIndexModel{},
			Names: []string{"idx_qc_queue", "idx_qc_tenant"},
		},
	}
}

type queueDefinitionIndexModel struct {
	TenantID    string `gorm:"column:tenant_id;index:idx_qd_name,unique,where:deleted_at IS NULL,priority:1;index:idx_qd_tenant,priority:1"`
	PartitionID string `gorm:"column:partition_id;index:idx_qd_tenant,priority:2"`
	Name        string `gorm:"column:name;index:idx_qd_name,unique,where:deleted_at IS NULL,priority:2"`
}

func (queueDefinitionIndexModel) TableName() string { return "queue_definitions" }

type queueItemIndexModel struct {
	TenantID    string    `gorm:"column:tenant_id;index:idx_qi_queue,where:deleted_at IS NULL,priority:1;index:idx_qi_tenant,priority:1"`
	PartitionID string    `gorm:"column:partition_id;index:idx_qi_tenant,priority:2"`
	QueueID     string    `gorm:"column:queue_id;index:idx_qi_waiting,where:status = 'waiting' AND deleted_at IS NULL,priority:1;index:idx_qi_queue,where:deleted_at IS NULL,priority:2;index:idx_qi_ticket,unique,where:deleted_at IS NULL,priority:1"`
	Priority    int       `gorm:"column:priority;index:idx_qi_waiting,sort:desc,where:status = 'waiting' AND deleted_at IS NULL,priority:2"`
	JoinedAt    time.Time `gorm:"column:joined_at;index:idx_qi_waiting,sort:asc,where:status = 'waiting' AND deleted_at IS NULL,priority:3"`
	CreatedAt   time.Time `gorm:"column:created_at;index:idx_qi_queue,sort:desc,where:deleted_at IS NULL,priority:3"`
	CounterID   string    `gorm:"column:counter_id;index:idx_qi_counter,where:status = 'serving'"`
	TicketNo    string    `gorm:"column:ticket_no;index:idx_qi_ticket,unique,where:deleted_at IS NULL,priority:2"`
}

func (queueItemIndexModel) TableName() string { return "queue_items" }

type queueCounterIndexModel struct {
	TenantID    string `gorm:"column:tenant_id;index:idx_qc_queue,where:deleted_at IS NULL,priority:1;index:idx_qc_tenant,priority:1"`
	PartitionID string `gorm:"column:partition_id;index:idx_qc_tenant,priority:2"`
	QueueID     string `gorm:"column:queue_id;index:idx_qc_queue,where:deleted_at IS NULL,priority:2"`
}

func (queueCounterIndexModel) TableName() string { return "queue_counters" }
