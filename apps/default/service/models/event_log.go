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
	"fmt"
	"time"

	"github.com/pitabwire/frame/data"
	"gorm.io/gorm"
)

// MaxEventLogPayloadBytes is the hard upper bound on event_log.Payload size.
// The 1 MiB limit matches the HTTP body cap applied upstream; enforcing it
// again at the GORM layer is the last line of defence against oversized
// payloads bypassing the HTTP gateway (e.g. internal callers or future
// protocol additions). At 20-events-per-batch outbox cycles, a hostile
// payload could push a 512 MiB pod toward its memory limit.
const MaxEventLogPayloadBytes = 1 << 20 // 1 MiB

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
func (*EventLog) TableName() string {
	return "event_log"
}

// BeforeCreate rejects oversized payloads at the GORM layer before any INSERT
// is attempted. Enforcement upstream via HTTP body limits is the primary guard;
// this hook is the last line of defence for internal callers and future code
// paths that bypass the HTTP gateway.
func (e *EventLog) BeforeCreate(_ *gorm.DB) error {
	if len(e.Payload) > MaxEventLogPayloadBytes {
		return fmt.Errorf(
			"event_log payload exceeds %d bytes (got %d): reduce payload size or split into multiple events",
			MaxEventLogPayloadBytes, len(e.Payload),
		)
	}

	return nil
}
