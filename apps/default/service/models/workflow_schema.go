package models

import (
	"time"
)

// SchemaType enumerates schema types.
type SchemaType string

const (
	SchemaTypeInput  SchemaType = "input"
	SchemaTypeOutput SchemaType = "output"
	SchemaTypeError  SchemaType = "error"
)

// WorkflowStateSchema stores immutable JSON Schema documents for state contracts.
type WorkflowStateSchema struct {
	ID              string     `gorm:"column:id;primaryKey"`
	TenantID        string     `gorm:"column:tenant_id;not null"`
	PartitionID     string     `gorm:"column:partition_id;not null"`
	WorkflowName    string     `gorm:"column:workflow_name;not null"`
	WorkflowVersion int        `gorm:"column:workflow_version;not null"`
	State           string     `gorm:"column:state;not null"`
	SchemaType      SchemaType `gorm:"column:schema_type;not null"`
	SchemaHash      string     `gorm:"column:schema_hash;not null"`
	SchemaBlob      string     `gorm:"column:schema_blob;type:jsonb;not null"`
	CreatedAt       time.Time  `gorm:"column:created_at;autoCreateTime"`
}

// TableName returns the database table name.
func (WorkflowStateSchema) TableName() string {
	return "workflow_state_schemas"
}
