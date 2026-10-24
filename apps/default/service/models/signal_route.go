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

import "github.com/pitabwire/frame/v3/data"

// SignalRoute delivers ingested events of EventType as signals to running
// instances of exactly (WorkflowName, WorkflowVersion). Exactly one of
// KeyExpr (match on the instance correlation key) and InstanceExpr (the
// instance id carried in the payload) is set.
type SignalRoute struct {
	data.BaseModel `gorm:"embedded"`

	EventType       string `gorm:"column:event_type;not null"`
	WorkflowName    string `gorm:"column:workflow_name;not null"`
	WorkflowVersion int    `gorm:"column:workflow_version;not null"`
	SignalName      string `gorm:"column:signal_name;not null"`
	Filter          string `gorm:"column:filter"`
	KeyExpr         string `gorm:"column:key_expr"`
	InstanceExpr    string `gorm:"column:instance_expr"`
	Active          bool   `gorm:"column:active;not null;default:true"`
}

// TableName returns the database table name.
func (SignalRoute) TableName() string {
	return "signal_routes"
}
