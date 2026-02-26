package models

import (
	"fmt"

	"github.com/pitabwire/frame/data"
)

// FormSubmissionStatus enumerates submission statuses.
type FormSubmissionStatus string

const (
	SubmissionStatusPending  FormSubmissionStatus = "pending"
	SubmissionStatusComplete FormSubmissionStatus = "complete"
	SubmissionStatusArchived FormSubmissionStatus = "archived"
)

// ValidSubmissionStatuses lists all valid statuses for validation.
var validSubmissionStatuses = map[FormSubmissionStatus]bool{ //nolint:gochecknoglobals // lookup map
	SubmissionStatusPending:  true,
	SubmissionStatusComplete: true,
	SubmissionStatusArchived: true,
}

// ValidateStatus checks whether the status is a known valid value.
func (s FormSubmissionStatus) ValidateStatus() error {
	if !validSubmissionStatuses[s] {
		return fmt.Errorf("invalid submission status: %q", s)
	}

	return nil
}

// FormSubmission stores submitted form data with file references.
type FormSubmission struct {
	data.BaseModel `gorm:"embedded"`

	FormID         string               `gorm:"column:form_id;not null"`
	SubmitterID    string               `gorm:"column:submitter_id"`
	Status         FormSubmissionStatus `gorm:"column:status;not null;default:pending"`
	Data           string               `gorm:"column:data;type:jsonb;not null"`
	FileCount      int                  `gorm:"column:file_count;not null;default:0"`
	IdempotencyKey string               `gorm:"column:idempotency_key"`
	Metadata       string               `gorm:"column:metadata;type:jsonb"`
}

// TableName returns the database table name.
func (FormSubmission) TableName() string {
	return "form_submissions"
}
