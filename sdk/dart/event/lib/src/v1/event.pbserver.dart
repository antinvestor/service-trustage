//
//  Generated code. Do not modify.
//  source: v1/event.proto
//
// @dart = 2.12

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_final_fields
// ignore_for_file: unnecessary_import, unnecessary_this, unused_import

import 'dart:async' as $async;
import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

import 'event.pb.dart' as $4;
import 'event.pbjson.dart';

export 'event.pb.dart';

abstract class EventServiceBase extends $pb.GeneratedService {
  $async.Future<$4.IngestEventResponse> ingestEvent($pb.ServerContext ctx, $4.IngestEventRequest request);
  $async.Future<$4.GetInstanceTimelineResponse> getInstanceTimeline($pb.ServerContext ctx, $4.GetInstanceTimelineRequest request);

  $pb.GeneratedMessage createRequest($core.String methodName) {
    switch (methodName) {
      case 'IngestEvent': return $4.IngestEventRequest();
      case 'GetInstanceTimeline': return $4.GetInstanceTimelineRequest();
      default: throw $core.ArgumentError('Unknown method: $methodName');
    }
  }

  $async.Future<$pb.GeneratedMessage> handleCall($pb.ServerContext ctx, $core.String methodName, $pb.GeneratedMessage request) {
    switch (methodName) {
      case 'IngestEvent': return this.ingestEvent(ctx, request as $4.IngestEventRequest);
      case 'GetInstanceTimeline': return this.getInstanceTimeline(ctx, request as $4.GetInstanceTimelineRequest);
      default: throw $core.ArgumentError('Unknown method: $methodName');
    }
  }

  $core.Map<$core.String, $core.dynamic> get $json => EventServiceBase$json;
  $core.Map<$core.String, $core.Map<$core.String, $core.dynamic>> get $messageJson => EventServiceBase$messageJson;
}

