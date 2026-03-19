package models

import (
	"time"

	"github.com/pitabwire/frame/data"
)

// WorkflowTimer stores durable wakeups for waiting workflow executions.
type WorkflowTimer struct {
	data.BaseModel `gorm:"embedded"`

	ExecutionID string     `gorm:"column:execution_id;not null"`
	InstanceID  string     `gorm:"column:instance_id;not null"`
	State       string     `gorm:"column:state;not null"`
	FiresAt     time.Time  `gorm:"column:fires_at;not null"`
	FiredAt     *time.Time `gorm:"column:fired_at"`

	ClaimUntil *time.Time `gorm:"column:claim_until"`
	ClaimOwner string     `gorm:"column:claim_owner"`
	Attempts   int        `gorm:"column:attempts;not null;default:0"`
}

// TableName returns the database table name.
func (WorkflowTimer) TableName() string {
	return "workflow_timers"
}
