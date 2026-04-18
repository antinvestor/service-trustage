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

// WorkflowSignalMessage stores a durable signal sent to a workflow instance.
type WorkflowSignalMessage struct {
	data.BaseModel `gorm:"embedded"`

	TargetInstanceID string     `gorm:"column:target_instance_id;not null"`
	SignalName       string     `gorm:"column:signal_name;not null"`
	Payload          string     `gorm:"column:payload;type:jsonb;default:'{}'"`
	Status           string     `gorm:"column:status;not null;default:pending"`
	DeliveredAt      *time.Time `gorm:"column:delivered_at"`
	WaitID           string     `gorm:"column:wait_id"`

	ClaimUntil *time.Time `gorm:"column:claim_until"`
	ClaimOwner string     `gorm:"column:claim_owner"`
	Attempts   int        `gorm:"column:attempts;not null;default:0"`
}

// TableName returns the database table name.
func (WorkflowSignalMessage) TableName() string {
	return "workflow_signal_messages"
}
