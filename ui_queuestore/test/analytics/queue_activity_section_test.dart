import 'dart:convert';

import 'package:antinvestor_ui_core/antinvestor_ui_core.dart';
import 'package:antinvestor_ui_queuestore/antinvestor_ui_queuestore.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;

http.Response ok(Object payload) => http.Response(
  json.encode(payload),
  200,
  headers: {'content-type': 'application/json'},
);

http.Response apiError(int status, String message) => http.Response(
  json.encode({'error': message}),
  status,
  headers: {'content-type': 'application/json'},
);

/// Wires the section into a ProviderScope with a stubbed gate transport.
Widget harness(
  http.Response Function(String path, Map<String, dynamic> body) handler,
) {
  Future<http.Response> transport(String path, {Object? body}) async {
    final decoded = json.decode(body! as String) as Map<String, dynamic>;
    return handler(path, decoded);
  }

  return ProviderScope(
    overrides: [
      analyticsDataSourceProvider.overrideWithValue(
        ThesaAnalyticsDataSource(transport, specs: const [queueAnalyticsSpec]),
      ),
    ],
    child: const MaterialApp(
      home: Scaffold(
        body: SingleChildScrollView(child: QueueActivitySection()),
      ),
    ),
  );
}

void main() {
  testWidgets('renders rejections KPI and combined activity charts', (
    tester,
  ) async {
    await tester.pumpWidget(
      harness((path, body) {
        if (path.endsWith('/scalar')) return ok({'value': 9});
        return ok({
          'points': [
            {'timestamp': '2026-06-01T00:00:00Z', 'value': 4},
            {'timestamp': '2026-06-02T00:00:00Z', 'value': 6},
          ],
        });
      }),
    );
    await tester.pumpAndSettle();

    expect(find.text('Service Activity'), findsOneWidget);
    expect(find.text('Capacity Rejections'), findsOneWidget);
    expect(find.text('9'), findsOneWidget);
    expect(find.text('Queue Operations'), findsOneWidget);
    expect(find.text('Operation Latency (avg ms)'), findsOneWidget);
    // Legends from both combined charts.
    expect(find.text('Enqueued'), findsOneWidget);
    expect(find.text('Dequeued'), findsOneWidget);
    expect(find.text('Completed'), findsOneWidget);
    expect(find.text('Cancelled'), findsOneWidget);
    expect(find.text('No-shows'), findsOneWidget);
    expect(find.text('Avg Enqueue Duration (ms)'), findsOneWidget);
    expect(find.text('Avg Dequeue Duration (ms)'), findsOneWidget);
    expect(find.byType(TimeSeriesChart), findsNWidgets(2));
  });

  testWidgets('shows access message on 403 instead of raw error', (
    tester,
  ) async {
    await tester.pumpWidget(
      harness((_, _) => apiError(403, 'tenant scope required')),
    );
    await tester.pumpAndSettle();

    expect(
      find.textContaining('You do not have access to analytics'),
      findsWidgets,
    );
    expect(find.textContaining('AnalyticsQueryException'), findsNothing);
    expect(find.textContaining('tenant scope required'), findsNothing);
  });

  testWidgets('shows temporary outage message on 503', (tester) async {
    await tester.pumpWidget(
      harness((_, _) => apiError(503, 'backend unavailable')),
    );
    await tester.pumpAndSettle();

    expect(
      find.textContaining('Analytics is temporarily unavailable'),
      findsWidgets,
    );
    expect(find.text('Retry'), findsWidgets);
  });

  testWidgets('shows unsupported-metric message on 400', (tester) async {
    await tester.pumpWidget(
      harness((_, _) => apiError(400, 'metric not allowed')),
    );
    await tester.pumpAndSettle();

    expect(
      find.textContaining('not supported by the analytics service'),
      findsWidgets,
    );
    expect(find.textContaining('metric not allowed'), findsNothing);
  });

  testWidgets('renders empty chart states when the gate returns no data', (
    tester,
  ) async {
    await tester.pumpWidget(
      harness((path, _) {
        if (path.endsWith('/scalar')) return ok({'value': 0});
        return ok({'points': []});
      }),
    );
    await tester.pumpAndSettle();

    expect(find.text('No data'), findsNWidgets(2));
    expect(find.text('0'), findsOneWidget);
  });
}
