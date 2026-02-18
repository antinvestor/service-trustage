package models

import (
	"fmt"

	"github.com/pitabwire/frame/data"
)

// WorkflowDefinitionStatus enumerates workflow definition statuses.
type WorkflowDefinitionStatus string

const (
	WorkflowStatusDraft    WorkflowDefinitionStatus = "draft"
	WorkflowStatusActive   WorkflowDefinitionStatus = "active"
	WorkflowStatusArchived WorkflowDefinitionStatus = "archived"
)

// WorkflowDefinition is a versioned workflow template.
type WorkflowDefinition struct { //nolint:recvcheck // TableName() must be value receiver for GORM
	data.BaseModel `gorm:"embedded"`

	Name            string                   `gorm:"column:name;not null"`
	WorkflowVersion int                      `gorm:"column:workflow_version;not null;default:1"`
	Status          WorkflowDefinitionStatus `gorm:"column:status;not null;default:draft"`
	DSLBlob         string                   `gorm:"column:dsl_blob;type:jsonb;not null"`
	InputSchemaHash string                   `gorm:"column:input_schema_hash"`
	TimeoutSeconds  int64                    `gorm:"column:timeout_seconds;default:0"`
}

// TableName returns the database table name.
func (WorkflowDefinition) TableName() string {
	return "workflow_definitions"
}

// ValidStatusTransitions defines allowed status transitions.
var validDefinitionTransitions = map[WorkflowDefinitionStatus][]WorkflowDefinitionStatus{ //nolint:gochecknoglobals // transition map
	WorkflowStatusDraft:    {WorkflowStatusActive, WorkflowStatusArchived},
	WorkflowStatusActive:   {WorkflowStatusArchived},
	WorkflowStatusArchived: {},
}

// TransitionTo validates and performs a status transition.
func (w *WorkflowDefinition) TransitionTo(newStatus WorkflowDefinitionStatus) error {
	allowed := validDefinitionTransitions[w.Status]
	for _, s := range allowed {
		if s == newStatus {
			w.Status = newStatus
			return nil
		}
	}

	return fmt.Errorf("invalid transition from %s to %s", w.Status, newStatus)
}
