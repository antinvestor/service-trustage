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

// WorkflowSignalWait stores a durable wait on a named signal for a waiting execution.
type WorkflowSignalWait struct {
	data.BaseModel `gorm:"embedded"`

	ExecutionID string `gorm:"column:execution_id;not null"`
	InstanceID  string `gorm:"column:instance_id;not null"`
	State       string `gorm:"column:state;not null"`

	SignalName string `gorm:"column:signal_name;not null"`
	OutputVar  string `gorm:"column:output_var"`
	Status     string `gorm:"column:status;not null;default:waiting"`

	TimeoutAt  *time.Time `gorm:"column:timeout_at"`
	MatchedAt  *time.Time `gorm:"column:matched_at"`
	TimedOutAt *time.Time `gorm:"column:timed_out_at"`
	MessageID  string     `gorm:"column:message_id"`

	ClaimUntil *time.Time `gorm:"column:claim_until"`
	ClaimOwner string     `gorm:"column:claim_owner"`
	Attempts   int        `gorm:"column:attempts;not null;default:0"`
}

// TableName returns the database table name.
func (WorkflowSignalWait) TableName() string {
	return "workflow_signal_waits"
}
