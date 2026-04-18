package dsl

import "fmt"

// ValidationErrorCode enumerates all possible DSL validation failures.
type ValidationErrorCode string

const (
	ErrCycleDetected       ValidationErrorCode = "CYCLE_DETECTED"
	ErrUnreachableState    ValidationErrorCode = "UNREACHABLE_STATE"
	ErrMissingSchema       ValidationErrorCode = "MISSING_SCHEMA"
	ErrMissingMapping      ValidationErrorCode = "MISSING_MAPPING"
	ErrIncompatibleMapping ValidationErrorCode = "INCOMPATIBLE_MAPPING"
	ErrDuplicateState      ValidationErrorCode = "DUPLICATE_STATE"
	ErrInvalidExpression   ValidationErrorCode = "INVALID_EXPRESSION"
	ErrInvalidTemplate     ValidationErrorCode = "INVALID_TEMPLATE"
	ErrInvalidRetry        ValidationErrorCode = "INVALID_RETRY"
	ErrInvalidTimeout      ValidationErrorCode = "INVALID_TIMEOUT"
	ErrDuplicateStepID     ValidationErrorCode = "DUPLICATE_STEP_ID"
	ErrInvalidStepType     ValidationErrorCode = "INVALID_STEP_TYPE"
	ErrMissingRequired     ValidationErrorCode = "MISSING_REQUIRED"
	ErrInvalidReference    ValidationErrorCode = "INVALID_REFERENCE"
	ErrInvalidSchedule     ValidationErrorCode = "INVALID_SCHEDULE"
)

// ValidationError represents a single validation failure in a DSL document.
type ValidationError struct {
	Code    ValidationErrorCode `json:"code"`
	Message string              `json:"message"`
	Path    string              `json:"path,omitempty"`
	StepID  string              `json:"step_id,omitempty"`
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	if e.StepID != "" {
		return fmt.Sprintf("%s (step %s): %s", e.Code, e.StepID, e.Message)
	}

	if e.Path != "" {
		return fmt.Sprintf("%s (%s): %s", e.Code, e.Path, e.Message)
	}

	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// ValidationResult collects all validation errors for a DSL document.
type ValidationResult struct {
	Errors []*ValidationError `json:"errors,omitempty"`
}

// Valid returns true if there are no validation errors.
func (r *ValidationResult) Valid() bool {
	return len(r.Errors) == 0
}

// AddError appends a validation error.
func (r *ValidationResult) AddError(code ValidationErrorCode, message string) {
	r.Errors = append(r.Errors, &ValidationError{
		Code:    code,
		Message: message,
	})
}

// AddErrorWithStep appends a validation error associated with a step ID.
func (r *ValidationResult) AddErrorWithStep(code ValidationErrorCode, stepID, message string) {
	r.Errors = append(r.Errors, &ValidationError{
		Code:    code,
		Message: message,
		StepID:  stepID,
	})
}

// AddErrorWithPath appends a validation error with a JSON path.
func (r *ValidationResult) AddErrorWithPath(code ValidationErrorCode, path, message string) {
	r.Errors = append(r.Errors, &ValidationError{
		Code:    code,
		Message: message,
		Path:    path,
	})
}

// Error returns a combined error message or nil if valid.
func (r *ValidationResult) Error() error {
	if r.Valid() {
		return nil
	}

	return fmt.Errorf("DSL validation failed with %d error(s): %s", len(r.Errors), r.Errors[0].Error())
}
