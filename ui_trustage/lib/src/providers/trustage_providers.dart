import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:antinvestor_api_runtime/antinvestor_api_runtime.dart' as rt;
import 'package:antinvestor_api_event/antinvestor_api_event.dart' as ev;
import 'package:antinvestor_api_signal/antinvestor_api_signal.dart' as sig;
import 'package:antinvestor_api_workflow/antinvestor_api_workflow.dart' as wf;

import 'trustage_transport_provider.dart';

// ---------------------------------------------------------------------------
// Query parameter records
// ---------------------------------------------------------------------------

/// Parameters for listing instances.
class InstanceQuery {
  const InstanceQuery({this.workflowName, this.status, this.query, this.limit = 50, this.cursor});
  final String? workflowName;
  final rt.InstanceStatus? status;
  final String? query;
  final int limit;
  final String? cursor;
}

/// Parameters for listing executions.
class ExecutionQuery {
  const ExecutionQuery({this.instanceId, this.status, this.query, this.limit = 50, this.cursor});
  final String? instanceId;
  final rt.ExecutionStatus? status;
  final String? query;
  final int limit;
  final String? cursor;
}

/// Parameters for listing workflows.
class WorkflowQuery {
  const WorkflowQuery({this.name, this.status, this.query, this.limit = 50, this.cursor});
  final String? name;
  final wf.WorkflowStatus? status;
  final String? query;
  final int limit;
  final String? cursor;
}

// ---------------------------------------------------------------------------
// Instance providers
// ---------------------------------------------------------------------------

/// Lists instances with optional filters.
final instanceListProvider =
    FutureProvider.family<rt.ListInstancesResponse, InstanceQuery>((ref, q) async {
  final client = ref.watch(runtimeClientProvider);
  final req = rt.ListInstancesRequest();
  if (q.workflowName != null && q.workflowName!.isNotEmpty) {
    req.workflowName = q.workflowName!;
  }
  if (q.status != null) req.status = q.status!;
  final search = rt.SearchRequest();
  if (q.query != null && q.query!.isNotEmpty) search.query = q.query!;
  final cursor = rt.PageCursor()..limit = q.limit;
  if (q.cursor != null && q.cursor!.isNotEmpty) cursor.page = q.cursor!;
  search.cursor = cursor;
  req.search = search;
  return client.listInstances(req);
});

/// Gets a full instance run with executions, timeline, outputs, scope runs, signals.
final instanceRunProvider =
    FutureProvider.family<rt.GetInstanceRunResponse, String>((ref, instanceId) async {
  final client = ref.watch(runtimeClientProvider);
  final req = rt.GetInstanceRunRequest()
    ..instanceId = instanceId
    ..includePayloads = true
    ..executionLimit = 150
    ..timelineLimit = 150;
  return client.getInstanceRun(req);
});

/// Gets the audit timeline for an instance.
final instanceTimelineProvider =
    FutureProvider.family<ev.GetInstanceTimelineResponse, String>((ref, instanceId) async {
  final client = ref.watch(eventClientProvider);
  final req = ev.GetInstanceTimelineRequest()..instanceId = instanceId;
  return client.getInstanceTimeline(req);
});

// ---------------------------------------------------------------------------
// Execution providers
// ---------------------------------------------------------------------------

/// Lists executions with optional filters.
final executionListProvider =
    FutureProvider.family<rt.ListExecutionsResponse, ExecutionQuery>((ref, q) async {
  final client = ref.watch(runtimeClientProvider);
  final req = rt.ListExecutionsRequest();
  if (q.instanceId != null && q.instanceId!.isNotEmpty) {
    req.instanceId = q.instanceId!;
  }
  if (q.status != null) req.status = q.status!;
  final search = rt.SearchRequest();
  if (q.query != null && q.query!.isNotEmpty) search.query = q.query!;
  final cursor = rt.PageCursor()..limit = q.limit;
  if (q.cursor != null && q.cursor!.isNotEmpty) cursor.page = q.cursor!;
  search.cursor = cursor;
  req.search = search;
  return client.listExecutions(req);
});

/// Gets a single execution with output.
final executionDetailProvider =
    FutureProvider.family<rt.GetExecutionResponse, String>((ref, executionId) async {
  final client = ref.watch(runtimeClientProvider);
  final req = rt.GetExecutionRequest()
    ..executionId = executionId
    ..includeOutput = true;
  return client.getExecution(req);
});

// ---------------------------------------------------------------------------
// Workflow providers
// ---------------------------------------------------------------------------

/// Lists workflow definitions.
final workflowListProvider =
    FutureProvider.family<wf.ListWorkflowsResponse, WorkflowQuery>((ref, q) async {
  final client = ref.watch(workflowClientProvider);
  final req = wf.ListWorkflowsRequest();
  if (q.name != null && q.name!.isNotEmpty) req.name = q.name!;
  if (q.status != null) req.status = q.status!;
  final search = wf.SearchRequest();
  if (q.query != null && q.query!.isNotEmpty) search.query = q.query!;
  final cursor = wf.PageCursor()..limit = q.limit;
  if (q.cursor != null && q.cursor!.isNotEmpty) cursor.page = q.cursor!;
  search.cursor = cursor;
  req.search = search;
  return client.listWorkflows(req);
});

// ---------------------------------------------------------------------------
// Action notifiers
// ---------------------------------------------------------------------------

/// Notifier for operator actions (retry, resume, signal, event ingest).
class TrustageActionNotifier extends Notifier<AsyncValue<void>> {
  @override
  AsyncValue<void> build() => const AsyncValue.data(null);

  /// Retry a specific execution.
  Future<rt.WorkflowExecution> retryExecution(String executionId) async {
    state = const AsyncValue.loading();
    try {
      final client = ref.read(runtimeClientProvider);
      final req = rt.RetryExecutionRequest()..executionId = executionId;
      final resp = await client.retryExecution(req);
      state = const AsyncValue.data(null);
      _invalidateAll();
      return resp.execution;
    } catch (e, st) {
      state = AsyncValue.error(e, st);
      rethrow;
    }
  }

  /// Retry an entire instance (creates new execution for current state).
  Future<rt.WorkflowExecution> retryInstance(String instanceId) async {
    state = const AsyncValue.loading();
    try {
      final client = ref.read(runtimeClientProvider);
      final req = rt.RetryInstanceRequest()..instanceId = instanceId;
      final resp = await client.retryInstance(req);
      state = const AsyncValue.data(null);
      _invalidateAll();
      return resp.execution;
    } catch (e, st) {
      state = AsyncValue.error(e, st);
      rethrow;
    }
  }

  /// Resume a waiting execution with a payload.
  Future<void> resumeExecution(String executionId, rt.Struct? payload) async {
    state = const AsyncValue.loading();
    try {
      final client = ref.read(runtimeClientProvider);
      final req = rt.ResumeExecutionRequest()..executionId = executionId;
      if (payload != null) req.payload = payload;
      await client.resumeExecution(req);
      state = const AsyncValue.data(null);
      _invalidateAll();
    } catch (e, st) {
      state = AsyncValue.error(e, st);
      rethrow;
    }
  }

  /// Send a signal to a workflow instance.
  Future<bool> sendSignal(String instanceId, String signalName, Map<String, dynamic>? payload) async {
    state = const AsyncValue.loading();
    try {
      final client = ref.read(signalClientProvider);
      final req = sig.SendSignalRequest()
        ..instanceId = instanceId
        ..signalName = signalName;
      if (payload != null) req.payload = (sig.Struct()..mergeFromJsonMap(payload));
      final resp = await client.sendSignal(req);
      state = const AsyncValue.data(null);
      _invalidateAll();
      return resp.delivered;
    } catch (e, st) {
      state = AsyncValue.error(e, st);
      rethrow;
    }
  }

  /// Ingest an external event to trigger workflows.
  Future<ev.IngestEventResponse> ingestEvent({
    required String eventType,
    required String source,
    String? idempotencyKey,
    Map<String, dynamic>? payload,
  }) async {
    state = const AsyncValue.loading();
    try {
      final client = ref.read(eventClientProvider);
      final req = ev.IngestEventRequest()
        ..eventType = eventType
        ..source = source;
      if (idempotencyKey != null && idempotencyKey.isNotEmpty) {
        req.idempotencyKey = idempotencyKey;
      }
      if (payload != null) req.payload = (ev.Struct()..mergeFromJsonMap(payload));
      final resp = await client.ingestEvent(req);
      state = const AsyncValue.data(null);
      _invalidateAll();
      return resp;
    } catch (e, st) {
      state = AsyncValue.error(e, st);
      rethrow;
    }
  }

  /// Activate a workflow definition.
  Future<wf.WorkflowDefinition> activateWorkflow(String workflowId) async {
    state = const AsyncValue.loading();
    try {
      final client = ref.read(workflowClientProvider);
      final req = wf.ActivateWorkflowRequest()..id = workflowId;
      final resp = await client.activateWorkflow(req);
      state = const AsyncValue.data(null);
      ref.invalidate(workflowListProvider);
      return resp.workflow;
    } catch (e, st) {
      state = AsyncValue.error(e, st);
      rethrow;
    }
  }

  void _invalidateAll() {
    ref.invalidate(instanceListProvider);
    ref.invalidate(executionListProvider);
    ref.invalidate(instanceRunProvider);
    ref.invalidate(instanceTimelineProvider);
  }
}

final trustageActionProvider =
    NotifierProvider<TrustageActionNotifier, AsyncValue<void>>(
        TrustageActionNotifier.new);
