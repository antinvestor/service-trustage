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

	"github.com/antinvestor/service-trustage/apps/formstore/service/models"
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
		&models.FormDefinition{},
		&models.FormSubmission{},
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
			Model: &formDefinitionIndexModel{},
			Names: []string{"idx_fd_form_id", "idx_fd_tenant"},
		},
		{
			Model: &formSubmissionIndexModel{},
			Names: []string{"idx_fs_form_id", "idx_fs_idempotency", "idx_fs_tenant"},
		},
	}
}

type formDefinitionIndexModel struct {
	TenantID    string `gorm:"column:tenant_id;index:idx_fd_form_id,unique,where:deleted_at IS NULL,priority:1;index:idx_fd_tenant,priority:1"`
	PartitionID string `gorm:"column:partition_id;index:idx_fd_tenant,priority:2"`
	FormID      string `gorm:"column:form_id;index:idx_fd_form_id,unique,where:deleted_at IS NULL,priority:2"`
}

func (formDefinitionIndexModel) TableName() string { return "form_definitions" }

type formSubmissionIndexModel struct {
	TenantID       string    `gorm:"column:tenant_id;index:idx_fs_form_id,where:deleted_at IS NULL,priority:1;index:idx_fs_idempotency,unique,where:idempotency_key IS NOT NULL AND idempotency_key <> '' AND deleted_at IS NULL,priority:1;index:idx_fs_tenant,priority:1"`
	PartitionID    string    `gorm:"column:partition_id;index:idx_fs_tenant,priority:2"`
	FormID         string    `gorm:"column:form_id;index:idx_fs_form_id,where:deleted_at IS NULL,priority:2"`
	CreatedAt      time.Time `gorm:"column:created_at;index:idx_fs_form_id,sort:desc,where:deleted_at IS NULL,priority:3"`
	IdempotencyKey string    `gorm:"column:idempotency_key;index:idx_fs_idempotency,unique,where:idempotency_key IS NOT NULL AND idempotency_key <> '' AND deleted_at IS NULL,priority:2"`
}

func (formSubmissionIndexModel) TableName() string { return "form_submissions" }
