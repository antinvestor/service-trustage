package repository

import (
	"context"
	"fmt"

	"github.com/pitabwire/frame/datastore"
	"github.com/pitabwire/util"

	"github.com/antinvestor/service-trustage/apps/default/service/models"
)

// Migrate runs GORM AutoMigrate for all models and creates partial indexes.
// Returns an error if migration fails — callers must treat this as fatal.
func Migrate(ctx context.Context, manager datastore.Manager) error {
	log := util.Log(ctx)

	dbPool := manager.GetPool(ctx, datastore.DefaultPoolName)
	db := dbPool.DB(ctx, false)

	err := db.AutoMigrate(
		&models.WorkflowDefinition{},
		&models.WorkflowInstance{},
		&models.WorkflowStateExecution{},
		&models.WorkflowStateSchema{},
		&models.WorkflowStateMapping{},
		&models.WorkflowStateOutput{},
		&models.WorkflowRetryPolicy{},
		&models.WorkflowAuditEvent{},
		&models.EventLog{},
		&models.TriggerBinding{},
		&models.ConnectorConfig{},
		&models.ConnectorCredential{},
		&models.ScheduleDefinition{},
	)
	if err != nil {
		return fmt.Errorf("auto-migrate database schema: %w", err)
	}

	indexes := []string{
		// Workflow definitions.
		"CREATE INDEX IF NOT EXISTS idx_wd_tenant ON workflow_definitions(tenant_id, partition_id)",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_wd_name_version ON workflow_definitions(tenant_id, name, workflow_version) WHERE deleted_at IS NULL",

		// Workflow instances.
		"CREATE INDEX IF NOT EXISTS idx_wi_tenant ON workflow_instances(tenant_id, partition_id)",
		"CREATE INDEX IF NOT EXISTS idx_wi_status ON workflow_instances(status) WHERE deleted_at IS NULL",
		"CREATE INDEX IF NOT EXISTS idx_wi_workflow ON workflow_instances(tenant_id, workflow_name, workflow_version)",
		"CREATE INDEX IF NOT EXISTS idx_wi_trigger ON workflow_instances(trigger_event_id) WHERE trigger_event_id IS NOT NULL",

		// Workflow state executions - scheduler indexes (partial).
		"CREATE INDEX IF NOT EXISTS idx_wse_tenant ON workflow_state_executions(tenant_id, partition_id)",
		"CREATE INDEX IF NOT EXISTS idx_wse_instance ON workflow_state_executions(instance_id, state)",
		"CREATE INDEX IF NOT EXISTS idx_wse_pending ON workflow_state_executions(status, created_at) WHERE status = 'pending'",
		"CREATE INDEX IF NOT EXISTS idx_wse_retry ON workflow_state_executions(next_retry_at) WHERE status = 'retry_scheduled'",
		"CREATE INDEX IF NOT EXISTS idx_wse_dispatched ON workflow_state_executions(status, created_at) WHERE status = 'dispatched'",

		// Schema registry.
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_wss_unique ON workflow_state_schemas(tenant_id, workflow_name, workflow_version, state, schema_type)",
		"CREATE INDEX IF NOT EXISTS idx_wss_hash ON workflow_state_schemas(schema_hash)",

		// Mappings.
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_wsm_unique ON workflow_state_mappings(tenant_id, workflow_name, workflow_version, from_state, to_state) WHERE deleted_at IS NULL",

		// Outputs.
		"CREATE INDEX IF NOT EXISTS idx_wso_instance ON workflow_state_outputs(instance_id, state)",
		"CREATE INDEX IF NOT EXISTS idx_wso_execution ON workflow_state_outputs(execution_id)",

		// Retry policies.
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_wrp_unique ON workflow_retry_policies(tenant_id, workflow_name, workflow_version, state) WHERE deleted_at IS NULL",

		// Audit events.
		"CREATE INDEX IF NOT EXISTS idx_wae_instance ON workflow_audit_events(instance_id, created_at)",
		"CREATE INDEX IF NOT EXISTS idx_wae_type ON workflow_audit_events(event_type)",

		// Event log - outbox pattern.
		"CREATE INDEX IF NOT EXISTS idx_el_unpublished ON event_log(published, created_at) WHERE published = false AND deleted_at IS NULL",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_el_idempotency_tenant ON event_log(tenant_id, partition_id, idempotency_key) WHERE idempotency_key IS NOT NULL AND deleted_at IS NULL",

		// Trigger bindings.
		"CREATE INDEX IF NOT EXISTS idx_tb_event ON trigger_bindings(tenant_id, event_type) WHERE active = true AND deleted_at IS NULL",

		// Schedule definitions.
		"CREATE INDEX IF NOT EXISTS idx_sd_tenant ON schedule_definitions(tenant_id, partition_id)",
		"CREATE INDEX IF NOT EXISTS idx_sd_due ON schedule_definitions(next_fire_at) WHERE active = true AND deleted_at IS NULL",
	}

	for _, sql := range indexes {
		if indexErr := db.Exec(sql).Error; indexErr != nil {
			return fmt.Errorf("create index %q: %w", sql, indexErr)
		}
	}

	log.Info("database auto-migration completed successfully")

	return nil
}
