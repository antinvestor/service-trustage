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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

//nolint:gochecknoglobals // metrics are process-wide singletons
var formstoreMeter = otel.Meter("service-trustage/formstore")

// Formstore business metrics.
//
//nolint:gochecknoglobals // metrics are process-wide singletons
var (
	submissionCreateCounter, _ = formstoreMeter.Int64Counter(
		"formstore.submission.create.total",
		metric.WithDescription("Total form submissions created"),
	)
	submissionErrorCounter, _ = formstoreMeter.Int64Counter(
		"formstore.submission.create.errors",
		metric.WithDescription("Total form submission errors"),
	)
	fileUploadCounter, _ = formstoreMeter.Int64Counter(
		"formstore.file.upload.total",
		metric.WithDescription("Total files uploaded via submissions"),
	)
	schemaValidationCounter, _ = formstoreMeter.Int64Counter(
		"formstore.schema.validation.total",
		metric.WithDescription("Total schema validations performed"),
	)
	schemaValidationErrors, _ = formstoreMeter.Int64Counter(
		"formstore.schema.validation.errors",
		metric.WithDescription("Total schema validation failures"),
	)
	definitionCreateCounter, _ = formstoreMeter.Int64Counter(
		"formstore.definition.create.total",
		metric.WithDescription("Total form definitions created"),
	)
	submissionHistogram, _ = formstoreMeter.Float64Histogram(
		"formstore.submission.create.duration_ms",
		metric.WithDescription("Submission creation duration in milliseconds"),
	)
)
