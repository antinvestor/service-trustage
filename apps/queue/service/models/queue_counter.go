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
