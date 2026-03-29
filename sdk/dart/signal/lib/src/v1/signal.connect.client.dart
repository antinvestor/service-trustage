//
//  Generated code. Do not modify.
//  source: v1/signal.proto
//

import "package:connectrpc/connect.dart" as connect;
import "signal.pb.dart" as v1signal;
import "signal.connect.spec.dart" as specs;

extension type SignalServiceClient (connect.Transport _transport) {
  Future<v1signal.SendSignalResponse> sendSignal(
    v1signal.SendSignalRequest input, {
    connect.Headers? headers,
    connect.AbortSignal? signal,
    Function(connect.Headers)? onHeader,
    Function(connect.Headers)? onTrailer,
  }) {
    return connect.Client(_transport).unary(
      specs.SignalService.sendSignal,
      input,
      signal: signal,
      headers: headers,
      onHeader: onHeader,
      onTrailer: onTrailer,
    );
  }
}
