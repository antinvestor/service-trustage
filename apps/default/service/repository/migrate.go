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
		&models.WorkflowScopeRun{},
		&models.WorkflowSignalWait{},
		&models.WorkflowSignalMessage{},
		&models.WorkflowRetryPolicy{},
		&models.WorkflowTimer{},
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

	// v1.1 housekeeping: drop the v1 idx_sd_due so the tightened predicate
	// (WHERE active = true AND deleted_at IS NULL AND next_fire_at IS NOT NULL)
	// is picked up on the next CreateIndex. One-time.
	if db.Migrator().HasIndex(&models.ScheduleDefinition{}, "idx_sd_due") {
		if dropErr := db.Migrator().DropIndex(&models.ScheduleDefinition{}, "idx_sd_due"); dropErr != nil {
			return fmt.Errorf("drop v1 idx_sd_due: %w", dropErr)
		}
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
			Model: &workflowDefinitionIndexModel{},
			Names: []string{"idx_wd_tenant", "idx_wd_name_version"},
		},
		{
			Model: &workflowInstanceIndexModel{},
			Names: []string{
				"idx_wi_tenant",
				"idx_wi_status",
				"idx_wi_workflow",
				"idx_wi_trigger",
				"idx_wi_parent_execution",
				"idx_wi_trigger_dedupe",
			},
		},
		{
			Model: &workflowExecutionIndexModel{},
			Names: []string{
				"idx_wse_tenant",
				"idx_wse_instance",
				"idx_wse_pending",
				"idx_wse_retry",
				"idx_wse_waiting",
				"idx_wse_dispatched",
			},
		},
		{
			Model: &workflowTimerIndexModel{},
			Names: []string{"idx_wt_execution_unique", "idx_wt_due"},
		},
		{
			Model: &workflowSchemaIndexModel{},
			Names: []string{"idx_wss_unique", "idx_wss_hash"},
		},
		{
			Model: &workflowMappingIndexModel{},
			Names: []string{"idx_wsm_unique"},
		},
		{
			Model: &workflowOutputIndexModel{},
			Names: []string{"idx_wso_instance", "idx_wso_execution"},
		},
		{
			Model: &workflowScopeRunIndexModel{},
			Names: []string{"idx_wsr_parent_execution_unique", "idx_wsr_running"},
		},
		{
			Model: &workflowSignalWaitIndexModel{},
			Names: []string{"idx_wsw_execution_unique", "idx_wsw_waiting"},
		},
		{
			Model: &workflowSignalMessageIndexModel{},
			Names: []string{"idx_wsm_pending"},
		},
		{
			Model: &workflowRetryPolicyIndexModel{},
			Names: []string{"idx_wrp_unique"},
		},
		{
			Model: &workflowAuditEventIndexModel{},
			Names: []string{"idx_wae_instance", "idx_wae_type"},
		},
		{
			Model: &eventLogIndexModel{},
			Names: []string{"idx_el_unpublished", "idx_el_claimable", "idx_el_idempotency_tenant"},
		},
		{
			Model: &triggerBindingIndexModel{},
			Names: []string{"idx_tb_event"},
		},
		{
			Model: &scheduleDefinitionIndexModel{},
			Names: []string{"idx_sd_tenant", "idx_sd_due", "idx_sd_workflow_unique"},
		},
	}
}

type workflowDefinitionIndexModel struct {
	TenantID        string `gorm:"column:tenant_id;index:idx_wd_tenant,priority:1;index:idx_wd_name_version,unique,where:deleted_at IS NULL,priority:1"`
	PartitionID     string `gorm:"column:partition_id;index:idx_wd_tenant,priority:2"`
	Name            string `gorm:"column:name;index:idx_wd_name_version,unique,where:deleted_at IS NULL,priority:2"`
	WorkflowVersion int    `gorm:"column:workflow_version;index:idx_wd_name_version,unique,where:deleted_at IS NULL,priority:3"`
}

func (workflowDefinitionIndexModel) TableName() string { return "workflow_definitions" }

type workflowInstanceIndexModel struct {
	TenantID          string `gorm:"column:tenant_id;index:idx_wi_tenant,priority:1;index:idx_wi_workflow,priority:1;index:idx_wi_trigger_dedupe,unique,where:trigger_event_id IS NOT NULL AND trigger_event_id <> '' AND deleted_at IS NULL,priority:1"`
	PartitionID       string `gorm:"column:partition_id;index:idx_wi_tenant,priority:2;index:idx_wi_trigger_dedupe,unique,where:trigger_event_id IS NOT NULL AND trigger_event_id <> '' AND deleted_at IS NULL,priority:2"`
	Status            string `gorm:"column:status;index:idx_wi_status,where:deleted_at IS NULL"`
	WorkflowName      string `gorm:"column:workflow_name;index:idx_wi_workflow,priority:2;index:idx_wi_trigger_dedupe,unique,where:trigger_event_id IS NOT NULL AND trigger_event_id <> '' AND deleted_at IS NULL,priority:3"`
	WorkflowVersion   int    `gorm:"column:workflow_version;index:idx_wi_workflow,priority:3;index:idx_wi_trigger_dedupe,unique,where:trigger_event_id IS NOT NULL AND trigger_event_id <> '' AND deleted_at IS NULL,priority:4"`
	TriggerEventID    string `gorm:"column:trigger_event_id;index:idx_wi_trigger,where:trigger_event_id IS NOT NULL;index:idx_wi_trigger_dedupe,unique,where:trigger_event_id IS NOT NULL AND trigger_event_id <> '' AND deleted_at IS NULL,priority:5"`
	ParentExecutionID string `gorm:"column:parent_execution_id;index:idx_wi_parent_execution,where:parent_execution_id IS NOT NULL AND parent_execution_id <> '' AND deleted_at IS NULL,priority:1"`
	ScopeIndex        int    `gorm:"column:scope_index;index:idx_wi_parent_execution,where:parent_execution_id IS NOT NULL AND parent_execution_id <> '' AND deleted_at IS NULL,priority:2"`
}

func (workflowInstanceIndexModel) TableName() string { return "workflow_instances" }

type workflowExecutionIndexModel struct {
	TenantID    string    `gorm:"column:tenant_id;index:idx_wse_tenant,priority:1"`
	PartitionID string    `gorm:"column:partition_id;index:idx_wse_tenant,priority:2"`
	InstanceID  string    `gorm:"column:instance_id;index:idx_wse_instance,priority:1"`
	State       string    `gorm:"column:state;index:idx_wse_instance,priority:2"`
	Status      string    `gorm:"column:status;index:idx_wse_pending,where:status = 'pending',priority:1;index:idx_wse_waiting,where:status = 'waiting',priority:1;index:idx_wse_dispatched,where:status = 'dispatched',priority:1"`
	CreatedAt   time.Time `gorm:"column:created_at;index:idx_wse_pending,where:status = 'pending',priority:2;index:idx_wse_waiting,where:status = 'waiting',priority:2;index:idx_wse_dispatched,where:status = 'dispatched',priority:2"`
	NextRetryAt time.Time `gorm:"column:next_retry_at;index:idx_wse_retry,where:status = 'retry_scheduled'"`
}

func (workflowExecutionIndexModel) TableName() string { return "workflow_state_executions" }

type workflowTimerIndexModel struct {
	ExecutionID string    `gorm:"column:execution_id;index:idx_wt_execution_unique,unique,where:deleted_at IS NULL"`
	FiresAt     time.Time `gorm:"column:fires_at;index:idx_wt_due,where:fired_at IS NULL AND deleted_at IS NULL"`
}

func (workflowTimerIndexModel) TableName() string { return "workflow_timers" }

type workflowSchemaIndexModel struct {
	TenantID        string `gorm:"column:tenant_id;index:idx_wss_unique,unique,priority:1"`
	WorkflowName    string `gorm:"column:workflow_name;index:idx_wss_unique,unique,priority:2"`
	WorkflowVersion int    `gorm:"column:workflow_version;index:idx_wss_unique,unique,priority:3"`
	State           string `gorm:"column:state;index:idx_wss_unique,unique,priority:4"`
	SchemaType      string `gorm:"column:schema_type;index:idx_wss_unique,unique,priority:5"`
	SchemaHash      string `gorm:"column:schema_hash;index:idx_wss_hash"`
}

func (workflowSchemaIndexModel) TableName() string { return "workflow_state_schemas" }

type workflowMappingIndexModel struct {
	TenantID        string `gorm:"column:tenant_id;index:idx_wsm_unique,unique,where:deleted_at IS NULL,priority:1"`
	WorkflowName    string `gorm:"column:workflow_name;index:idx_wsm_unique,unique,where:deleted_at IS NULL,priority:2"`
	WorkflowVersion int    `gorm:"column:workflow_version;index:idx_wsm_unique,unique,where:deleted_at IS NULL,priority:3"`
	FromState       string `gorm:"column:from_state;index:idx_wsm_unique,unique,where:deleted_at IS NULL,priority:4"`
	ToState         string `gorm:"column:to_state;index:idx_wsm_unique,unique,where:deleted_at IS NULL,priority:5"`
}

func (workflowMappingIndexModel) TableName() string { return "workflow_state_mappings" }

type workflowOutputIndexModel struct {
	InstanceID  string `gorm:"column:instance_id;index:idx_wso_instance,priority:1"`
	State       string `gorm:"column:state;index:idx_wso_instance,priority:2"`
	ExecutionID string `gorm:"column:execution_id;index:idx_wso_execution"`
}

func (workflowOutputIndexModel) TableName() string { return "workflow_state_outputs" }

type workflowScopeRunIndexModel struct {
	ParentExecutionID string    `gorm:"column:parent_execution_id;index:idx_wsr_parent_execution_unique,unique,where:deleted_at IS NULL;index:idx_wsr_running,where:status = 'running' AND deleted_at IS NULL,priority:2"`
	Status            string    `gorm:"column:status;index:idx_wsr_running,where:status = 'running' AND deleted_at IS NULL,priority:1"`
	CreatedAt         time.Time `gorm:"column:created_at;index:idx_wsr_running,where:status = 'running' AND deleted_at IS NULL,priority:3"`
}

func (workflowScopeRunIndexModel) TableName() string { return "workflow_scope_runs" }

type workflowSignalWaitIndexModel struct {
	ExecutionID string    `gorm:"column:execution_id;index:idx_wsw_execution_unique,unique,where:deleted_at IS NULL"`
	InstanceID  string    `gorm:"column:instance_id;index:idx_wsw_waiting,where:status = 'waiting' AND deleted_at IS NULL,priority:1"`
	SignalName  string    `gorm:"column:signal_name;index:idx_wsw_waiting,where:status = 'waiting' AND deleted_at IS NULL,priority:2"`
	TimeoutAt   time.Time `gorm:"column:timeout_at;index:idx_wsw_waiting,where:status = 'waiting' AND deleted_at IS NULL,priority:3"`
}

func (workflowSignalWaitIndexModel) TableName() string { return "workflow_signal_waits" }

type workflowSignalMessageIndexModel struct {
	TargetInstanceID string    `gorm:"column:target_instance_id;index:idx_wsm_pending,where:status = 'pending' AND deleted_at IS NULL,priority:1"`
	SignalName       string    `gorm:"column:signal_name;index:idx_wsm_pending,where:status = 'pending' AND deleted_at IS NULL,priority:2"`
	CreatedAt        time.Time `gorm:"column:created_at;index:idx_wsm_pending,where:status = 'pending' AND deleted_at IS NULL,priority:3"`
}

func (workflowSignalMessageIndexModel) TableName() string { return "workflow_signal_messages" }

type workflowRetryPolicyIndexModel struct {
	TenantID        string `gorm:"column:tenant_id;index:idx_wrp_unique,unique,where:deleted_at IS NULL,priority:1"`
	WorkflowName    string `gorm:"column:workflow_name;index:idx_wrp_unique,unique,where:deleted_at IS NULL,priority:2"`
	WorkflowVersion int    `gorm:"column:workflow_version;index:idx_wrp_unique,unique,where:deleted_at IS NULL,priority:3"`
	State           string `gorm:"column:state;index:idx_wrp_unique,unique,where:deleted_at IS NULL,priority:4"`
}

func (workflowRetryPolicyIndexModel) TableName() string { return "workflow_retry_policies" }

type workflowAuditEventIndexModel struct {
	InstanceID string    `gorm:"column:instance_id;index:idx_wae_instance,priority:1"`
	CreatedAt  time.Time `gorm:"column:created_at;index:idx_wae_instance,priority:2"`
	EventType  string    `gorm:"column:event_type;index:idx_wae_type"`
}

func (workflowAuditEventIndexModel) TableName() string { return "workflow_audit_events" }

type eventLogIndexModel struct {
	TenantID          string    `gorm:"column:tenant_id;index:idx_el_idempotency_tenant,unique,where:idempotency_key IS NOT NULL AND deleted_at IS NULL,priority:1"`
	PartitionID       string    `gorm:"column:partition_id;index:idx_el_idempotency_tenant,unique,where:idempotency_key IS NOT NULL AND deleted_at IS NULL,priority:2"`
	IdempotencyKey    string    `gorm:"column:idempotency_key;index:idx_el_idempotency_tenant,unique,where:idempotency_key IS NOT NULL AND deleted_at IS NULL,priority:3"`
	Published         bool      `gorm:"column:published;index:idx_el_unpublished,where:published = false AND deleted_at IS NULL,priority:1;index:idx_el_claimable,where:published = false AND deleted_at IS NULL,priority:1"`
	CreatedAt         time.Time `gorm:"column:created_at;index:idx_el_unpublished,where:published = false AND deleted_at IS NULL,priority:2;index:idx_el_claimable,where:published = false AND deleted_at IS NULL,priority:3"`
	PublishClaimUntil time.Time `gorm:"column:publish_claim_until;index:idx_el_claimable,where:published = false AND deleted_at IS NULL,priority:2"`
}

func (eventLogIndexModel) TableName() string { return "event_log" }

type triggerBindingIndexModel struct {
	TenantID  string `gorm:"column:tenant_id;index:idx_tb_event,where:active = true AND deleted_at IS NULL,priority:1"`
	EventType string `gorm:"column:event_type;index:idx_tb_event,where:active = true AND deleted_at IS NULL,priority:2"`
}

func (triggerBindingIndexModel) TableName() string { return "trigger_bindings" }

type scheduleDefinitionIndexModel struct {
	TenantID        string    `gorm:"column:tenant_id;index:idx_sd_tenant,priority:1;index:idx_sd_workflow_unique,unique,where:deleted_at IS NULL,priority:1"`
	PartitionID     string    `gorm:"column:partition_id;index:idx_sd_tenant,priority:2;index:idx_sd_workflow_unique,unique,where:deleted_at IS NULL,priority:2"`
	WorkflowName    string    `gorm:"column:workflow_name;index:idx_sd_workflow_unique,unique,where:deleted_at IS NULL,priority:3"`
	WorkflowVersion int       `gorm:"column:workflow_version;index:idx_sd_workflow_unique,unique,where:deleted_at IS NULL,priority:4"`
	Name            string    `gorm:"column:name;index:idx_sd_workflow_unique,unique,where:deleted_at IS NULL,priority:5"`
	NextFireAt      time.Time `gorm:"column:next_fire_at;index:idx_sd_due,where:active = true AND deleted_at IS NULL AND next_fire_at IS NOT NULL,priority:1"`
}

func (scheduleDefinitionIndexModel) TableName() string { return "schedule_definitions" }
