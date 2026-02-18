package models

import "github.com/pitabwire/frame/data"

// WorkflowStateMapping stores data mapping expressions between states.
type WorkflowStateMapping struct {
	data.BaseModel `gorm:"embedded"`

	WorkflowName    string `gorm:"column:workflow_name;not null"`
	WorkflowVersion int    `gorm:"column:workflow_version;not null"`
	FromState       string `gorm:"column:from_state;not null"`
	ToState         string `gorm:"column:to_state;not null"`
	MappingExpr     string `gorm:"column:mapping_expr;type:jsonb;not null"`
}

// TableName returns the database table name.
func (WorkflowStateMapping) TableName() string {
	return "workflow_state_mappings"
}
