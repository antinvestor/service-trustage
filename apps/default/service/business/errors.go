package business

import "errors"

// Sentinel errors for the business layer.
var (
	ErrWorkflowNotFound        = errors.New("workflow not found")
	ErrInstanceNotFound        = errors.New("instance not found")
	ErrExecutionNotFound       = errors.New("execution not found")
	ErrStaleExecution          = errors.New("stale execution: CAS transition failed")
	ErrInvalidToken            = errors.New("invalid execution token")
	ErrInputContractViolation  = errors.New("input contract violation")
	ErrOutputContractViolation = errors.New("output contract violation")
	ErrWorkflowAlreadyActive   = errors.New("workflow already active")
	ErrInvalidWorkflowStatus   = errors.New("invalid workflow status transition")
	ErrSchemaNotFound          = errors.New("schema not found")
	ErrMappingNotFound         = errors.New("mapping not found")
	ErrDSLValidationFailed     = errors.New("DSL validation failed")
	ErrTriggerNotFound         = errors.New("trigger binding not found")
)
