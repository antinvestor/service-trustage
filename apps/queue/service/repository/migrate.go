package repository

import (
	"context"
	"fmt"

	"github.com/pitabwire/frame/datastore"
	"github.com/pitabwire/util"

	"github.com/antinvestor/service-trustage/apps/queue/service/models"
)

// Migrate runs GORM AutoMigrate for all models and creates partial indexes.
// Returns an error if migration fails — callers must treat this as fatal.
func Migrate(ctx context.Context, manager datastore.Manager) error {
	log := util.Log(ctx)

	dbPool := manager.GetPool(ctx, datastore.DefaultPoolName)
	db := dbPool.DB(ctx, false)

	err := db.AutoMigrate(
		&models.QueueDefinition{},
		&models.QueueItem{},
		&models.QueueCounter{},
	)
	if err != nil {
		return fmt.Errorf("auto-migrate database schema: %w", err)
	}

	indexes := []string{
		// Queue definitions.
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_qd_name ON queue_definitions(tenant_id, name) WHERE deleted_at IS NULL",
		"CREATE INDEX IF NOT EXISTS idx_qd_tenant ON queue_definitions(tenant_id, partition_id)",

		// Queue items — critical for priority queue ordering.
		"CREATE INDEX IF NOT EXISTS idx_qi_waiting ON queue_items(queue_id, priority DESC, joined_at ASC) WHERE status = 'waiting' AND deleted_at IS NULL",
		"CREATE INDEX IF NOT EXISTS idx_qi_queue ON queue_items(tenant_id, queue_id, created_at DESC) WHERE deleted_at IS NULL",
		"CREATE INDEX IF NOT EXISTS idx_qi_tenant ON queue_items(tenant_id, partition_id)",
		"CREATE INDEX IF NOT EXISTS idx_qi_counter ON queue_items(counter_id) WHERE status = 'serving'",

		// Queue item ticket uniqueness per queue.
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_qi_ticket ON queue_items(queue_id, ticket_no) WHERE deleted_at IS NULL",

		// Queue counters.
		"CREATE INDEX IF NOT EXISTS idx_qc_queue ON queue_counters(tenant_id, queue_id) WHERE deleted_at IS NULL",
		"CREATE INDEX IF NOT EXISTS idx_qc_tenant ON queue_counters(tenant_id, partition_id)",
	}

	for _, sql := range indexes {
		if indexErr := db.Exec(sql).Error; indexErr != nil {
			return fmt.Errorf("create index %q: %w", sql, indexErr)
		}
	}

	log.Info("database auto-migration completed successfully")

	return nil
}
