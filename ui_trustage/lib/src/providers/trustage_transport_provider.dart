import 'package:connectrpc/connect.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:antinvestor_ui_core/api/api_base.dart';

import 'package:antinvestor_api_runtime/antinvestor_api_runtime.dart';
import 'package:antinvestor_api_event/antinvestor_api_event.dart';
import 'package:antinvestor_api_signal/antinvestor_api_signal.dart';
import 'package:antinvestor_api_workflow/antinvestor_api_workflow.dart';

const _trustageUrl = String.fromEnvironment(
  'TRUSTAGE_URL',
  defaultValue: 'https://api.stawi.dev/trustage',
);

/// ConnectRPC transport for the trustage service.
final trustageTransportProvider = Provider<Transport>((ref) {
  final tokenProvider = ref.watch(authTokenProviderProvider);
  return createTransport(tokenProvider, baseUrl: _trustageUrl);
});

/// Runtime service client (instances, executions, retries).
final runtimeClientProvider = Provider<RuntimeServiceClient>((ref) {
  final transport = ref.watch(trustageTransportProvider);
  return RuntimeServiceClient(transport);
});

/// Event service client (ingest events, timelines).
final eventClientProvider = Provider<EventServiceClient>((ref) {
  final transport = ref.watch(trustageTransportProvider);
  return EventServiceClient(transport);
});

/// Signal service client (send signals).
final signalClientProvider = Provider<SignalServiceClient>((ref) {
  final transport = ref.watch(trustageTransportProvider);
  return SignalServiceClient(transport);
});

/// Workflow service client (definitions, activation).
final workflowClientProvider = Provider<WorkflowServiceClient>((ref) {
  final transport = ref.watch(trustageTransportProvider);
  return WorkflowServiceClient(transport);
});
