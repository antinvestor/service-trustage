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

// WorkflowScopeRun tracks durable branch orchestration for parallel and foreach steps.
type WorkflowScopeRun struct {
	data.BaseModel `gorm:"embedded"`

	ParentExecutionID string `gorm:"column:parent_execution_id;not null"`
	ParentInstanceID  string `gorm:"column:parent_instance_id;not null"`
	ParentState       string `gorm:"column:parent_state;not null"`
	ScopeType         string `gorm:"column:scope_type;not null"`
	Status            string `gorm:"column:status;not null;default:running"`
	WaitAll           bool   `gorm:"column:wait_all;not null;default:true"`

	TotalChildren     int `gorm:"column:total_children;not null;default:0"`
	CompletedChildren int `gorm:"column:completed_children;not null;default:0"`
	FailedChildren    int `gorm:"column:failed_children;not null;default:0"`
	NextChildIndex    int `gorm:"column:next_child_index;not null;default:0"`
	MaxConcurrency    int `gorm:"column:max_concurrency;not null;default:1"`

	ItemVar        string `gorm:"column:item_var"`
	IndexVar       string `gorm:"column:index_var"`
	ItemsPayload   string `gorm:"column:items_payload;type:jsonb;default:'[]'"`
	ResultsPayload string `gorm:"column:results_payload;type:jsonb;default:'[]'"`

	ClaimUntil *time.Time `gorm:"column:claim_until"`
	ClaimOwner string     `gorm:"column:claim_owner"`
}

// TableName returns the database table name.
func (WorkflowScopeRun) TableName() string {
	return "workflow_scope_runs"
}
