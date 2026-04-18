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

// WorkflowRetryPolicy configures retry behavior for a specific state.
type WorkflowRetryPolicy struct {
	data.BaseModel `gorm:"embedded"`

	WorkflowName    string `gorm:"column:workflow_name;not null"`
	WorkflowVersion int    `gorm:"column:workflow_version;not null"`
	State           string `gorm:"column:state;not null"`
	MaxAttempts     int    `gorm:"column:max_attempts;not null;default:3"`
	BackoffStrategy string `gorm:"column:backoff_strategy;not null;default:exponential"`
	InitialDelayMs  int64  `gorm:"column:initial_delay_ms;not null;default:1000"`
	MaxDelayMs      int64  `gorm:"column:max_delay_ms;not null;default:300000"`
	RetryOn         string `gorm:"column:retry_on;type:text[];not null;default:'{retryable,external_dependency}'"`
}

// TableName returns the database table name.
func (WorkflowRetryPolicy) TableName() string {
	return "workflow_retry_policies"
}
