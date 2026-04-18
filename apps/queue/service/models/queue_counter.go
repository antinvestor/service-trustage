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

	"github.com/pitabwire/frame/data"
)

// CounterStatus enumerates counter statuses.
type CounterStatus string

const (
	CounterStatusOpen   CounterStatus = "open"
	CounterStatusClosed CounterStatus = "closed"
	CounterStatusPaused CounterStatus = "paused"
)

// QueueCounter represents a service point (window, room, desk).
type QueueCounter struct { //nolint:recvcheck // TableName() must be value receiver for GORM
	data.BaseModel `gorm:"embedded"`

	QueueID       string        `gorm:"column:queue_id;not null"`
	Name          string        `gorm:"column:name;not null"`
	Status        CounterStatus `gorm:"column:status;not null;default:closed"`
	CurrentItemID string        `gorm:"column:current_item_id"`
	ServedBy      string        `gorm:"column:served_by"`
	Categories    string        `gorm:"column:categories;type:jsonb"`
	TotalServed   int           `gorm:"column:total_served;not null;default:0"`
}

// TableName returns the database table name.
func (QueueCounter) TableName() string {
	return "queue_counters"
}

// ValidCounterTransitions defines allowed status transitions.
var validCounterTransitions = map[CounterStatus][]CounterStatus{ //nolint:gochecknoglobals // transition map
	CounterStatusClosed: {CounterStatusOpen},
	CounterStatusOpen:   {CounterStatusClosed, CounterStatusPaused},
	CounterStatusPaused: {CounterStatusOpen, CounterStatusClosed},
}

// TransitionTo validates and performs a status transition.
func (c *QueueCounter) TransitionTo(newStatus CounterStatus) error {
	allowed := validCounterTransitions[c.Status]
	for _, s := range allowed {
		if s == newStatus {
			c.Status = newStatus
			return nil
		}
	}

	return fmt.Errorf("invalid transition from %s to %s", c.Status, newStatus)
}
