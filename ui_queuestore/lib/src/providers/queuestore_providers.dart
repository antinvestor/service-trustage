import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:http/http.dart' as http;

import 'package:antinvestor_ui_core/api/api_base.dart';

import '../api/auth_http_client.dart';
import '../api/queuestore_client.dart';
import '../models/queue_counter.dart';
import '../models/queue_definition.dart';
import '../models/queue_item.dart';
import '../models/queue_stats.dart';

/// Base URL for the Queuestore service. Override via compile-time env.
const _queuestoreUrl = String.fromEnvironment(
  'QUEUESTORE_URL',
  defaultValue: 'https://api.stawi.dev/queuestore',
);

/// Authenticated HTTP client for the queuestore service.
final queuestoreHttpClientProvider = Provider<http.Client>((ref) {
  final tokenProvider = ref.watch(authTokenProviderProvider);
  return AuthHttpClient(http.Client(), tokenProvider);
});

/// Queuestore API client.
final queuestoreClientProvider = Provider<QueuestoreClient>((ref) {
  final client = ref.watch(queuestoreHttpClientProvider);
  return QueuestoreClient(client, _queuestoreUrl);
});

// ---------------------------------------------------------------------------
// Queue definitions
// ---------------------------------------------------------------------------

final queueListProvider =
    FutureProvider.family<List<QueueDefinition>, bool?>((ref, active) async {
  final client = ref.watch(queuestoreClientProvider);
  return client.listQueues(active: active);
});

final queueDetailProvider =
    FutureProvider.family<QueueDefinition, String>((ref, id) async {
  final client = ref.watch(queuestoreClientProvider);
  return client.getQueue(id);
});

class QueueDefinitionNotifier extends Notifier<AsyncValue<void>> {
  @override
  AsyncValue<void> build() => const AsyncValue.data(null);

  QueuestoreClient get _client => ref.read(queuestoreClientProvider);

  Future<QueueDefinition> create(QueueDefinition def) async {
    state = const AsyncValue.loading();
    try {
      final result = await _client.createQueue(def);
      state = const AsyncValue.data(null);
      ref.invalidate(queueListProvider);
      return result;
    } catch (e, st) {
      state = AsyncValue.error(e, st);
      rethrow;
    }
  }

  Future<QueueDefinition> update(
      String id, Map<String, dynamic> updates) async {
    state = const AsyncValue.loading();
    try {
      final result = await _client.updateQueue(id, updates);
      state = const AsyncValue.data(null);
      ref.invalidate(queueListProvider);
      ref.invalidate(queueDetailProvider);
      return result;
    } catch (e, st) {
      state = AsyncValue.error(e, st);
      rethrow;
    }
  }

  Future<void> delete(String id) async {
    state = const AsyncValue.loading();
    try {
      await _client.deleteQueue(id);
      state = const AsyncValue.data(null);
      ref.invalidate(queueListProvider);
    } catch (e, st) {
      state = AsyncValue.error(e, st);
      rethrow;
    }
  }
}

final queueDefinitionNotifierProvider =
    NotifierProvider<QueueDefinitionNotifier, AsyncValue<void>>(
        QueueDefinitionNotifier.new);

// ---------------------------------------------------------------------------
// Queue items
// ---------------------------------------------------------------------------

final queueItemListProvider =
    FutureProvider.family<List<QueueItem>, String>((ref, queueId) async {
  final client = ref.watch(queuestoreClientProvider);
  return client.listItems(queueId);
});

final queueItemProvider =
    FutureProvider.family<QueueItem, String>((ref, id) async {
  final client = ref.watch(queuestoreClientProvider);
  return client.getItem(id);
});

class QueueItemNotifier extends Notifier<AsyncValue<void>> {
  @override
  AsyncValue<void> build() => const AsyncValue.data(null);

  QueuestoreClient get _client => ref.read(queuestoreClientProvider);

  Future<QueueItem> enqueue(String queueId, QueueItem item) async {
    state = const AsyncValue.loading();
    try {
      final result = await _client.enqueue(queueId, item);
      state = const AsyncValue.data(null);
      ref.invalidate(queueItemListProvider);
      ref.invalidate(queueStatsProvider);
      return result;
    } catch (e, st) {
      state = AsyncValue.error(e, st);
      rethrow;
    }
  }

  Future<void> cancel(String id) async {
    state = const AsyncValue.loading();
    try {
      await _client.cancelItem(id);
      state = const AsyncValue.data(null);
      _invalidateAll();
    } catch (e, st) {
      state = AsyncValue.error(e, st);
      rethrow;
    }
  }

  Future<void> noShow(String id) async {
    state = const AsyncValue.loading();
    try {
      await _client.noShowItem(id);
      state = const AsyncValue.data(null);
      _invalidateAll();
    } catch (e, st) {
      state = AsyncValue.error(e, st);
      rethrow;
    }
  }

  Future<void> requeue(String id) async {
    state = const AsyncValue.loading();
    try {
      await _client.requeueItem(id);
      state = const AsyncValue.data(null);
      _invalidateAll();
    } catch (e, st) {
      state = AsyncValue.error(e, st);
      rethrow;
    }
  }

  Future<void> transfer(String id, String targetQueueId) async {
    state = const AsyncValue.loading();
    try {
      await _client.transferItem(id, targetQueueId);
      state = const AsyncValue.data(null);
      _invalidateAll();
    } catch (e, st) {
      state = AsyncValue.error(e, st);
      rethrow;
    }
  }

  void _invalidateAll() {
    ref.invalidate(queueItemListProvider);
    ref.invalidate(queueStatsProvider);
  }
}

final queueItemNotifierProvider =
    NotifierProvider<QueueItemNotifier, AsyncValue<void>>(
        QueueItemNotifier.new);

// ---------------------------------------------------------------------------
// Counters
// ---------------------------------------------------------------------------

final counterListProvider =
    FutureProvider.family<List<QueueCounter>, String>((ref, queueId) async {
  final client = ref.watch(queuestoreClientProvider);
  return client.listCounters(queueId);
});

class CounterNotifier extends Notifier<AsyncValue<void>> {
  @override
  AsyncValue<void> build() => const AsyncValue.data(null);

  QueuestoreClient get _client => ref.read(queuestoreClientProvider);

  Future<QueueCounter> create(String queueId, QueueCounter counter) async {
    state = const AsyncValue.loading();
    try {
      final result = await _client.createCounter(queueId, counter);
      state = const AsyncValue.data(null);
      ref.invalidate(counterListProvider);
      return result;
    } catch (e, st) {
      state = AsyncValue.error(e, st);
      rethrow;
    }
  }

  Future<void> open(String id, {String? staffId}) async {
    state = const AsyncValue.loading();
    try {
      await _client.openCounter(id, staffId: staffId);
      state = const AsyncValue.data(null);
      ref.invalidate(counterListProvider);
    } catch (e, st) {
      state = AsyncValue.error(e, st);
      rethrow;
    }
  }

  Future<void> close(String id) async {
    state = const AsyncValue.loading();
    try {
      await _client.closeCounter(id);
      state = const AsyncValue.data(null);
      ref.invalidate(counterListProvider);
    } catch (e, st) {
      state = AsyncValue.error(e, st);
      rethrow;
    }
  }

  Future<void> pause(String id) async {
    state = const AsyncValue.loading();
    try {
      await _client.pauseCounter(id);
      state = const AsyncValue.data(null);
      ref.invalidate(counterListProvider);
    } catch (e, st) {
      state = AsyncValue.error(e, st);
      rethrow;
    }
  }

  Future<QueueItem> callNext(String counterId) async {
    state = const AsyncValue.loading();
    try {
      final result = await _client.callNext(counterId);
      state = const AsyncValue.data(null);
      ref.invalidate(counterListProvider);
      ref.invalidate(queueItemListProvider);
      ref.invalidate(queueStatsProvider);
      return result;
    } catch (e, st) {
      state = AsyncValue.error(e, st);
      rethrow;
    }
  }

  Future<void> beginService(String counterId) async {
    state = const AsyncValue.loading();
    try {
      await _client.beginService(counterId);
      state = const AsyncValue.data(null);
      ref.invalidate(counterListProvider);
    } catch (e, st) {
      state = AsyncValue.error(e, st);
      rethrow;
    }
  }

  Future<void> completeService(String counterId) async {
    state = const AsyncValue.loading();
    try {
      await _client.completeService(counterId);
      state = const AsyncValue.data(null);
      ref.invalidate(counterListProvider);
      ref.invalidate(queueStatsProvider);
    } catch (e, st) {
      state = AsyncValue.error(e, st);
      rethrow;
    }
  }
}

final counterNotifierProvider =
    NotifierProvider<CounterNotifier, AsyncValue<void>>(CounterNotifier.new);

// ---------------------------------------------------------------------------
// Stats
// ---------------------------------------------------------------------------

final queueStatsProvider =
    FutureProvider.family<QueueStats, String>((ref, queueId) async {
  final client = ref.watch(queuestoreClientProvider);
  return client.getStats(queueId);
});
