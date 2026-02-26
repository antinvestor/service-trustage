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
