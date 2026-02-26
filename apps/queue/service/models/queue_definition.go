package models

import "github.com/pitabwire/frame/data"

// QueueDefinition represents a named queue with configuration.
type QueueDefinition struct {
	data.BaseModel `gorm:"embedded"`

	Name           string `gorm:"column:name;not null"`
	Description    string `gorm:"column:description"`
	Active         bool   `gorm:"column:active;not null;default:true"`
	PriorityLevels int    `gorm:"column:priority_levels;not null;default:3"`
	MaxCapacity    int    `gorm:"column:max_capacity;not null;default:0"`
	SLAMinutes     int    `gorm:"column:sla_minutes;not null;default:30"`
	Config         string `gorm:"column:config;type:jsonb"`
}

// TableName returns the database table name.
func (QueueDefinition) TableName() string {
	return "queue_definitions"
}
