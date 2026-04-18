//
//  Generated code. Do not modify.
//  source: v1/event.proto
//
// @dart = 2.12

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_final_fields
// ignore_for_file: unnecessary_import, unnecessary_this, unused_import

import 'dart:async' as $async;
import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

import '../google/protobuf/struct.pb.dart' as $2;
import '../google/protobuf/timestamp.pb.dart' as $3;

class EventRecord extends $pb.GeneratedMessage {
  factory EventRecord({
    $core.String? eventId,
    $core.String? eventType,
    $core.String? source,
    $core.String? idempotencyKey,
    $2.Struct? payload,
  }) {
    final $result = create();
    if (eventId != null) {
      $result.eventId = eventId;
    }
    if (eventType != null) {
      $result.eventType = eventType;
    }
    if (source != null) {
      $result.source = source;
    }
    if (idempotencyKey != null) {
      $result.idempotencyKey = idempotencyKey;
    }
    if (payload != null) {
      $result.payload = payload;
    }
    return $result;
  }
  EventRecord._() : super();
  factory EventRecord.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory EventRecord.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'EventRecord', package: const $pb.PackageName(_omitMessageNames ? '' : 'event.v1'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'eventId')
    ..aOS(2, _omitFieldNames ? '' : 'eventType')
    ..aOS(3, _omitFieldNames ? '' : 'source')
    ..aOS(4, _omitFieldNames ? '' : 'idempotencyKey')
    ..aOM<$2.Struct>(5, _omitFieldNames ? '' : 'payload', subBuilder: $2.Struct.create)
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  EventRecord clone() => EventRecord()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  EventRecord copyWith(void Function(EventRecord) updates) => super.copyWith((message) => updates(message as EventRecord)) as EventRecord;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EventRecord create() => EventRecord._();
  EventRecord createEmptyInstance() => create();
  static $pb.PbList<EventRecord> createRepeated() => $pb.PbList<EventRecord>();
  @$core.pragma('dart2js:noInline')
  static EventRecord getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<EventRecord>(create);
  static EventRecord? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get eventId => $_getSZ(0);
  @$pb.TagNumber(1)
  set eventId($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasEventId() => $_has(0);
  @$pb.TagNumber(1)
  void clearEventId() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get eventType => $_getSZ(1);
  @$pb.TagNumber(2)
  set eventType($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasEventType() => $_has(1);
  @$pb.TagNumber(2)
  void clearEventType() => clearField(2);

  @$pb.TagNumber(3)
  $core.String get source => $_getSZ(2);
  @$pb.TagNumber(3)
  set source($core.String v) { $_setString(2, v); }
  @$pb.TagNumber(3)
  $core.bool hasSource() => $_has(2);
  @$pb.TagNumber(3)
  void clearSource() => clearField(3);

  @$pb.TagNumber(4)
  $core.String get idempotencyKey => $_getSZ(3);
  @$pb.TagNumber(4)
  set idempotencyKey($core.String v) { $_setString(3, v); }
  @$pb.TagNumber(4)
  $core.bool hasIdempotencyKey() => $_has(3);
  @$pb.TagNumber(4)
  void clearIdempotencyKey() => clearField(4);

  @$pb.TagNumber(5)
  $2.Struct get payload => $_getN(4);
  @$pb.TagNumber(5)
  set payload($2.Struct v) { setField(5, v); }
  @$pb.TagNumber(5)
  $core.bool hasPayload() => $_has(4);
  @$pb.TagNumber(5)
  void clearPayload() => clearField(5);
  @$pb.TagNumber(5)
  $2.Struct ensurePayload() => $_ensure(4);
}

class IngestEventRequest extends $pb.GeneratedMessage {
  factory IngestEventRequest({
    $core.String? eventType,
    $core.String? source,
    $core.String? idempotencyKey,
    $2.Struct? payload,
  }) {
    final $result = create();
    if (eventType != null) {
      $result.eventType = eventType;
    }
    if (source != null) {
      $result.source = source;
    }
    if (idempotencyKey != null) {
      $result.idempotencyKey = idempotencyKey;
    }
    if (payload != null) {
      $result.payload = payload;
    }
    return $result;
  }
  IngestEventRequest._() : super();
  factory IngestEventRequest.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory IngestEventRequest.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'IngestEventRequest', package: const $pb.PackageName(_omitMessageNames ? '' : 'event.v1'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'eventType')
    ..aOS(2, _omitFieldNames ? '' : 'source')
    ..aOS(3, _omitFieldNames ? '' : 'idempotencyKey')
    ..aOM<$2.Struct>(4, _omitFieldNames ? '' : 'payload', subBuilder: $2.Struct.create)
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  IngestEventRequest clone() => IngestEventRequest()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  IngestEventRequest copyWith(void Function(IngestEventRequest) updates) => super.copyWith((message) => updates(message as IngestEventRequest)) as IngestEventRequest;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static IngestEventRequest create() => IngestEventRequest._();
  IngestEventRequest createEmptyInstance() => create();
  static $pb.PbList<IngestEventRequest> createRepeated() => $pb.PbList<IngestEventRequest>();
  @$core.pragma('dart2js:noInline')
  static IngestEventRequest getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<IngestEventRequest>(create);
  static IngestEventRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get eventType => $_getSZ(0);
  @$pb.TagNumber(1)
  set eventType($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasEventType() => $_has(0);
  @$pb.TagNumber(1)
  void clearEventType() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get source => $_getSZ(1);
  @$pb.TagNumber(2)
  set source($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasSource() => $_has(1);
  @$pb.TagNumber(2)
  void clearSource() => clearField(2);

  @$pb.TagNumber(3)
  $core.String get idempotencyKey => $_getSZ(2);
  @$pb.TagNumber(3)
  set idempotencyKey($core.String v) { $_setString(2, v); }
  @$pb.TagNumber(3)
  $core.bool hasIdempotencyKey() => $_has(2);
  @$pb.TagNumber(3)
  void clearIdempotencyKey() => clearField(3);

  @$pb.TagNumber(4)
  $2.Struct get payload => $_getN(3);
  @$pb.TagNumber(4)
  set payload($2.Struct v) { setField(4, v); }
  @$pb.TagNumber(4)
  $core.bool hasPayload() => $_has(3);
  @$pb.TagNumber(4)
  void clearPayload() => clearField(4);
  @$pb.TagNumber(4)
  $2.Struct ensurePayload() => $_ensure(3);
}

class IngestEventResponse extends $pb.GeneratedMessage {
  factory IngestEventResponse({
    EventRecord? event,
    $core.bool? idempotent,
  }) {
    final $result = create();
    if (event != null) {
      $result.event = event;
    }
    if (idempotent != null) {
      $result.idempotent = idempotent;
    }
    return $result;
  }
  IngestEventResponse._() : super();
  factory IngestEventResponse.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory IngestEventResponse.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'IngestEventResponse', package: const $pb.PackageName(_omitMessageNames ? '' : 'event.v1'), createEmptyInstance: create)
    ..aOM<EventRecord>(1, _omitFieldNames ? '' : 'event', subBuilder: EventRecord.create)
    ..aOB(2, _omitFieldNames ? '' : 'idempotent')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  IngestEventResponse clone() => IngestEventResponse()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  IngestEventResponse copyWith(void Function(IngestEventResponse) updates) => super.copyWith((message) => updates(message as IngestEventResponse)) as IngestEventResponse;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static IngestEventResponse create() => IngestEventResponse._();
  IngestEventResponse createEmptyInstance() => create();
  static $pb.PbList<IngestEventResponse> createRepeated() => $pb.PbList<IngestEventResponse>();
  @$core.pragma('dart2js:noInline')
  static IngestEventResponse getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<IngestEventResponse>(create);
  static IngestEventResponse? _defaultInstance;

  @$pb.TagNumber(1)
  EventRecord get event => $_getN(0);
  @$pb.TagNumber(1)
  set event(EventRecord v) { setField(1, v); }
  @$pb.TagNumber(1)
  $core.bool hasEvent() => $_has(0);
  @$pb.TagNumber(1)
  void clearEvent() => clearField(1);
  @$pb.TagNumber(1)
  EventRecord ensureEvent() => $_ensure(0);

  @$pb.TagNumber(2)
  $core.bool get idempotent => $_getBF(1);
  @$pb.TagNumber(2)
  set idempotent($core.bool v) { $_setBool(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasIdempotent() => $_has(1);
  @$pb.TagNumber(2)
  void clearIdempotent() => clearField(2);
}

class GetInstanceTimelineRequest extends $pb.GeneratedMessage {
  factory GetInstanceTimelineRequest({
    $core.String? instanceId,
  }) {
    final $result = create();
    if (instanceId != null) {
      $result.instanceId = instanceId;
    }
    return $result;
  }
  GetInstanceTimelineRequest._() : super();
  factory GetInstanceTimelineRequest.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory GetInstanceTimelineRequest.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'GetInstanceTimelineRequest', package: const $pb.PackageName(_omitMessageNames ? '' : 'event.v1'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'instanceId')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  GetInstanceTimelineRequest clone() => GetInstanceTimelineRequest()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  GetInstanceTimelineRequest copyWith(void Function(GetInstanceTimelineRequest) updates) => super.copyWith((message) => updates(message as GetInstanceTimelineRequest)) as GetInstanceTimelineRequest;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetInstanceTimelineRequest create() => GetInstanceTimelineRequest._();
  GetInstanceTimelineRequest createEmptyInstance() => create();
  static $pb.PbList<GetInstanceTimelineRequest> createRepeated() => $pb.PbList<GetInstanceTimelineRequest>();
  @$core.pragma('dart2js:noInline')
  static GetInstanceTimelineRequest getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<GetInstanceTimelineRequest>(create);
  static GetInstanceTimelineRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get instanceId => $_getSZ(0);
  @$pb.TagNumber(1)
  set instanceId($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasInstanceId() => $_has(0);
  @$pb.TagNumber(1)
  void clearInstanceId() => clearField(1);
}

class TimelineEntry extends $pb.GeneratedMessage {
  factory TimelineEntry({
    $core.String? eventType,
    $core.String? state,
    $core.String? fromState,
    $core.String? toState,
    $core.String? executionId,
    $core.String? traceId,
    $2.Struct? payload,
    $3.Timestamp? createdAt,
  }) {
    final $result = create();
    if (eventType != null) {
      $result.eventType = eventType;
    }
    if (state != null) {
      $result.state = state;
    }
    if (fromState != null) {
      $result.fromState = fromState;
    }
    if (toState != null) {
      $result.toState = toState;
    }
    if (executionId != null) {
      $result.executionId = executionId;
    }
    if (traceId != null) {
      $result.traceId = traceId;
    }
    if (payload != null) {
      $result.payload = payload;
    }
    if (createdAt != null) {
      $result.createdAt = createdAt;
    }
    return $result;
  }
  TimelineEntry._() : super();
  factory TimelineEntry.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory TimelineEntry.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'TimelineEntry', package: const $pb.PackageName(_omitMessageNames ? '' : 'event.v1'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'eventType')
    ..aOS(2, _omitFieldNames ? '' : 'state')
    ..aOS(3, _omitFieldNames ? '' : 'fromState')
    ..aOS(4, _omitFieldNames ? '' : 'toState')
    ..aOS(5, _omitFieldNames ? '' : 'executionId')
    ..aOS(6, _omitFieldNames ? '' : 'traceId')
    ..aOM<$2.Struct>(7, _omitFieldNames ? '' : 'payload', subBuilder: $2.Struct.create)
    ..aOM<$3.Timestamp>(8, _omitFieldNames ? '' : 'createdAt', subBuilder: $3.Timestamp.create)
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  TimelineEntry clone() => TimelineEntry()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  TimelineEntry copyWith(void Function(TimelineEntry) updates) => super.copyWith((message) => updates(message as TimelineEntry)) as TimelineEntry;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static TimelineEntry create() => TimelineEntry._();
  TimelineEntry createEmptyInstance() => create();
  static $pb.PbList<TimelineEntry> createRepeated() => $pb.PbList<TimelineEntry>();
  @$core.pragma('dart2js:noInline')
  static TimelineEntry getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<TimelineEntry>(create);
  static TimelineEntry? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get eventType => $_getSZ(0);
  @$pb.TagNumber(1)
  set eventType($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasEventType() => $_has(0);
  @$pb.TagNumber(1)
  void clearEventType() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get state => $_getSZ(1);
  @$pb.TagNumber(2)
  set state($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasState() => $_has(1);
  @$pb.TagNumber(2)
  void clearState() => clearField(2);

  @$pb.TagNumber(3)
  $core.String get fromState => $_getSZ(2);
  @$pb.TagNumber(3)
  set fromState($core.String v) { $_setString(2, v); }
  @$pb.TagNumber(3)
  $core.bool hasFromState() => $_has(2);
  @$pb.TagNumber(3)
  void clearFromState() => clearField(3);

  @$pb.TagNumber(4)
  $core.String get toState => $_getSZ(3);
  @$pb.TagNumber(4)
  set toState($core.String v) { $_setString(3, v); }
  @$pb.TagNumber(4)
  $core.bool hasToState() => $_has(3);
  @$pb.TagNumber(4)
  void clearToState() => clearField(4);

  @$pb.TagNumber(5)
  $core.String get executionId => $_getSZ(4);
  @$pb.TagNumber(5)
  set executionId($core.String v) { $_setString(4, v); }
  @$pb.TagNumber(5)
  $core.bool hasExecutionId() => $_has(4);
  @$pb.TagNumber(5)
  void clearExecutionId() => clearField(5);

  @$pb.TagNumber(6)
  $core.String get traceId => $_getSZ(5);
  @$pb.TagNumber(6)
  set traceId($core.String v) { $_setString(5, v); }
  @$pb.TagNumber(6)
  $core.bool hasTraceId() => $_has(5);
  @$pb.TagNumber(6)
  void clearTraceId() => clearField(6);

  @$pb.TagNumber(7)
  $2.Struct get payload => $_getN(6);
  @$pb.TagNumber(7)
  set payload($2.Struct v) { setField(7, v); }
  @$pb.TagNumber(7)
  $core.bool hasPayload() => $_has(6);
  @$pb.TagNumber(7)
  void clearPayload() => clearField(7);
  @$pb.TagNumber(7)
  $2.Struct ensurePayload() => $_ensure(6);

  @$pb.TagNumber(8)
  $3.Timestamp get createdAt => $_getN(7);
  @$pb.TagNumber(8)
  set createdAt($3.Timestamp v) { setField(8, v); }
  @$pb.TagNumber(8)
  $core.bool hasCreatedAt() => $_has(7);
  @$pb.TagNumber(8)
  void clearCreatedAt() => clearField(8);
  @$pb.TagNumber(8)
  $3.Timestamp ensureCreatedAt() => $_ensure(7);
}

class GetInstanceTimelineResponse extends $pb.GeneratedMessage {
  factory GetInstanceTimelineResponse({
    $core.Iterable<TimelineEntry>? items,
  }) {
    final $result = create();
    if (items != null) {
      $result.items.addAll(items);
    }
    return $result;
  }
  GetInstanceTimelineResponse._() : super();
  factory GetInstanceTimelineResponse.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory GetInstanceTimelineResponse.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'GetInstanceTimelineResponse', package: const $pb.PackageName(_omitMessageNames ? '' : 'event.v1'), createEmptyInstance: create)
    ..pc<TimelineEntry>(1, _omitFieldNames ? '' : 'items', $pb.PbFieldType.PM, subBuilder: TimelineEntry.create)
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  GetInstanceTimelineResponse clone() => GetInstanceTimelineResponse()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  GetInstanceTimelineResponse copyWith(void Function(GetInstanceTimelineResponse) updates) => super.copyWith((message) => updates(message as GetInstanceTimelineResponse)) as GetInstanceTimelineResponse;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetInstanceTimelineResponse create() => GetInstanceTimelineResponse._();
  GetInstanceTimelineResponse createEmptyInstance() => create();
  static $pb.PbList<GetInstanceTimelineResponse> createRepeated() => $pb.PbList<GetInstanceTimelineResponse>();
  @$core.pragma('dart2js:noInline')
  static GetInstanceTimelineResponse getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<GetInstanceTimelineResponse>(create);
  static GetInstanceTimelineResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.List<TimelineEntry> get items => $_getList(0);
}

class EventServiceApi {
  $pb.RpcClient _client;
  EventServiceApi(this._client);

  $async.Future<IngestEventResponse> ingestEvent($pb.ClientContext? ctx, IngestEventRequest request) =>
    _client.invoke<IngestEventResponse>(ctx, 'EventService', 'IngestEvent', request, IngestEventResponse())
  ;
  $async.Future<GetInstanceTimelineResponse> getInstanceTimeline($pb.ClientContext? ctx, GetInstanceTimelineRequest request) =>
    _client.invoke<GetInstanceTimelineResponse>(ctx, 'EventService', 'GetInstanceTimeline', request, GetInstanceTimelineResponse())
  ;
}


const _omitFieldNames = $core.bool.fromEnvironment('protobuf.omit_field_names');
const _omitMessageNames = $core.bool.fromEnvironment('protobuf.omit_message_names');
