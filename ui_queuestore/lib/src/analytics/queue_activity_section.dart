import 'package:antinvestor_ui_core/antinvestor_ui_core.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'analytics_states.dart';
import 'queue_analytics_spec.dart';

/// Service-level queue activity from the Thesa analytics gate.
///
/// Complements the per-queue snapshot statistics (queue REST API) with
/// tenant-scoped operation trends: enqueue/dequeue/complete/cancel/no-show
/// time series, average operation latency, and a capacity-rejections KPI.
class QueueActivitySection extends ConsumerStatefulWidget {
  const QueueActivitySection({super.key});

  @override
  ConsumerState<QueueActivitySection> createState() =>
      _QueueActivitySectionState();
}

class _QueueActivitySectionState extends ConsumerState<QueueActivitySection> {
  AnalyticsTimeRange _range = AnalyticsTimeRange.last30Days();

  void _refresh() {
    ref.invalidate(serviceMetricsProvider);
    ref.invalidate(serviceTimeSeriesProvider);
  }

  AsyncValue<List<TimeSeries>> _watchSeries(String metric) {
    return ref.watch(
      serviceTimeSeriesProvider(
        ServiceTimeSeriesParams(
          queueAnalyticsSpec.service,
          metric,
          timeRange: _range,
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final metricsAsync = ref.watch(
      serviceMetricsProvider(
        ServiceMetricsParams(queueAnalyticsSpec.service, timeRange: _range),
      ),
    );

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            Expanded(
              child: Text(
                'Service Activity',
                style: theme.textTheme.titleMedium?.copyWith(
                  fontWeight: FontWeight.w600,
                ),
              ),
            ),
            IconButton(
              icon: const Icon(Icons.refresh),
              tooltip: 'Refresh activity',
              onPressed: _refresh,
            ),
          ],
        ),
        Text(
          'Activity across all queues for your tenant',
          style: theme.textTheme.bodySmall?.copyWith(
            color: theme.colorScheme.onSurfaceVariant,
          ),
        ),
        const SizedBox(height: 12),
        SingleChildScrollView(
          scrollDirection: Axis.horizontal,
          child: TimeRangeSelector(
            value: _range,
            onChanged: (range) => setState(() => _range = range),
          ),
        ),
        const SizedBox(height: 16),
        metricsAsync.when(
          data: (metrics) => MetricsRow(metrics: metrics),
          loading: () =>
              const MetricsRow(metrics: [], isLoading: true, skeletonCount: 1),
          error: (error, _) =>
              AnalyticsErrorCard(error: error, onRetry: _refresh),
        ),
        const SizedBox(height: 16),
        _chartCard(
          title: 'Queue Operations',
          asyncSeries: [
            for (final metric in queueActivityMetrics) _watchSeries(metric),
          ],
        ),
        const SizedBox(height: 16),
        _chartCard(
          title: 'Operation Latency (avg ms)',
          asyncSeries: [
            for (final metric in queueLatencyMetrics) _watchSeries(metric),
          ],
        ),
      ],
    );
  }

  /// A card combining several single-series queries into one chart.
  Widget _chartCard({
    required String title,
    required List<AsyncValue<List<TimeSeries>>> asyncSeries,
  }) {
    final theme = Theme.of(context);
    final cs = theme.colorScheme;

    Widget body;
    final failed = asyncSeries.where((a) => a.hasError).toList();
    if (failed.isNotEmpty) {
      body = AnalyticsErrorCard(error: failed.first.error!, onRetry: _refresh);
    } else if (asyncSeries.any((a) => a.isLoading)) {
      body = const SizedBox(
        height: 240,
        child: Center(child: CircularProgressIndicator()),
      );
    } else {
      body = TimeSeriesChart(
        series: [for (final a in asyncSeries) ...a.requireValue],
        granularity: _range.granularity,
      );
    }

    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: cs.surface,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: cs.outlineVariant),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            title,
            style: theme.textTheme.titleSmall?.copyWith(
              fontWeight: FontWeight.w600,
            ),
          ),
          const SizedBox(height: 16),
          body,
        ],
      ),
    );
  }
}
