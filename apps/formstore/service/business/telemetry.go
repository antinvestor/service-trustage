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

import (
	"github.com/pitabwire/frame/telemetry"
)

//nolint:gochecknoglobals // metrics are process-wide singletons
var formstoreMetrics = telemetry.NewBusinessMetrics("service-trustage/formstore")

// Formstore business metrics. Tenant-scoped transparently: every measurement
// carries tenant_id/partition_id from the claims context.
//
//nolint:gochecknoglobals // metrics are process-wide singletons
var (
	submissionCreateCounter = formstoreMetrics.Counter(
		"formstore.submission.create.total",
		"Total form submissions created",
	)
	submissionErrorCounter = formstoreMetrics.Counter(
		"formstore.submission.create.errors",
		"Total form submission errors",
	)
	fileUploadCounter = formstoreMetrics.Counter(
		"formstore.file.upload.total",
		"Total files uploaded via submissions",
	)
	schemaValidationCounter = formstoreMetrics.Counter(
		"formstore.schema.validation.total",
		"Total schema validations performed",
	)
	schemaValidationErrors = formstoreMetrics.Counter(
		"formstore.schema.validation.errors",
		"Total schema validation failures",
	)
	definitionCreateCounter = formstoreMetrics.Counter(
		"formstore.definition.create.total",
		"Total form definitions created",
	)
	submissionHistogram = formstoreMetrics.Histogram(
		"formstore.submission.create.duration_ms",
		"Submission creation duration in milliseconds",
	)
)
