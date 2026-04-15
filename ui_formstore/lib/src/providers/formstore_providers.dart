import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:http/http.dart' as http;

import 'package:antinvestor_ui_core/api/api_base.dart';

import '../api/auth_http_client.dart';
import '../api/formstore_client.dart';
import '../models/form_definition.dart';
import '../models/form_submission.dart';

/// Base URL for the Formstore service. Override via compile-time env.
const _formstoreUrl = String.fromEnvironment(
  'FORMSTORE_URL',
  defaultValue: 'https://api.stawi.dev/formstore',
);

/// Authenticated HTTP client for the formstore service.
final formstoreHttpClientProvider = Provider<http.Client>((ref) {
  final tokenProvider = ref.watch(authTokenProviderProvider);
  return AuthHttpClient(http.Client(), tokenProvider);
});

/// Formstore API client.
final formstoreClientProvider = Provider<FormstoreClient>((ref) {
  final client = ref.watch(formstoreHttpClientProvider);
  return FormstoreClient(client, _formstoreUrl);
});

// ---------------------------------------------------------------------------
// Form definitions
// ---------------------------------------------------------------------------

/// Lists form definitions. Optionally filter by [active] status.
final formDefinitionListProvider =
    FutureProvider.family<List<FormDefinition>, bool?>((ref, active) async {
  final client = ref.watch(formstoreClientProvider);
  return client.listFormDefinitions(active: active);
});

/// Gets a single form definition by ID.
final formDefinitionProvider =
    FutureProvider.family<FormDefinition, String>((ref, id) async {
  final client = ref.watch(formstoreClientProvider);
  return client.getFormDefinition(id);
});

/// Notifier for form definition mutations (create / update / delete).
class FormDefinitionNotifier extends Notifier<AsyncValue<void>> {
  @override
  AsyncValue<void> build() => const AsyncValue.data(null);

  FormstoreClient get _client => ref.read(formstoreClientProvider);

  Future<FormDefinition> create(FormDefinition def) async {
    state = const AsyncValue.loading();
    try {
      final result = await _client.createFormDefinition(def);
      state = const AsyncValue.data(null);
      ref.invalidate(formDefinitionListProvider);
      return result;
    } catch (e, st) {
      state = AsyncValue.error(e, st);
      rethrow;
    }
  }

  Future<FormDefinition> update(
      String id, Map<String, dynamic> updates) async {
    state = const AsyncValue.loading();
    try {
      final result = await _client.updateFormDefinition(id, updates);
      state = const AsyncValue.data(null);
      ref.invalidate(formDefinitionListProvider);
      ref.invalidate(formDefinitionProvider);
      return result;
    } catch (e, st) {
      state = AsyncValue.error(e, st);
      rethrow;
    }
  }

  Future<void> delete(String id) async {
    state = const AsyncValue.loading();
    try {
      await _client.deleteFormDefinition(id);
      state = const AsyncValue.data(null);
      ref.invalidate(formDefinitionListProvider);
    } catch (e, st) {
      state = AsyncValue.error(e, st);
      rethrow;
    }
  }
}

final formDefinitionNotifierProvider =
    NotifierProvider<FormDefinitionNotifier, AsyncValue<void>>(
        FormDefinitionNotifier.new);

// ---------------------------------------------------------------------------
// Submissions
// ---------------------------------------------------------------------------

/// Lists submissions for a form.
final submissionListProvider =
    FutureProvider.family<List<FormSubmission>, String>((ref, formId) async {
  final client = ref.watch(formstoreClientProvider);
  return client.listSubmissions(formId);
});

/// Gets a single submission by ID.
final submissionProvider =
    FutureProvider.family<FormSubmission, String>((ref, id) async {
  final client = ref.watch(formstoreClientProvider);
  return client.getSubmission(id);
});

/// Notifier for submission mutations.
class SubmissionNotifier extends Notifier<AsyncValue<void>> {
  @override
  AsyncValue<void> build() => const AsyncValue.data(null);

  FormstoreClient get _client => ref.read(formstoreClientProvider);

  Future<FormSubmission> submit(String formId, FormSubmission sub) async {
    state = const AsyncValue.loading();
    try {
      final result = await _client.createSubmission(formId, sub);
      state = const AsyncValue.data(null);
      ref.invalidate(submissionListProvider);
      return result;
    } catch (e, st) {
      state = AsyncValue.error(e, st);
      rethrow;
    }
  }

  Future<FormSubmission> update(
      String id, Map<String, dynamic> updates) async {
    state = const AsyncValue.loading();
    try {
      final result = await _client.updateSubmission(id, updates);
      state = const AsyncValue.data(null);
      ref.invalidate(submissionListProvider);
      ref.invalidate(submissionProvider);
      return result;
    } catch (e, st) {
      state = AsyncValue.error(e, st);
      rethrow;
    }
  }

  Future<void> delete(String id) async {
    state = const AsyncValue.loading();
    try {
      await _client.deleteSubmission(id);
      state = const AsyncValue.data(null);
      ref.invalidate(submissionListProvider);
    } catch (e, st) {
      state = AsyncValue.error(e, st);
      rethrow;
    }
  }
}

final submissionNotifierProvider =
    NotifierProvider<SubmissionNotifier, AsyncValue<void>>(
        SubmissionNotifier.new);
