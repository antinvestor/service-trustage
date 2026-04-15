import 'dart:convert';

import 'package:http/http.dart' as http;

import '../models/form_definition.dart';
import '../models/form_submission.dart';

/// REST API client for the Formstore service.
class FormstoreClient {
  FormstoreClient(this._client, this._baseUrl);

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
      throw FormstoreApiException(
        response.statusCode,
        response.body.isNotEmpty ? response.body : 'Request failed',
      );
    }
  }

  // -- Form definitions --

  Future<List<FormDefinition>> listFormDefinitions({bool? active}) async {
    final params = <String, String>{};
    if (active != null) params['active'] = active.toString();
    final response = await _client.get(_uri('/api/v1/form-definitions', params));
    _checkResponse(response);
    final body = jsonDecode(response.body) as Map<String, dynamic>;
    final items = body['items'] as List<dynamic>? ?? [];
    return items
        .map((e) => FormDefinition.fromJson(e as Map<String, dynamic>))
        .toList();
  }

  Future<FormDefinition> getFormDefinition(String id) async {
    final response = await _client.get(_uri('/api/v1/form-definitions/$id'));
    _checkResponse(response);
    return FormDefinition.fromJson(
        jsonDecode(response.body) as Map<String, dynamic>);
  }

  Future<FormDefinition> createFormDefinition(FormDefinition def) async {
    final response = await _client.post(
      _uri('/api/v1/form-definitions'),
      headers: _jsonHeaders,
      body: jsonEncode(def.toJson()),
    );
    _checkResponse(response);
    return FormDefinition.fromJson(
        jsonDecode(response.body) as Map<String, dynamic>);
  }

  Future<FormDefinition> updateFormDefinition(
      String id, Map<String, dynamic> updates) async {
    final response = await _client.put(
      _uri('/api/v1/form-definitions/$id'),
      headers: _jsonHeaders,
      body: jsonEncode(updates),
    );
    _checkResponse(response);
    return FormDefinition.fromJson(
        jsonDecode(response.body) as Map<String, dynamic>);
  }

  Future<void> deleteFormDefinition(String id) async {
    final response =
        await _client.delete(_uri('/api/v1/form-definitions/$id'));
    _checkResponse(response);
  }

  // -- Submissions --

  Future<List<FormSubmission>> listSubmissions(String formId,
      {int? limit, int? offset}) async {
    final params = <String, String>{};
    if (limit != null) params['limit'] = limit.toString();
    if (offset != null) params['offset'] = offset.toString();
    final response = await _client
        .get(_uri('/api/v1/forms/$formId/submissions', params));
    _checkResponse(response);
    final body = jsonDecode(response.body) as Map<String, dynamic>;
    final items = body['items'] as List<dynamic>? ?? [];
    return items
        .map((e) => FormSubmission.fromJson(e as Map<String, dynamic>))
        .toList();
  }

  Future<FormSubmission> getSubmission(String id) async {
    final response = await _client.get(_uri('/api/v1/submissions/$id'));
    _checkResponse(response);
    return FormSubmission.fromJson(
        jsonDecode(response.body) as Map<String, dynamic>);
  }

  Future<FormSubmission> createSubmission(
      String formId, FormSubmission sub) async {
    final response = await _client.post(
      _uri('/api/v1/forms/$formId/submissions'),
      headers: _jsonHeaders,
      body: jsonEncode(sub.toJson()),
    );
    _checkResponse(response);
    return FormSubmission.fromJson(
        jsonDecode(response.body) as Map<String, dynamic>);
  }

  Future<FormSubmission> updateSubmission(
      String id, Map<String, dynamic> updates) async {
    final response = await _client.put(
      _uri('/api/v1/submissions/$id'),
      headers: _jsonHeaders,
      body: jsonEncode(updates),
    );
    _checkResponse(response);
    return FormSubmission.fromJson(
        jsonDecode(response.body) as Map<String, dynamic>);
  }

  Future<void> deleteSubmission(String id) async {
    final response = await _client.delete(_uri('/api/v1/submissions/$id'));
    _checkResponse(response);
  }
}

/// Exception thrown when the Formstore API returns an error.
class FormstoreApiException implements Exception {
  const FormstoreApiException(this.statusCode, this.message);
  final int statusCode;
  final String message;

  @override
  String toString() => 'FormstoreApiException($statusCode): $message';
}
