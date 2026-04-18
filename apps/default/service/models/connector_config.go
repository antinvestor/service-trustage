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

// ConnectorConfig stores connector adapter configuration.
type ConnectorConfig struct {
	data.BaseModel `gorm:"embedded"`

	ConnectorType string `gorm:"column:connector_type;not null"`
	Name          string `gorm:"column:name;not null"`
	Config        string `gorm:"column:config;type:jsonb;default:'{}'"`
	Active        bool   `gorm:"column:active;not null;default:true"`
}

// TableName returns the database table name.
func (ConnectorConfig) TableName() string {
	return "connector_configs"
}
