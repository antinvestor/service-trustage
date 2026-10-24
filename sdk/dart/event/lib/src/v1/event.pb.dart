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
import 'package:protobuf/well_known_types/google/protobuf/struct.pb.dart' as $0;
import 'package:protobuf/well_known_types/google/protobuf/timestamp.pb.dart'
    as $1;

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

class EventRecord extends $pb.GeneratedMessage {
  factory EventRecord({
    $core.String? eventId,
    $core.String? eventType,
    $core.String? source,
    $core.String? idempotencyKey,
    $0.Struct? payload,
  }) {
    final result = EventRecord._();
    if (eventId != null) result.eventId = eventId;
    if (eventType != null) result.eventType = eventType;
    if (source != null) result.source = source;
    if (idempotencyKey != null) result.idempotencyKey = idempotencyKey;
    if (payload != null) result.payload = payload;
    return result;
  }

  EventRecord._();

  factory EventRecord.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      EventRecord()..mergeFromBuffer(data, registry);
  factory EventRecord.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      EventRecord()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EventRecord',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'event.v1'),
      createEmptyInstance: EventRecord.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'eventId')
    ..aOS(2, _omitFieldNames ? '' : 'eventType')
    ..aOS(3, _omitFieldNames ? '' : 'source')
    ..aOS(4, _omitFieldNames ? '' : 'idempotencyKey')
    ..aOM<$0.Struct>(5, _omitFieldNames ? '' : 'payload',
        subBuilder: $0.Struct.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EventRecord clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EventRecord copyWith(void Function(EventRecord) updates) =>
      super.copyWith((message) => updates(message as EventRecord))
          as EventRecord;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use EventRecord() / EventRecord.new instead')
  static EventRecord create() => EventRecord._();
  static $pb.GeneratedMessage $_createMessage() => EventRecord._();
  @$core.override
  EventRecord createEmptyInstance() => EventRecord._();
  @$core.pragma('dart2js:noInline')
  static EventRecord getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<EventRecord>(
          EventRecord.$_createMessage);
  static EventRecord? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get eventId => $_getSZ(0);
  @$pb.TagNumber(1)
  set eventId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasEventId() => $_has(0);
  @$pb.TagNumber(1)
  void clearEventId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get eventType => $_getSZ(1);
  @$pb.TagNumber(2)
  set eventType($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasEventType() => $_has(1);
  @$pb.TagNumber(2)
  void clearEventType() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get source => $_getSZ(2);
  @$pb.TagNumber(3)
  set source($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasSource() => $_has(2);
  @$pb.TagNumber(3)
  void clearSource() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get idempotencyKey => $_getSZ(3);
  @$pb.TagNumber(4)
  set idempotencyKey($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasIdempotencyKey() => $_has(3);
  @$pb.TagNumber(4)
  void clearIdempotencyKey() => $_clearField(4);

  @$pb.TagNumber(5)
  $0.Struct get payload => $_getN(4);
  @$pb.TagNumber(5)
  set payload($0.Struct value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasPayload() => $_has(4);
  @$pb.TagNumber(5)
  void clearPayload() => $_clearField(5);
  @$pb.TagNumber(5)
  $0.Struct ensurePayload() => $_ensure(4);
}

class IngestEventRequest extends $pb.GeneratedMessage {
  factory IngestEventRequest({
    $core.String? eventType,
    $core.String? source,
    $core.String? idempotencyKey,
    $0.Struct? payload,
  }) {
    final result = IngestEventRequest._();
    if (eventType != null) result.eventType = eventType;
    if (source != null) result.source = source;
    if (idempotencyKey != null) result.idempotencyKey = idempotencyKey;
    if (payload != null) result.payload = payload;
    return result;
  }

  IngestEventRequest._();

  factory IngestEventRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      IngestEventRequest()..mergeFromBuffer(data, registry);
  factory IngestEventRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      IngestEventRequest()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'IngestEventRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'event.v1'),
      createEmptyInstance: IngestEventRequest.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'eventType')
    ..aOS(2, _omitFieldNames ? '' : 'source')
    ..aOS(3, _omitFieldNames ? '' : 'idempotencyKey')
    ..aOM<$0.Struct>(4, _omitFieldNames ? '' : 'payload',
        subBuilder: $0.Struct.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  IngestEventRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  IngestEventRequest copyWith(void Function(IngestEventRequest) updates) =>
      super.copyWith((message) => updates(message as IngestEventRequest))
          as IngestEventRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use IngestEventRequest() / IngestEventRequest.new instead')
  static IngestEventRequest create() => IngestEventRequest._();
  static $pb.GeneratedMessage $_createMessage() => IngestEventRequest._();
  @$core.override
  IngestEventRequest createEmptyInstance() => IngestEventRequest._();
  @$core.pragma('dart2js:noInline')
  static IngestEventRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<IngestEventRequest>(
          IngestEventRequest.$_createMessage);
  static IngestEventRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get eventType => $_getSZ(0);
  @$pb.TagNumber(1)
  set eventType($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasEventType() => $_has(0);
  @$pb.TagNumber(1)
  void clearEventType() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get source => $_getSZ(1);
  @$pb.TagNumber(2)
  set source($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasSource() => $_has(1);
  @$pb.TagNumber(2)
  void clearSource() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get idempotencyKey => $_getSZ(2);
  @$pb.TagNumber(3)
  set idempotencyKey($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasIdempotencyKey() => $_has(2);
  @$pb.TagNumber(3)
  void clearIdempotencyKey() => $_clearField(3);

  @$pb.TagNumber(4)
  $0.Struct get payload => $_getN(3);
  @$pb.TagNumber(4)
  set payload($0.Struct value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasPayload() => $_has(3);
  @$pb.TagNumber(4)
  void clearPayload() => $_clearField(4);
  @$pb.TagNumber(4)
  $0.Struct ensurePayload() => $_ensure(3);
}

class IngestEventResponse extends $pb.GeneratedMessage {
  factory IngestEventResponse({
    EventRecord? event,
    $core.bool? idempotent,
  }) {
    final result = IngestEventResponse._();
    if (event != null) result.event = event;
    if (idempotent != null) result.idempotent = idempotent;
    return result;
  }

  IngestEventResponse._();

  factory IngestEventResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      IngestEventResponse()..mergeFromBuffer(data, registry);
  factory IngestEventResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      IngestEventResponse()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'IngestEventResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'event.v1'),
      createEmptyInstance: IngestEventResponse.$_createMessage)
    ..aOM<EventRecord>(1, _omitFieldNames ? '' : 'event',
        subBuilder: EventRecord.$_createMessage)
    ..aOB(2, _omitFieldNames ? '' : 'idempotent')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  IngestEventResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  IngestEventResponse copyWith(void Function(IngestEventResponse) updates) =>
      super.copyWith((message) => updates(message as IngestEventResponse))
          as IngestEventResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core
      .Deprecated('Use IngestEventResponse() / IngestEventResponse.new instead')
  static IngestEventResponse create() => IngestEventResponse._();
  static $pb.GeneratedMessage $_createMessage() => IngestEventResponse._();
  @$core.override
  IngestEventResponse createEmptyInstance() => IngestEventResponse._();
  @$core.pragma('dart2js:noInline')
  static IngestEventResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<IngestEventResponse>(
          IngestEventResponse.$_createMessage);
  static IngestEventResponse? _defaultInstance;

  @$pb.TagNumber(1)
  EventRecord get event => $_getN(0);
  @$pb.TagNumber(1)
  set event(EventRecord value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasEvent() => $_has(0);
  @$pb.TagNumber(1)
  void clearEvent() => $_clearField(1);
  @$pb.TagNumber(1)
  EventRecord ensureEvent() => $_ensure(0);

  @$pb.TagNumber(2)
  $core.bool get idempotent => $_getBF(1);
  @$pb.TagNumber(2)
  set idempotent($core.bool value) => $_setBool(1, value);
  @$pb.TagNumber(2)
  $core.bool hasIdempotent() => $_has(1);
  @$pb.TagNumber(2)
  void clearIdempotent() => $_clearField(2);
}

class GetInstanceTimelineRequest extends $pb.GeneratedMessage {
  factory GetInstanceTimelineRequest({
    $core.String? instanceId,
  }) {
    final result = GetInstanceTimelineRequest._();
    if (instanceId != null) result.instanceId = instanceId;
    return result;
  }

  GetInstanceTimelineRequest._();

  factory GetInstanceTimelineRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      GetInstanceTimelineRequest()..mergeFromBuffer(data, registry);
  factory GetInstanceTimelineRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      GetInstanceTimelineRequest()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetInstanceTimelineRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'event.v1'),
      createEmptyInstance: GetInstanceTimelineRequest.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'instanceId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetInstanceTimelineRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetInstanceTimelineRequest copyWith(
          void Function(GetInstanceTimelineRequest) updates) =>
      super.copyWith(
              (message) => updates(message as GetInstanceTimelineRequest))
          as GetInstanceTimelineRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use GetInstanceTimelineRequest() / GetInstanceTimelineRequest.new instead')
  static GetInstanceTimelineRequest create() => GetInstanceTimelineRequest._();
  static $pb.GeneratedMessage $_createMessage() =>
      GetInstanceTimelineRequest._();
  @$core.override
  GetInstanceTimelineRequest createEmptyInstance() =>
      GetInstanceTimelineRequest._();
  @$core.pragma('dart2js:noInline')
  static GetInstanceTimelineRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetInstanceTimelineRequest>(
          GetInstanceTimelineRequest.$_createMessage);
  static GetInstanceTimelineRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get instanceId => $_getSZ(0);
  @$pb.TagNumber(1)
  set instanceId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasInstanceId() => $_has(0);
  @$pb.TagNumber(1)
  void clearInstanceId() => $_clearField(1);
}

class TimelineEntry extends $pb.GeneratedMessage {
  factory TimelineEntry({
    $core.String? eventType,
    $core.String? state,
    $core.String? fromState,
    $core.String? toState,
    $core.String? executionId,
    $core.String? traceId,
    $0.Struct? payload,
    $1.Timestamp? createdAt,
  }) {
    final result = TimelineEntry._();
    if (eventType != null) result.eventType = eventType;
    if (state != null) result.state = state;
    if (fromState != null) result.fromState = fromState;
    if (toState != null) result.toState = toState;
    if (executionId != null) result.executionId = executionId;
    if (traceId != null) result.traceId = traceId;
    if (payload != null) result.payload = payload;
    if (createdAt != null) result.createdAt = createdAt;
    return result;
  }

  TimelineEntry._();

  factory TimelineEntry.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      TimelineEntry()..mergeFromBuffer(data, registry);
  factory TimelineEntry.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      TimelineEntry()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'TimelineEntry',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'event.v1'),
      createEmptyInstance: TimelineEntry.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'eventType')
    ..aOS(2, _omitFieldNames ? '' : 'state')
    ..aOS(3, _omitFieldNames ? '' : 'fromState')
    ..aOS(4, _omitFieldNames ? '' : 'toState')
    ..aOS(5, _omitFieldNames ? '' : 'executionId')
    ..aOS(6, _omitFieldNames ? '' : 'traceId')
    ..aOM<$0.Struct>(7, _omitFieldNames ? '' : 'payload',
        subBuilder: $0.Struct.$_createMessage)
    ..aOM<$1.Timestamp>(8, _omitFieldNames ? '' : 'createdAt',
        subBuilder: $1.Timestamp.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TimelineEntry clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TimelineEntry copyWith(void Function(TimelineEntry) updates) =>
      super.copyWith((message) => updates(message as TimelineEntry))
          as TimelineEntry;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use TimelineEntry() / TimelineEntry.new instead')
  static TimelineEntry create() => TimelineEntry._();
  static $pb.GeneratedMessage $_createMessage() => TimelineEntry._();
  @$core.override
  TimelineEntry createEmptyInstance() => TimelineEntry._();
  @$core.pragma('dart2js:noInline')
  static TimelineEntry getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<TimelineEntry>(
          TimelineEntry.$_createMessage);
  static TimelineEntry? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get eventType => $_getSZ(0);
  @$pb.TagNumber(1)
  set eventType($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasEventType() => $_has(0);
  @$pb.TagNumber(1)
  void clearEventType() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get state => $_getSZ(1);
  @$pb.TagNumber(2)
  set state($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasState() => $_has(1);
  @$pb.TagNumber(2)
  void clearState() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get fromState => $_getSZ(2);
  @$pb.TagNumber(3)
  set fromState($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasFromState() => $_has(2);
  @$pb.TagNumber(3)
  void clearFromState() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get toState => $_getSZ(3);
  @$pb.TagNumber(4)
  set toState($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasToState() => $_has(3);
  @$pb.TagNumber(4)
  void clearToState() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get executionId => $_getSZ(4);
  @$pb.TagNumber(5)
  set executionId($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasExecutionId() => $_has(4);
  @$pb.TagNumber(5)
  void clearExecutionId() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get traceId => $_getSZ(5);
  @$pb.TagNumber(6)
  set traceId($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasTraceId() => $_has(5);
  @$pb.TagNumber(6)
  void clearTraceId() => $_clearField(6);

  @$pb.TagNumber(7)
  $0.Struct get payload => $_getN(6);
  @$pb.TagNumber(7)
  set payload($0.Struct value) => $_setField(7, value);
  @$pb.TagNumber(7)
  $core.bool hasPayload() => $_has(6);
  @$pb.TagNumber(7)
  void clearPayload() => $_clearField(7);
  @$pb.TagNumber(7)
  $0.Struct ensurePayload() => $_ensure(6);

  @$pb.TagNumber(8)
  $1.Timestamp get createdAt => $_getN(7);
  @$pb.TagNumber(8)
  set createdAt($1.Timestamp value) => $_setField(8, value);
  @$pb.TagNumber(8)
  $core.bool hasCreatedAt() => $_has(7);
  @$pb.TagNumber(8)
  void clearCreatedAt() => $_clearField(8);
  @$pb.TagNumber(8)
  $1.Timestamp ensureCreatedAt() => $_ensure(7);
}

class GetInstanceTimelineResponse extends $pb.GeneratedMessage {
  factory GetInstanceTimelineResponse({
    $core.Iterable<TimelineEntry>? items,
  }) {
    final result = GetInstanceTimelineResponse._();
    if (items != null) result.items.addAll(items);
    return result;
  }

  GetInstanceTimelineResponse._();

  factory GetInstanceTimelineResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      GetInstanceTimelineResponse()..mergeFromBuffer(data, registry);
  factory GetInstanceTimelineResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      GetInstanceTimelineResponse()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetInstanceTimelineResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'event.v1'),
      createEmptyInstance: GetInstanceTimelineResponse.$_createMessage)
    ..pPM<TimelineEntry>(1, _omitFieldNames ? '' : 'items',
        subBuilder: TimelineEntry.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetInstanceTimelineResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetInstanceTimelineResponse copyWith(
          void Function(GetInstanceTimelineResponse) updates) =>
      super.copyWith(
              (message) => updates(message as GetInstanceTimelineResponse))
          as GetInstanceTimelineResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use GetInstanceTimelineResponse() / GetInstanceTimelineResponse.new instead')
  static GetInstanceTimelineResponse create() =>
      GetInstanceTimelineResponse._();
  static $pb.GeneratedMessage $_createMessage() =>
      GetInstanceTimelineResponse._();
  @$core.override
  GetInstanceTimelineResponse createEmptyInstance() =>
      GetInstanceTimelineResponse._();
  @$core.pragma('dart2js:noInline')
  static GetInstanceTimelineResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetInstanceTimelineResponse>(
          GetInstanceTimelineResponse.$_createMessage);
  static GetInstanceTimelineResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<TimelineEntry> get items => $_getList(0);
}

class EventServiceApi {
  final $pb.RpcClient _client;

  EventServiceApi(this._client);

  $async.Future<IngestEventResponse> ingestEvent(
          $pb.ClientContext? ctx, IngestEventRequest request) =>
      _client.invoke<IngestEventResponse>(
          ctx, 'EventService', 'IngestEvent', request, IngestEventResponse());
  $async.Future<GetInstanceTimelineResponse> getInstanceTimeline(
          $pb.ClientContext? ctx, GetInstanceTimelineRequest request) =>
      _client.invoke<GetInstanceTimelineResponse>(ctx, 'EventService',
          'GetInstanceTimeline', request, GetInstanceTimelineResponse());
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
