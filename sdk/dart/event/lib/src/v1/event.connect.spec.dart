//
//  Generated code. Do not modify.
//  source: v1/event.proto
//

import "package:connectrpc/connect.dart" as connect;
import "event.pb.dart" as v1event;

abstract final class EventService {
  /// Fully-qualified name of the EventService service.
  static const name = 'event.v1.EventService';

  static const ingestEvent = connect.Spec(
    '/$name/IngestEvent',
    connect.StreamType.unary,
    v1event.IngestEventRequest.new,
    v1event.IngestEventResponse.new,
  );

  static const getInstanceTimeline = connect.Spec(
    '/$name/GetInstanceTimeline',
    connect.StreamType.unary,
    v1event.GetInstanceTimelineRequest.new,
    v1event.GetInstanceTimelineResponse.new,
  );
}
