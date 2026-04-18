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
