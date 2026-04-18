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
