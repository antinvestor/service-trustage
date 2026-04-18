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

import "github.com/pitabwire/frame/data"

// WorkflowAuditEvent records an append-only audit trail for workflow state transitions.
type WorkflowAuditEvent struct {
	data.BaseModel `gorm:"embedded"`

	InstanceID  string `gorm:"column:instance_id;not null"`
	ExecutionID string `gorm:"column:execution_id"`
	EventType   string `gorm:"column:event_type;not null"`
	State       string `gorm:"column:state"`
	FromState   string `gorm:"column:from_state"`
	ToState     string `gorm:"column:to_state"`
	Payload     string `gorm:"column:payload;type:jsonb;default:'{}'"`
	TraceID     string `gorm:"column:trace_id"`
}

// TableName returns the database table name.
func (WorkflowAuditEvent) TableName() string {
	return "workflow_audit_events"
}
