package models

import "github.com/pitabwire/frame/data"

// WorkflowAuditEvent records an append-only audit trail for workflow state transitions.
type WorkflowAuditEvent struct {
	data.BaseModel `gorm:"embedded"`

	InstanceID  string    `gorm:"column:instance_id;not null"`
	ExecutionID string    `gorm:"column:execution_id"`
	EventType   string    `gorm:"column:event_type;not null"`
	State       string    `gorm:"column:state"`
	FromState   string    `gorm:"column:from_state"`
	ToState     string    `gorm:"column:to_state"`
	Payload     string    `gorm:"column:payload;type:jsonb;default:'{}'"`
	TraceID     string    `gorm:"column:trace_id"`
}

// TableName returns the database table name.
func (WorkflowAuditEvent) TableName() string {
	return "workflow_audit_events"
}
