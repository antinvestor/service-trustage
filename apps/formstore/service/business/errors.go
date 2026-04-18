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
