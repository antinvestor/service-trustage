package models

import "github.com/pitabwire/frame/data"

// WorkflowStateOutput stores the validated output of each state execution.
type WorkflowStateOutput struct {
	data.BaseModel `gorm:"embedded"`

	ExecutionID string `gorm:"column:execution_id;not null"`
	InstanceID  string `gorm:"column:instance_id;not null"`
	State       string `gorm:"column:state;not null"`
	SchemaHash  string `gorm:"column:schema_hash;not null"`
	Payload     string `gorm:"column:payload;type:jsonb;not null"`
}

// TableName returns the database table name.
func (WorkflowStateOutput) TableName() string {
	return "workflow_state_outputs"
}
