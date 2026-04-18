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

// WorkflowStateOutput stores the validated output of each state execution.
type WorkflowStateOutput struct {
	data.BaseModel `gorm:"embedded"`

	ExecutionID string `gorm:"column:execution_id;not null"`
	InstanceID  string `gorm:"column:instance_id;not null"`
	State       string `gorm:"column:state;not null"`
	SchemaHash  string `gorm:"column:schema_hash;not null"`
	Payload     string `gorm:"column:payload;type:jsonb;not null"`
}

// TableName returns the database table name.
func (WorkflowStateOutput) TableName() string {
	return "workflow_state_outputs"
}
