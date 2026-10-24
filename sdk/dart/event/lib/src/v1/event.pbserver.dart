// This is a generated file - do not edit.
//
// Generated from v1/event.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:async' as $async;
import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

import 'event.pb.dart' as $2;
import 'event.pbjson.dart';

export 'event.pb.dart';

abstract class EventServiceBase extends $pb.GeneratedService {
  $async.Future<$2.IngestEventResponse> ingestEvent(
      $pb.ServerContext ctx, $2.IngestEventRequest request);
  $async.Future<$2.GetInstanceTimelineResponse> getInstanceTimeline(
      $pb.ServerContext ctx, $2.GetInstanceTimelineRequest request);

  $pb.GeneratedMessage createRequest($core.String methodName) {
    switch (methodName) {
      case 'IngestEvent':
        return $2.IngestEventRequest();
      case 'GetInstanceTimeline':
        return $2.GetInstanceTimelineRequest();
      default:
        throw $core.ArgumentError('Unknown method: $methodName');
    }
  }

  $async.Future<$pb.GeneratedMessage> handleCall($pb.ServerContext ctx,
      $core.String methodName, $pb.GeneratedMessage request) {
    switch (methodName) {
      case 'IngestEvent':
        return ingestEvent(ctx, request as $2.IngestEventRequest);
      case 'GetInstanceTimeline':
        return getInstanceTimeline(
            ctx, request as $2.GetInstanceTimelineRequest);
      default:
        throw $core.ArgumentError('Unknown method: $methodName');
    }
  }

  $core.Map<$core.String, $core.dynamic> get $json => EventServiceBase$json;
  $core.Map<$core.String, $core.Map<$core.String, $core.dynamic>>
      get $messageJson => EventServiceBase$messageJson;
}
