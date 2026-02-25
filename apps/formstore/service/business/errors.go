package business

import "errors"

// Sentinel errors for the form store business layer.
var (
	ErrFormDefinitionNotFound = errors.New("form definition not found")
	ErrFormSubmissionNotFound = errors.New("form submission not found")
	ErrDuplicateFormID        = errors.New("form_id already exists")
	ErrInvalidFormData        = errors.New("invalid form data")
	ErrSchemaValidationFailed = errors.New("form data does not match schema")
	ErrFileUploadFailed       = errors.New("file upload failed")
	ErrSubmissionTooLarge     = errors.New("submission exceeds maximum size")
	ErrInvalidStatus          = errors.New("invalid submission status")
)
