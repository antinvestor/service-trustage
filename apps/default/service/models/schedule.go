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
