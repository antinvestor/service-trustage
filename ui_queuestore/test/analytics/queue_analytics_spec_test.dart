import 'dart:convert';

import 'package:antinvestor_ui_core/antinvestor_ui_core.dart';
import 'package:antinvestor_ui_queuestore/antinvestor_ui_queuestore.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;

/// A transport call recorded with its decoded JSON body.
class RecordedRequest {
  RecordedRequest(this.path, this.body);
  final String path;
  final Map<String, dynamic> body;
}

/// Records every request and answers via a per-test handler.
class MockTransport {
  MockTransport(this.handler);

  final http.Response Function(String path, Map<String, dynamic> body) handler;
  final List<RecordedRequest> requests = [];

  Future<http.Response> call(String path, {Object? body}) async {
    final decoded = json.decode(body! as String) as Map<String, dynamic>;
    requests.add(RecordedRequest(path, decoded));
    return handler(path, decoded);
  }
}

http.Response ok(Object payload) => http.Response(
  json.encode(payload),
  200,
  headers: {'content-type': 'application/json'},
);

final fixedRange = AnalyticsTimeRange(
  start: DateTime.utc(2026, 6, 1),
  end: DateTime.utc(2026, 6, 8),
  granularity: TimeGranularity.day,
);

const fixedRangeJson = {
  'start': '2026-06-01T00:00:00.000Z',
  'end': '2026-06-08T00:00:00.000Z',
};

ThesaAnalyticsDataSource sourceWith(MockTransport transport) =>
    ThesaAnalyticsDataSource(transport.call, specs: const [queueAnalyticsSpec]);

void main() {
  group('rejections KPI contract', () {
    test(
      'getMetrics posts one exact scalar body for queue.full.total',
      () async {
        final transport = MockTransport((_, _) => ok({'value': 3}));
        final source = sourceWith(transport);

        final metrics = await source.getMetrics(
          'queuestore',
          timeRange: fixedRange,
        );

        expect(transport.requests.single.path, '/api/analytics/query/scalar');
        expect(transport.requests.single.body, {
          'metric': 'queue.full.total',
          'aggregation': 'sum',
          'time_range': fixedRangeJson,
        });
        expect(metrics.single.key, 'rejections');
        expect(metrics.single.label, 'Capacity Rejections');
        expect(metrics.single.value, 3.0);
      },
    );
  });

  group('operation counter time series contracts', () {
    const expectedLabels = {
      'queue.enqueue.total': 'Enqueued',
      'queue.dequeue.total': 'Dequeued',
      'queue.complete.total': 'Completed',
      'queue.cancel.total': 'Cancelled',
      'queue.noshow.total': 'No-shows',
    };

    for (final entry in expectedLabels.entries) {
      test('${entry.key} posts an exact sum time-series body', () async {
        final transport = MockTransport(
          (_, _) => ok({
            'points': [
              {'timestamp': '2026-06-01T00:00:00Z', 'value': 7},
            ],
          }),
        );
        final source = sourceWith(transport);

        final series = await source.getTimeSeries(
          'queuestore',
          entry.key,
          timeRange: fixedRange,
        );

        expect(
          transport.requests.single.path,
          '/api/analytics/query/timeseries',
        );
        expect(transport.requests.single.body, {
          'metric': entry.key,
          'aggregation': 'sum',
          'time_range': fixedRangeJson,
          'step': 'day',
        });
        expect(series.single.label, entry.value);
        expect(series.single.points.single.value, 7.0);
      });
    }
  });

  group('latency trend contracts', () {
    for (final metric in queueLatencyMetrics) {
      test('$metric posts an exact avg time-series body', () async {
        final transport = MockTransport((_, _) => ok({'points': []}));
        final source = sourceWith(transport);

        await source.getTimeSeries('queuestore', metric, timeRange: fixedRange);

        expect(transport.requests.single.body, {
          'metric': metric,
          'aggregation': 'avg',
          'time_range': fixedRangeJson,
          'step': 'day',
        });
      });
    }
  });

  group('tenancy', () {
    test('never sends tenant_id or partition_id', () async {
      final transport = MockTransport((_, _) => ok({'value': 1}));
      final source = sourceWith(transport);

      await source.getMetrics('queuestore', timeRange: fixedRange);
      for (final metric in [...queueActivityMetrics, ...queueLatencyMetrics]) {
        await source.getTimeSeries('queuestore', metric, timeRange: fixedRange);
      }

      for (final request in transport.requests) {
        final filters =
            request.body['filters'] as Map<String, dynamic>? ?? const {};
        expect(filters.keys, isNot(contains('tenant_id')));
        expect(filters.keys, isNot(contains('partition_id')));
      }
    });

    test('spec declares no tenancy filters anywhere', () {
      expect(queueAnalyticsSpec.baseFilters, isEmpty);
      for (final kpi in queueAnalyticsSpec.kpis) {
        expect(kpi.filters, isNull);
      }
      for (final chart in queueAnalyticsSpec.charts) {
        expect(chart.filters, isNull);
      }
    });
  });

  group('spec declaration', () {
    test('covers all queue activity and latency metrics', () {
      expect(queueAnalyticsSpec.service, 'queuestore');
      expect(queueAnalyticsSpec.metricKeys, ['rejections']);
      for (final metric in queueActivityMetrics) {
        final chart = queueAnalyticsSpec.chartFor(metric);
        expect(chart, isNotNull, reason: metric);
        expect(chart!.aggregation, AnalyticsAggregation.sum, reason: metric);
      }
      for (final metric in queueLatencyMetrics) {
        final chart = queueAnalyticsSpec.chartFor(metric);
        expect(chart, isNotNull, reason: metric);
        expect(chart!.aggregation, AnalyticsAggregation.avg, reason: metric);
      }
    });
  });
}
