//
//  Generated code. Do not modify.
//  source: v1/event.proto
//

import "package:connectrpc/connect.dart" as connect;
import "event.pb.dart" as v1event;
import "event.connect.spec.dart" as specs;

extension type EventServiceClient (connect.Transport _transport) {
  Future<v1event.IngestEventResponse> ingestEvent(
    v1event.IngestEventRequest input, {
    connect.Headers? headers,
    connect.AbortSignal? signal,
    Function(connect.Headers)? onHeader,
    Function(connect.Headers)? onTrailer,
  }) {
    return connect.Client(_transport).unary(
      specs.EventService.ingestEvent,
      input,
      signal: signal,
      headers: headers,
      onHeader: onHeader,
      onTrailer: onTrailer,
    );
  }

  Future<v1event.GetInstanceTimelineResponse> getInstanceTimeline(
    v1event.GetInstanceTimelineRequest input, {
    connect.Headers? headers,
    connect.AbortSignal? signal,
    Function(connect.Headers)? onHeader,
    Function(connect.Headers)? onTrailer,
  }) {
    return connect.Client(_transport).unary(
      specs.EventService.getInstanceTimeline,
      input,
      signal: signal,
      headers: headers,
      onHeader: onHeader,
      onTrailer: onTrailer,
    );
  }
}
