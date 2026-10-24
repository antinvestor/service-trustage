// This is a generated file - do not edit.
//
// Generated from v1/workflow.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:async' as $async;
import 'dart:core' as $core;

import 'package:fixnum/fixnum.dart' as $fixnum;
import 'package:protobuf/protobuf.dart' as $pb;
import 'package:protobuf/well_known_types/google/protobuf/struct.pb.dart' as $0;
import 'package:protobuf/well_known_types/google/protobuf/timestamp.pb.dart'
    as $1;

import '../common/v1/common.pb.dart' as $2;
import 'workflow.pbenum.dart';

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

export 'workflow.pbenum.dart';

class WorkflowDefinition extends $pb.GeneratedMessage {
  factory WorkflowDefinition({
    $core.String? id,
    $core.String? name,
    $core.int? version,
    WorkflowStatus? status,
    $0.Struct? dsl,
    $core.String? inputSchemaHash,
    $fixnum.Int64? timeoutSeconds,
    $1.Timestamp? createdAt,
    $1.Timestamp? updatedAt,
  }) {
    final result = WorkflowDefinition._();
    if (id != null) result.id = id;
    if (name != null) result.name = name;
    if (version != null) result.version = version;
    if (status != null) result.status = status;
    if (dsl != null) result.dsl = dsl;
    if (inputSchemaHash != null) result.inputSchemaHash = inputSchemaHash;
    if (timeoutSeconds != null) result.timeoutSeconds = timeoutSeconds;
    if (createdAt != null) result.createdAt = createdAt;
    if (updatedAt != null) result.updatedAt = updatedAt;
    return result;
  }

  WorkflowDefinition._();

  factory WorkflowDefinition.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      WorkflowDefinition()..mergeFromBuffer(data, registry);
  factory WorkflowDefinition.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      WorkflowDefinition()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'WorkflowDefinition',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'workflow.v1'),
      createEmptyInstance: WorkflowDefinition.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'id')
    ..aOS(2, _omitFieldNames ? '' : 'name')
    ..aI(3, _omitFieldNames ? '' : 'version')
    ..aE<WorkflowStatus>(4, _omitFieldNames ? '' : 'status',
        enumValues: WorkflowStatus.values)
    ..aOM<$0.Struct>(5, _omitFieldNames ? '' : 'dsl',
        subBuilder: $0.Struct.$_createMessage)
    ..aOS(6, _omitFieldNames ? '' : 'inputSchemaHash')
    ..aInt64(7, _omitFieldNames ? '' : 'timeoutSeconds')
    ..aOM<$1.Timestamp>(8, _omitFieldNames ? '' : 'createdAt',
        subBuilder: $1.Timestamp.$_createMessage)
    ..aOM<$1.Timestamp>(9, _omitFieldNames ? '' : 'updatedAt',
        subBuilder: $1.Timestamp.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  WorkflowDefinition clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  WorkflowDefinition copyWith(void Function(WorkflowDefinition) updates) =>
      super.copyWith((message) => updates(message as WorkflowDefinition))
          as WorkflowDefinition;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use WorkflowDefinition() / WorkflowDefinition.new instead')
  static WorkflowDefinition create() => WorkflowDefinition._();
  static $pb.GeneratedMessage $_createMessage() => WorkflowDefinition._();
  @$core.override
  WorkflowDefinition createEmptyInstance() => WorkflowDefinition._();
  @$core.pragma('dart2js:noInline')
  static WorkflowDefinition getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<WorkflowDefinition>(
          WorkflowDefinition.$_createMessage);
  static WorkflowDefinition? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get id => $_getSZ(0);
  @$pb.TagNumber(1)
  set id($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get name => $_getSZ(1);
  @$pb.TagNumber(2)
  set name($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasName() => $_has(1);
  @$pb.TagNumber(2)
  void clearName() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.int get version => $_getIZ(2);
  @$pb.TagNumber(3)
  set version($core.int value) => $_setSignedInt32(2, value);
  @$pb.TagNumber(3)
  $core.bool hasVersion() => $_has(2);
  @$pb.TagNumber(3)
  void clearVersion() => $_clearField(3);

  @$pb.TagNumber(4)
  WorkflowStatus get status => $_getN(3);
  @$pb.TagNumber(4)
  set status(WorkflowStatus value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasStatus() => $_has(3);
  @$pb.TagNumber(4)
  void clearStatus() => $_clearField(4);

  @$pb.TagNumber(5)
  $0.Struct get dsl => $_getN(4);
  @$pb.TagNumber(5)
  set dsl($0.Struct value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasDsl() => $_has(4);
  @$pb.TagNumber(5)
  void clearDsl() => $_clearField(5);
  @$pb.TagNumber(5)
  $0.Struct ensureDsl() => $_ensure(4);

  @$pb.TagNumber(6)
  $core.String get inputSchemaHash => $_getSZ(5);
  @$pb.TagNumber(6)
  set inputSchemaHash($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasInputSchemaHash() => $_has(5);
  @$pb.TagNumber(6)
  void clearInputSchemaHash() => $_clearField(6);

  @$pb.TagNumber(7)
  $fixnum.Int64 get timeoutSeconds => $_getI64(6);
  @$pb.TagNumber(7)
  set timeoutSeconds($fixnum.Int64 value) => $_setInt64(6, value);
  @$pb.TagNumber(7)
  $core.bool hasTimeoutSeconds() => $_has(6);
  @$pb.TagNumber(7)
  void clearTimeoutSeconds() => $_clearField(7);

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

  @$pb.TagNumber(9)
  $1.Timestamp get updatedAt => $_getN(8);
  @$pb.TagNumber(9)
  set updatedAt($1.Timestamp value) => $_setField(9, value);
  @$pb.TagNumber(9)
  $core.bool hasUpdatedAt() => $_has(8);
  @$pb.TagNumber(9)
  void clearUpdatedAt() => $_clearField(9);
  @$pb.TagNumber(9)
  $1.Timestamp ensureUpdatedAt() => $_ensure(8);
}

class CreateWorkflowRequest extends $pb.GeneratedMessage {
  factory CreateWorkflowRequest({
    $0.Struct? dsl,
  }) {
    final result = CreateWorkflowRequest._();
    if (dsl != null) result.dsl = dsl;
    return result;
  }

  CreateWorkflowRequest._();

  factory CreateWorkflowRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      CreateWorkflowRequest()..mergeFromBuffer(data, registry);
  factory CreateWorkflowRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      CreateWorkflowRequest()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CreateWorkflowRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'workflow.v1'),
      createEmptyInstance: CreateWorkflowRequest.$_createMessage)
    ..aOM<$0.Struct>(1, _omitFieldNames ? '' : 'dsl',
        subBuilder: $0.Struct.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CreateWorkflowRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CreateWorkflowRequest copyWith(
          void Function(CreateWorkflowRequest) updates) =>
      super.copyWith((message) => updates(message as CreateWorkflowRequest))
          as CreateWorkflowRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use CreateWorkflowRequest() / CreateWorkflowRequest.new instead')
  static CreateWorkflowRequest create() => CreateWorkflowRequest._();
  static $pb.GeneratedMessage $_createMessage() => CreateWorkflowRequest._();
  @$core.override
  CreateWorkflowRequest createEmptyInstance() => CreateWorkflowRequest._();
  @$core.pragma('dart2js:noInline')
  static CreateWorkflowRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CreateWorkflowRequest>(
          CreateWorkflowRequest.$_createMessage);
  static CreateWorkflowRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $0.Struct get dsl => $_getN(0);
  @$pb.TagNumber(1)
  set dsl($0.Struct value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasDsl() => $_has(0);
  @$pb.TagNumber(1)
  void clearDsl() => $_clearField(1);
  @$pb.TagNumber(1)
  $0.Struct ensureDsl() => $_ensure(0);
}

class CreateWorkflowResponse extends $pb.GeneratedMessage {
  factory CreateWorkflowResponse({
    WorkflowDefinition? workflow,
  }) {
    final result = CreateWorkflowResponse._();
    if (workflow != null) result.workflow = workflow;
    return result;
  }

  CreateWorkflowResponse._();

  factory CreateWorkflowResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      CreateWorkflowResponse()..mergeFromBuffer(data, registry);
  factory CreateWorkflowResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      CreateWorkflowResponse()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CreateWorkflowResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'workflow.v1'),
      createEmptyInstance: CreateWorkflowResponse.$_createMessage)
    ..aOM<WorkflowDefinition>(1, _omitFieldNames ? '' : 'workflow',
        subBuilder: WorkflowDefinition.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CreateWorkflowResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CreateWorkflowResponse copyWith(
          void Function(CreateWorkflowResponse) updates) =>
      super.copyWith((message) => updates(message as CreateWorkflowResponse))
          as CreateWorkflowResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use CreateWorkflowResponse() / CreateWorkflowResponse.new instead')
  static CreateWorkflowResponse create() => CreateWorkflowResponse._();
  static $pb.GeneratedMessage $_createMessage() => CreateWorkflowResponse._();
  @$core.override
  CreateWorkflowResponse createEmptyInstance() => CreateWorkflowResponse._();
  @$core.pragma('dart2js:noInline')
  static CreateWorkflowResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CreateWorkflowResponse>(
          CreateWorkflowResponse.$_createMessage);
  static CreateWorkflowResponse? _defaultInstance;

  @$pb.TagNumber(1)
  WorkflowDefinition get workflow => $_getN(0);
  @$pb.TagNumber(1)
  set workflow(WorkflowDefinition value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasWorkflow() => $_has(0);
  @$pb.TagNumber(1)
  void clearWorkflow() => $_clearField(1);
  @$pb.TagNumber(1)
  WorkflowDefinition ensureWorkflow() => $_ensure(0);
}

class ScheduleDefinition extends $pb.GeneratedMessage {
  factory ScheduleDefinition({
    $core.String? id,
    $core.String? name,
    $core.String? cronExpr,
    $core.String? workflowName,
    $core.int? workflowVersion,
    $core.bool? active,
    $1.Timestamp? nextFireAt,
    $1.Timestamp? lastFiredAt,
    $core.int? jitterSeconds,
    $1.Timestamp? createdAt,
    $1.Timestamp? updatedAt,
    $core.String? timezone,
  }) {
    final result = ScheduleDefinition._();
    if (id != null) result.id = id;
    if (name != null) result.name = name;
    if (cronExpr != null) result.cronExpr = cronExpr;
    if (workflowName != null) result.workflowName = workflowName;
    if (workflowVersion != null) result.workflowVersion = workflowVersion;
    if (active != null) result.active = active;
    if (nextFireAt != null) result.nextFireAt = nextFireAt;
    if (lastFiredAt != null) result.lastFiredAt = lastFiredAt;
    if (jitterSeconds != null) result.jitterSeconds = jitterSeconds;
    if (createdAt != null) result.createdAt = createdAt;
    if (updatedAt != null) result.updatedAt = updatedAt;
    if (timezone != null) result.timezone = timezone;
    return result;
  }

  ScheduleDefinition._();

  factory ScheduleDefinition.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ScheduleDefinition()..mergeFromBuffer(data, registry);
  factory ScheduleDefinition.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ScheduleDefinition()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ScheduleDefinition',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'workflow.v1'),
      createEmptyInstance: ScheduleDefinition.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'id')
    ..aOS(2, _omitFieldNames ? '' : 'name')
    ..aOS(3, _omitFieldNames ? '' : 'cronExpr')
    ..aOS(4, _omitFieldNames ? '' : 'workflowName')
    ..aI(5, _omitFieldNames ? '' : 'workflowVersion')
    ..aOB(6, _omitFieldNames ? '' : 'active')
    ..aOM<$1.Timestamp>(7, _omitFieldNames ? '' : 'nextFireAt',
        subBuilder: $1.Timestamp.$_createMessage)
    ..aOM<$1.Timestamp>(8, _omitFieldNames ? '' : 'lastFiredAt',
        subBuilder: $1.Timestamp.$_createMessage)
    ..aI(9, _omitFieldNames ? '' : 'jitterSeconds')
    ..aOM<$1.Timestamp>(10, _omitFieldNames ? '' : 'createdAt',
        subBuilder: $1.Timestamp.$_createMessage)
    ..aOM<$1.Timestamp>(11, _omitFieldNames ? '' : 'updatedAt',
        subBuilder: $1.Timestamp.$_createMessage)
    ..aOS(12, _omitFieldNames ? '' : 'timezone')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ScheduleDefinition clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ScheduleDefinition copyWith(void Function(ScheduleDefinition) updates) =>
      super.copyWith((message) => updates(message as ScheduleDefinition))
          as ScheduleDefinition;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use ScheduleDefinition() / ScheduleDefinition.new instead')
  static ScheduleDefinition create() => ScheduleDefinition._();
  static $pb.GeneratedMessage $_createMessage() => ScheduleDefinition._();
  @$core.override
  ScheduleDefinition createEmptyInstance() => ScheduleDefinition._();
  @$core.pragma('dart2js:noInline')
  static ScheduleDefinition getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ScheduleDefinition>(
          ScheduleDefinition.$_createMessage);
  static ScheduleDefinition? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get id => $_getSZ(0);
  @$pb.TagNumber(1)
  set id($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get name => $_getSZ(1);
  @$pb.TagNumber(2)
  set name($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasName() => $_has(1);
  @$pb.TagNumber(2)
  void clearName() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get cronExpr => $_getSZ(2);
  @$pb.TagNumber(3)
  set cronExpr($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasCronExpr() => $_has(2);
  @$pb.TagNumber(3)
  void clearCronExpr() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get workflowName => $_getSZ(3);
  @$pb.TagNumber(4)
  set workflowName($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasWorkflowName() => $_has(3);
  @$pb.TagNumber(4)
  void clearWorkflowName() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.int get workflowVersion => $_getIZ(4);
  @$pb.TagNumber(5)
  set workflowVersion($core.int value) => $_setSignedInt32(4, value);
  @$pb.TagNumber(5)
  $core.bool hasWorkflowVersion() => $_has(4);
  @$pb.TagNumber(5)
  void clearWorkflowVersion() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.bool get active => $_getBF(5);
  @$pb.TagNumber(6)
  set active($core.bool value) => $_setBool(5, value);
  @$pb.TagNumber(6)
  $core.bool hasActive() => $_has(5);
  @$pb.TagNumber(6)
  void clearActive() => $_clearField(6);

  @$pb.TagNumber(7)
  $1.Timestamp get nextFireAt => $_getN(6);
  @$pb.TagNumber(7)
  set nextFireAt($1.Timestamp value) => $_setField(7, value);
  @$pb.TagNumber(7)
  $core.bool hasNextFireAt() => $_has(6);
  @$pb.TagNumber(7)
  void clearNextFireAt() => $_clearField(7);
  @$pb.TagNumber(7)
  $1.Timestamp ensureNextFireAt() => $_ensure(6);

  @$pb.TagNumber(8)
  $1.Timestamp get lastFiredAt => $_getN(7);
  @$pb.TagNumber(8)
  set lastFiredAt($1.Timestamp value) => $_setField(8, value);
  @$pb.TagNumber(8)
  $core.bool hasLastFiredAt() => $_has(7);
  @$pb.TagNumber(8)
  void clearLastFiredAt() => $_clearField(8);
  @$pb.TagNumber(8)
  $1.Timestamp ensureLastFiredAt() => $_ensure(7);

  @$pb.TagNumber(9)
  $core.int get jitterSeconds => $_getIZ(8);
  @$pb.TagNumber(9)
  set jitterSeconds($core.int value) => $_setSignedInt32(8, value);
  @$pb.TagNumber(9)
  $core.bool hasJitterSeconds() => $_has(8);
  @$pb.TagNumber(9)
  void clearJitterSeconds() => $_clearField(9);

  @$pb.TagNumber(10)
  $1.Timestamp get createdAt => $_getN(9);
  @$pb.TagNumber(10)
  set createdAt($1.Timestamp value) => $_setField(10, value);
  @$pb.TagNumber(10)
  $core.bool hasCreatedAt() => $_has(9);
  @$pb.TagNumber(10)
  void clearCreatedAt() => $_clearField(10);
  @$pb.TagNumber(10)
  $1.Timestamp ensureCreatedAt() => $_ensure(9);

  @$pb.TagNumber(11)
  $1.Timestamp get updatedAt => $_getN(10);
  @$pb.TagNumber(11)
  set updatedAt($1.Timestamp value) => $_setField(11, value);
  @$pb.TagNumber(11)
  $core.bool hasUpdatedAt() => $_has(10);
  @$pb.TagNumber(11)
  void clearUpdatedAt() => $_clearField(11);
  @$pb.TagNumber(11)
  $1.Timestamp ensureUpdatedAt() => $_ensure(10);

  @$pb.TagNumber(12)
  $core.String get timezone => $_getSZ(11);
  @$pb.TagNumber(12)
  set timezone($core.String value) => $_setString(11, value);
  @$pb.TagNumber(12)
  $core.bool hasTimezone() => $_has(11);
  @$pb.TagNumber(12)
  void clearTimezone() => $_clearField(12);
}

class GetWorkflowRequest extends $pb.GeneratedMessage {
  factory GetWorkflowRequest({
    $core.String? id,
  }) {
    final result = GetWorkflowRequest._();
    if (id != null) result.id = id;
    return result;
  }

  GetWorkflowRequest._();

  factory GetWorkflowRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      GetWorkflowRequest()..mergeFromBuffer(data, registry);
  factory GetWorkflowRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      GetWorkflowRequest()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetWorkflowRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'workflow.v1'),
      createEmptyInstance: GetWorkflowRequest.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'id')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetWorkflowRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetWorkflowRequest copyWith(void Function(GetWorkflowRequest) updates) =>
      super.copyWith((message) => updates(message as GetWorkflowRequest))
          as GetWorkflowRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use GetWorkflowRequest() / GetWorkflowRequest.new instead')
  static GetWorkflowRequest create() => GetWorkflowRequest._();
  static $pb.GeneratedMessage $_createMessage() => GetWorkflowRequest._();
  @$core.override
  GetWorkflowRequest createEmptyInstance() => GetWorkflowRequest._();
  @$core.pragma('dart2js:noInline')
  static GetWorkflowRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetWorkflowRequest>(
          GetWorkflowRequest.$_createMessage);
  static GetWorkflowRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get id => $_getSZ(0);
  @$pb.TagNumber(1)
  set id($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => $_clearField(1);
}

class GetWorkflowResponse extends $pb.GeneratedMessage {
  factory GetWorkflowResponse({
    WorkflowDefinition? workflow,
    $core.Iterable<ScheduleDefinition>? schedules,
  }) {
    final result = GetWorkflowResponse._();
    if (workflow != null) result.workflow = workflow;
    if (schedules != null) result.schedules.addAll(schedules);
    return result;
  }

  GetWorkflowResponse._();

  factory GetWorkflowResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      GetWorkflowResponse()..mergeFromBuffer(data, registry);
  factory GetWorkflowResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      GetWorkflowResponse()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetWorkflowResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'workflow.v1'),
      createEmptyInstance: GetWorkflowResponse.$_createMessage)
    ..aOM<WorkflowDefinition>(1, _omitFieldNames ? '' : 'workflow',
        subBuilder: WorkflowDefinition.$_createMessage)
    ..pPM<ScheduleDefinition>(2, _omitFieldNames ? '' : 'schedules',
        subBuilder: ScheduleDefinition.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetWorkflowResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetWorkflowResponse copyWith(void Function(GetWorkflowResponse) updates) =>
      super.copyWith((message) => updates(message as GetWorkflowResponse))
          as GetWorkflowResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core
      .Deprecated('Use GetWorkflowResponse() / GetWorkflowResponse.new instead')
  static GetWorkflowResponse create() => GetWorkflowResponse._();
  static $pb.GeneratedMessage $_createMessage() => GetWorkflowResponse._();
  @$core.override
  GetWorkflowResponse createEmptyInstance() => GetWorkflowResponse._();
  @$core.pragma('dart2js:noInline')
  static GetWorkflowResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetWorkflowResponse>(
          GetWorkflowResponse.$_createMessage);
  static GetWorkflowResponse? _defaultInstance;

  @$pb.TagNumber(1)
  WorkflowDefinition get workflow => $_getN(0);
  @$pb.TagNumber(1)
  set workflow(WorkflowDefinition value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasWorkflow() => $_has(0);
  @$pb.TagNumber(1)
  void clearWorkflow() => $_clearField(1);
  @$pb.TagNumber(1)
  WorkflowDefinition ensureWorkflow() => $_ensure(0);

  @$pb.TagNumber(2)
  $pb.PbList<ScheduleDefinition> get schedules => $_getList(1);
}

class ListWorkflowsRequest extends $pb.GeneratedMessage {
  factory ListWorkflowsRequest({
    $core.String? name,
    WorkflowStatus? status,
    $2.SearchRequest? search,
  }) {
    final result = ListWorkflowsRequest._();
    if (name != null) result.name = name;
    if (status != null) result.status = status;
    if (search != null) result.search = search;
    return result;
  }

  ListWorkflowsRequest._();

  factory ListWorkflowsRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ListWorkflowsRequest()..mergeFromBuffer(data, registry);
  factory ListWorkflowsRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ListWorkflowsRequest()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ListWorkflowsRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'workflow.v1'),
      createEmptyInstance: ListWorkflowsRequest.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'name')
    ..aE<WorkflowStatus>(2, _omitFieldNames ? '' : 'status',
        enumValues: WorkflowStatus.values)
    ..aOM<$2.SearchRequest>(3, _omitFieldNames ? '' : 'search',
        subBuilder: $2.SearchRequest.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListWorkflowsRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListWorkflowsRequest copyWith(void Function(ListWorkflowsRequest) updates) =>
      super.copyWith((message) => updates(message as ListWorkflowsRequest))
          as ListWorkflowsRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use ListWorkflowsRequest() / ListWorkflowsRequest.new instead')
  static ListWorkflowsRequest create() => ListWorkflowsRequest._();
  static $pb.GeneratedMessage $_createMessage() => ListWorkflowsRequest._();
  @$core.override
  ListWorkflowsRequest createEmptyInstance() => ListWorkflowsRequest._();
  @$core.pragma('dart2js:noInline')
  static ListWorkflowsRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ListWorkflowsRequest>(
          ListWorkflowsRequest.$_createMessage);
  static ListWorkflowsRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get name => $_getSZ(0);
  @$pb.TagNumber(1)
  set name($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasName() => $_has(0);
  @$pb.TagNumber(1)
  void clearName() => $_clearField(1);

  @$pb.TagNumber(2)
  WorkflowStatus get status => $_getN(1);
  @$pb.TagNumber(2)
  set status(WorkflowStatus value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasStatus() => $_has(1);
  @$pb.TagNumber(2)
  void clearStatus() => $_clearField(2);

  @$pb.TagNumber(3)
  $2.SearchRequest get search => $_getN(2);
  @$pb.TagNumber(3)
  set search($2.SearchRequest value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasSearch() => $_has(2);
  @$pb.TagNumber(3)
  void clearSearch() => $_clearField(3);
  @$pb.TagNumber(3)
  $2.SearchRequest ensureSearch() => $_ensure(2);
}

class ListWorkflowsResponse extends $pb.GeneratedMessage {
  factory ListWorkflowsResponse({
    $core.Iterable<WorkflowDefinition>? items,
    $2.PageCursor? nextCursor,
  }) {
    final result = ListWorkflowsResponse._();
    if (items != null) result.items.addAll(items);
    if (nextCursor != null) result.nextCursor = nextCursor;
    return result;
  }

  ListWorkflowsResponse._();

  factory ListWorkflowsResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ListWorkflowsResponse()..mergeFromBuffer(data, registry);
  factory ListWorkflowsResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ListWorkflowsResponse()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ListWorkflowsResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'workflow.v1'),
      createEmptyInstance: ListWorkflowsResponse.$_createMessage)
    ..pPM<WorkflowDefinition>(1, _omitFieldNames ? '' : 'items',
        subBuilder: WorkflowDefinition.$_createMessage)
    ..aOM<$2.PageCursor>(2, _omitFieldNames ? '' : 'nextCursor',
        subBuilder: $2.PageCursor.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListWorkflowsResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListWorkflowsResponse copyWith(
          void Function(ListWorkflowsResponse) updates) =>
      super.copyWith((message) => updates(message as ListWorkflowsResponse))
          as ListWorkflowsResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use ListWorkflowsResponse() / ListWorkflowsResponse.new instead')
  static ListWorkflowsResponse create() => ListWorkflowsResponse._();
  static $pb.GeneratedMessage $_createMessage() => ListWorkflowsResponse._();
  @$core.override
  ListWorkflowsResponse createEmptyInstance() => ListWorkflowsResponse._();
  @$core.pragma('dart2js:noInline')
  static ListWorkflowsResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ListWorkflowsResponse>(
          ListWorkflowsResponse.$_createMessage);
  static ListWorkflowsResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<WorkflowDefinition> get items => $_getList(0);

  @$pb.TagNumber(2)
  $2.PageCursor get nextCursor => $_getN(1);
  @$pb.TagNumber(2)
  set nextCursor($2.PageCursor value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasNextCursor() => $_has(1);
  @$pb.TagNumber(2)
  void clearNextCursor() => $_clearField(2);
  @$pb.TagNumber(2)
  $2.PageCursor ensureNextCursor() => $_ensure(1);
}

class ActivateWorkflowRequest extends $pb.GeneratedMessage {
  factory ActivateWorkflowRequest({
    $core.String? id,
  }) {
    final result = ActivateWorkflowRequest._();
    if (id != null) result.id = id;
    return result;
  }

  ActivateWorkflowRequest._();

  factory ActivateWorkflowRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ActivateWorkflowRequest()..mergeFromBuffer(data, registry);
  factory ActivateWorkflowRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ActivateWorkflowRequest()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ActivateWorkflowRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'workflow.v1'),
      createEmptyInstance: ActivateWorkflowRequest.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'id')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ActivateWorkflowRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ActivateWorkflowRequest copyWith(
          void Function(ActivateWorkflowRequest) updates) =>
      super.copyWith((message) => updates(message as ActivateWorkflowRequest))
          as ActivateWorkflowRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use ActivateWorkflowRequest() / ActivateWorkflowRequest.new instead')
  static ActivateWorkflowRequest create() => ActivateWorkflowRequest._();
  static $pb.GeneratedMessage $_createMessage() => ActivateWorkflowRequest._();
  @$core.override
  ActivateWorkflowRequest createEmptyInstance() => ActivateWorkflowRequest._();
  @$core.pragma('dart2js:noInline')
  static ActivateWorkflowRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ActivateWorkflowRequest>(
          ActivateWorkflowRequest.$_createMessage);
  static ActivateWorkflowRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get id => $_getSZ(0);
  @$pb.TagNumber(1)
  set id($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => $_clearField(1);
}

class ActivateWorkflowResponse extends $pb.GeneratedMessage {
  factory ActivateWorkflowResponse({
    WorkflowDefinition? workflow,
  }) {
    final result = ActivateWorkflowResponse._();
    if (workflow != null) result.workflow = workflow;
    return result;
  }

  ActivateWorkflowResponse._();

  factory ActivateWorkflowResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ActivateWorkflowResponse()..mergeFromBuffer(data, registry);
  factory ActivateWorkflowResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ActivateWorkflowResponse()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ActivateWorkflowResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'workflow.v1'),
      createEmptyInstance: ActivateWorkflowResponse.$_createMessage)
    ..aOM<WorkflowDefinition>(1, _omitFieldNames ? '' : 'workflow',
        subBuilder: WorkflowDefinition.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ActivateWorkflowResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ActivateWorkflowResponse copyWith(
          void Function(ActivateWorkflowResponse) updates) =>
      super.copyWith((message) => updates(message as ActivateWorkflowResponse))
          as ActivateWorkflowResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use ActivateWorkflowResponse() / ActivateWorkflowResponse.new instead')
  static ActivateWorkflowResponse create() => ActivateWorkflowResponse._();
  static $pb.GeneratedMessage $_createMessage() => ActivateWorkflowResponse._();
  @$core.override
  ActivateWorkflowResponse createEmptyInstance() =>
      ActivateWorkflowResponse._();
  @$core.pragma('dart2js:noInline')
  static ActivateWorkflowResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ActivateWorkflowResponse>(
          ActivateWorkflowResponse.$_createMessage);
  static ActivateWorkflowResponse? _defaultInstance;

  @$pb.TagNumber(1)
  WorkflowDefinition get workflow => $_getN(0);
  @$pb.TagNumber(1)
  set workflow(WorkflowDefinition value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasWorkflow() => $_has(0);
  @$pb.TagNumber(1)
  void clearWorkflow() => $_clearField(1);
  @$pb.TagNumber(1)
  WorkflowDefinition ensureWorkflow() => $_ensure(0);
}

class ArchiveWorkflowRequest extends $pb.GeneratedMessage {
  factory ArchiveWorkflowRequest({
    $core.String? id,
  }) {
    final result = ArchiveWorkflowRequest._();
    if (id != null) result.id = id;
    return result;
  }

  ArchiveWorkflowRequest._();

  factory ArchiveWorkflowRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ArchiveWorkflowRequest()..mergeFromBuffer(data, registry);
  factory ArchiveWorkflowRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ArchiveWorkflowRequest()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ArchiveWorkflowRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'workflow.v1'),
      createEmptyInstance: ArchiveWorkflowRequest.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'id')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ArchiveWorkflowRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ArchiveWorkflowRequest copyWith(
          void Function(ArchiveWorkflowRequest) updates) =>
      super.copyWith((message) => updates(message as ArchiveWorkflowRequest))
          as ArchiveWorkflowRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use ArchiveWorkflowRequest() / ArchiveWorkflowRequest.new instead')
  static ArchiveWorkflowRequest create() => ArchiveWorkflowRequest._();
  static $pb.GeneratedMessage $_createMessage() => ArchiveWorkflowRequest._();
  @$core.override
  ArchiveWorkflowRequest createEmptyInstance() => ArchiveWorkflowRequest._();
  @$core.pragma('dart2js:noInline')
  static ArchiveWorkflowRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ArchiveWorkflowRequest>(
          ArchiveWorkflowRequest.$_createMessage);
  static ArchiveWorkflowRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get id => $_getSZ(0);
  @$pb.TagNumber(1)
  set id($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => $_clearField(1);
}

class ArchiveWorkflowResponse extends $pb.GeneratedMessage {
  factory ArchiveWorkflowResponse({
    WorkflowDefinition? workflow,
  }) {
    final result = ArchiveWorkflowResponse._();
    if (workflow != null) result.workflow = workflow;
    return result;
  }

  ArchiveWorkflowResponse._();

  factory ArchiveWorkflowResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ArchiveWorkflowResponse()..mergeFromBuffer(data, registry);
  factory ArchiveWorkflowResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ArchiveWorkflowResponse()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ArchiveWorkflowResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'workflow.v1'),
      createEmptyInstance: ArchiveWorkflowResponse.$_createMessage)
    ..aOM<WorkflowDefinition>(1, _omitFieldNames ? '' : 'workflow',
        subBuilder: WorkflowDefinition.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ArchiveWorkflowResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ArchiveWorkflowResponse copyWith(
          void Function(ArchiveWorkflowResponse) updates) =>
      super.copyWith((message) => updates(message as ArchiveWorkflowResponse))
          as ArchiveWorkflowResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use ArchiveWorkflowResponse() / ArchiveWorkflowResponse.new instead')
  static ArchiveWorkflowResponse create() => ArchiveWorkflowResponse._();
  static $pb.GeneratedMessage $_createMessage() => ArchiveWorkflowResponse._();
  @$core.override
  ArchiveWorkflowResponse createEmptyInstance() => ArchiveWorkflowResponse._();
  @$core.pragma('dart2js:noInline')
  static ArchiveWorkflowResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ArchiveWorkflowResponse>(
          ArchiveWorkflowResponse.$_createMessage);
  static ArchiveWorkflowResponse? _defaultInstance;

  @$pb.TagNumber(1)
  WorkflowDefinition get workflow => $_getN(0);
  @$pb.TagNumber(1)
  set workflow(WorkflowDefinition value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasWorkflow() => $_has(0);
  @$pb.TagNumber(1)
  void clearWorkflow() => $_clearField(1);
  @$pb.TagNumber(1)
  WorkflowDefinition ensureWorkflow() => $_ensure(0);
}

class WorkflowServiceApi {
  final $pb.RpcClient _client;

  WorkflowServiceApi(this._client);

  $async.Future<CreateWorkflowResponse> createWorkflow(
          $pb.ClientContext? ctx, CreateWorkflowRequest request) =>
      _client.invoke<CreateWorkflowResponse>(ctx, 'WorkflowService',
          'CreateWorkflow', request, CreateWorkflowResponse());
  $async.Future<GetWorkflowResponse> getWorkflow(
          $pb.ClientContext? ctx, GetWorkflowRequest request) =>
      _client.invoke<GetWorkflowResponse>(ctx, 'WorkflowService', 'GetWorkflow',
          request, GetWorkflowResponse());
  $async.Future<ListWorkflowsResponse> listWorkflows(
          $pb.ClientContext? ctx, ListWorkflowsRequest request) =>
      _client.invoke<ListWorkflowsResponse>(ctx, 'WorkflowService',
          'ListWorkflows', request, ListWorkflowsResponse());
  $async.Future<ActivateWorkflowResponse> activateWorkflow(
          $pb.ClientContext? ctx, ActivateWorkflowRequest request) =>
      _client.invoke<ActivateWorkflowResponse>(ctx, 'WorkflowService',
          'ActivateWorkflow', request, ActivateWorkflowResponse());
  $async.Future<ArchiveWorkflowResponse> archiveWorkflow(
          $pb.ClientContext? ctx, ArchiveWorkflowRequest request) =>
      _client.invoke<ArchiveWorkflowResponse>(ctx, 'WorkflowService',
          'ArchiveWorkflow', request, ArchiveWorkflowResponse());
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
