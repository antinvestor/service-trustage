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
var queueMeter = otel.Meter("service-trustage/queue")

// Queue business metrics.
//
//nolint:gochecknoglobals // metrics are process-wide singletons
var (
	enqueueCounter, _ = queueMeter.Int64Counter(
		"queue.enqueue.total",
		metric.WithDescription("Total enqueue operations"),
	)
	enqueueErrorCounter, _ = queueMeter.Int64Counter(
		"queue.enqueue.errors",
		metric.WithDescription("Total enqueue errors"),
	)
	dequeueCounter, _ = queueMeter.Int64Counter(
		"queue.dequeue.total",
		metric.WithDescription("Total dequeue (call-next) operations"),
	)
	dequeueErrorCounter, _ = queueMeter.Int64Counter(
		"queue.dequeue.errors",
		metric.WithDescription("Total dequeue errors"),
	)
	completeCounter, _ = queueMeter.Int64Counter(
		"queue.complete.total",
		metric.WithDescription("Total service completions"),
	)
	cancelCounter, _ = queueMeter.Int64Counter(
		"queue.cancel.total",
		metric.WithDescription("Total cancellations"),
	)
	noShowCounter, _    = queueMeter.Int64Counter("queue.noshow.total", metric.WithDescription("Total no-shows"))
	transferCounter, _  = queueMeter.Int64Counter("queue.transfer.total", metric.WithDescription("Total transfers"))
	enqueueHistogram, _ = queueMeter.Float64Histogram(
		"queue.enqueue.duration_ms",
		metric.WithDescription("Enqueue duration in milliseconds"),
	)
	dequeueHistogram, _ = queueMeter.Float64Histogram(
		"queue.dequeue.duration_ms",
		metric.WithDescription("Dequeue duration in milliseconds"),
	)
	queueFullCounter, _ = queueMeter.Int64Counter(
		"queue.full.total",
		metric.WithDescription("Times enqueue rejected due to capacity"),
	)
)
