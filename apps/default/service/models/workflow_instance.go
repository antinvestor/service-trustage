package models

import (
	"fmt"
	"time"

	"github.com/pitabwire/frame/data"
)

// WorkflowInstanceStatus enumerates workflow instance statuses.
type WorkflowInstanceStatus string

const (
	InstanceStatusRunning   WorkflowInstanceStatus = "running"
	InstanceStatusCompleted WorkflowInstanceStatus = "completed"
	InstanceStatusFailed    WorkflowInstanceStatus = "failed"
	InstanceStatusCancelled WorkflowInstanceStatus = "cancelled"
	InstanceStatusSuspended WorkflowInstanceStatus = "suspended"
)

// WorkflowInstance is a running copy of a workflow definition.
type WorkflowInstance struct { //nolint:recvcheck // TableName() must be value receiver for GORM
	data.BaseModel `gorm:"embedded"`

	WorkflowName      string                 `gorm:"column:workflow_name;not null"`
	WorkflowVersion   int                    `gorm:"column:workflow_version;not null"`
	CurrentState      string                 `gorm:"column:current_state;not null"`
	Status            WorkflowInstanceStatus `gorm:"column:status;not null;default:running"`
	Revision          int64                  `gorm:"column:revision;not null;default:1"`
	TriggerEventID    string                 `gorm:"column:trigger_event_id"`
	ParentInstanceID  string                 `gorm:"column:parent_instance_id"`
	ParentExecutionID string                 `gorm:"column:parent_execution_id"`
	ScopeType         string                 `gorm:"column:scope_type"`
	ScopeParentState  string                 `gorm:"column:scope_parent_state"`
	ScopeEntryState   string                 `gorm:"column:scope_entry_state"`
	ScopeIndex        int                    `gorm:"column:scope_index;not null;default:0"`
	Metadata          string                 `gorm:"column:metadata;type:jsonb;default:'{}'"`
	StartedAt         *time.Time             `gorm:"column:started_at"`
	FinishedAt        *time.Time             `gorm:"column:finished_at"`
}

// TableName returns the database table name.
func (WorkflowInstance) TableName() string {
	return "workflow_instances"
}

// ValidInstanceTransitions defines allowed status transitions.
var validInstanceTransitions = map[WorkflowInstanceStatus][]WorkflowInstanceStatus{ //nolint:gochecknoglobals // transition map
	InstanceStatusRunning: {
		InstanceStatusCompleted,
		InstanceStatusFailed,
		InstanceStatusCancelled,
		InstanceStatusSuspended,
	},
	InstanceStatusSuspended: {InstanceStatusRunning, InstanceStatusCancelled},
	InstanceStatusCompleted: {},
	InstanceStatusFailed:    {},
	InstanceStatusCancelled: {},
}

// TransitionTo validates and performs a status transition.
func (i *WorkflowInstance) TransitionTo(newStatus WorkflowInstanceStatus) error {
	allowed := validInstanceTransitions[i.Status]
	for _, s := range allowed {
		if s == newStatus {
			i.Status = newStatus
			return nil
		}
	}

	return fmt.Errorf("invalid instance transition from %s to %s", i.Status, newStatus)
}
