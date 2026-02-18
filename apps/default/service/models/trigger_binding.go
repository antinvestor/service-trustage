package models

import "github.com/pitabwire/frame/data"

// TriggerBinding maps event types to workflow instantiation.
type TriggerBinding struct {
	data.BaseModel `gorm:"embedded"`

	EventType       string `gorm:"column:event_type;not null"`
	EventFilter     string `gorm:"column:event_filter"`
	WorkflowName    string `gorm:"column:workflow_name;not null"`
	WorkflowVersion int    `gorm:"column:workflow_version;not null"`
	InputMapping    string `gorm:"column:input_mapping;type:jsonb"`
	Active          bool   `gorm:"column:active;not null;default:true"`
}

// TableName returns the database table name.
func (TriggerBinding) TableName() string {
	return "trigger_bindings"
}
