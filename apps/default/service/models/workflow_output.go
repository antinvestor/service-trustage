package models

import "time"

// WorkflowStateOutput stores the validated output of each state execution.
type WorkflowStateOutput struct {
	ID          string    `gorm:"column:id;primaryKey"`
	TenantID    string    `gorm:"column:tenant_id;not null"`
	PartitionID string    `gorm:"column:partition_id;not null"`
	ExecutionID string    `gorm:"column:execution_id;not null"`
	InstanceID  string    `gorm:"column:instance_id;not null"`
	State       string    `gorm:"column:state;not null"`
	SchemaHash  string    `gorm:"column:schema_hash;not null"`
	Payload     string    `gorm:"column:payload;type:jsonb;not null"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime"`
}

// TableName returns the database table name.
func (WorkflowStateOutput) TableName() string {
	return "workflow_state_outputs"
}
