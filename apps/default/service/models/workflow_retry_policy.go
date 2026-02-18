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
