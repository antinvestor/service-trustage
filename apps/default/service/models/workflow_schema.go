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
	"encoding/json"

	"github.com/pitabwire/frame/data"
)

// SchemaType enumerates schema types.
type SchemaType string

const (
	SchemaTypeInput  SchemaType = "input"
	SchemaTypeOutput SchemaType = "output"
	SchemaTypeError  SchemaType = "error"
)

// WorkflowStateSchema stores immutable JSON Schema documents for state contracts.
type WorkflowStateSchema struct {
	data.BaseModel `gorm:"embedded"`

	WorkflowName    string          `gorm:"column:workflow_name;not null"`
	WorkflowVersion int             `gorm:"column:workflow_version;not null"`
	State           string          `gorm:"column:state;not null"`
	SchemaType      SchemaType      `gorm:"column:schema_type;not null"`
	SchemaHash      string          `gorm:"column:schema_hash;not null;index:idx_wss_hash"`
	SchemaBlob      json.RawMessage `gorm:"column:schema_blob;type:jsonb;not null"`
}

// TableName returns the database table name.
func (WorkflowStateSchema) TableName() string {
	return "workflow_state_schemas"
}
