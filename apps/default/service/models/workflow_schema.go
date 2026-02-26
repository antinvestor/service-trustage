package models

import (
	"encoding/json"

	"github.com/pitabwire/frame/data"
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
	data.BaseModel `gorm:"embedded"`

	WorkflowName    string          `gorm:"column:workflow_name;not null;uniqueIndex:uniq_workflow_state_schema"`
	WorkflowVersion int             `gorm:"column:workflow_version;not null;uniqueIndex:uniq_workflow_state_schema"`
	State           string          `gorm:"column:state;not null;uniqueIndex:uniq_workflow_state_schema"`
	SchemaType      SchemaType      `gorm:"column:schema_type;not null;uniqueIndex:uniq_workflow_state_schema"`
	SchemaHash      string          `gorm:"column:schema_hash;not null"`
	SchemaBlob      json.RawMessage `gorm:"column:schema_blob;type:jsonb;not null"`
}

// TableName returns the database table name.
func (WorkflowStateSchema) TableName() string {
	return "workflow_state_schemas"
}
