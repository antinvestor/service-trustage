package models

import (
	"time"

	"github.com/pitabwire/frame/data"
)

// EventLog records events for outbox publishing.
type EventLog struct {
	data.BaseModel `gorm:"embedded"`

	EventType      string     `gorm:"column:event_type;not null"`
	Source         string     `gorm:"column:source"`
	IdempotencyKey string     `gorm:"column:idempotency_key;uniqueIndex:idx_event_log_idempotency"`
	Payload        string     `gorm:"column:payload;type:jsonb;not null"`
	Published      bool       `gorm:"column:published;not null;default:false"`
	PublishedAt    *time.Time `gorm:"column:published_at"`
}

// TableName returns the database table name.
func (EventLog) TableName() string {
	return "event_log"
}
