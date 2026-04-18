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

// EventLog records events for outbox publishing.
type EventLog struct {
	data.BaseModel `gorm:"embedded"`

	EventType         string     `gorm:"column:event_type;not null"`
	Source            string     `gorm:"column:source"`
	IdempotencyKey    string     `gorm:"column:idempotency_key"`
	Payload           string     `gorm:"column:payload;type:jsonb;not null"`
	Published         bool       `gorm:"column:published;not null;default:false"`
	PublishedAt       *time.Time `gorm:"column:published_at"`
	PublishClaimUntil *time.Time `gorm:"column:publish_claim_until"`
	PublishClaimOwner string     `gorm:"column:publish_claim_owner"`
	PublishAttempts   int        `gorm:"column:publish_attempts;not null;default:0"`
}

// TableName returns the database table name.
func (EventLog) TableName() string {
	return "event_log"
}
