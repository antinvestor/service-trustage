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
var queueMetrics = telemetry.NewBusinessMetrics("service-trustage/queue")

// Queue business metrics. Tenant-scoped transparently: every measurement
// carries tenant_id/partition_id from the claims context.
//
//nolint:gochecknoglobals // metrics are process-wide singletons
var (
	enqueueCounter = queueMetrics.Counter(
		"queue.enqueue.total",
		"Total enqueue operations",
	)
	enqueueErrorCounter = queueMetrics.Counter(
		"queue.enqueue.errors",
		"Total enqueue errors",
	)
	dequeueCounter = queueMetrics.Counter(
		"queue.dequeue.total",
		"Total dequeue (call-next) operations",
	)
	dequeueErrorCounter = queueMetrics.Counter(
		"queue.dequeue.errors",
		"Total dequeue errors",
	)
	completeCounter = queueMetrics.Counter(
		"queue.complete.total",
		"Total service completions",
	)
	cancelCounter = queueMetrics.Counter(
		"queue.cancel.total",
		"Total cancellations",
	)
	noShowCounter    = queueMetrics.Counter("queue.noshow.total", "Total no-shows")
	transferCounter  = queueMetrics.Counter("queue.transfer.total", "Total transfers")
	enqueueHistogram = queueMetrics.Histogram(
		"queue.enqueue.duration_ms",
		"Enqueue duration in milliseconds",
	)
	dequeueHistogram = queueMetrics.Histogram(
		"queue.dequeue.duration_ms",
		"Dequeue duration in milliseconds",
	)
	queueFullCounter = queueMetrics.Counter(
		"queue.full.total",
		"Times enqueue rejected due to capacity",
	)
)
