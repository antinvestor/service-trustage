// Copyright 2023-2026 Ant Investor Ltd
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package models

import (
	"time"

	"github.com/pitabwire/frame/data"
)

// ExecutionStatus enumerates all possible execution statuses.
type ExecutionStatus string

const (
	ExecStatusPending               ExecutionStatus = "pending"
	ExecStatusDispatched            ExecutionStatus = "dispatched"
	ExecStatusRunning               ExecutionStatus = "running"
	ExecStatusWaiting               ExecutionStatus = "waiting"
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
type WorkflowStateExecution struct {
	data.BaseModel `gorm:"embedded"`

	InstanceID       string          `gorm:"column:instance_id;not null"`
	State            string          `gorm:"column:state;not null"`
	StateVersion     int             `gorm:"column:state_version;not null;default:1"`
	Attempt          int             `gorm:"column:attempt;not null;default:1"`
	Status           ExecutionStatus `gorm:"column:status;not null;default:pending"`
	ExecutionToken   string          `gorm:"column:execution_token;not null"`
	InputSchemaHash  string          `gorm:"column:input_schema_hash;not null"`
	InputPayload     string          `gorm:"column:input_payload;type:jsonb;default:'{}'"`
	OutputSchemaHash string          `gorm:"column:output_schema_hash"`
	ErrorClass       string          `gorm:"column:error_class"`
	ErrorMessage     string          `gorm:"column:error_message"`
	NextRetryAt      *time.Time      `gorm:"column:next_retry_at"`
	TraceID          string          `gorm:"column:trace_id"`
	StartedAt        *time.Time      `gorm:"column:started_at"`
	FinishedAt       *time.Time      `gorm:"column:finished_at"`
}

// TableName returns the database table name.
func (WorkflowStateExecution) TableName() string {
	return "workflow_state_executions"
}
