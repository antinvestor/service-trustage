import 'package:antinvestor_ui_core/antinvestor_ui_core.dart';
import 'package:flutter/material.dart';

/// Metric names emitted by `apps/queue/service/business/telemetry.go`.
const queueEnqueueTotalMetric = 'queue.enqueue.total';
const queueDequeueTotalMetric = 'queue.dequeue.total';
const queueCompleteTotalMetric = 'queue.complete.total';
const queueCancelTotalMetric = 'queue.cancel.total';
const queueNoShowTotalMetric = 'queue.noshow.total';
const queueEnqueueDurationMetric = 'queue.enqueue.duration_ms';
const queueDequeueDurationMetric = 'queue.dequeue.duration_ms';
const queueFullTotalMetric = 'queue.full.total';

/// The five queue operation counters charted as activity time series.
const queueActivityMetrics = [
  queueEnqueueTotalMetric,
  queueDequeueTotalMetric,
  queueCompleteTotalMetric,
  queueCancelTotalMetric,
  queueNoShowTotalMetric,
];

/// The two operation latency histograms charted as average trends.
const queueLatencyMetrics = [
  queueEnqueueDurationMetric,
  queueDequeueDurationMetric,
];

/// Analytics catalog for the queue service, served by the Thesa analytics
/// gate.
///
/// Per-queue snapshot statistics stay on the queue REST API; this spec only
/// covers service-level activity. Tenant scoping is injected server-side
/// from the caller's JWT; no tenant filters are (or may be) declared here.
const queueAnalyticsSpec = ServiceAnalyticsSpec(
  service: 'queuestore',
  kpis: [
    KpiSpec(
      'rejections',
      label: 'Capacity Rejections',
      metric: queueFullTotalMetric,
      unit: 'count',
      icon: Icons.block_outlined,
    ),
  ],
  charts: [
    ChartConfig.timeSeries(queueEnqueueTotalMetric, label: 'Enqueued'),
    ChartConfig.timeSeries(queueDequeueTotalMetric, label: 'Dequeued'),
    ChartConfig.timeSeries(queueCompleteTotalMetric, label: 'Completed'),
    ChartConfig.timeSeries(queueCancelTotalMetric, label: 'Cancelled'),
    ChartConfig.timeSeries(queueNoShowTotalMetric, label: 'No-shows'),
    ChartConfig.timeSeries(
      queueEnqueueDurationMetric,
      label: 'Avg Enqueue Duration (ms)',
      aggregation: AnalyticsAggregation.avg,
    ),
    ChartConfig.timeSeries(
      queueDequeueDurationMetric,
      label: 'Avg Dequeue Duration (ms)',
      aggregation: AnalyticsAggregation.avg,
    ),
  ],
);
