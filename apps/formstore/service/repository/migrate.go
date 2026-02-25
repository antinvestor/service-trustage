package repository

import (
	"context"
	"fmt"

	"github.com/pitabwire/frame/datastore"
	"github.com/pitabwire/util"

	"github.com/antinvestor/service-trustage/apps/formstore/service/models"
)

// Migrate runs GORM AutoMigrate for all models and creates partial indexes.
// Returns an error if migration fails — callers must treat this as fatal.
func Migrate(ctx context.Context, manager datastore.Manager) error {
	log := util.Log(ctx)

	dbPool := manager.GetPool(ctx, datastore.DefaultPoolName)
	db := dbPool.DB(ctx, false)

	err := db.AutoMigrate(
		&models.FormDefinition{},
		&models.FormSubmission{},
	)
	if err != nil {
		return fmt.Errorf("auto-migrate database schema: %w", err)
	}

	indexes := []string{
		// Form definitions.
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_fd_form_id ON form_definitions(tenant_id, form_id) WHERE deleted_at IS NULL",
		"CREATE INDEX IF NOT EXISTS idx_fd_tenant ON form_definitions(tenant_id, partition_id)",

		// Form submissions.
		"CREATE INDEX IF NOT EXISTS idx_fs_form_id ON form_submissions(tenant_id, form_id, created_at DESC) WHERE deleted_at IS NULL",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_fs_idempotency ON form_submissions(tenant_id, idempotency_key) WHERE idempotency_key IS NOT NULL AND idempotency_key != '' AND deleted_at IS NULL",
		"CREATE INDEX IF NOT EXISTS idx_fs_tenant ON form_submissions(tenant_id, partition_id)",
	}

	for _, sql := range indexes {
		if indexErr := db.Exec(sql).Error; indexErr != nil {
			return fmt.Errorf("create index %q: %w", sql, indexErr)
		}
	}

	log.Info("database auto-migration completed successfully")

	return nil
}
