//
//  Generated code. Do not modify.
//  source: v1/signal.proto
//

import "package:connectrpc/connect.dart" as connect;
import "signal.pb.dart" as v1signal;

abstract final class SignalService {
  /// Fully-qualified name of the SignalService service.
  static const name = 'signal.v1.SignalService';

  static const sendSignal = connect.Spec(
    '/$name/SendSignal',
    connect.StreamType.unary,
    v1signal.SendSignalRequest.new,
    v1signal.SendSignalResponse.new,
  );
}
