package models

import (
	"time"
)

// ExecutionStatus enumerates all possible execution statuses.
type ExecutionStatus string

const (
	ExecStatusPending               ExecutionStatus = "pending"
	ExecStatusDispatched            ExecutionStatus = "dispatched"
	ExecStatusRunning               ExecutionStatus = "running"
	ExecStatusCompleted             ExecutionStatus = "completed"
	ExecStatusFailed                ExecutionStatus = "failed"
	ExecStatusFatal                 ExecutionStatus = "fatal"
	ExecStatusTimedOut              ExecutionStatus = "timed_out"
	ExecStatusInvalidInputContract  ExecutionStatus = "invalid_input_contract"
	ExecStatusInvalidOutputContract ExecutionStatus = "invalid_output_contract"
	ExecStatusStale                 ExecutionStatus = "stale"
	ExecStatusRetryScheduled        ExecutionStatus = "retry_scheduled"
)

// WorkflowStateExecution tracks each attempt to execute a state.
// Uses a custom primary key (ExecutionID) instead of BaseModel.
type WorkflowStateExecution struct {
	ExecutionID      string          `gorm:"column:execution_id;primaryKey"`
	TenantID         string          `gorm:"column:tenant_id;not null"`
	PartitionID      string          `gorm:"column:partition_id;not null"`
	InstanceID       string          `gorm:"column:instance_id;not null"`
	State            string          `gorm:"column:state;not null"`
	StateVersion     int             `gorm:"column:state_version;not null;default:1"`
	Attempt          int             `gorm:"column:attempt;not null;default:1"`
	Status           ExecutionStatus `gorm:"column:status;not null;default:pending"`
	ExecutionToken   string          `gorm:"column:execution_token;not null"`
	InputSchemaHash  string          `gorm:"column:input_schema_hash;not null"`
	OutputSchemaHash string          `gorm:"column:output_schema_hash"`
	ErrorClass       string          `gorm:"column:error_class"`
	ErrorMessage     string          `gorm:"column:error_message"`
	NextRetryAt      *time.Time      `gorm:"column:next_retry_at"`
	TraceID          string          `gorm:"column:trace_id"`
	StartedAt        *time.Time      `gorm:"column:started_at"`
	FinishedAt       *time.Time      `gorm:"column:finished_at"`
	CreatedAt        time.Time       `gorm:"column:created_at;autoCreateTime"`
}

// TableName returns the database table name.
func (WorkflowStateExecution) TableName() string {
	return "workflow_state_executions"
}
