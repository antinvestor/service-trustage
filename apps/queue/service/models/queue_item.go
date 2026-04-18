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
)

// QueueItemStatus enumerates queue item statuses.
type QueueItemStatus string

const (
	ItemStatusWaiting   QueueItemStatus = "waiting"
	ItemStatusServing   QueueItemStatus = "serving"
	ItemStatusCompleted QueueItemStatus = "completed"
	ItemStatusCancelled QueueItemStatus = "cancelled"
	ItemStatusNoShow    QueueItemStatus = "no_show"
	ItemStatusExpired   QueueItemStatus = "expired"
)

// QueueItem represents an item in a queue.
type QueueItem struct { //nolint:recvcheck // TableName() must be value receiver for GORM
	data.BaseModel `gorm:"embedded"`

	QueueID      string          `gorm:"column:queue_id;not null"`
	Priority     int             `gorm:"column:priority;not null;default:1"`
	Status       QueueItemStatus `gorm:"column:status;not null;default:waiting"`
	TicketNo     string          `gorm:"column:ticket_no;not null"`
	Category     string          `gorm:"column:category"`
	CustomerID   string          `gorm:"column:customer_id"`
	Metadata     string          `gorm:"column:metadata;type:jsonb"`
	CounterID    string          `gorm:"column:counter_id"`
	ServedBy     string          `gorm:"column:served_by"`
	CalledAt     *time.Time      `gorm:"column:called_at"`
	ServiceStart *time.Time      `gorm:"column:service_start"`
	ServiceEnd   *time.Time      `gorm:"column:service_end"`
	JoinedAt     time.Time       `gorm:"column:joined_at;not null"`
}

// TableName returns the database table name.
func (QueueItem) TableName() string {
	return "queue_items"
}

// ValidItemTransitions defines allowed status transitions.
var validItemTransitions = map[QueueItemStatus][]QueueItemStatus{ //nolint:gochecknoglobals // transition map
	ItemStatusWaiting:   {ItemStatusServing, ItemStatusCancelled, ItemStatusNoShow, ItemStatusExpired},
	ItemStatusServing:   {ItemStatusCompleted, ItemStatusCancelled, ItemStatusNoShow, ItemStatusWaiting},
	ItemStatusNoShow:    {ItemStatusWaiting}, // re-queue
	ItemStatusCompleted: {},
	ItemStatusCancelled: {},
	ItemStatusExpired:   {},
}

// TransitionTo validates and performs a status transition.
func (q *QueueItem) TransitionTo(newStatus QueueItemStatus) error {
	allowed := validItemTransitions[q.Status]
	for _, s := range allowed {
		if s == newStatus {
			q.Status = newStatus
			return nil
		}
	}

	return fmt.Errorf("invalid transition from %s to %s", q.Status, newStatus)
}
