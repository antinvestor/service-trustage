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

// WorkflowStateMapping stores data mapping expressions between states.
type WorkflowStateMapping struct {
	data.BaseModel `gorm:"embedded"`

	WorkflowName    string `gorm:"column:workflow_name;not null"`
	WorkflowVersion int    `gorm:"column:workflow_version;not null"`
	FromState       string `gorm:"column:from_state;not null"`
	ToState         string `gorm:"column:to_state;not null"`
	MappingExpr     string `gorm:"column:mapping_expr;type:jsonb;not null"`
}

// TableName returns the database table name.
func (WorkflowStateMapping) TableName() string {
	return "workflow_state_mappings"
}
