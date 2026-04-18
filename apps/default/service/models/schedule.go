package models

import (
	"time"

	"github.com/pitabwire/frame/data"
)

// ScheduleDefinition defines a cron schedule that triggers workflow events.
type ScheduleDefinition struct {
	data.BaseModel `gorm:"embedded"`

	Name            string     `gorm:"column:name;not null"`
	CronExpr        string     `gorm:"column:cron_expr;not null"`
	WorkflowName    string     `gorm:"column:workflow_name;not null"`
	WorkflowVersion int        `gorm:"column:workflow_version;not null"`
	InputPayload    string     `gorm:"column:input_payload;type:jsonb;default:'{}'"`
	Active          bool       `gorm:"column:active;not null;default:false"`
	NextFireAt      *time.Time `gorm:"column:next_fire_at"`
	LastFiredAt     *time.Time `gorm:"column:last_fired_at"`
	JitterSeconds   int        `gorm:"column:jitter_seconds;not null;default:0"`
}

// TableName returns the database table name.
func (ScheduleDefinition) TableName() string {
	return "schedule_definitions"
}
