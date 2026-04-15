import 'dart:convert';

import 'package:http/http.dart' as http;

import '../models/queue_counter.dart';
import '../models/queue_definition.dart';
import '../models/queue_item.dart';
import '../models/queue_stats.dart';

/// REST API client for the Queue service.
class QueuestoreClient {
  QueuestoreClient(this._client, this._baseUrl);

  final http.Client _client;
  final String _baseUrl;

  Uri _uri(String path, [Map<String, String>? queryParams]) =>
      Uri.parse('$_baseUrl$path').replace(queryParameters: queryParams);

  Map<String, String> get _jsonHeaders => {
        'Content-Type': 'application/json',
        'Accept': 'application/json',
      };

  void _checkResponse(http.Response response) {
    if (response.statusCode >= 400) {
      throw QueuestoreApiException(
        response.statusCode,
        response.body.isNotEmpty ? response.body : 'Request failed',
      );
    }
  }

  // -- Queue definitions --

  Future<List<QueueDefinition>> listQueues({bool? active}) async {
    final params = <String, String>{};
    if (active != null) params['active'] = active.toString();
    final response = await _client.get(_uri('/api/v1/queues', params));
    _checkResponse(response);
    final body = jsonDecode(response.body) as Map<String, dynamic>;
    final items = body['items'] as List<dynamic>? ?? [];
    return items
        .map((e) => QueueDefinition.fromJson(e as Map<String, dynamic>))
        .toList();
  }

  Future<QueueDefinition> getQueue(String id) async {
    final response = await _client.get(_uri('/api/v1/queues/$id'));
    _checkResponse(response);
    return QueueDefinition.fromJson(
        jsonDecode(response.body) as Map<String, dynamic>);
  }

  Future<QueueDefinition> createQueue(QueueDefinition def) async {
    final response = await _client.post(
      _uri('/api/v1/queues'),
      headers: _jsonHeaders,
      body: jsonEncode(def.toJson()),
    );
    _checkResponse(response);
    return QueueDefinition.fromJson(
        jsonDecode(response.body) as Map<String, dynamic>);
  }

  Future<QueueDefinition> updateQueue(
      String id, Map<String, dynamic> updates) async {
    final response = await _client.put(
      _uri('/api/v1/queues/$id'),
      headers: _jsonHeaders,
      body: jsonEncode(updates),
    );
    _checkResponse(response);
    return QueueDefinition.fromJson(
        jsonDecode(response.body) as Map<String, dynamic>);
  }

  Future<void> deleteQueue(String id) async {
    final response = await _client.delete(_uri('/api/v1/queues/$id'));
    _checkResponse(response);
  }

  // -- Queue items --

  Future<List<QueueItem>> listItems(String queueId) async {
    final response =
        await _client.get(_uri('/api/v1/queues/$queueId/items'));
    _checkResponse(response);
    final body = jsonDecode(response.body) as Map<String, dynamic>;
    final items = body['items'] as List<dynamic>? ?? [];
    return items
        .map((e) => QueueItem.fromJson(e as Map<String, dynamic>))
        .toList();
  }

  Future<QueueItem> getItem(String id) async {
    final response = await _client.get(_uri('/api/v1/items/$id'));
    _checkResponse(response);
    return QueueItem.fromJson(
        jsonDecode(response.body) as Map<String, dynamic>);
  }

  Future<QueueItem> enqueue(String queueId, QueueItem item) async {
    final response = await _client.post(
      _uri('/api/v1/queues/$queueId/items'),
      headers: _jsonHeaders,
      body: jsonEncode(item.toJson()),
    );
    _checkResponse(response);
    return QueueItem.fromJson(
        jsonDecode(response.body) as Map<String, dynamic>);
  }

  Future<Map<String, dynamic>> getPosition(String itemId) async {
    final response =
        await _client.get(_uri('/api/v1/items/$itemId/position'));
    _checkResponse(response);
    return jsonDecode(response.body) as Map<String, dynamic>;
  }

  Future<void> cancelItem(String id) async {
    final response =
        await _client.post(_uri('/api/v1/items/$id/cancel'));
    _checkResponse(response);
  }

  Future<void> noShowItem(String id) async {
    final response =
        await _client.post(_uri('/api/v1/items/$id/no-show'));
    _checkResponse(response);
  }

  Future<void> requeueItem(String id) async {
    final response =
        await _client.post(_uri('/api/v1/items/$id/requeue'));
    _checkResponse(response);
  }

  Future<void> transferItem(String id, String targetQueueId) async {
    final response = await _client.post(
      _uri('/api/v1/items/$id/transfer'),
      headers: _jsonHeaders,
      body: jsonEncode({'queue_id': targetQueueId}),
    );
    _checkResponse(response);
  }

  // -- Counters --

  Future<List<QueueCounter>> listCounters(String queueId) async {
    final response =
        await _client.get(_uri('/api/v1/queues/$queueId/counters'));
    _checkResponse(response);
    final body = jsonDecode(response.body) as Map<String, dynamic>;
    final items = body['items'] as List<dynamic>? ?? [];
    return items
        .map((e) => QueueCounter.fromJson(e as Map<String, dynamic>))
        .toList();
  }

  Future<QueueCounter> createCounter(
      String queueId, QueueCounter counter) async {
    final response = await _client.post(
      _uri('/api/v1/queues/$queueId/counters'),
      headers: _jsonHeaders,
      body: jsonEncode(counter.toJson()),
    );
    _checkResponse(response);
    return QueueCounter.fromJson(
        jsonDecode(response.body) as Map<String, dynamic>);
  }

  Future<void> openCounter(String id, {String? staffId}) async {
    final response = await _client.post(
      _uri('/api/v1/counters/$id/open'),
      headers: _jsonHeaders,
      body: jsonEncode({if (staffId != null) 'staff_id': staffId}),
    );
    _checkResponse(response);
  }

  Future<void> closeCounter(String id) async {
    final response =
        await _client.post(_uri('/api/v1/counters/$id/close'));
    _checkResponse(response);
  }

  Future<void> pauseCounter(String id) async {
    final response =
        await _client.post(_uri('/api/v1/counters/$id/pause'));
    _checkResponse(response);
  }

  Future<QueueItem> callNext(String counterId) async {
    final response =
        await _client.post(_uri('/api/v1/counters/$counterId/call-next'));
    _checkResponse(response);
    return QueueItem.fromJson(
        jsonDecode(response.body) as Map<String, dynamic>);
  }

  Future<void> beginService(String counterId) async {
    final response = await _client
        .post(_uri('/api/v1/counters/$counterId/begin-service'));
    _checkResponse(response);
  }

  Future<void> completeService(String counterId) async {
    final response = await _client
        .post(_uri('/api/v1/counters/$counterId/complete-service'));
    _checkResponse(response);
  }

  // -- Stats --

  Future<QueueStats> getStats(String queueId) async {
    final response =
        await _client.get(_uri('/api/v1/queues/$queueId/stats'));
    _checkResponse(response);
    return QueueStats.fromJson(
        jsonDecode(response.body) as Map<String, dynamic>);
  }
}

/// Exception thrown when the Queue API returns an error.
class QueuestoreApiException implements Exception {
  const QueuestoreApiException(this.statusCode, this.message);
  final int statusCode;
  final String message;

  @override
  String toString() => 'QueuestoreApiException($statusCode): $message';
}
