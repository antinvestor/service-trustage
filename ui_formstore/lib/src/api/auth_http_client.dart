import 'package:http/http.dart' as http;

import 'package:antinvestor_ui_core/auth/auth_token_provider.dart';

/// HTTP client that injects the Bearer token into every request
/// and retries once on 401 after refreshing the token.
class AuthHttpClient extends http.BaseClient {
  AuthHttpClient(this._inner, this._tokenProvider);

  final http.Client _inner;
  final AuthTokenProvider _tokenProvider;

  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    final token = await _tokenProvider.ensureValidAccessToken();
    if (token != null) {
      request.headers['Authorization'] = 'Bearer $token';
    }

    final response = await _inner.send(request);

    if (response.statusCode == 401) {
      final newToken = await _tokenProvider.forceRefreshAccessToken();
      if (newToken != null) {
        final retry = _copyRequest(request);
        retry.headers['Authorization'] = 'Bearer $newToken';
        return _inner.send(retry);
      }
    }

    return response;
  }

  http.Request _copyRequest(http.BaseRequest original) {
    final copy = http.Request(original.method, original.url);
    copy.headers.addAll(original.headers);
    if (original is http.Request) {
      copy.body = original.body;
    }
    return copy;
  }
}
