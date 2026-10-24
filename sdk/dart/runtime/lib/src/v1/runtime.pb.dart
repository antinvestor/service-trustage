// This is a generated file - do not edit.
//
// Generated from v1/runtime.proto.

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
import 'runtime.pbenum.dart';

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

export 'runtime.pbenum.dart';

class WorkflowInstance extends $pb.GeneratedMessage {
  factory WorkflowInstance({
    $core.String? id,
    $core.String? workflowName,
    $core.int? workflowVersion,
    $core.String? currentState,
    InstanceStatus? status,
    $fixnum.Int64? revision,
    $core.String? triggerEventId,
    $0.Struct? metadata,
    $1.Timestamp? startedAt,
    $1.Timestamp? finishedAt,
    $1.Timestamp? createdAt,
    $1.Timestamp? updatedAt,
    $core.String? parentInstanceId,
    $core.String? parentExecutionId,
    $core.String? scopeType,
    $core.String? scopeParentState,
    $core.String? scopeEntryState,
    $core.int? scopeIndex,
  }) {
    final result = WorkflowInstance._();
    if (id != null) result.id = id;
    if (workflowName != null) result.workflowName = workflowName;
    if (workflowVersion != null) result.workflowVersion = workflowVersion;
    if (currentState != null) result.currentState = currentState;
    if (status != null) result.status = status;
    if (revision != null) result.revision = revision;
    if (triggerEventId != null) result.triggerEventId = triggerEventId;
    if (metadata != null) result.metadata = metadata;
    if (startedAt != null) result.startedAt = startedAt;
    if (finishedAt != null) result.finishedAt = finishedAt;
    if (createdAt != null) result.createdAt = createdAt;
    if (updatedAt != null) result.updatedAt = updatedAt;
    if (parentInstanceId != null) result.parentInstanceId = parentInstanceId;
    if (parentExecutionId != null) result.parentExecutionId = parentExecutionId;
    if (scopeType != null) result.scopeType = scopeType;
    if (scopeParentState != null) result.scopeParentState = scopeParentState;
    if (scopeEntryState != null) result.scopeEntryState = scopeEntryState;
    if (scopeIndex != null) result.scopeIndex = scopeIndex;
    return result;
  }

  WorkflowInstance._();

  factory WorkflowInstance.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      WorkflowInstance()..mergeFromBuffer(data, registry);
  factory WorkflowInstance.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      WorkflowInstance()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'WorkflowInstance',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'runtime.v1'),
      createEmptyInstance: WorkflowInstance.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'id')
    ..aOS(2, _omitFieldNames ? '' : 'workflowName')
    ..aI(3, _omitFieldNames ? '' : 'workflowVersion')
    ..aOS(4, _omitFieldNames ? '' : 'currentState')
    ..aE<InstanceStatus>(5, _omitFieldNames ? '' : 'status',
        enumValues: InstanceStatus.values)
    ..aInt64(6, _omitFieldNames ? '' : 'revision')
    ..aOS(7, _omitFieldNames ? '' : 'triggerEventId')
    ..aOM<$0.Struct>(8, _omitFieldNames ? '' : 'metadata',
        subBuilder: $0.Struct.$_createMessage)
    ..aOM<$1.Timestamp>(9, _omitFieldNames ? '' : 'startedAt',
        subBuilder: $1.Timestamp.$_createMessage)
    ..aOM<$1.Timestamp>(10, _omitFieldNames ? '' : 'finishedAt',
        subBuilder: $1.Timestamp.$_createMessage)
    ..aOM<$1.Timestamp>(11, _omitFieldNames ? '' : 'createdAt',
        subBuilder: $1.Timestamp.$_createMessage)
    ..aOM<$1.Timestamp>(12, _omitFieldNames ? '' : 'updatedAt',
        subBuilder: $1.Timestamp.$_createMessage)
    ..aOS(13, _omitFieldNames ? '' : 'parentInstanceId')
    ..aOS(14, _omitFieldNames ? '' : 'parentExecutionId')
    ..aOS(15, _omitFieldNames ? '' : 'scopeType')
    ..aOS(16, _omitFieldNames ? '' : 'scopeParentState')
    ..aOS(17, _omitFieldNames ? '' : 'scopeEntryState')
    ..aI(18, _omitFieldNames ? '' : 'scopeIndex')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  WorkflowInstance clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  WorkflowInstance copyWith(void Function(WorkflowInstance) updates) =>
      super.copyWith((message) => updates(message as WorkflowInstance))
          as WorkflowInstance;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use WorkflowInstance() / WorkflowInstance.new instead')
  static WorkflowInstance create() => WorkflowInstance._();
  static $pb.GeneratedMessage $_createMessage() => WorkflowInstance._();
  @$core.override
  WorkflowInstance createEmptyInstance() => WorkflowInstance._();
  @$core.pragma('dart2js:noInline')
  static WorkflowInstance getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<WorkflowInstance>(
          WorkflowInstance.$_createMessage);
  static WorkflowInstance? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get id => $_getSZ(0);
  @$pb.TagNumber(1)
  set id($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get workflowName => $_getSZ(1);
  @$pb.TagNumber(2)
  set workflowName($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasWorkflowName() => $_has(1);
  @$pb.TagNumber(2)
  void clearWorkflowName() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.int get workflowVersion => $_getIZ(2);
  @$pb.TagNumber(3)
  set workflowVersion($core.int value) => $_setSignedInt32(2, value);
  @$pb.TagNumber(3)
  $core.bool hasWorkflowVersion() => $_has(2);
  @$pb.TagNumber(3)
  void clearWorkflowVersion() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get currentState => $_getSZ(3);
  @$pb.TagNumber(4)
  set currentState($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasCurrentState() => $_has(3);
  @$pb.TagNumber(4)
  void clearCurrentState() => $_clearField(4);

  @$pb.TagNumber(5)
  InstanceStatus get status => $_getN(4);
  @$pb.TagNumber(5)
  set status(InstanceStatus value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasStatus() => $_has(4);
  @$pb.TagNumber(5)
  void clearStatus() => $_clearField(5);

  @$pb.TagNumber(6)
  $fixnum.Int64 get revision => $_getI64(5);
  @$pb.TagNumber(6)
  set revision($fixnum.Int64 value) => $_setInt64(5, value);
  @$pb.TagNumber(6)
  $core.bool hasRevision() => $_has(5);
  @$pb.TagNumber(6)
  void clearRevision() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.String get triggerEventId => $_getSZ(6);
  @$pb.TagNumber(7)
  set triggerEventId($core.String value) => $_setString(6, value);
  @$pb.TagNumber(7)
  $core.bool hasTriggerEventId() => $_has(6);
  @$pb.TagNumber(7)
  void clearTriggerEventId() => $_clearField(7);

  @$pb.TagNumber(8)
  $0.Struct get metadata => $_getN(7);
  @$pb.TagNumber(8)
  set metadata($0.Struct value) => $_setField(8, value);
  @$pb.TagNumber(8)
  $core.bool hasMetadata() => $_has(7);
  @$pb.TagNumber(8)
  void clearMetadata() => $_clearField(8);
  @$pb.TagNumber(8)
  $0.Struct ensureMetadata() => $_ensure(7);

  @$pb.TagNumber(9)
  $1.Timestamp get startedAt => $_getN(8);
  @$pb.TagNumber(9)
  set startedAt($1.Timestamp value) => $_setField(9, value);
  @$pb.TagNumber(9)
  $core.bool hasStartedAt() => $_has(8);
  @$pb.TagNumber(9)
  void clearStartedAt() => $_clearField(9);
  @$pb.TagNumber(9)
  $1.Timestamp ensureStartedAt() => $_ensure(8);

  @$pb.TagNumber(10)
  $1.Timestamp get finishedAt => $_getN(9);
  @$pb.TagNumber(10)
  set finishedAt($1.Timestamp value) => $_setField(10, value);
  @$pb.TagNumber(10)
  $core.bool hasFinishedAt() => $_has(9);
  @$pb.TagNumber(10)
  void clearFinishedAt() => $_clearField(10);
  @$pb.TagNumber(10)
  $1.Timestamp ensureFinishedAt() => $_ensure(9);

  @$pb.TagNumber(11)
  $1.Timestamp get createdAt => $_getN(10);
  @$pb.TagNumber(11)
  set createdAt($1.Timestamp value) => $_setField(11, value);
  @$pb.TagNumber(11)
  $core.bool hasCreatedAt() => $_has(10);
  @$pb.TagNumber(11)
  void clearCreatedAt() => $_clearField(11);
  @$pb.TagNumber(11)
  $1.Timestamp ensureCreatedAt() => $_ensure(10);

  @$pb.TagNumber(12)
  $1.Timestamp get updatedAt => $_getN(11);
  @$pb.TagNumber(12)
  set updatedAt($1.Timestamp value) => $_setField(12, value);
  @$pb.TagNumber(12)
  $core.bool hasUpdatedAt() => $_has(11);
  @$pb.TagNumber(12)
  void clearUpdatedAt() => $_clearField(12);
  @$pb.TagNumber(12)
  $1.Timestamp ensureUpdatedAt() => $_ensure(11);

  @$pb.TagNumber(13)
  $core.String get parentInstanceId => $_getSZ(12);
  @$pb.TagNumber(13)
  set parentInstanceId($core.String value) => $_setString(12, value);
  @$pb.TagNumber(13)
  $core.bool hasParentInstanceId() => $_has(12);
  @$pb.TagNumber(13)
  void clearParentInstanceId() => $_clearField(13);

  @$pb.TagNumber(14)
  $core.String get parentExecutionId => $_getSZ(13);
  @$pb.TagNumber(14)
  set parentExecutionId($core.String value) => $_setString(13, value);
  @$pb.TagNumber(14)
  $core.bool hasParentExecutionId() => $_has(13);
  @$pb.TagNumber(14)
  void clearParentExecutionId() => $_clearField(14);

  @$pb.TagNumber(15)
  $core.String get scopeType => $_getSZ(14);
  @$pb.TagNumber(15)
  set scopeType($core.String value) => $_setString(14, value);
  @$pb.TagNumber(15)
  $core.bool hasScopeType() => $_has(14);
  @$pb.TagNumber(15)
  void clearScopeType() => $_clearField(15);

  @$pb.TagNumber(16)
  $core.String get scopeParentState => $_getSZ(15);
  @$pb.TagNumber(16)
  set scopeParentState($core.String value) => $_setString(15, value);
  @$pb.TagNumber(16)
  $core.bool hasScopeParentState() => $_has(15);
  @$pb.TagNumber(16)
  void clearScopeParentState() => $_clearField(16);

  @$pb.TagNumber(17)
  $core.String get scopeEntryState => $_getSZ(16);
  @$pb.TagNumber(17)
  set scopeEntryState($core.String value) => $_setString(16, value);
  @$pb.TagNumber(17)
  $core.bool hasScopeEntryState() => $_has(16);
  @$pb.TagNumber(17)
  void clearScopeEntryState() => $_clearField(17);

  @$pb.TagNumber(18)
  $core.int get scopeIndex => $_getIZ(17);
  @$pb.TagNumber(18)
  set scopeIndex($core.int value) => $_setSignedInt32(17, value);
  @$pb.TagNumber(18)
  $core.bool hasScopeIndex() => $_has(17);
  @$pb.TagNumber(18)
  void clearScopeIndex() => $_clearField(18);
}

class WorkflowExecution extends $pb.GeneratedMessage {
  factory WorkflowExecution({
    $core.String? id,
    $core.String? instanceId,
    $core.String? state,
    $core.int? stateVersion,
    $core.int? attempt,
    ExecutionStatus? status,
    $core.String? errorClass,
    $core.String? errorMessage,
    $1.Timestamp? nextRetryAt,
    $1.Timestamp? startedAt,
    $1.Timestamp? finishedAt,
    $1.Timestamp? createdAt,
    $1.Timestamp? updatedAt,
    $core.String? traceId,
    $core.String? inputSchemaHash,
    $core.String? outputSchemaHash,
    $0.Struct? inputPayload,
    $0.Struct? output,
  }) {
    final result = WorkflowExecution._();
    if (id != null) result.id = id;
    if (instanceId != null) result.instanceId = instanceId;
    if (state != null) result.state = state;
    if (stateVersion != null) result.stateVersion = stateVersion;
    if (attempt != null) result.attempt = attempt;
    if (status != null) result.status = status;
    if (errorClass != null) result.errorClass = errorClass;
    if (errorMessage != null) result.errorMessage = errorMessage;
    if (nextRetryAt != null) result.nextRetryAt = nextRetryAt;
    if (startedAt != null) result.startedAt = startedAt;
    if (finishedAt != null) result.finishedAt = finishedAt;
    if (createdAt != null) result.createdAt = createdAt;
    if (updatedAt != null) result.updatedAt = updatedAt;
    if (traceId != null) result.traceId = traceId;
    if (inputSchemaHash != null) result.inputSchemaHash = inputSchemaHash;
    if (outputSchemaHash != null) result.outputSchemaHash = outputSchemaHash;
    if (inputPayload != null) result.inputPayload = inputPayload;
    if (output != null) result.output = output;
    return result;
  }

  WorkflowExecution._();

  factory WorkflowExecution.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      WorkflowExecution()..mergeFromBuffer(data, registry);
  factory WorkflowExecution.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      WorkflowExecution()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'WorkflowExecution',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'runtime.v1'),
      createEmptyInstance: WorkflowExecution.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'id')
    ..aOS(2, _omitFieldNames ? '' : 'instanceId')
    ..aOS(3, _omitFieldNames ? '' : 'state')
    ..aI(4, _omitFieldNames ? '' : 'stateVersion')
    ..aI(5, _omitFieldNames ? '' : 'attempt')
    ..aE<ExecutionStatus>(6, _omitFieldNames ? '' : 'status',
        enumValues: ExecutionStatus.values)
    ..aOS(7, _omitFieldNames ? '' : 'errorClass')
    ..aOS(8, _omitFieldNames ? '' : 'errorMessage')
    ..aOM<$1.Timestamp>(9, _omitFieldNames ? '' : 'nextRetryAt',
        subBuilder: $1.Timestamp.$_createMessage)
    ..aOM<$1.Timestamp>(10, _omitFieldNames ? '' : 'startedAt',
        subBuilder: $1.Timestamp.$_createMessage)
    ..aOM<$1.Timestamp>(11, _omitFieldNames ? '' : 'finishedAt',
        subBuilder: $1.Timestamp.$_createMessage)
    ..aOM<$1.Timestamp>(12, _omitFieldNames ? '' : 'createdAt',
        subBuilder: $1.Timestamp.$_createMessage)
    ..aOM<$1.Timestamp>(13, _omitFieldNames ? '' : 'updatedAt',
        subBuilder: $1.Timestamp.$_createMessage)
    ..aOS(14, _omitFieldNames ? '' : 'traceId')
    ..aOS(15, _omitFieldNames ? '' : 'inputSchemaHash')
    ..aOS(16, _omitFieldNames ? '' : 'outputSchemaHash')
    ..aOM<$0.Struct>(17, _omitFieldNames ? '' : 'inputPayload',
        subBuilder: $0.Struct.$_createMessage)
    ..aOM<$0.Struct>(18, _omitFieldNames ? '' : 'output',
        subBuilder: $0.Struct.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  WorkflowExecution clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  WorkflowExecution copyWith(void Function(WorkflowExecution) updates) =>
      super.copyWith((message) => updates(message as WorkflowExecution))
          as WorkflowExecution;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use WorkflowExecution() / WorkflowExecution.new instead')
  static WorkflowExecution create() => WorkflowExecution._();
  static $pb.GeneratedMessage $_createMessage() => WorkflowExecution._();
  @$core.override
  WorkflowExecution createEmptyInstance() => WorkflowExecution._();
  @$core.pragma('dart2js:noInline')
  static WorkflowExecution getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<WorkflowExecution>(
          WorkflowExecution.$_createMessage);
  static WorkflowExecution? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get id => $_getSZ(0);
  @$pb.TagNumber(1)
  set id($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get instanceId => $_getSZ(1);
  @$pb.TagNumber(2)
  set instanceId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasInstanceId() => $_has(1);
  @$pb.TagNumber(2)
  void clearInstanceId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get state => $_getSZ(2);
  @$pb.TagNumber(3)
  set state($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasState() => $_has(2);
  @$pb.TagNumber(3)
  void clearState() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.int get stateVersion => $_getIZ(3);
  @$pb.TagNumber(4)
  set stateVersion($core.int value) => $_setSignedInt32(3, value);
  @$pb.TagNumber(4)
  $core.bool hasStateVersion() => $_has(3);
  @$pb.TagNumber(4)
  void clearStateVersion() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.int get attempt => $_getIZ(4);
  @$pb.TagNumber(5)
  set attempt($core.int value) => $_setSignedInt32(4, value);
  @$pb.TagNumber(5)
  $core.bool hasAttempt() => $_has(4);
  @$pb.TagNumber(5)
  void clearAttempt() => $_clearField(5);

  @$pb.TagNumber(6)
  ExecutionStatus get status => $_getN(5);
  @$pb.TagNumber(6)
  set status(ExecutionStatus value) => $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasStatus() => $_has(5);
  @$pb.TagNumber(6)
  void clearStatus() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.String get errorClass => $_getSZ(6);
  @$pb.TagNumber(7)
  set errorClass($core.String value) => $_setString(6, value);
  @$pb.TagNumber(7)
  $core.bool hasErrorClass() => $_has(6);
  @$pb.TagNumber(7)
  void clearErrorClass() => $_clearField(7);

  @$pb.TagNumber(8)
  $core.String get errorMessage => $_getSZ(7);
  @$pb.TagNumber(8)
  set errorMessage($core.String value) => $_setString(7, value);
  @$pb.TagNumber(8)
  $core.bool hasErrorMessage() => $_has(7);
  @$pb.TagNumber(8)
  void clearErrorMessage() => $_clearField(8);

  @$pb.TagNumber(9)
  $1.Timestamp get nextRetryAt => $_getN(8);
  @$pb.TagNumber(9)
  set nextRetryAt($1.Timestamp value) => $_setField(9, value);
  @$pb.TagNumber(9)
  $core.bool hasNextRetryAt() => $_has(8);
  @$pb.TagNumber(9)
  void clearNextRetryAt() => $_clearField(9);
  @$pb.TagNumber(9)
  $1.Timestamp ensureNextRetryAt() => $_ensure(8);

  @$pb.TagNumber(10)
  $1.Timestamp get startedAt => $_getN(9);
  @$pb.TagNumber(10)
  set startedAt($1.Timestamp value) => $_setField(10, value);
  @$pb.TagNumber(10)
  $core.bool hasStartedAt() => $_has(9);
  @$pb.TagNumber(10)
  void clearStartedAt() => $_clearField(10);
  @$pb.TagNumber(10)
  $1.Timestamp ensureStartedAt() => $_ensure(9);

  @$pb.TagNumber(11)
  $1.Timestamp get finishedAt => $_getN(10);
  @$pb.TagNumber(11)
  set finishedAt($1.Timestamp value) => $_setField(11, value);
  @$pb.TagNumber(11)
  $core.bool hasFinishedAt() => $_has(10);
  @$pb.TagNumber(11)
  void clearFinishedAt() => $_clearField(11);
  @$pb.TagNumber(11)
  $1.Timestamp ensureFinishedAt() => $_ensure(10);

  @$pb.TagNumber(12)
  $1.Timestamp get createdAt => $_getN(11);
  @$pb.TagNumber(12)
  set createdAt($1.Timestamp value) => $_setField(12, value);
  @$pb.TagNumber(12)
  $core.bool hasCreatedAt() => $_has(11);
  @$pb.TagNumber(12)
  void clearCreatedAt() => $_clearField(12);
  @$pb.TagNumber(12)
  $1.Timestamp ensureCreatedAt() => $_ensure(11);

  @$pb.TagNumber(13)
  $1.Timestamp get updatedAt => $_getN(12);
  @$pb.TagNumber(13)
  set updatedAt($1.Timestamp value) => $_setField(13, value);
  @$pb.TagNumber(13)
  $core.bool hasUpdatedAt() => $_has(12);
  @$pb.TagNumber(13)
  void clearUpdatedAt() => $_clearField(13);
  @$pb.TagNumber(13)
  $1.Timestamp ensureUpdatedAt() => $_ensure(12);

  @$pb.TagNumber(14)
  $core.String get traceId => $_getSZ(13);
  @$pb.TagNumber(14)
  set traceId($core.String value) => $_setString(13, value);
  @$pb.TagNumber(14)
  $core.bool hasTraceId() => $_has(13);
  @$pb.TagNumber(14)
  void clearTraceId() => $_clearField(14);

  @$pb.TagNumber(15)
  $core.String get inputSchemaHash => $_getSZ(14);
  @$pb.TagNumber(15)
  set inputSchemaHash($core.String value) => $_setString(14, value);
  @$pb.TagNumber(15)
  $core.bool hasInputSchemaHash() => $_has(14);
  @$pb.TagNumber(15)
  void clearInputSchemaHash() => $_clearField(15);

  @$pb.TagNumber(16)
  $core.String get outputSchemaHash => $_getSZ(15);
  @$pb.TagNumber(16)
  set outputSchemaHash($core.String value) => $_setString(15, value);
  @$pb.TagNumber(16)
  $core.bool hasOutputSchemaHash() => $_has(15);
  @$pb.TagNumber(16)
  void clearOutputSchemaHash() => $_clearField(16);

  @$pb.TagNumber(17)
  $0.Struct get inputPayload => $_getN(16);
  @$pb.TagNumber(17)
  set inputPayload($0.Struct value) => $_setField(17, value);
  @$pb.TagNumber(17)
  $core.bool hasInputPayload() => $_has(16);
  @$pb.TagNumber(17)
  void clearInputPayload() => $_clearField(17);
  @$pb.TagNumber(17)
  $0.Struct ensureInputPayload() => $_ensure(16);

  @$pb.TagNumber(18)
  $0.Struct get output => $_getN(17);
  @$pb.TagNumber(18)
  set output($0.Struct value) => $_setField(18, value);
  @$pb.TagNumber(18)
  $core.bool hasOutput() => $_has(17);
  @$pb.TagNumber(18)
  void clearOutput() => $_clearField(18);
  @$pb.TagNumber(18)
  $0.Struct ensureOutput() => $_ensure(17);
}

class ListInstancesRequest extends $pb.GeneratedMessage {
  factory ListInstancesRequest({
    $core.String? workflowName,
    InstanceStatus? status,
    $2.SearchRequest? search,
  }) {
    final result = ListInstancesRequest._();
    if (workflowName != null) result.workflowName = workflowName;
    if (status != null) result.status = status;
    if (search != null) result.search = search;
    return result;
  }

  ListInstancesRequest._();

  factory ListInstancesRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ListInstancesRequest()..mergeFromBuffer(data, registry);
  factory ListInstancesRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ListInstancesRequest()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ListInstancesRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'runtime.v1'),
      createEmptyInstance: ListInstancesRequest.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'workflowName')
    ..aE<InstanceStatus>(2, _omitFieldNames ? '' : 'status',
        enumValues: InstanceStatus.values)
    ..aOM<$2.SearchRequest>(3, _omitFieldNames ? '' : 'search',
        subBuilder: $2.SearchRequest.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListInstancesRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListInstancesRequest copyWith(void Function(ListInstancesRequest) updates) =>
      super.copyWith((message) => updates(message as ListInstancesRequest))
          as ListInstancesRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use ListInstancesRequest() / ListInstancesRequest.new instead')
  static ListInstancesRequest create() => ListInstancesRequest._();
  static $pb.GeneratedMessage $_createMessage() => ListInstancesRequest._();
  @$core.override
  ListInstancesRequest createEmptyInstance() => ListInstancesRequest._();
  @$core.pragma('dart2js:noInline')
  static ListInstancesRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ListInstancesRequest>(
          ListInstancesRequest.$_createMessage);
  static ListInstancesRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get workflowName => $_getSZ(0);
  @$pb.TagNumber(1)
  set workflowName($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasWorkflowName() => $_has(0);
  @$pb.TagNumber(1)
  void clearWorkflowName() => $_clearField(1);

  @$pb.TagNumber(2)
  InstanceStatus get status => $_getN(1);
  @$pb.TagNumber(2)
  set status(InstanceStatus value) => $_setField(2, value);
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

class ListInstancesResponse extends $pb.GeneratedMessage {
  factory ListInstancesResponse({
    $core.Iterable<WorkflowInstance>? items,
    $2.PageCursor? nextCursor,
  }) {
    final result = ListInstancesResponse._();
    if (items != null) result.items.addAll(items);
    if (nextCursor != null) result.nextCursor = nextCursor;
    return result;
  }

  ListInstancesResponse._();

  factory ListInstancesResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ListInstancesResponse()..mergeFromBuffer(data, registry);
  factory ListInstancesResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ListInstancesResponse()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ListInstancesResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'runtime.v1'),
      createEmptyInstance: ListInstancesResponse.$_createMessage)
    ..pPM<WorkflowInstance>(1, _omitFieldNames ? '' : 'items',
        subBuilder: WorkflowInstance.$_createMessage)
    ..aOM<$2.PageCursor>(2, _omitFieldNames ? '' : 'nextCursor',
        subBuilder: $2.PageCursor.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListInstancesResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListInstancesResponse copyWith(
          void Function(ListInstancesResponse) updates) =>
      super.copyWith((message) => updates(message as ListInstancesResponse))
          as ListInstancesResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use ListInstancesResponse() / ListInstancesResponse.new instead')
  static ListInstancesResponse create() => ListInstancesResponse._();
  static $pb.GeneratedMessage $_createMessage() => ListInstancesResponse._();
  @$core.override
  ListInstancesResponse createEmptyInstance() => ListInstancesResponse._();
  @$core.pragma('dart2js:noInline')
  static ListInstancesResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ListInstancesResponse>(
          ListInstancesResponse.$_createMessage);
  static ListInstancesResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<WorkflowInstance> get items => $_getList(0);

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

class RetryInstanceRequest extends $pb.GeneratedMessage {
  factory RetryInstanceRequest({
    $core.String? instanceId,
  }) {
    final result = RetryInstanceRequest._();
    if (instanceId != null) result.instanceId = instanceId;
    return result;
  }

  RetryInstanceRequest._();

  factory RetryInstanceRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      RetryInstanceRequest()..mergeFromBuffer(data, registry);
  factory RetryInstanceRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      RetryInstanceRequest()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RetryInstanceRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'runtime.v1'),
      createEmptyInstance: RetryInstanceRequest.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'instanceId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RetryInstanceRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RetryInstanceRequest copyWith(void Function(RetryInstanceRequest) updates) =>
      super.copyWith((message) => updates(message as RetryInstanceRequest))
          as RetryInstanceRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use RetryInstanceRequest() / RetryInstanceRequest.new instead')
  static RetryInstanceRequest create() => RetryInstanceRequest._();
  static $pb.GeneratedMessage $_createMessage() => RetryInstanceRequest._();
  @$core.override
  RetryInstanceRequest createEmptyInstance() => RetryInstanceRequest._();
  @$core.pragma('dart2js:noInline')
  static RetryInstanceRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RetryInstanceRequest>(
          RetryInstanceRequest.$_createMessage);
  static RetryInstanceRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get instanceId => $_getSZ(0);
  @$pb.TagNumber(1)
  set instanceId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasInstanceId() => $_has(0);
  @$pb.TagNumber(1)
  void clearInstanceId() => $_clearField(1);
}

class RetryInstanceResponse extends $pb.GeneratedMessage {
  factory RetryInstanceResponse({
    WorkflowExecution? execution,
  }) {
    final result = RetryInstanceResponse._();
    if (execution != null) result.execution = execution;
    return result;
  }

  RetryInstanceResponse._();

  factory RetryInstanceResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      RetryInstanceResponse()..mergeFromBuffer(data, registry);
  factory RetryInstanceResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      RetryInstanceResponse()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RetryInstanceResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'runtime.v1'),
      createEmptyInstance: RetryInstanceResponse.$_createMessage)
    ..aOM<WorkflowExecution>(1, _omitFieldNames ? '' : 'execution',
        subBuilder: WorkflowExecution.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RetryInstanceResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RetryInstanceResponse copyWith(
          void Function(RetryInstanceResponse) updates) =>
      super.copyWith((message) => updates(message as RetryInstanceResponse))
          as RetryInstanceResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use RetryInstanceResponse() / RetryInstanceResponse.new instead')
  static RetryInstanceResponse create() => RetryInstanceResponse._();
  static $pb.GeneratedMessage $_createMessage() => RetryInstanceResponse._();
  @$core.override
  RetryInstanceResponse createEmptyInstance() => RetryInstanceResponse._();
  @$core.pragma('dart2js:noInline')
  static RetryInstanceResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RetryInstanceResponse>(
          RetryInstanceResponse.$_createMessage);
  static RetryInstanceResponse? _defaultInstance;

  @$pb.TagNumber(1)
  WorkflowExecution get execution => $_getN(0);
  @$pb.TagNumber(1)
  set execution(WorkflowExecution value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasExecution() => $_has(0);
  @$pb.TagNumber(1)
  void clearExecution() => $_clearField(1);
  @$pb.TagNumber(1)
  WorkflowExecution ensureExecution() => $_ensure(0);
}

class ListExecutionsRequest extends $pb.GeneratedMessage {
  factory ListExecutionsRequest({
    $core.String? instanceId,
    ExecutionStatus? status,
    $2.SearchRequest? search,
  }) {
    final result = ListExecutionsRequest._();
    if (instanceId != null) result.instanceId = instanceId;
    if (status != null) result.status = status;
    if (search != null) result.search = search;
    return result;
  }

  ListExecutionsRequest._();

  factory ListExecutionsRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ListExecutionsRequest()..mergeFromBuffer(data, registry);
  factory ListExecutionsRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ListExecutionsRequest()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ListExecutionsRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'runtime.v1'),
      createEmptyInstance: ListExecutionsRequest.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'instanceId')
    ..aE<ExecutionStatus>(2, _omitFieldNames ? '' : 'status',
        enumValues: ExecutionStatus.values)
    ..aOM<$2.SearchRequest>(3, _omitFieldNames ? '' : 'search',
        subBuilder: $2.SearchRequest.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListExecutionsRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListExecutionsRequest copyWith(
          void Function(ListExecutionsRequest) updates) =>
      super.copyWith((message) => updates(message as ListExecutionsRequest))
          as ListExecutionsRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use ListExecutionsRequest() / ListExecutionsRequest.new instead')
  static ListExecutionsRequest create() => ListExecutionsRequest._();
  static $pb.GeneratedMessage $_createMessage() => ListExecutionsRequest._();
  @$core.override
  ListExecutionsRequest createEmptyInstance() => ListExecutionsRequest._();
  @$core.pragma('dart2js:noInline')
  static ListExecutionsRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ListExecutionsRequest>(
          ListExecutionsRequest.$_createMessage);
  static ListExecutionsRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get instanceId => $_getSZ(0);
  @$pb.TagNumber(1)
  set instanceId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasInstanceId() => $_has(0);
  @$pb.TagNumber(1)
  void clearInstanceId() => $_clearField(1);

  @$pb.TagNumber(2)
  ExecutionStatus get status => $_getN(1);
  @$pb.TagNumber(2)
  set status(ExecutionStatus value) => $_setField(2, value);
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

class ListExecutionsResponse extends $pb.GeneratedMessage {
  factory ListExecutionsResponse({
    $core.Iterable<WorkflowExecution>? items,
    $2.PageCursor? nextCursor,
  }) {
    final result = ListExecutionsResponse._();
    if (items != null) result.items.addAll(items);
    if (nextCursor != null) result.nextCursor = nextCursor;
    return result;
  }

  ListExecutionsResponse._();

  factory ListExecutionsResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ListExecutionsResponse()..mergeFromBuffer(data, registry);
  factory ListExecutionsResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ListExecutionsResponse()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ListExecutionsResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'runtime.v1'),
      createEmptyInstance: ListExecutionsResponse.$_createMessage)
    ..pPM<WorkflowExecution>(1, _omitFieldNames ? '' : 'items',
        subBuilder: WorkflowExecution.$_createMessage)
    ..aOM<$2.PageCursor>(2, _omitFieldNames ? '' : 'nextCursor',
        subBuilder: $2.PageCursor.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListExecutionsResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ListExecutionsResponse copyWith(
          void Function(ListExecutionsResponse) updates) =>
      super.copyWith((message) => updates(message as ListExecutionsResponse))
          as ListExecutionsResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use ListExecutionsResponse() / ListExecutionsResponse.new instead')
  static ListExecutionsResponse create() => ListExecutionsResponse._();
  static $pb.GeneratedMessage $_createMessage() => ListExecutionsResponse._();
  @$core.override
  ListExecutionsResponse createEmptyInstance() => ListExecutionsResponse._();
  @$core.pragma('dart2js:noInline')
  static ListExecutionsResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ListExecutionsResponse>(
          ListExecutionsResponse.$_createMessage);
  static ListExecutionsResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<WorkflowExecution> get items => $_getList(0);

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

class GetExecutionRequest extends $pb.GeneratedMessage {
  factory GetExecutionRequest({
    $core.String? executionId,
    $core.bool? includeOutput,
  }) {
    final result = GetExecutionRequest._();
    if (executionId != null) result.executionId = executionId;
    if (includeOutput != null) result.includeOutput = includeOutput;
    return result;
  }

  GetExecutionRequest._();

  factory GetExecutionRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      GetExecutionRequest()..mergeFromBuffer(data, registry);
  factory GetExecutionRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      GetExecutionRequest()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetExecutionRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'runtime.v1'),
      createEmptyInstance: GetExecutionRequest.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'executionId')
    ..aOB(2, _omitFieldNames ? '' : 'includeOutput')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetExecutionRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetExecutionRequest copyWith(void Function(GetExecutionRequest) updates) =>
      super.copyWith((message) => updates(message as GetExecutionRequest))
          as GetExecutionRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core
      .Deprecated('Use GetExecutionRequest() / GetExecutionRequest.new instead')
  static GetExecutionRequest create() => GetExecutionRequest._();
  static $pb.GeneratedMessage $_createMessage() => GetExecutionRequest._();
  @$core.override
  GetExecutionRequest createEmptyInstance() => GetExecutionRequest._();
  @$core.pragma('dart2js:noInline')
  static GetExecutionRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetExecutionRequest>(
          GetExecutionRequest.$_createMessage);
  static GetExecutionRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get executionId => $_getSZ(0);
  @$pb.TagNumber(1)
  set executionId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasExecutionId() => $_has(0);
  @$pb.TagNumber(1)
  void clearExecutionId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.bool get includeOutput => $_getBF(1);
  @$pb.TagNumber(2)
  set includeOutput($core.bool value) => $_setBool(1, value);
  @$pb.TagNumber(2)
  $core.bool hasIncludeOutput() => $_has(1);
  @$pb.TagNumber(2)
  void clearIncludeOutput() => $_clearField(2);
}

class GetExecutionResponse extends $pb.GeneratedMessage {
  factory GetExecutionResponse({
    WorkflowExecution? execution,
  }) {
    final result = GetExecutionResponse._();
    if (execution != null) result.execution = execution;
    return result;
  }

  GetExecutionResponse._();

  factory GetExecutionResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      GetExecutionResponse()..mergeFromBuffer(data, registry);
  factory GetExecutionResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      GetExecutionResponse()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetExecutionResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'runtime.v1'),
      createEmptyInstance: GetExecutionResponse.$_createMessage)
    ..aOM<WorkflowExecution>(1, _omitFieldNames ? '' : 'execution',
        subBuilder: WorkflowExecution.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetExecutionResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetExecutionResponse copyWith(void Function(GetExecutionResponse) updates) =>
      super.copyWith((message) => updates(message as GetExecutionResponse))
          as GetExecutionResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use GetExecutionResponse() / GetExecutionResponse.new instead')
  static GetExecutionResponse create() => GetExecutionResponse._();
  static $pb.GeneratedMessage $_createMessage() => GetExecutionResponse._();
  @$core.override
  GetExecutionResponse createEmptyInstance() => GetExecutionResponse._();
  @$core.pragma('dart2js:noInline')
  static GetExecutionResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetExecutionResponse>(
          GetExecutionResponse.$_createMessage);
  static GetExecutionResponse? _defaultInstance;

  @$pb.TagNumber(1)
  WorkflowExecution get execution => $_getN(0);
  @$pb.TagNumber(1)
  set execution(WorkflowExecution value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasExecution() => $_has(0);
  @$pb.TagNumber(1)
  void clearExecution() => $_clearField(1);
  @$pb.TagNumber(1)
  WorkflowExecution ensureExecution() => $_ensure(0);
}

class RetryExecutionRequest extends $pb.GeneratedMessage {
  factory RetryExecutionRequest({
    $core.String? executionId,
  }) {
    final result = RetryExecutionRequest._();
    if (executionId != null) result.executionId = executionId;
    return result;
  }

  RetryExecutionRequest._();

  factory RetryExecutionRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      RetryExecutionRequest()..mergeFromBuffer(data, registry);
  factory RetryExecutionRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      RetryExecutionRequest()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RetryExecutionRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'runtime.v1'),
      createEmptyInstance: RetryExecutionRequest.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'executionId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RetryExecutionRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RetryExecutionRequest copyWith(
          void Function(RetryExecutionRequest) updates) =>
      super.copyWith((message) => updates(message as RetryExecutionRequest))
          as RetryExecutionRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use RetryExecutionRequest() / RetryExecutionRequest.new instead')
  static RetryExecutionRequest create() => RetryExecutionRequest._();
  static $pb.GeneratedMessage $_createMessage() => RetryExecutionRequest._();
  @$core.override
  RetryExecutionRequest createEmptyInstance() => RetryExecutionRequest._();
  @$core.pragma('dart2js:noInline')
  static RetryExecutionRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RetryExecutionRequest>(
          RetryExecutionRequest.$_createMessage);
  static RetryExecutionRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get executionId => $_getSZ(0);
  @$pb.TagNumber(1)
  set executionId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasExecutionId() => $_has(0);
  @$pb.TagNumber(1)
  void clearExecutionId() => $_clearField(1);
}

class RetryExecutionResponse extends $pb.GeneratedMessage {
  factory RetryExecutionResponse({
    WorkflowExecution? execution,
  }) {
    final result = RetryExecutionResponse._();
    if (execution != null) result.execution = execution;
    return result;
  }

  RetryExecutionResponse._();

  factory RetryExecutionResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      RetryExecutionResponse()..mergeFromBuffer(data, registry);
  factory RetryExecutionResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      RetryExecutionResponse()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RetryExecutionResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'runtime.v1'),
      createEmptyInstance: RetryExecutionResponse.$_createMessage)
    ..aOM<WorkflowExecution>(1, _omitFieldNames ? '' : 'execution',
        subBuilder: WorkflowExecution.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RetryExecutionResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RetryExecutionResponse copyWith(
          void Function(RetryExecutionResponse) updates) =>
      super.copyWith((message) => updates(message as RetryExecutionResponse))
          as RetryExecutionResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use RetryExecutionResponse() / RetryExecutionResponse.new instead')
  static RetryExecutionResponse create() => RetryExecutionResponse._();
  static $pb.GeneratedMessage $_createMessage() => RetryExecutionResponse._();
  @$core.override
  RetryExecutionResponse createEmptyInstance() => RetryExecutionResponse._();
  @$core.pragma('dart2js:noInline')
  static RetryExecutionResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RetryExecutionResponse>(
          RetryExecutionResponse.$_createMessage);
  static RetryExecutionResponse? _defaultInstance;

  @$pb.TagNumber(1)
  WorkflowExecution get execution => $_getN(0);
  @$pb.TagNumber(1)
  set execution(WorkflowExecution value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasExecution() => $_has(0);
  @$pb.TagNumber(1)
  void clearExecution() => $_clearField(1);
  @$pb.TagNumber(1)
  WorkflowExecution ensureExecution() => $_ensure(0);
}

class ResumeExecutionRequest extends $pb.GeneratedMessage {
  factory ResumeExecutionRequest({
    $core.String? executionId,
    $0.Struct? payload,
  }) {
    final result = ResumeExecutionRequest._();
    if (executionId != null) result.executionId = executionId;
    if (payload != null) result.payload = payload;
    return result;
  }

  ResumeExecutionRequest._();

  factory ResumeExecutionRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ResumeExecutionRequest()..mergeFromBuffer(data, registry);
  factory ResumeExecutionRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ResumeExecutionRequest()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ResumeExecutionRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'runtime.v1'),
      createEmptyInstance: ResumeExecutionRequest.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'executionId')
    ..aOM<$0.Struct>(2, _omitFieldNames ? '' : 'payload',
        subBuilder: $0.Struct.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ResumeExecutionRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ResumeExecutionRequest copyWith(
          void Function(ResumeExecutionRequest) updates) =>
      super.copyWith((message) => updates(message as ResumeExecutionRequest))
          as ResumeExecutionRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use ResumeExecutionRequest() / ResumeExecutionRequest.new instead')
  static ResumeExecutionRequest create() => ResumeExecutionRequest._();
  static $pb.GeneratedMessage $_createMessage() => ResumeExecutionRequest._();
  @$core.override
  ResumeExecutionRequest createEmptyInstance() => ResumeExecutionRequest._();
  @$core.pragma('dart2js:noInline')
  static ResumeExecutionRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ResumeExecutionRequest>(
          ResumeExecutionRequest.$_createMessage);
  static ResumeExecutionRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get executionId => $_getSZ(0);
  @$pb.TagNumber(1)
  set executionId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasExecutionId() => $_has(0);
  @$pb.TagNumber(1)
  void clearExecutionId() => $_clearField(1);

  @$pb.TagNumber(2)
  $0.Struct get payload => $_getN(1);
  @$pb.TagNumber(2)
  set payload($0.Struct value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasPayload() => $_has(1);
  @$pb.TagNumber(2)
  void clearPayload() => $_clearField(2);
  @$pb.TagNumber(2)
  $0.Struct ensurePayload() => $_ensure(1);
}

class ResumeExecutionResponse extends $pb.GeneratedMessage {
  factory ResumeExecutionResponse({
    WorkflowExecution? execution,
    $core.String? action,
  }) {
    final result = ResumeExecutionResponse._();
    if (execution != null) result.execution = execution;
    if (action != null) result.action = action;
    return result;
  }

  ResumeExecutionResponse._();

  factory ResumeExecutionResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ResumeExecutionResponse()..mergeFromBuffer(data, registry);
  factory ResumeExecutionResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ResumeExecutionResponse()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ResumeExecutionResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'runtime.v1'),
      createEmptyInstance: ResumeExecutionResponse.$_createMessage)
    ..aOM<WorkflowExecution>(1, _omitFieldNames ? '' : 'execution',
        subBuilder: WorkflowExecution.$_createMessage)
    ..aOS(2, _omitFieldNames ? '' : 'action')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ResumeExecutionResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ResumeExecutionResponse copyWith(
          void Function(ResumeExecutionResponse) updates) =>
      super.copyWith((message) => updates(message as ResumeExecutionResponse))
          as ResumeExecutionResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use ResumeExecutionResponse() / ResumeExecutionResponse.new instead')
  static ResumeExecutionResponse create() => ResumeExecutionResponse._();
  static $pb.GeneratedMessage $_createMessage() => ResumeExecutionResponse._();
  @$core.override
  ResumeExecutionResponse createEmptyInstance() => ResumeExecutionResponse._();
  @$core.pragma('dart2js:noInline')
  static ResumeExecutionResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ResumeExecutionResponse>(
          ResumeExecutionResponse.$_createMessage);
  static ResumeExecutionResponse? _defaultInstance;

  @$pb.TagNumber(1)
  WorkflowExecution get execution => $_getN(0);
  @$pb.TagNumber(1)
  set execution(WorkflowExecution value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasExecution() => $_has(0);
  @$pb.TagNumber(1)
  void clearExecution() => $_clearField(1);
  @$pb.TagNumber(1)
  WorkflowExecution ensureExecution() => $_ensure(0);

  @$pb.TagNumber(2)
  $core.String get action => $_getSZ(1);
  @$pb.TagNumber(2)
  set action($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasAction() => $_has(1);
  @$pb.TagNumber(2)
  void clearAction() => $_clearField(2);
}

class RunTimelineEntry extends $pb.GeneratedMessage {
  factory RunTimelineEntry({
    $core.String? eventType,
    $core.String? state,
    $core.String? fromState,
    $core.String? toState,
    $core.String? executionId,
    $core.String? traceId,
    $0.Struct? payload,
    $1.Timestamp? createdAt,
  }) {
    final result = RunTimelineEntry._();
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

  RunTimelineEntry._();

  factory RunTimelineEntry.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      RunTimelineEntry()..mergeFromBuffer(data, registry);
  factory RunTimelineEntry.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      RunTimelineEntry()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RunTimelineEntry',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'runtime.v1'),
      createEmptyInstance: RunTimelineEntry.$_createMessage)
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
  RunTimelineEntry clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RunTimelineEntry copyWith(void Function(RunTimelineEntry) updates) =>
      super.copyWith((message) => updates(message as RunTimelineEntry))
          as RunTimelineEntry;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use RunTimelineEntry() / RunTimelineEntry.new instead')
  static RunTimelineEntry create() => RunTimelineEntry._();
  static $pb.GeneratedMessage $_createMessage() => RunTimelineEntry._();
  @$core.override
  RunTimelineEntry createEmptyInstance() => RunTimelineEntry._();
  @$core.pragma('dart2js:noInline')
  static RunTimelineEntry getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<RunTimelineEntry>(
          RunTimelineEntry.$_createMessage);
  static RunTimelineEntry? _defaultInstance;

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

class StateOutput extends $pb.GeneratedMessage {
  factory StateOutput({
    $core.String? executionId,
    $core.String? state,
    $core.String? schemaHash,
    $0.Struct? payload,
    $1.Timestamp? createdAt,
  }) {
    final result = StateOutput._();
    if (executionId != null) result.executionId = executionId;
    if (state != null) result.state = state;
    if (schemaHash != null) result.schemaHash = schemaHash;
    if (payload != null) result.payload = payload;
    if (createdAt != null) result.createdAt = createdAt;
    return result;
  }

  StateOutput._();

  factory StateOutput.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      StateOutput()..mergeFromBuffer(data, registry);
  factory StateOutput.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      StateOutput()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'StateOutput',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'runtime.v1'),
      createEmptyInstance: StateOutput.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'executionId')
    ..aOS(2, _omitFieldNames ? '' : 'state')
    ..aOS(3, _omitFieldNames ? '' : 'schemaHash')
    ..aOM<$0.Struct>(4, _omitFieldNames ? '' : 'payload',
        subBuilder: $0.Struct.$_createMessage)
    ..aOM<$1.Timestamp>(5, _omitFieldNames ? '' : 'createdAt',
        subBuilder: $1.Timestamp.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StateOutput clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StateOutput copyWith(void Function(StateOutput) updates) =>
      super.copyWith((message) => updates(message as StateOutput))
          as StateOutput;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use StateOutput() / StateOutput.new instead')
  static StateOutput create() => StateOutput._();
  static $pb.GeneratedMessage $_createMessage() => StateOutput._();
  @$core.override
  StateOutput createEmptyInstance() => StateOutput._();
  @$core.pragma('dart2js:noInline')
  static StateOutput getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<StateOutput>(
          StateOutput.$_createMessage);
  static StateOutput? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get executionId => $_getSZ(0);
  @$pb.TagNumber(1)
  set executionId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasExecutionId() => $_has(0);
  @$pb.TagNumber(1)
  void clearExecutionId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get state => $_getSZ(1);
  @$pb.TagNumber(2)
  set state($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasState() => $_has(1);
  @$pb.TagNumber(2)
  void clearState() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get schemaHash => $_getSZ(2);
  @$pb.TagNumber(3)
  set schemaHash($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasSchemaHash() => $_has(2);
  @$pb.TagNumber(3)
  void clearSchemaHash() => $_clearField(3);

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

  @$pb.TagNumber(5)
  $1.Timestamp get createdAt => $_getN(4);
  @$pb.TagNumber(5)
  set createdAt($1.Timestamp value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasCreatedAt() => $_has(4);
  @$pb.TagNumber(5)
  void clearCreatedAt() => $_clearField(5);
  @$pb.TagNumber(5)
  $1.Timestamp ensureCreatedAt() => $_ensure(4);
}

class ScopeRun extends $pb.GeneratedMessage {
  factory ScopeRun({
    $core.String? id,
    $core.String? parentExecutionId,
    $core.String? parentState,
    $core.String? scopeType,
    $core.String? status,
    $core.bool? waitAll,
    $core.int? totalChildren,
    $core.int? completedChildren,
    $core.int? failedChildren,
    $core.int? nextChildIndex,
    $core.int? maxConcurrency,
    $core.String? itemVar,
    $core.String? indexVar,
    $0.Struct? itemsPayload,
    $0.Struct? resultsPayload,
    $1.Timestamp? createdAt,
    $1.Timestamp? updatedAt,
  }) {
    final result = ScopeRun._();
    if (id != null) result.id = id;
    if (parentExecutionId != null) result.parentExecutionId = parentExecutionId;
    if (parentState != null) result.parentState = parentState;
    if (scopeType != null) result.scopeType = scopeType;
    if (status != null) result.status = status;
    if (waitAll != null) result.waitAll = waitAll;
    if (totalChildren != null) result.totalChildren = totalChildren;
    if (completedChildren != null) result.completedChildren = completedChildren;
    if (failedChildren != null) result.failedChildren = failedChildren;
    if (nextChildIndex != null) result.nextChildIndex = nextChildIndex;
    if (maxConcurrency != null) result.maxConcurrency = maxConcurrency;
    if (itemVar != null) result.itemVar = itemVar;
    if (indexVar != null) result.indexVar = indexVar;
    if (itemsPayload != null) result.itemsPayload = itemsPayload;
    if (resultsPayload != null) result.resultsPayload = resultsPayload;
    if (createdAt != null) result.createdAt = createdAt;
    if (updatedAt != null) result.updatedAt = updatedAt;
    return result;
  }

  ScopeRun._();

  factory ScopeRun.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ScopeRun()..mergeFromBuffer(data, registry);
  factory ScopeRun.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      ScopeRun()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ScopeRun',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'runtime.v1'),
      createEmptyInstance: ScopeRun.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'id')
    ..aOS(2, _omitFieldNames ? '' : 'parentExecutionId')
    ..aOS(3, _omitFieldNames ? '' : 'parentState')
    ..aOS(4, _omitFieldNames ? '' : 'scopeType')
    ..aOS(5, _omitFieldNames ? '' : 'status')
    ..aOB(6, _omitFieldNames ? '' : 'waitAll')
    ..aI(7, _omitFieldNames ? '' : 'totalChildren')
    ..aI(8, _omitFieldNames ? '' : 'completedChildren')
    ..aI(9, _omitFieldNames ? '' : 'failedChildren')
    ..aI(10, _omitFieldNames ? '' : 'nextChildIndex')
    ..aI(11, _omitFieldNames ? '' : 'maxConcurrency')
    ..aOS(12, _omitFieldNames ? '' : 'itemVar')
    ..aOS(13, _omitFieldNames ? '' : 'indexVar')
    ..aOM<$0.Struct>(14, _omitFieldNames ? '' : 'itemsPayload',
        subBuilder: $0.Struct.$_createMessage)
    ..aOM<$0.Struct>(15, _omitFieldNames ? '' : 'resultsPayload',
        subBuilder: $0.Struct.$_createMessage)
    ..aOM<$1.Timestamp>(16, _omitFieldNames ? '' : 'createdAt',
        subBuilder: $1.Timestamp.$_createMessage)
    ..aOM<$1.Timestamp>(17, _omitFieldNames ? '' : 'updatedAt',
        subBuilder: $1.Timestamp.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ScopeRun clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ScopeRun copyWith(void Function(ScopeRun) updates) =>
      super.copyWith((message) => updates(message as ScopeRun)) as ScopeRun;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use ScopeRun() / ScopeRun.new instead')
  static ScopeRun create() => ScopeRun._();
  static $pb.GeneratedMessage $_createMessage() => ScopeRun._();
  @$core.override
  ScopeRun createEmptyInstance() => ScopeRun._();
  @$core.pragma('dart2js:noInline')
  static ScopeRun getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ScopeRun>(ScopeRun.$_createMessage);
  static ScopeRun? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get id => $_getSZ(0);
  @$pb.TagNumber(1)
  set id($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get parentExecutionId => $_getSZ(1);
  @$pb.TagNumber(2)
  set parentExecutionId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasParentExecutionId() => $_has(1);
  @$pb.TagNumber(2)
  void clearParentExecutionId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get parentState => $_getSZ(2);
  @$pb.TagNumber(3)
  set parentState($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasParentState() => $_has(2);
  @$pb.TagNumber(3)
  void clearParentState() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get scopeType => $_getSZ(3);
  @$pb.TagNumber(4)
  set scopeType($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasScopeType() => $_has(3);
  @$pb.TagNumber(4)
  void clearScopeType() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get status => $_getSZ(4);
  @$pb.TagNumber(5)
  set status($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasStatus() => $_has(4);
  @$pb.TagNumber(5)
  void clearStatus() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.bool get waitAll => $_getBF(5);
  @$pb.TagNumber(6)
  set waitAll($core.bool value) => $_setBool(5, value);
  @$pb.TagNumber(6)
  $core.bool hasWaitAll() => $_has(5);
  @$pb.TagNumber(6)
  void clearWaitAll() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.int get totalChildren => $_getIZ(6);
  @$pb.TagNumber(7)
  set totalChildren($core.int value) => $_setSignedInt32(6, value);
  @$pb.TagNumber(7)
  $core.bool hasTotalChildren() => $_has(6);
  @$pb.TagNumber(7)
  void clearTotalChildren() => $_clearField(7);

  @$pb.TagNumber(8)
  $core.int get completedChildren => $_getIZ(7);
  @$pb.TagNumber(8)
  set completedChildren($core.int value) => $_setSignedInt32(7, value);
  @$pb.TagNumber(8)
  $core.bool hasCompletedChildren() => $_has(7);
  @$pb.TagNumber(8)
  void clearCompletedChildren() => $_clearField(8);

  @$pb.TagNumber(9)
  $core.int get failedChildren => $_getIZ(8);
  @$pb.TagNumber(9)
  set failedChildren($core.int value) => $_setSignedInt32(8, value);
  @$pb.TagNumber(9)
  $core.bool hasFailedChildren() => $_has(8);
  @$pb.TagNumber(9)
  void clearFailedChildren() => $_clearField(9);

  @$pb.TagNumber(10)
  $core.int get nextChildIndex => $_getIZ(9);
  @$pb.TagNumber(10)
  set nextChildIndex($core.int value) => $_setSignedInt32(9, value);
  @$pb.TagNumber(10)
  $core.bool hasNextChildIndex() => $_has(9);
  @$pb.TagNumber(10)
  void clearNextChildIndex() => $_clearField(10);

  @$pb.TagNumber(11)
  $core.int get maxConcurrency => $_getIZ(10);
  @$pb.TagNumber(11)
  set maxConcurrency($core.int value) => $_setSignedInt32(10, value);
  @$pb.TagNumber(11)
  $core.bool hasMaxConcurrency() => $_has(10);
  @$pb.TagNumber(11)
  void clearMaxConcurrency() => $_clearField(11);

  @$pb.TagNumber(12)
  $core.String get itemVar => $_getSZ(11);
  @$pb.TagNumber(12)
  set itemVar($core.String value) => $_setString(11, value);
  @$pb.TagNumber(12)
  $core.bool hasItemVar() => $_has(11);
  @$pb.TagNumber(12)
  void clearItemVar() => $_clearField(12);

  @$pb.TagNumber(13)
  $core.String get indexVar => $_getSZ(12);
  @$pb.TagNumber(13)
  set indexVar($core.String value) => $_setString(12, value);
  @$pb.TagNumber(13)
  $core.bool hasIndexVar() => $_has(12);
  @$pb.TagNumber(13)
  void clearIndexVar() => $_clearField(13);

  @$pb.TagNumber(14)
  $0.Struct get itemsPayload => $_getN(13);
  @$pb.TagNumber(14)
  set itemsPayload($0.Struct value) => $_setField(14, value);
  @$pb.TagNumber(14)
  $core.bool hasItemsPayload() => $_has(13);
  @$pb.TagNumber(14)
  void clearItemsPayload() => $_clearField(14);
  @$pb.TagNumber(14)
  $0.Struct ensureItemsPayload() => $_ensure(13);

  @$pb.TagNumber(15)
  $0.Struct get resultsPayload => $_getN(14);
  @$pb.TagNumber(15)
  set resultsPayload($0.Struct value) => $_setField(15, value);
  @$pb.TagNumber(15)
  $core.bool hasResultsPayload() => $_has(14);
  @$pb.TagNumber(15)
  void clearResultsPayload() => $_clearField(15);
  @$pb.TagNumber(15)
  $0.Struct ensureResultsPayload() => $_ensure(14);

  @$pb.TagNumber(16)
  $1.Timestamp get createdAt => $_getN(15);
  @$pb.TagNumber(16)
  set createdAt($1.Timestamp value) => $_setField(16, value);
  @$pb.TagNumber(16)
  $core.bool hasCreatedAt() => $_has(15);
  @$pb.TagNumber(16)
  void clearCreatedAt() => $_clearField(16);
  @$pb.TagNumber(16)
  $1.Timestamp ensureCreatedAt() => $_ensure(15);

  @$pb.TagNumber(17)
  $1.Timestamp get updatedAt => $_getN(16);
  @$pb.TagNumber(17)
  set updatedAt($1.Timestamp value) => $_setField(17, value);
  @$pb.TagNumber(17)
  $core.bool hasUpdatedAt() => $_has(16);
  @$pb.TagNumber(17)
  void clearUpdatedAt() => $_clearField(17);
  @$pb.TagNumber(17)
  $1.Timestamp ensureUpdatedAt() => $_ensure(16);
}

class SignalWait extends $pb.GeneratedMessage {
  factory SignalWait({
    $core.String? id,
    $core.String? executionId,
    $core.String? state,
    $core.String? signalName,
    $core.String? outputVar,
    $core.String? status,
    $1.Timestamp? timeoutAt,
    $1.Timestamp? matchedAt,
    $1.Timestamp? timedOutAt,
    $core.String? messageId,
    $core.int? attempts,
    $1.Timestamp? createdAt,
    $1.Timestamp? updatedAt,
  }) {
    final result = SignalWait._();
    if (id != null) result.id = id;
    if (executionId != null) result.executionId = executionId;
    if (state != null) result.state = state;
    if (signalName != null) result.signalName = signalName;
    if (outputVar != null) result.outputVar = outputVar;
    if (status != null) result.status = status;
    if (timeoutAt != null) result.timeoutAt = timeoutAt;
    if (matchedAt != null) result.matchedAt = matchedAt;
    if (timedOutAt != null) result.timedOutAt = timedOutAt;
    if (messageId != null) result.messageId = messageId;
    if (attempts != null) result.attempts = attempts;
    if (createdAt != null) result.createdAt = createdAt;
    if (updatedAt != null) result.updatedAt = updatedAt;
    return result;
  }

  SignalWait._();

  factory SignalWait.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      SignalWait()..mergeFromBuffer(data, registry);
  factory SignalWait.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      SignalWait()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SignalWait',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'runtime.v1'),
      createEmptyInstance: SignalWait.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'id')
    ..aOS(2, _omitFieldNames ? '' : 'executionId')
    ..aOS(3, _omitFieldNames ? '' : 'state')
    ..aOS(4, _omitFieldNames ? '' : 'signalName')
    ..aOS(5, _omitFieldNames ? '' : 'outputVar')
    ..aOS(6, _omitFieldNames ? '' : 'status')
    ..aOM<$1.Timestamp>(7, _omitFieldNames ? '' : 'timeoutAt',
        subBuilder: $1.Timestamp.$_createMessage)
    ..aOM<$1.Timestamp>(8, _omitFieldNames ? '' : 'matchedAt',
        subBuilder: $1.Timestamp.$_createMessage)
    ..aOM<$1.Timestamp>(9, _omitFieldNames ? '' : 'timedOutAt',
        subBuilder: $1.Timestamp.$_createMessage)
    ..aOS(10, _omitFieldNames ? '' : 'messageId')
    ..aI(11, _omitFieldNames ? '' : 'attempts')
    ..aOM<$1.Timestamp>(12, _omitFieldNames ? '' : 'createdAt',
        subBuilder: $1.Timestamp.$_createMessage)
    ..aOM<$1.Timestamp>(13, _omitFieldNames ? '' : 'updatedAt',
        subBuilder: $1.Timestamp.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SignalWait clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SignalWait copyWith(void Function(SignalWait) updates) =>
      super.copyWith((message) => updates(message as SignalWait)) as SignalWait;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use SignalWait() / SignalWait.new instead')
  static SignalWait create() => SignalWait._();
  static $pb.GeneratedMessage $_createMessage() => SignalWait._();
  @$core.override
  SignalWait createEmptyInstance() => SignalWait._();
  @$core.pragma('dart2js:noInline')
  static SignalWait getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SignalWait>(SignalWait.$_createMessage);
  static SignalWait? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get id => $_getSZ(0);
  @$pb.TagNumber(1)
  set id($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get executionId => $_getSZ(1);
  @$pb.TagNumber(2)
  set executionId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasExecutionId() => $_has(1);
  @$pb.TagNumber(2)
  void clearExecutionId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get state => $_getSZ(2);
  @$pb.TagNumber(3)
  set state($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasState() => $_has(2);
  @$pb.TagNumber(3)
  void clearState() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get signalName => $_getSZ(3);
  @$pb.TagNumber(4)
  set signalName($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasSignalName() => $_has(3);
  @$pb.TagNumber(4)
  void clearSignalName() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get outputVar => $_getSZ(4);
  @$pb.TagNumber(5)
  set outputVar($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasOutputVar() => $_has(4);
  @$pb.TagNumber(5)
  void clearOutputVar() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get status => $_getSZ(5);
  @$pb.TagNumber(6)
  set status($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasStatus() => $_has(5);
  @$pb.TagNumber(6)
  void clearStatus() => $_clearField(6);

  @$pb.TagNumber(7)
  $1.Timestamp get timeoutAt => $_getN(6);
  @$pb.TagNumber(7)
  set timeoutAt($1.Timestamp value) => $_setField(7, value);
  @$pb.TagNumber(7)
  $core.bool hasTimeoutAt() => $_has(6);
  @$pb.TagNumber(7)
  void clearTimeoutAt() => $_clearField(7);
  @$pb.TagNumber(7)
  $1.Timestamp ensureTimeoutAt() => $_ensure(6);

  @$pb.TagNumber(8)
  $1.Timestamp get matchedAt => $_getN(7);
  @$pb.TagNumber(8)
  set matchedAt($1.Timestamp value) => $_setField(8, value);
  @$pb.TagNumber(8)
  $core.bool hasMatchedAt() => $_has(7);
  @$pb.TagNumber(8)
  void clearMatchedAt() => $_clearField(8);
  @$pb.TagNumber(8)
  $1.Timestamp ensureMatchedAt() => $_ensure(7);

  @$pb.TagNumber(9)
  $1.Timestamp get timedOutAt => $_getN(8);
  @$pb.TagNumber(9)
  set timedOutAt($1.Timestamp value) => $_setField(9, value);
  @$pb.TagNumber(9)
  $core.bool hasTimedOutAt() => $_has(8);
  @$pb.TagNumber(9)
  void clearTimedOutAt() => $_clearField(9);
  @$pb.TagNumber(9)
  $1.Timestamp ensureTimedOutAt() => $_ensure(8);

  @$pb.TagNumber(10)
  $core.String get messageId => $_getSZ(9);
  @$pb.TagNumber(10)
  set messageId($core.String value) => $_setString(9, value);
  @$pb.TagNumber(10)
  $core.bool hasMessageId() => $_has(9);
  @$pb.TagNumber(10)
  void clearMessageId() => $_clearField(10);

  @$pb.TagNumber(11)
  $core.int get attempts => $_getIZ(10);
  @$pb.TagNumber(11)
  set attempts($core.int value) => $_setSignedInt32(10, value);
  @$pb.TagNumber(11)
  $core.bool hasAttempts() => $_has(10);
  @$pb.TagNumber(11)
  void clearAttempts() => $_clearField(11);

  @$pb.TagNumber(12)
  $1.Timestamp get createdAt => $_getN(11);
  @$pb.TagNumber(12)
  set createdAt($1.Timestamp value) => $_setField(12, value);
  @$pb.TagNumber(12)
  $core.bool hasCreatedAt() => $_has(11);
  @$pb.TagNumber(12)
  void clearCreatedAt() => $_clearField(12);
  @$pb.TagNumber(12)
  $1.Timestamp ensureCreatedAt() => $_ensure(11);

  @$pb.TagNumber(13)
  $1.Timestamp get updatedAt => $_getN(12);
  @$pb.TagNumber(13)
  set updatedAt($1.Timestamp value) => $_setField(13, value);
  @$pb.TagNumber(13)
  $core.bool hasUpdatedAt() => $_has(12);
  @$pb.TagNumber(13)
  void clearUpdatedAt() => $_clearField(13);
  @$pb.TagNumber(13)
  $1.Timestamp ensureUpdatedAt() => $_ensure(12);
}

class SignalMessage extends $pb.GeneratedMessage {
  factory SignalMessage({
    $core.String? id,
    $core.String? signalName,
    $0.Struct? payload,
    $core.String? status,
    $1.Timestamp? deliveredAt,
    $core.String? waitId,
    $core.int? attempts,
    $1.Timestamp? createdAt,
    $1.Timestamp? updatedAt,
  }) {
    final result = SignalMessage._();
    if (id != null) result.id = id;
    if (signalName != null) result.signalName = signalName;
    if (payload != null) result.payload = payload;
    if (status != null) result.status = status;
    if (deliveredAt != null) result.deliveredAt = deliveredAt;
    if (waitId != null) result.waitId = waitId;
    if (attempts != null) result.attempts = attempts;
    if (createdAt != null) result.createdAt = createdAt;
    if (updatedAt != null) result.updatedAt = updatedAt;
    return result;
  }

  SignalMessage._();

  factory SignalMessage.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      SignalMessage()..mergeFromBuffer(data, registry);
  factory SignalMessage.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      SignalMessage()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SignalMessage',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'runtime.v1'),
      createEmptyInstance: SignalMessage.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'id')
    ..aOS(2, _omitFieldNames ? '' : 'signalName')
    ..aOM<$0.Struct>(3, _omitFieldNames ? '' : 'payload',
        subBuilder: $0.Struct.$_createMessage)
    ..aOS(4, _omitFieldNames ? '' : 'status')
    ..aOM<$1.Timestamp>(5, _omitFieldNames ? '' : 'deliveredAt',
        subBuilder: $1.Timestamp.$_createMessage)
    ..aOS(6, _omitFieldNames ? '' : 'waitId')
    ..aI(7, _omitFieldNames ? '' : 'attempts')
    ..aOM<$1.Timestamp>(8, _omitFieldNames ? '' : 'createdAt',
        subBuilder: $1.Timestamp.$_createMessage)
    ..aOM<$1.Timestamp>(9, _omitFieldNames ? '' : 'updatedAt',
        subBuilder: $1.Timestamp.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SignalMessage clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SignalMessage copyWith(void Function(SignalMessage) updates) =>
      super.copyWith((message) => updates(message as SignalMessage))
          as SignalMessage;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use SignalMessage() / SignalMessage.new instead')
  static SignalMessage create() => SignalMessage._();
  static $pb.GeneratedMessage $_createMessage() => SignalMessage._();
  @$core.override
  SignalMessage createEmptyInstance() => SignalMessage._();
  @$core.pragma('dart2js:noInline')
  static SignalMessage getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<SignalMessage>(
          SignalMessage.$_createMessage);
  static SignalMessage? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get id => $_getSZ(0);
  @$pb.TagNumber(1)
  set id($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get signalName => $_getSZ(1);
  @$pb.TagNumber(2)
  set signalName($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasSignalName() => $_has(1);
  @$pb.TagNumber(2)
  void clearSignalName() => $_clearField(2);

  @$pb.TagNumber(3)
  $0.Struct get payload => $_getN(2);
  @$pb.TagNumber(3)
  set payload($0.Struct value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasPayload() => $_has(2);
  @$pb.TagNumber(3)
  void clearPayload() => $_clearField(3);
  @$pb.TagNumber(3)
  $0.Struct ensurePayload() => $_ensure(2);

  @$pb.TagNumber(4)
  $core.String get status => $_getSZ(3);
  @$pb.TagNumber(4)
  set status($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasStatus() => $_has(3);
  @$pb.TagNumber(4)
  void clearStatus() => $_clearField(4);

  @$pb.TagNumber(5)
  $1.Timestamp get deliveredAt => $_getN(4);
  @$pb.TagNumber(5)
  set deliveredAt($1.Timestamp value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasDeliveredAt() => $_has(4);
  @$pb.TagNumber(5)
  void clearDeliveredAt() => $_clearField(5);
  @$pb.TagNumber(5)
  $1.Timestamp ensureDeliveredAt() => $_ensure(4);

  @$pb.TagNumber(6)
  $core.String get waitId => $_getSZ(5);
  @$pb.TagNumber(6)
  set waitId($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasWaitId() => $_has(5);
  @$pb.TagNumber(6)
  void clearWaitId() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.int get attempts => $_getIZ(6);
  @$pb.TagNumber(7)
  set attempts($core.int value) => $_setSignedInt32(6, value);
  @$pb.TagNumber(7)
  $core.bool hasAttempts() => $_has(6);
  @$pb.TagNumber(7)
  void clearAttempts() => $_clearField(7);

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

class GetInstanceRunRequest extends $pb.GeneratedMessage {
  factory GetInstanceRunRequest({
    $core.String? instanceId,
    $core.bool? includePayloads,
    $core.int? executionLimit,
    $core.int? timelineLimit,
  }) {
    final result = GetInstanceRunRequest._();
    if (instanceId != null) result.instanceId = instanceId;
    if (includePayloads != null) result.includePayloads = includePayloads;
    if (executionLimit != null) result.executionLimit = executionLimit;
    if (timelineLimit != null) result.timelineLimit = timelineLimit;
    return result;
  }

  GetInstanceRunRequest._();

  factory GetInstanceRunRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      GetInstanceRunRequest()..mergeFromBuffer(data, registry);
  factory GetInstanceRunRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      GetInstanceRunRequest()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetInstanceRunRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'runtime.v1'),
      createEmptyInstance: GetInstanceRunRequest.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'instanceId')
    ..aOB(2, _omitFieldNames ? '' : 'includePayloads')
    ..aI(3, _omitFieldNames ? '' : 'executionLimit')
    ..aI(4, _omitFieldNames ? '' : 'timelineLimit')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetInstanceRunRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetInstanceRunRequest copyWith(
          void Function(GetInstanceRunRequest) updates) =>
      super.copyWith((message) => updates(message as GetInstanceRunRequest))
          as GetInstanceRunRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use GetInstanceRunRequest() / GetInstanceRunRequest.new instead')
  static GetInstanceRunRequest create() => GetInstanceRunRequest._();
  static $pb.GeneratedMessage $_createMessage() => GetInstanceRunRequest._();
  @$core.override
  GetInstanceRunRequest createEmptyInstance() => GetInstanceRunRequest._();
  @$core.pragma('dart2js:noInline')
  static GetInstanceRunRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetInstanceRunRequest>(
          GetInstanceRunRequest.$_createMessage);
  static GetInstanceRunRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get instanceId => $_getSZ(0);
  @$pb.TagNumber(1)
  set instanceId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasInstanceId() => $_has(0);
  @$pb.TagNumber(1)
  void clearInstanceId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.bool get includePayloads => $_getBF(1);
  @$pb.TagNumber(2)
  set includePayloads($core.bool value) => $_setBool(1, value);
  @$pb.TagNumber(2)
  $core.bool hasIncludePayloads() => $_has(1);
  @$pb.TagNumber(2)
  void clearIncludePayloads() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.int get executionLimit => $_getIZ(2);
  @$pb.TagNumber(3)
  set executionLimit($core.int value) => $_setSignedInt32(2, value);
  @$pb.TagNumber(3)
  $core.bool hasExecutionLimit() => $_has(2);
  @$pb.TagNumber(3)
  void clearExecutionLimit() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.int get timelineLimit => $_getIZ(3);
  @$pb.TagNumber(4)
  set timelineLimit($core.int value) => $_setSignedInt32(3, value);
  @$pb.TagNumber(4)
  $core.bool hasTimelineLimit() => $_has(3);
  @$pb.TagNumber(4)
  void clearTimelineLimit() => $_clearField(4);
}

class GetInstanceRunResponse extends $pb.GeneratedMessage {
  factory GetInstanceRunResponse({
    WorkflowInstance? instance,
    WorkflowExecution? latestExecution,
    $core.String? traceId,
    $core.String? resumeStrategy,
    $core.Iterable<WorkflowExecution>? executions,
    $core.Iterable<RunTimelineEntry>? timeline,
    $core.Iterable<StateOutput>? outputs,
    $core.Iterable<ScopeRun>? scopeRuns,
    $core.Iterable<SignalWait>? signalWaits,
    $core.Iterable<SignalMessage>? signalMessages,
  }) {
    final result = GetInstanceRunResponse._();
    if (instance != null) result.instance = instance;
    if (latestExecution != null) result.latestExecution = latestExecution;
    if (traceId != null) result.traceId = traceId;
    if (resumeStrategy != null) result.resumeStrategy = resumeStrategy;
    if (executions != null) result.executions.addAll(executions);
    if (timeline != null) result.timeline.addAll(timeline);
    if (outputs != null) result.outputs.addAll(outputs);
    if (scopeRuns != null) result.scopeRuns.addAll(scopeRuns);
    if (signalWaits != null) result.signalWaits.addAll(signalWaits);
    if (signalMessages != null) result.signalMessages.addAll(signalMessages);
    return result;
  }

  GetInstanceRunResponse._();

  factory GetInstanceRunResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      GetInstanceRunResponse()..mergeFromBuffer(data, registry);
  factory GetInstanceRunResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      GetInstanceRunResponse()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GetInstanceRunResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'runtime.v1'),
      createEmptyInstance: GetInstanceRunResponse.$_createMessage)
    ..aOM<WorkflowInstance>(1, _omitFieldNames ? '' : 'instance',
        subBuilder: WorkflowInstance.$_createMessage)
    ..aOM<WorkflowExecution>(2, _omitFieldNames ? '' : 'latestExecution',
        subBuilder: WorkflowExecution.$_createMessage)
    ..aOS(3, _omitFieldNames ? '' : 'traceId')
    ..aOS(4, _omitFieldNames ? '' : 'resumeStrategy')
    ..pPM<WorkflowExecution>(5, _omitFieldNames ? '' : 'executions',
        subBuilder: WorkflowExecution.$_createMessage)
    ..pPM<RunTimelineEntry>(6, _omitFieldNames ? '' : 'timeline',
        subBuilder: RunTimelineEntry.$_createMessage)
    ..pPM<StateOutput>(7, _omitFieldNames ? '' : 'outputs',
        subBuilder: StateOutput.$_createMessage)
    ..pPM<ScopeRun>(8, _omitFieldNames ? '' : 'scopeRuns',
        subBuilder: ScopeRun.$_createMessage)
    ..pPM<SignalWait>(9, _omitFieldNames ? '' : 'signalWaits',
        subBuilder: SignalWait.$_createMessage)
    ..pPM<SignalMessage>(10, _omitFieldNames ? '' : 'signalMessages',
        subBuilder: SignalMessage.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetInstanceRunResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GetInstanceRunResponse copyWith(
          void Function(GetInstanceRunResponse) updates) =>
      super.copyWith((message) => updates(message as GetInstanceRunResponse))
          as GetInstanceRunResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated(
      'Use GetInstanceRunResponse() / GetInstanceRunResponse.new instead')
  static GetInstanceRunResponse create() => GetInstanceRunResponse._();
  static $pb.GeneratedMessage $_createMessage() => GetInstanceRunResponse._();
  @$core.override
  GetInstanceRunResponse createEmptyInstance() => GetInstanceRunResponse._();
  @$core.pragma('dart2js:noInline')
  static GetInstanceRunResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GetInstanceRunResponse>(
          GetInstanceRunResponse.$_createMessage);
  static GetInstanceRunResponse? _defaultInstance;

  @$pb.TagNumber(1)
  WorkflowInstance get instance => $_getN(0);
  @$pb.TagNumber(1)
  set instance(WorkflowInstance value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasInstance() => $_has(0);
  @$pb.TagNumber(1)
  void clearInstance() => $_clearField(1);
  @$pb.TagNumber(1)
  WorkflowInstance ensureInstance() => $_ensure(0);

  @$pb.TagNumber(2)
  WorkflowExecution get latestExecution => $_getN(1);
  @$pb.TagNumber(2)
  set latestExecution(WorkflowExecution value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasLatestExecution() => $_has(1);
  @$pb.TagNumber(2)
  void clearLatestExecution() => $_clearField(2);
  @$pb.TagNumber(2)
  WorkflowExecution ensureLatestExecution() => $_ensure(1);

  @$pb.TagNumber(3)
  $core.String get traceId => $_getSZ(2);
  @$pb.TagNumber(3)
  set traceId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasTraceId() => $_has(2);
  @$pb.TagNumber(3)
  void clearTraceId() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get resumeStrategy => $_getSZ(3);
  @$pb.TagNumber(4)
  set resumeStrategy($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasResumeStrategy() => $_has(3);
  @$pb.TagNumber(4)
  void clearResumeStrategy() => $_clearField(4);

  @$pb.TagNumber(5)
  $pb.PbList<WorkflowExecution> get executions => $_getList(4);

  @$pb.TagNumber(6)
  $pb.PbList<RunTimelineEntry> get timeline => $_getList(5);

  @$pb.TagNumber(7)
  $pb.PbList<StateOutput> get outputs => $_getList(6);

  @$pb.TagNumber(8)
  $pb.PbList<ScopeRun> get scopeRuns => $_getList(7);

  @$pb.TagNumber(9)
  $pb.PbList<SignalWait> get signalWaits => $_getList(8);

  @$pb.TagNumber(10)
  $pb.PbList<SignalMessage> get signalMessages => $_getList(9);
}

class RuntimeServiceApi {
  final $pb.RpcClient _client;

  RuntimeServiceApi(this._client);

  $async.Future<ListInstancesResponse> listInstances(
          $pb.ClientContext? ctx, ListInstancesRequest request) =>
      _client.invoke<ListInstancesResponse>(ctx, 'RuntimeService',
          'ListInstances', request, ListInstancesResponse());
  $async.Future<RetryInstanceResponse> retryInstance(
          $pb.ClientContext? ctx, RetryInstanceRequest request) =>
      _client.invoke<RetryInstanceResponse>(ctx, 'RuntimeService',
          'RetryInstance', request, RetryInstanceResponse());
  $async.Future<ListExecutionsResponse> listExecutions(
          $pb.ClientContext? ctx, ListExecutionsRequest request) =>
      _client.invoke<ListExecutionsResponse>(ctx, 'RuntimeService',
          'ListExecutions', request, ListExecutionsResponse());
  $async.Future<GetExecutionResponse> getExecution(
          $pb.ClientContext? ctx, GetExecutionRequest request) =>
      _client.invoke<GetExecutionResponse>(ctx, 'RuntimeService',
          'GetExecution', request, GetExecutionResponse());
  $async.Future<RetryExecutionResponse> retryExecution(
          $pb.ClientContext? ctx, RetryExecutionRequest request) =>
      _client.invoke<RetryExecutionResponse>(ctx, 'RuntimeService',
          'RetryExecution', request, RetryExecutionResponse());
  $async.Future<ResumeExecutionResponse> resumeExecution(
          $pb.ClientContext? ctx, ResumeExecutionRequest request) =>
      _client.invoke<ResumeExecutionResponse>(ctx, 'RuntimeService',
          'ResumeExecution', request, ResumeExecutionResponse());
  $async.Future<GetInstanceRunResponse> getInstanceRun(
          $pb.ClientContext? ctx, GetInstanceRunRequest request) =>
      _client.invoke<GetInstanceRunResponse>(ctx, 'RuntimeService',
          'GetInstanceRun', request, GetInstanceRunResponse());
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
