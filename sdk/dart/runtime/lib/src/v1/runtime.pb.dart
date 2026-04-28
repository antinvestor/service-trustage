//
//  Generated code. Do not modify.
//  source: v1/runtime.proto
//
// @dart = 2.12

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_final_fields
// ignore_for_file: unnecessary_import, unnecessary_this, unused_import

import 'dart:async' as $async;
import 'dart:core' as $core;

import 'package:fixnum/fixnum.dart' as $fixnum;
import 'package:protobuf/protobuf.dart' as $pb;

import '../common/v1/common.pb.dart' as $7;
import '../google/protobuf/struct.pb.dart' as $6;
import '../google/protobuf/timestamp.pb.dart' as $2;
import 'runtime.pbenum.dart';

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
    $6.Struct? metadata,
    $2.Timestamp? startedAt,
    $2.Timestamp? finishedAt,
    $2.Timestamp? createdAt,
    $2.Timestamp? updatedAt,
    $core.String? parentInstanceId,
    $core.String? parentExecutionId,
    $core.String? scopeType,
    $core.String? scopeParentState,
    $core.String? scopeEntryState,
    $core.int? scopeIndex,
  }) {
    final $result = create();
    if (id != null) {
      $result.id = id;
    }
    if (workflowName != null) {
      $result.workflowName = workflowName;
    }
    if (workflowVersion != null) {
      $result.workflowVersion = workflowVersion;
    }
    if (currentState != null) {
      $result.currentState = currentState;
    }
    if (status != null) {
      $result.status = status;
    }
    if (revision != null) {
      $result.revision = revision;
    }
    if (triggerEventId != null) {
      $result.triggerEventId = triggerEventId;
    }
    if (metadata != null) {
      $result.metadata = metadata;
    }
    if (startedAt != null) {
      $result.startedAt = startedAt;
    }
    if (finishedAt != null) {
      $result.finishedAt = finishedAt;
    }
    if (createdAt != null) {
      $result.createdAt = createdAt;
    }
    if (updatedAt != null) {
      $result.updatedAt = updatedAt;
    }
    if (parentInstanceId != null) {
      $result.parentInstanceId = parentInstanceId;
    }
    if (parentExecutionId != null) {
      $result.parentExecutionId = parentExecutionId;
    }
    if (scopeType != null) {
      $result.scopeType = scopeType;
    }
    if (scopeParentState != null) {
      $result.scopeParentState = scopeParentState;
    }
    if (scopeEntryState != null) {
      $result.scopeEntryState = scopeEntryState;
    }
    if (scopeIndex != null) {
      $result.scopeIndex = scopeIndex;
    }
    return $result;
  }
  WorkflowInstance._() : super();
  factory WorkflowInstance.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory WorkflowInstance.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'WorkflowInstance', package: const $pb.PackageName(_omitMessageNames ? '' : 'runtime.v1'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'id')
    ..aOS(2, _omitFieldNames ? '' : 'workflowName')
    ..a<$core.int>(3, _omitFieldNames ? '' : 'workflowVersion', $pb.PbFieldType.O3)
    ..aOS(4, _omitFieldNames ? '' : 'currentState')
    ..e<InstanceStatus>(5, _omitFieldNames ? '' : 'status', $pb.PbFieldType.OE, defaultOrMaker: InstanceStatus.INSTANCE_STATUS_UNSPECIFIED, valueOf: InstanceStatus.valueOf, enumValues: InstanceStatus.values)
    ..aInt64(6, _omitFieldNames ? '' : 'revision')
    ..aOS(7, _omitFieldNames ? '' : 'triggerEventId')
    ..aOM<$6.Struct>(8, _omitFieldNames ? '' : 'metadata', subBuilder: $6.Struct.create)
    ..aOM<$2.Timestamp>(9, _omitFieldNames ? '' : 'startedAt', subBuilder: $2.Timestamp.create)
    ..aOM<$2.Timestamp>(10, _omitFieldNames ? '' : 'finishedAt', subBuilder: $2.Timestamp.create)
    ..aOM<$2.Timestamp>(11, _omitFieldNames ? '' : 'createdAt', subBuilder: $2.Timestamp.create)
    ..aOM<$2.Timestamp>(12, _omitFieldNames ? '' : 'updatedAt', subBuilder: $2.Timestamp.create)
    ..aOS(13, _omitFieldNames ? '' : 'parentInstanceId')
    ..aOS(14, _omitFieldNames ? '' : 'parentExecutionId')
    ..aOS(15, _omitFieldNames ? '' : 'scopeType')
    ..aOS(16, _omitFieldNames ? '' : 'scopeParentState')
    ..aOS(17, _omitFieldNames ? '' : 'scopeEntryState')
    ..a<$core.int>(18, _omitFieldNames ? '' : 'scopeIndex', $pb.PbFieldType.O3)
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  WorkflowInstance clone() => WorkflowInstance()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  WorkflowInstance copyWith(void Function(WorkflowInstance) updates) => super.copyWith((message) => updates(message as WorkflowInstance)) as WorkflowInstance;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static WorkflowInstance create() => WorkflowInstance._();
  WorkflowInstance createEmptyInstance() => create();
  static $pb.PbList<WorkflowInstance> createRepeated() => $pb.PbList<WorkflowInstance>();
  @$core.pragma('dart2js:noInline')
  static WorkflowInstance getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<WorkflowInstance>(create);
  static WorkflowInstance? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get id => $_getSZ(0);
  @$pb.TagNumber(1)
  set id($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get workflowName => $_getSZ(1);
  @$pb.TagNumber(2)
  set workflowName($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasWorkflowName() => $_has(1);
  @$pb.TagNumber(2)
  void clearWorkflowName() => clearField(2);

  @$pb.TagNumber(3)
  $core.int get workflowVersion => $_getIZ(2);
  @$pb.TagNumber(3)
  set workflowVersion($core.int v) { $_setSignedInt32(2, v); }
  @$pb.TagNumber(3)
  $core.bool hasWorkflowVersion() => $_has(2);
  @$pb.TagNumber(3)
  void clearWorkflowVersion() => clearField(3);

  @$pb.TagNumber(4)
  $core.String get currentState => $_getSZ(3);
  @$pb.TagNumber(4)
  set currentState($core.String v) { $_setString(3, v); }
  @$pb.TagNumber(4)
  $core.bool hasCurrentState() => $_has(3);
  @$pb.TagNumber(4)
  void clearCurrentState() => clearField(4);

  @$pb.TagNumber(5)
  InstanceStatus get status => $_getN(4);
  @$pb.TagNumber(5)
  set status(InstanceStatus v) { setField(5, v); }
  @$pb.TagNumber(5)
  $core.bool hasStatus() => $_has(4);
  @$pb.TagNumber(5)
  void clearStatus() => clearField(5);

  @$pb.TagNumber(6)
  $fixnum.Int64 get revision => $_getI64(5);
  @$pb.TagNumber(6)
  set revision($fixnum.Int64 v) { $_setInt64(5, v); }
  @$pb.TagNumber(6)
  $core.bool hasRevision() => $_has(5);
  @$pb.TagNumber(6)
  void clearRevision() => clearField(6);

  @$pb.TagNumber(7)
  $core.String get triggerEventId => $_getSZ(6);
  @$pb.TagNumber(7)
  set triggerEventId($core.String v) { $_setString(6, v); }
  @$pb.TagNumber(7)
  $core.bool hasTriggerEventId() => $_has(6);
  @$pb.TagNumber(7)
  void clearTriggerEventId() => clearField(7);

  @$pb.TagNumber(8)
  $6.Struct get metadata => $_getN(7);
  @$pb.TagNumber(8)
  set metadata($6.Struct v) { setField(8, v); }
  @$pb.TagNumber(8)
  $core.bool hasMetadata() => $_has(7);
  @$pb.TagNumber(8)
  void clearMetadata() => clearField(8);
  @$pb.TagNumber(8)
  $6.Struct ensureMetadata() => $_ensure(7);

  @$pb.TagNumber(9)
  $2.Timestamp get startedAt => $_getN(8);
  @$pb.TagNumber(9)
  set startedAt($2.Timestamp v) { setField(9, v); }
  @$pb.TagNumber(9)
  $core.bool hasStartedAt() => $_has(8);
  @$pb.TagNumber(9)
  void clearStartedAt() => clearField(9);
  @$pb.TagNumber(9)
  $2.Timestamp ensureStartedAt() => $_ensure(8);

  @$pb.TagNumber(10)
  $2.Timestamp get finishedAt => $_getN(9);
  @$pb.TagNumber(10)
  set finishedAt($2.Timestamp v) { setField(10, v); }
  @$pb.TagNumber(10)
  $core.bool hasFinishedAt() => $_has(9);
  @$pb.TagNumber(10)
  void clearFinishedAt() => clearField(10);
  @$pb.TagNumber(10)
  $2.Timestamp ensureFinishedAt() => $_ensure(9);

  @$pb.TagNumber(11)
  $2.Timestamp get createdAt => $_getN(10);
  @$pb.TagNumber(11)
  set createdAt($2.Timestamp v) { setField(11, v); }
  @$pb.TagNumber(11)
  $core.bool hasCreatedAt() => $_has(10);
  @$pb.TagNumber(11)
  void clearCreatedAt() => clearField(11);
  @$pb.TagNumber(11)
  $2.Timestamp ensureCreatedAt() => $_ensure(10);

  @$pb.TagNumber(12)
  $2.Timestamp get updatedAt => $_getN(11);
  @$pb.TagNumber(12)
  set updatedAt($2.Timestamp v) { setField(12, v); }
  @$pb.TagNumber(12)
  $core.bool hasUpdatedAt() => $_has(11);
  @$pb.TagNumber(12)
  void clearUpdatedAt() => clearField(12);
  @$pb.TagNumber(12)
  $2.Timestamp ensureUpdatedAt() => $_ensure(11);

  @$pb.TagNumber(13)
  $core.String get parentInstanceId => $_getSZ(12);
  @$pb.TagNumber(13)
  set parentInstanceId($core.String v) { $_setString(12, v); }
  @$pb.TagNumber(13)
  $core.bool hasParentInstanceId() => $_has(12);
  @$pb.TagNumber(13)
  void clearParentInstanceId() => clearField(13);

  @$pb.TagNumber(14)
  $core.String get parentExecutionId => $_getSZ(13);
  @$pb.TagNumber(14)
  set parentExecutionId($core.String v) { $_setString(13, v); }
  @$pb.TagNumber(14)
  $core.bool hasParentExecutionId() => $_has(13);
  @$pb.TagNumber(14)
  void clearParentExecutionId() => clearField(14);

  @$pb.TagNumber(15)
  $core.String get scopeType => $_getSZ(14);
  @$pb.TagNumber(15)
  set scopeType($core.String v) { $_setString(14, v); }
  @$pb.TagNumber(15)
  $core.bool hasScopeType() => $_has(14);
  @$pb.TagNumber(15)
  void clearScopeType() => clearField(15);

  @$pb.TagNumber(16)
  $core.String get scopeParentState => $_getSZ(15);
  @$pb.TagNumber(16)
  set scopeParentState($core.String v) { $_setString(15, v); }
  @$pb.TagNumber(16)
  $core.bool hasScopeParentState() => $_has(15);
  @$pb.TagNumber(16)
  void clearScopeParentState() => clearField(16);

  @$pb.TagNumber(17)
  $core.String get scopeEntryState => $_getSZ(16);
  @$pb.TagNumber(17)
  set scopeEntryState($core.String v) { $_setString(16, v); }
  @$pb.TagNumber(17)
  $core.bool hasScopeEntryState() => $_has(16);
  @$pb.TagNumber(17)
  void clearScopeEntryState() => clearField(17);

  @$pb.TagNumber(18)
  $core.int get scopeIndex => $_getIZ(17);
  @$pb.TagNumber(18)
  set scopeIndex($core.int v) { $_setSignedInt32(17, v); }
  @$pb.TagNumber(18)
  $core.bool hasScopeIndex() => $_has(17);
  @$pb.TagNumber(18)
  void clearScopeIndex() => clearField(18);
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
    $2.Timestamp? nextRetryAt,
    $2.Timestamp? startedAt,
    $2.Timestamp? finishedAt,
    $2.Timestamp? createdAt,
    $2.Timestamp? updatedAt,
    $core.String? traceId,
    $core.String? inputSchemaHash,
    $core.String? outputSchemaHash,
    $6.Struct? inputPayload,
    $6.Struct? output,
  }) {
    final $result = create();
    if (id != null) {
      $result.id = id;
    }
    if (instanceId != null) {
      $result.instanceId = instanceId;
    }
    if (state != null) {
      $result.state = state;
    }
    if (stateVersion != null) {
      $result.stateVersion = stateVersion;
    }
    if (attempt != null) {
      $result.attempt = attempt;
    }
    if (status != null) {
      $result.status = status;
    }
    if (errorClass != null) {
      $result.errorClass = errorClass;
    }
    if (errorMessage != null) {
      $result.errorMessage = errorMessage;
    }
    if (nextRetryAt != null) {
      $result.nextRetryAt = nextRetryAt;
    }
    if (startedAt != null) {
      $result.startedAt = startedAt;
    }
    if (finishedAt != null) {
      $result.finishedAt = finishedAt;
    }
    if (createdAt != null) {
      $result.createdAt = createdAt;
    }
    if (updatedAt != null) {
      $result.updatedAt = updatedAt;
    }
    if (traceId != null) {
      $result.traceId = traceId;
    }
    if (inputSchemaHash != null) {
      $result.inputSchemaHash = inputSchemaHash;
    }
    if (outputSchemaHash != null) {
      $result.outputSchemaHash = outputSchemaHash;
    }
    if (inputPayload != null) {
      $result.inputPayload = inputPayload;
    }
    if (output != null) {
      $result.output = output;
    }
    return $result;
  }
  WorkflowExecution._() : super();
  factory WorkflowExecution.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory WorkflowExecution.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'WorkflowExecution', package: const $pb.PackageName(_omitMessageNames ? '' : 'runtime.v1'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'id')
    ..aOS(2, _omitFieldNames ? '' : 'instanceId')
    ..aOS(3, _omitFieldNames ? '' : 'state')
    ..a<$core.int>(4, _omitFieldNames ? '' : 'stateVersion', $pb.PbFieldType.O3)
    ..a<$core.int>(5, _omitFieldNames ? '' : 'attempt', $pb.PbFieldType.O3)
    ..e<ExecutionStatus>(6, _omitFieldNames ? '' : 'status', $pb.PbFieldType.OE, defaultOrMaker: ExecutionStatus.EXECUTION_STATUS_UNSPECIFIED, valueOf: ExecutionStatus.valueOf, enumValues: ExecutionStatus.values)
    ..aOS(7, _omitFieldNames ? '' : 'errorClass')
    ..aOS(8, _omitFieldNames ? '' : 'errorMessage')
    ..aOM<$2.Timestamp>(9, _omitFieldNames ? '' : 'nextRetryAt', subBuilder: $2.Timestamp.create)
    ..aOM<$2.Timestamp>(10, _omitFieldNames ? '' : 'startedAt', subBuilder: $2.Timestamp.create)
    ..aOM<$2.Timestamp>(11, _omitFieldNames ? '' : 'finishedAt', subBuilder: $2.Timestamp.create)
    ..aOM<$2.Timestamp>(12, _omitFieldNames ? '' : 'createdAt', subBuilder: $2.Timestamp.create)
    ..aOM<$2.Timestamp>(13, _omitFieldNames ? '' : 'updatedAt', subBuilder: $2.Timestamp.create)
    ..aOS(14, _omitFieldNames ? '' : 'traceId')
    ..aOS(15, _omitFieldNames ? '' : 'inputSchemaHash')
    ..aOS(16, _omitFieldNames ? '' : 'outputSchemaHash')
    ..aOM<$6.Struct>(17, _omitFieldNames ? '' : 'inputPayload', subBuilder: $6.Struct.create)
    ..aOM<$6.Struct>(18, _omitFieldNames ? '' : 'output', subBuilder: $6.Struct.create)
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  WorkflowExecution clone() => WorkflowExecution()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  WorkflowExecution copyWith(void Function(WorkflowExecution) updates) => super.copyWith((message) => updates(message as WorkflowExecution)) as WorkflowExecution;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static WorkflowExecution create() => WorkflowExecution._();
  WorkflowExecution createEmptyInstance() => create();
  static $pb.PbList<WorkflowExecution> createRepeated() => $pb.PbList<WorkflowExecution>();
  @$core.pragma('dart2js:noInline')
  static WorkflowExecution getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<WorkflowExecution>(create);
  static WorkflowExecution? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get id => $_getSZ(0);
  @$pb.TagNumber(1)
  set id($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get instanceId => $_getSZ(1);
  @$pb.TagNumber(2)
  set instanceId($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasInstanceId() => $_has(1);
  @$pb.TagNumber(2)
  void clearInstanceId() => clearField(2);

  @$pb.TagNumber(3)
  $core.String get state => $_getSZ(2);
  @$pb.TagNumber(3)
  set state($core.String v) { $_setString(2, v); }
  @$pb.TagNumber(3)
  $core.bool hasState() => $_has(2);
  @$pb.TagNumber(3)
  void clearState() => clearField(3);

  @$pb.TagNumber(4)
  $core.int get stateVersion => $_getIZ(3);
  @$pb.TagNumber(4)
  set stateVersion($core.int v) { $_setSignedInt32(3, v); }
  @$pb.TagNumber(4)
  $core.bool hasStateVersion() => $_has(3);
  @$pb.TagNumber(4)
  void clearStateVersion() => clearField(4);

  @$pb.TagNumber(5)
  $core.int get attempt => $_getIZ(4);
  @$pb.TagNumber(5)
  set attempt($core.int v) { $_setSignedInt32(4, v); }
  @$pb.TagNumber(5)
  $core.bool hasAttempt() => $_has(4);
  @$pb.TagNumber(5)
  void clearAttempt() => clearField(5);

  @$pb.TagNumber(6)
  ExecutionStatus get status => $_getN(5);
  @$pb.TagNumber(6)
  set status(ExecutionStatus v) { setField(6, v); }
  @$pb.TagNumber(6)
  $core.bool hasStatus() => $_has(5);
  @$pb.TagNumber(6)
  void clearStatus() => clearField(6);

  @$pb.TagNumber(7)
  $core.String get errorClass => $_getSZ(6);
  @$pb.TagNumber(7)
  set errorClass($core.String v) { $_setString(6, v); }
  @$pb.TagNumber(7)
  $core.bool hasErrorClass() => $_has(6);
  @$pb.TagNumber(7)
  void clearErrorClass() => clearField(7);

  @$pb.TagNumber(8)
  $core.String get errorMessage => $_getSZ(7);
  @$pb.TagNumber(8)
  set errorMessage($core.String v) { $_setString(7, v); }
  @$pb.TagNumber(8)
  $core.bool hasErrorMessage() => $_has(7);
  @$pb.TagNumber(8)
  void clearErrorMessage() => clearField(8);

  @$pb.TagNumber(9)
  $2.Timestamp get nextRetryAt => $_getN(8);
  @$pb.TagNumber(9)
  set nextRetryAt($2.Timestamp v) { setField(9, v); }
  @$pb.TagNumber(9)
  $core.bool hasNextRetryAt() => $_has(8);
  @$pb.TagNumber(9)
  void clearNextRetryAt() => clearField(9);
  @$pb.TagNumber(9)
  $2.Timestamp ensureNextRetryAt() => $_ensure(8);

  @$pb.TagNumber(10)
  $2.Timestamp get startedAt => $_getN(9);
  @$pb.TagNumber(10)
  set startedAt($2.Timestamp v) { setField(10, v); }
  @$pb.TagNumber(10)
  $core.bool hasStartedAt() => $_has(9);
  @$pb.TagNumber(10)
  void clearStartedAt() => clearField(10);
  @$pb.TagNumber(10)
  $2.Timestamp ensureStartedAt() => $_ensure(9);

  @$pb.TagNumber(11)
  $2.Timestamp get finishedAt => $_getN(10);
  @$pb.TagNumber(11)
  set finishedAt($2.Timestamp v) { setField(11, v); }
  @$pb.TagNumber(11)
  $core.bool hasFinishedAt() => $_has(10);
  @$pb.TagNumber(11)
  void clearFinishedAt() => clearField(11);
  @$pb.TagNumber(11)
  $2.Timestamp ensureFinishedAt() => $_ensure(10);

  @$pb.TagNumber(12)
  $2.Timestamp get createdAt => $_getN(11);
  @$pb.TagNumber(12)
  set createdAt($2.Timestamp v) { setField(12, v); }
  @$pb.TagNumber(12)
  $core.bool hasCreatedAt() => $_has(11);
  @$pb.TagNumber(12)
  void clearCreatedAt() => clearField(12);
  @$pb.TagNumber(12)
  $2.Timestamp ensureCreatedAt() => $_ensure(11);

  @$pb.TagNumber(13)
  $2.Timestamp get updatedAt => $_getN(12);
  @$pb.TagNumber(13)
  set updatedAt($2.Timestamp v) { setField(13, v); }
  @$pb.TagNumber(13)
  $core.bool hasUpdatedAt() => $_has(12);
  @$pb.TagNumber(13)
  void clearUpdatedAt() => clearField(13);
  @$pb.TagNumber(13)
  $2.Timestamp ensureUpdatedAt() => $_ensure(12);

  @$pb.TagNumber(14)
  $core.String get traceId => $_getSZ(13);
  @$pb.TagNumber(14)
  set traceId($core.String v) { $_setString(13, v); }
  @$pb.TagNumber(14)
  $core.bool hasTraceId() => $_has(13);
  @$pb.TagNumber(14)
  void clearTraceId() => clearField(14);

  @$pb.TagNumber(15)
  $core.String get inputSchemaHash => $_getSZ(14);
  @$pb.TagNumber(15)
  set inputSchemaHash($core.String v) { $_setString(14, v); }
  @$pb.TagNumber(15)
  $core.bool hasInputSchemaHash() => $_has(14);
  @$pb.TagNumber(15)
  void clearInputSchemaHash() => clearField(15);

  @$pb.TagNumber(16)
  $core.String get outputSchemaHash => $_getSZ(15);
  @$pb.TagNumber(16)
  set outputSchemaHash($core.String v) { $_setString(15, v); }
  @$pb.TagNumber(16)
  $core.bool hasOutputSchemaHash() => $_has(15);
  @$pb.TagNumber(16)
  void clearOutputSchemaHash() => clearField(16);

  @$pb.TagNumber(17)
  $6.Struct get inputPayload => $_getN(16);
  @$pb.TagNumber(17)
  set inputPayload($6.Struct v) { setField(17, v); }
  @$pb.TagNumber(17)
  $core.bool hasInputPayload() => $_has(16);
  @$pb.TagNumber(17)
  void clearInputPayload() => clearField(17);
  @$pb.TagNumber(17)
  $6.Struct ensureInputPayload() => $_ensure(16);

  @$pb.TagNumber(18)
  $6.Struct get output => $_getN(17);
  @$pb.TagNumber(18)
  set output($6.Struct v) { setField(18, v); }
  @$pb.TagNumber(18)
  $core.bool hasOutput() => $_has(17);
  @$pb.TagNumber(18)
  void clearOutput() => clearField(18);
  @$pb.TagNumber(18)
  $6.Struct ensureOutput() => $_ensure(17);
}

class ListInstancesRequest extends $pb.GeneratedMessage {
  factory ListInstancesRequest({
    $core.String? workflowName,
    InstanceStatus? status,
    $7.SearchRequest? search,
  }) {
    final $result = create();
    if (workflowName != null) {
      $result.workflowName = workflowName;
    }
    if (status != null) {
      $result.status = status;
    }
    if (search != null) {
      $result.search = search;
    }
    return $result;
  }
  ListInstancesRequest._() : super();
  factory ListInstancesRequest.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory ListInstancesRequest.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'ListInstancesRequest', package: const $pb.PackageName(_omitMessageNames ? '' : 'runtime.v1'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'workflowName')
    ..e<InstanceStatus>(2, _omitFieldNames ? '' : 'status', $pb.PbFieldType.OE, defaultOrMaker: InstanceStatus.INSTANCE_STATUS_UNSPECIFIED, valueOf: InstanceStatus.valueOf, enumValues: InstanceStatus.values)
    ..aOM<$7.SearchRequest>(3, _omitFieldNames ? '' : 'search', subBuilder: $7.SearchRequest.create)
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  ListInstancesRequest clone() => ListInstancesRequest()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  ListInstancesRequest copyWith(void Function(ListInstancesRequest) updates) => super.copyWith((message) => updates(message as ListInstancesRequest)) as ListInstancesRequest;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ListInstancesRequest create() => ListInstancesRequest._();
  ListInstancesRequest createEmptyInstance() => create();
  static $pb.PbList<ListInstancesRequest> createRepeated() => $pb.PbList<ListInstancesRequest>();
  @$core.pragma('dart2js:noInline')
  static ListInstancesRequest getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<ListInstancesRequest>(create);
  static ListInstancesRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get workflowName => $_getSZ(0);
  @$pb.TagNumber(1)
  set workflowName($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasWorkflowName() => $_has(0);
  @$pb.TagNumber(1)
  void clearWorkflowName() => clearField(1);

  @$pb.TagNumber(2)
  InstanceStatus get status => $_getN(1);
  @$pb.TagNumber(2)
  set status(InstanceStatus v) { setField(2, v); }
  @$pb.TagNumber(2)
  $core.bool hasStatus() => $_has(1);
  @$pb.TagNumber(2)
  void clearStatus() => clearField(2);

  @$pb.TagNumber(3)
  $7.SearchRequest get search => $_getN(2);
  @$pb.TagNumber(3)
  set search($7.SearchRequest v) { setField(3, v); }
  @$pb.TagNumber(3)
  $core.bool hasSearch() => $_has(2);
  @$pb.TagNumber(3)
  void clearSearch() => clearField(3);
  @$pb.TagNumber(3)
  $7.SearchRequest ensureSearch() => $_ensure(2);
}

class ListInstancesResponse extends $pb.GeneratedMessage {
  factory ListInstancesResponse({
    $core.Iterable<WorkflowInstance>? items,
    $7.PageCursor? nextCursor,
  }) {
    final $result = create();
    if (items != null) {
      $result.items.addAll(items);
    }
    if (nextCursor != null) {
      $result.nextCursor = nextCursor;
    }
    return $result;
  }
  ListInstancesResponse._() : super();
  factory ListInstancesResponse.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory ListInstancesResponse.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'ListInstancesResponse', package: const $pb.PackageName(_omitMessageNames ? '' : 'runtime.v1'), createEmptyInstance: create)
    ..pc<WorkflowInstance>(1, _omitFieldNames ? '' : 'items', $pb.PbFieldType.PM, subBuilder: WorkflowInstance.create)
    ..aOM<$7.PageCursor>(2, _omitFieldNames ? '' : 'nextCursor', subBuilder: $7.PageCursor.create)
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  ListInstancesResponse clone() => ListInstancesResponse()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  ListInstancesResponse copyWith(void Function(ListInstancesResponse) updates) => super.copyWith((message) => updates(message as ListInstancesResponse)) as ListInstancesResponse;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ListInstancesResponse create() => ListInstancesResponse._();
  ListInstancesResponse createEmptyInstance() => create();
  static $pb.PbList<ListInstancesResponse> createRepeated() => $pb.PbList<ListInstancesResponse>();
  @$core.pragma('dart2js:noInline')
  static ListInstancesResponse getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<ListInstancesResponse>(create);
  static ListInstancesResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.List<WorkflowInstance> get items => $_getList(0);

  @$pb.TagNumber(2)
  $7.PageCursor get nextCursor => $_getN(1);
  @$pb.TagNumber(2)
  set nextCursor($7.PageCursor v) { setField(2, v); }
  @$pb.TagNumber(2)
  $core.bool hasNextCursor() => $_has(1);
  @$pb.TagNumber(2)
  void clearNextCursor() => clearField(2);
  @$pb.TagNumber(2)
  $7.PageCursor ensureNextCursor() => $_ensure(1);
}

class RetryInstanceRequest extends $pb.GeneratedMessage {
  factory RetryInstanceRequest({
    $core.String? instanceId,
  }) {
    final $result = create();
    if (instanceId != null) {
      $result.instanceId = instanceId;
    }
    return $result;
  }
  RetryInstanceRequest._() : super();
  factory RetryInstanceRequest.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory RetryInstanceRequest.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'RetryInstanceRequest', package: const $pb.PackageName(_omitMessageNames ? '' : 'runtime.v1'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'instanceId')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  RetryInstanceRequest clone() => RetryInstanceRequest()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  RetryInstanceRequest copyWith(void Function(RetryInstanceRequest) updates) => super.copyWith((message) => updates(message as RetryInstanceRequest)) as RetryInstanceRequest;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RetryInstanceRequest create() => RetryInstanceRequest._();
  RetryInstanceRequest createEmptyInstance() => create();
  static $pb.PbList<RetryInstanceRequest> createRepeated() => $pb.PbList<RetryInstanceRequest>();
  @$core.pragma('dart2js:noInline')
  static RetryInstanceRequest getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<RetryInstanceRequest>(create);
  static RetryInstanceRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get instanceId => $_getSZ(0);
  @$pb.TagNumber(1)
  set instanceId($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasInstanceId() => $_has(0);
  @$pb.TagNumber(1)
  void clearInstanceId() => clearField(1);
}

class RetryInstanceResponse extends $pb.GeneratedMessage {
  factory RetryInstanceResponse({
    WorkflowExecution? execution,
  }) {
    final $result = create();
    if (execution != null) {
      $result.execution = execution;
    }
    return $result;
  }
  RetryInstanceResponse._() : super();
  factory RetryInstanceResponse.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory RetryInstanceResponse.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'RetryInstanceResponse', package: const $pb.PackageName(_omitMessageNames ? '' : 'runtime.v1'), createEmptyInstance: create)
    ..aOM<WorkflowExecution>(1, _omitFieldNames ? '' : 'execution', subBuilder: WorkflowExecution.create)
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  RetryInstanceResponse clone() => RetryInstanceResponse()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  RetryInstanceResponse copyWith(void Function(RetryInstanceResponse) updates) => super.copyWith((message) => updates(message as RetryInstanceResponse)) as RetryInstanceResponse;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RetryInstanceResponse create() => RetryInstanceResponse._();
  RetryInstanceResponse createEmptyInstance() => create();
  static $pb.PbList<RetryInstanceResponse> createRepeated() => $pb.PbList<RetryInstanceResponse>();
  @$core.pragma('dart2js:noInline')
  static RetryInstanceResponse getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<RetryInstanceResponse>(create);
  static RetryInstanceResponse? _defaultInstance;

  @$pb.TagNumber(1)
  WorkflowExecution get execution => $_getN(0);
  @$pb.TagNumber(1)
  set execution(WorkflowExecution v) { setField(1, v); }
  @$pb.TagNumber(1)
  $core.bool hasExecution() => $_has(0);
  @$pb.TagNumber(1)
  void clearExecution() => clearField(1);
  @$pb.TagNumber(1)
  WorkflowExecution ensureExecution() => $_ensure(0);
}

class ListExecutionsRequest extends $pb.GeneratedMessage {
  factory ListExecutionsRequest({
    $core.String? instanceId,
    ExecutionStatus? status,
    $7.SearchRequest? search,
  }) {
    final $result = create();
    if (instanceId != null) {
      $result.instanceId = instanceId;
    }
    if (status != null) {
      $result.status = status;
    }
    if (search != null) {
      $result.search = search;
    }
    return $result;
  }
  ListExecutionsRequest._() : super();
  factory ListExecutionsRequest.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory ListExecutionsRequest.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'ListExecutionsRequest', package: const $pb.PackageName(_omitMessageNames ? '' : 'runtime.v1'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'instanceId')
    ..e<ExecutionStatus>(2, _omitFieldNames ? '' : 'status', $pb.PbFieldType.OE, defaultOrMaker: ExecutionStatus.EXECUTION_STATUS_UNSPECIFIED, valueOf: ExecutionStatus.valueOf, enumValues: ExecutionStatus.values)
    ..aOM<$7.SearchRequest>(3, _omitFieldNames ? '' : 'search', subBuilder: $7.SearchRequest.create)
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  ListExecutionsRequest clone() => ListExecutionsRequest()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  ListExecutionsRequest copyWith(void Function(ListExecutionsRequest) updates) => super.copyWith((message) => updates(message as ListExecutionsRequest)) as ListExecutionsRequest;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ListExecutionsRequest create() => ListExecutionsRequest._();
  ListExecutionsRequest createEmptyInstance() => create();
  static $pb.PbList<ListExecutionsRequest> createRepeated() => $pb.PbList<ListExecutionsRequest>();
  @$core.pragma('dart2js:noInline')
  static ListExecutionsRequest getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<ListExecutionsRequest>(create);
  static ListExecutionsRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get instanceId => $_getSZ(0);
  @$pb.TagNumber(1)
  set instanceId($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasInstanceId() => $_has(0);
  @$pb.TagNumber(1)
  void clearInstanceId() => clearField(1);

  @$pb.TagNumber(2)
  ExecutionStatus get status => $_getN(1);
  @$pb.TagNumber(2)
  set status(ExecutionStatus v) { setField(2, v); }
  @$pb.TagNumber(2)
  $core.bool hasStatus() => $_has(1);
  @$pb.TagNumber(2)
  void clearStatus() => clearField(2);

  @$pb.TagNumber(3)
  $7.SearchRequest get search => $_getN(2);
  @$pb.TagNumber(3)
  set search($7.SearchRequest v) { setField(3, v); }
  @$pb.TagNumber(3)
  $core.bool hasSearch() => $_has(2);
  @$pb.TagNumber(3)
  void clearSearch() => clearField(3);
  @$pb.TagNumber(3)
  $7.SearchRequest ensureSearch() => $_ensure(2);
}

class ListExecutionsResponse extends $pb.GeneratedMessage {
  factory ListExecutionsResponse({
    $core.Iterable<WorkflowExecution>? items,
    $7.PageCursor? nextCursor,
  }) {
    final $result = create();
    if (items != null) {
      $result.items.addAll(items);
    }
    if (nextCursor != null) {
      $result.nextCursor = nextCursor;
    }
    return $result;
  }
  ListExecutionsResponse._() : super();
  factory ListExecutionsResponse.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory ListExecutionsResponse.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'ListExecutionsResponse', package: const $pb.PackageName(_omitMessageNames ? '' : 'runtime.v1'), createEmptyInstance: create)
    ..pc<WorkflowExecution>(1, _omitFieldNames ? '' : 'items', $pb.PbFieldType.PM, subBuilder: WorkflowExecution.create)
    ..aOM<$7.PageCursor>(2, _omitFieldNames ? '' : 'nextCursor', subBuilder: $7.PageCursor.create)
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  ListExecutionsResponse clone() => ListExecutionsResponse()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  ListExecutionsResponse copyWith(void Function(ListExecutionsResponse) updates) => super.copyWith((message) => updates(message as ListExecutionsResponse)) as ListExecutionsResponse;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ListExecutionsResponse create() => ListExecutionsResponse._();
  ListExecutionsResponse createEmptyInstance() => create();
  static $pb.PbList<ListExecutionsResponse> createRepeated() => $pb.PbList<ListExecutionsResponse>();
  @$core.pragma('dart2js:noInline')
  static ListExecutionsResponse getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<ListExecutionsResponse>(create);
  static ListExecutionsResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.List<WorkflowExecution> get items => $_getList(0);

  @$pb.TagNumber(2)
  $7.PageCursor get nextCursor => $_getN(1);
  @$pb.TagNumber(2)
  set nextCursor($7.PageCursor v) { setField(2, v); }
  @$pb.TagNumber(2)
  $core.bool hasNextCursor() => $_has(1);
  @$pb.TagNumber(2)
  void clearNextCursor() => clearField(2);
  @$pb.TagNumber(2)
  $7.PageCursor ensureNextCursor() => $_ensure(1);
}

class GetExecutionRequest extends $pb.GeneratedMessage {
  factory GetExecutionRequest({
    $core.String? executionId,
    $core.bool? includeOutput,
  }) {
    final $result = create();
    if (executionId != null) {
      $result.executionId = executionId;
    }
    if (includeOutput != null) {
      $result.includeOutput = includeOutput;
    }
    return $result;
  }
  GetExecutionRequest._() : super();
  factory GetExecutionRequest.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory GetExecutionRequest.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'GetExecutionRequest', package: const $pb.PackageName(_omitMessageNames ? '' : 'runtime.v1'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'executionId')
    ..aOB(2, _omitFieldNames ? '' : 'includeOutput')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  GetExecutionRequest clone() => GetExecutionRequest()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  GetExecutionRequest copyWith(void Function(GetExecutionRequest) updates) => super.copyWith((message) => updates(message as GetExecutionRequest)) as GetExecutionRequest;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetExecutionRequest create() => GetExecutionRequest._();
  GetExecutionRequest createEmptyInstance() => create();
  static $pb.PbList<GetExecutionRequest> createRepeated() => $pb.PbList<GetExecutionRequest>();
  @$core.pragma('dart2js:noInline')
  static GetExecutionRequest getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<GetExecutionRequest>(create);
  static GetExecutionRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get executionId => $_getSZ(0);
  @$pb.TagNumber(1)
  set executionId($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasExecutionId() => $_has(0);
  @$pb.TagNumber(1)
  void clearExecutionId() => clearField(1);

  @$pb.TagNumber(2)
  $core.bool get includeOutput => $_getBF(1);
  @$pb.TagNumber(2)
  set includeOutput($core.bool v) { $_setBool(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasIncludeOutput() => $_has(1);
  @$pb.TagNumber(2)
  void clearIncludeOutput() => clearField(2);
}

class GetExecutionResponse extends $pb.GeneratedMessage {
  factory GetExecutionResponse({
    WorkflowExecution? execution,
  }) {
    final $result = create();
    if (execution != null) {
      $result.execution = execution;
    }
    return $result;
  }
  GetExecutionResponse._() : super();
  factory GetExecutionResponse.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory GetExecutionResponse.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'GetExecutionResponse', package: const $pb.PackageName(_omitMessageNames ? '' : 'runtime.v1'), createEmptyInstance: create)
    ..aOM<WorkflowExecution>(1, _omitFieldNames ? '' : 'execution', subBuilder: WorkflowExecution.create)
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  GetExecutionResponse clone() => GetExecutionResponse()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  GetExecutionResponse copyWith(void Function(GetExecutionResponse) updates) => super.copyWith((message) => updates(message as GetExecutionResponse)) as GetExecutionResponse;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetExecutionResponse create() => GetExecutionResponse._();
  GetExecutionResponse createEmptyInstance() => create();
  static $pb.PbList<GetExecutionResponse> createRepeated() => $pb.PbList<GetExecutionResponse>();
  @$core.pragma('dart2js:noInline')
  static GetExecutionResponse getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<GetExecutionResponse>(create);
  static GetExecutionResponse? _defaultInstance;

  @$pb.TagNumber(1)
  WorkflowExecution get execution => $_getN(0);
  @$pb.TagNumber(1)
  set execution(WorkflowExecution v) { setField(1, v); }
  @$pb.TagNumber(1)
  $core.bool hasExecution() => $_has(0);
  @$pb.TagNumber(1)
  void clearExecution() => clearField(1);
  @$pb.TagNumber(1)
  WorkflowExecution ensureExecution() => $_ensure(0);
}

class RetryExecutionRequest extends $pb.GeneratedMessage {
  factory RetryExecutionRequest({
    $core.String? executionId,
  }) {
    final $result = create();
    if (executionId != null) {
      $result.executionId = executionId;
    }
    return $result;
  }
  RetryExecutionRequest._() : super();
  factory RetryExecutionRequest.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory RetryExecutionRequest.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'RetryExecutionRequest', package: const $pb.PackageName(_omitMessageNames ? '' : 'runtime.v1'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'executionId')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  RetryExecutionRequest clone() => RetryExecutionRequest()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  RetryExecutionRequest copyWith(void Function(RetryExecutionRequest) updates) => super.copyWith((message) => updates(message as RetryExecutionRequest)) as RetryExecutionRequest;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RetryExecutionRequest create() => RetryExecutionRequest._();
  RetryExecutionRequest createEmptyInstance() => create();
  static $pb.PbList<RetryExecutionRequest> createRepeated() => $pb.PbList<RetryExecutionRequest>();
  @$core.pragma('dart2js:noInline')
  static RetryExecutionRequest getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<RetryExecutionRequest>(create);
  static RetryExecutionRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get executionId => $_getSZ(0);
  @$pb.TagNumber(1)
  set executionId($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasExecutionId() => $_has(0);
  @$pb.TagNumber(1)
  void clearExecutionId() => clearField(1);
}

class RetryExecutionResponse extends $pb.GeneratedMessage {
  factory RetryExecutionResponse({
    WorkflowExecution? execution,
  }) {
    final $result = create();
    if (execution != null) {
      $result.execution = execution;
    }
    return $result;
  }
  RetryExecutionResponse._() : super();
  factory RetryExecutionResponse.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory RetryExecutionResponse.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'RetryExecutionResponse', package: const $pb.PackageName(_omitMessageNames ? '' : 'runtime.v1'), createEmptyInstance: create)
    ..aOM<WorkflowExecution>(1, _omitFieldNames ? '' : 'execution', subBuilder: WorkflowExecution.create)
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  RetryExecutionResponse clone() => RetryExecutionResponse()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  RetryExecutionResponse copyWith(void Function(RetryExecutionResponse) updates) => super.copyWith((message) => updates(message as RetryExecutionResponse)) as RetryExecutionResponse;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RetryExecutionResponse create() => RetryExecutionResponse._();
  RetryExecutionResponse createEmptyInstance() => create();
  static $pb.PbList<RetryExecutionResponse> createRepeated() => $pb.PbList<RetryExecutionResponse>();
  @$core.pragma('dart2js:noInline')
  static RetryExecutionResponse getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<RetryExecutionResponse>(create);
  static RetryExecutionResponse? _defaultInstance;

  @$pb.TagNumber(1)
  WorkflowExecution get execution => $_getN(0);
  @$pb.TagNumber(1)
  set execution(WorkflowExecution v) { setField(1, v); }
  @$pb.TagNumber(1)
  $core.bool hasExecution() => $_has(0);
  @$pb.TagNumber(1)
  void clearExecution() => clearField(1);
  @$pb.TagNumber(1)
  WorkflowExecution ensureExecution() => $_ensure(0);
}

class ResumeExecutionRequest extends $pb.GeneratedMessage {
  factory ResumeExecutionRequest({
    $core.String? executionId,
    $6.Struct? payload,
  }) {
    final $result = create();
    if (executionId != null) {
      $result.executionId = executionId;
    }
    if (payload != null) {
      $result.payload = payload;
    }
    return $result;
  }
  ResumeExecutionRequest._() : super();
  factory ResumeExecutionRequest.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory ResumeExecutionRequest.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'ResumeExecutionRequest', package: const $pb.PackageName(_omitMessageNames ? '' : 'runtime.v1'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'executionId')
    ..aOM<$6.Struct>(2, _omitFieldNames ? '' : 'payload', subBuilder: $6.Struct.create)
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  ResumeExecutionRequest clone() => ResumeExecutionRequest()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  ResumeExecutionRequest copyWith(void Function(ResumeExecutionRequest) updates) => super.copyWith((message) => updates(message as ResumeExecutionRequest)) as ResumeExecutionRequest;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ResumeExecutionRequest create() => ResumeExecutionRequest._();
  ResumeExecutionRequest createEmptyInstance() => create();
  static $pb.PbList<ResumeExecutionRequest> createRepeated() => $pb.PbList<ResumeExecutionRequest>();
  @$core.pragma('dart2js:noInline')
  static ResumeExecutionRequest getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<ResumeExecutionRequest>(create);
  static ResumeExecutionRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get executionId => $_getSZ(0);
  @$pb.TagNumber(1)
  set executionId($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasExecutionId() => $_has(0);
  @$pb.TagNumber(1)
  void clearExecutionId() => clearField(1);

  @$pb.TagNumber(2)
  $6.Struct get payload => $_getN(1);
  @$pb.TagNumber(2)
  set payload($6.Struct v) { setField(2, v); }
  @$pb.TagNumber(2)
  $core.bool hasPayload() => $_has(1);
  @$pb.TagNumber(2)
  void clearPayload() => clearField(2);
  @$pb.TagNumber(2)
  $6.Struct ensurePayload() => $_ensure(1);
}

class ResumeExecutionResponse extends $pb.GeneratedMessage {
  factory ResumeExecutionResponse({
    WorkflowExecution? execution,
    $core.String? action,
  }) {
    final $result = create();
    if (execution != null) {
      $result.execution = execution;
    }
    if (action != null) {
      $result.action = action;
    }
    return $result;
  }
  ResumeExecutionResponse._() : super();
  factory ResumeExecutionResponse.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory ResumeExecutionResponse.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'ResumeExecutionResponse', package: const $pb.PackageName(_omitMessageNames ? '' : 'runtime.v1'), createEmptyInstance: create)
    ..aOM<WorkflowExecution>(1, _omitFieldNames ? '' : 'execution', subBuilder: WorkflowExecution.create)
    ..aOS(2, _omitFieldNames ? '' : 'action')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  ResumeExecutionResponse clone() => ResumeExecutionResponse()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  ResumeExecutionResponse copyWith(void Function(ResumeExecutionResponse) updates) => super.copyWith((message) => updates(message as ResumeExecutionResponse)) as ResumeExecutionResponse;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ResumeExecutionResponse create() => ResumeExecutionResponse._();
  ResumeExecutionResponse createEmptyInstance() => create();
  static $pb.PbList<ResumeExecutionResponse> createRepeated() => $pb.PbList<ResumeExecutionResponse>();
  @$core.pragma('dart2js:noInline')
  static ResumeExecutionResponse getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<ResumeExecutionResponse>(create);
  static ResumeExecutionResponse? _defaultInstance;

  @$pb.TagNumber(1)
  WorkflowExecution get execution => $_getN(0);
  @$pb.TagNumber(1)
  set execution(WorkflowExecution v) { setField(1, v); }
  @$pb.TagNumber(1)
  $core.bool hasExecution() => $_has(0);
  @$pb.TagNumber(1)
  void clearExecution() => clearField(1);
  @$pb.TagNumber(1)
  WorkflowExecution ensureExecution() => $_ensure(0);

  @$pb.TagNumber(2)
  $core.String get action => $_getSZ(1);
  @$pb.TagNumber(2)
  set action($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasAction() => $_has(1);
  @$pb.TagNumber(2)
  void clearAction() => clearField(2);
}

class RunTimelineEntry extends $pb.GeneratedMessage {
  factory RunTimelineEntry({
    $core.String? eventType,
    $core.String? state,
    $core.String? fromState,
    $core.String? toState,
    $core.String? executionId,
    $core.String? traceId,
    $6.Struct? payload,
    $2.Timestamp? createdAt,
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
  RunTimelineEntry._() : super();
  factory RunTimelineEntry.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory RunTimelineEntry.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'RunTimelineEntry', package: const $pb.PackageName(_omitMessageNames ? '' : 'runtime.v1'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'eventType')
    ..aOS(2, _omitFieldNames ? '' : 'state')
    ..aOS(3, _omitFieldNames ? '' : 'fromState')
    ..aOS(4, _omitFieldNames ? '' : 'toState')
    ..aOS(5, _omitFieldNames ? '' : 'executionId')
    ..aOS(6, _omitFieldNames ? '' : 'traceId')
    ..aOM<$6.Struct>(7, _omitFieldNames ? '' : 'payload', subBuilder: $6.Struct.create)
    ..aOM<$2.Timestamp>(8, _omitFieldNames ? '' : 'createdAt', subBuilder: $2.Timestamp.create)
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  RunTimelineEntry clone() => RunTimelineEntry()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  RunTimelineEntry copyWith(void Function(RunTimelineEntry) updates) => super.copyWith((message) => updates(message as RunTimelineEntry)) as RunTimelineEntry;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RunTimelineEntry create() => RunTimelineEntry._();
  RunTimelineEntry createEmptyInstance() => create();
  static $pb.PbList<RunTimelineEntry> createRepeated() => $pb.PbList<RunTimelineEntry>();
  @$core.pragma('dart2js:noInline')
  static RunTimelineEntry getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<RunTimelineEntry>(create);
  static RunTimelineEntry? _defaultInstance;

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
  $6.Struct get payload => $_getN(6);
  @$pb.TagNumber(7)
  set payload($6.Struct v) { setField(7, v); }
  @$pb.TagNumber(7)
  $core.bool hasPayload() => $_has(6);
  @$pb.TagNumber(7)
  void clearPayload() => clearField(7);
  @$pb.TagNumber(7)
  $6.Struct ensurePayload() => $_ensure(6);

  @$pb.TagNumber(8)
  $2.Timestamp get createdAt => $_getN(7);
  @$pb.TagNumber(8)
  set createdAt($2.Timestamp v) { setField(8, v); }
  @$pb.TagNumber(8)
  $core.bool hasCreatedAt() => $_has(7);
  @$pb.TagNumber(8)
  void clearCreatedAt() => clearField(8);
  @$pb.TagNumber(8)
  $2.Timestamp ensureCreatedAt() => $_ensure(7);
}

class StateOutput extends $pb.GeneratedMessage {
  factory StateOutput({
    $core.String? executionId,
    $core.String? state,
    $core.String? schemaHash,
    $6.Struct? payload,
    $2.Timestamp? createdAt,
  }) {
    final $result = create();
    if (executionId != null) {
      $result.executionId = executionId;
    }
    if (state != null) {
      $result.state = state;
    }
    if (schemaHash != null) {
      $result.schemaHash = schemaHash;
    }
    if (payload != null) {
      $result.payload = payload;
    }
    if (createdAt != null) {
      $result.createdAt = createdAt;
    }
    return $result;
  }
  StateOutput._() : super();
  factory StateOutput.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory StateOutput.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'StateOutput', package: const $pb.PackageName(_omitMessageNames ? '' : 'runtime.v1'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'executionId')
    ..aOS(2, _omitFieldNames ? '' : 'state')
    ..aOS(3, _omitFieldNames ? '' : 'schemaHash')
    ..aOM<$6.Struct>(4, _omitFieldNames ? '' : 'payload', subBuilder: $6.Struct.create)
    ..aOM<$2.Timestamp>(5, _omitFieldNames ? '' : 'createdAt', subBuilder: $2.Timestamp.create)
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  StateOutput clone() => StateOutput()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  StateOutput copyWith(void Function(StateOutput) updates) => super.copyWith((message) => updates(message as StateOutput)) as StateOutput;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static StateOutput create() => StateOutput._();
  StateOutput createEmptyInstance() => create();
  static $pb.PbList<StateOutput> createRepeated() => $pb.PbList<StateOutput>();
  @$core.pragma('dart2js:noInline')
  static StateOutput getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<StateOutput>(create);
  static StateOutput? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get executionId => $_getSZ(0);
  @$pb.TagNumber(1)
  set executionId($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasExecutionId() => $_has(0);
  @$pb.TagNumber(1)
  void clearExecutionId() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get state => $_getSZ(1);
  @$pb.TagNumber(2)
  set state($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasState() => $_has(1);
  @$pb.TagNumber(2)
  void clearState() => clearField(2);

  @$pb.TagNumber(3)
  $core.String get schemaHash => $_getSZ(2);
  @$pb.TagNumber(3)
  set schemaHash($core.String v) { $_setString(2, v); }
  @$pb.TagNumber(3)
  $core.bool hasSchemaHash() => $_has(2);
  @$pb.TagNumber(3)
  void clearSchemaHash() => clearField(3);

  @$pb.TagNumber(4)
  $6.Struct get payload => $_getN(3);
  @$pb.TagNumber(4)
  set payload($6.Struct v) { setField(4, v); }
  @$pb.TagNumber(4)
  $core.bool hasPayload() => $_has(3);
  @$pb.TagNumber(4)
  void clearPayload() => clearField(4);
  @$pb.TagNumber(4)
  $6.Struct ensurePayload() => $_ensure(3);

  @$pb.TagNumber(5)
  $2.Timestamp get createdAt => $_getN(4);
  @$pb.TagNumber(5)
  set createdAt($2.Timestamp v) { setField(5, v); }
  @$pb.TagNumber(5)
  $core.bool hasCreatedAt() => $_has(4);
  @$pb.TagNumber(5)
  void clearCreatedAt() => clearField(5);
  @$pb.TagNumber(5)
  $2.Timestamp ensureCreatedAt() => $_ensure(4);
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
    $6.Struct? itemsPayload,
    $6.Struct? resultsPayload,
    $2.Timestamp? createdAt,
    $2.Timestamp? updatedAt,
  }) {
    final $result = create();
    if (id != null) {
      $result.id = id;
    }
    if (parentExecutionId != null) {
      $result.parentExecutionId = parentExecutionId;
    }
    if (parentState != null) {
      $result.parentState = parentState;
    }
    if (scopeType != null) {
      $result.scopeType = scopeType;
    }
    if (status != null) {
      $result.status = status;
    }
    if (waitAll != null) {
      $result.waitAll = waitAll;
    }
    if (totalChildren != null) {
      $result.totalChildren = totalChildren;
    }
    if (completedChildren != null) {
      $result.completedChildren = completedChildren;
    }
    if (failedChildren != null) {
      $result.failedChildren = failedChildren;
    }
    if (nextChildIndex != null) {
      $result.nextChildIndex = nextChildIndex;
    }
    if (maxConcurrency != null) {
      $result.maxConcurrency = maxConcurrency;
    }
    if (itemVar != null) {
      $result.itemVar = itemVar;
    }
    if (indexVar != null) {
      $result.indexVar = indexVar;
    }
    if (itemsPayload != null) {
      $result.itemsPayload = itemsPayload;
    }
    if (resultsPayload != null) {
      $result.resultsPayload = resultsPayload;
    }
    if (createdAt != null) {
      $result.createdAt = createdAt;
    }
    if (updatedAt != null) {
      $result.updatedAt = updatedAt;
    }
    return $result;
  }
  ScopeRun._() : super();
  factory ScopeRun.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory ScopeRun.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'ScopeRun', package: const $pb.PackageName(_omitMessageNames ? '' : 'runtime.v1'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'id')
    ..aOS(2, _omitFieldNames ? '' : 'parentExecutionId')
    ..aOS(3, _omitFieldNames ? '' : 'parentState')
    ..aOS(4, _omitFieldNames ? '' : 'scopeType')
    ..aOS(5, _omitFieldNames ? '' : 'status')
    ..aOB(6, _omitFieldNames ? '' : 'waitAll')
    ..a<$core.int>(7, _omitFieldNames ? '' : 'totalChildren', $pb.PbFieldType.O3)
    ..a<$core.int>(8, _omitFieldNames ? '' : 'completedChildren', $pb.PbFieldType.O3)
    ..a<$core.int>(9, _omitFieldNames ? '' : 'failedChildren', $pb.PbFieldType.O3)
    ..a<$core.int>(10, _omitFieldNames ? '' : 'nextChildIndex', $pb.PbFieldType.O3)
    ..a<$core.int>(11, _omitFieldNames ? '' : 'maxConcurrency', $pb.PbFieldType.O3)
    ..aOS(12, _omitFieldNames ? '' : 'itemVar')
    ..aOS(13, _omitFieldNames ? '' : 'indexVar')
    ..aOM<$6.Struct>(14, _omitFieldNames ? '' : 'itemsPayload', subBuilder: $6.Struct.create)
    ..aOM<$6.Struct>(15, _omitFieldNames ? '' : 'resultsPayload', subBuilder: $6.Struct.create)
    ..aOM<$2.Timestamp>(16, _omitFieldNames ? '' : 'createdAt', subBuilder: $2.Timestamp.create)
    ..aOM<$2.Timestamp>(17, _omitFieldNames ? '' : 'updatedAt', subBuilder: $2.Timestamp.create)
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  ScopeRun clone() => ScopeRun()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  ScopeRun copyWith(void Function(ScopeRun) updates) => super.copyWith((message) => updates(message as ScopeRun)) as ScopeRun;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ScopeRun create() => ScopeRun._();
  ScopeRun createEmptyInstance() => create();
  static $pb.PbList<ScopeRun> createRepeated() => $pb.PbList<ScopeRun>();
  @$core.pragma('dart2js:noInline')
  static ScopeRun getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<ScopeRun>(create);
  static ScopeRun? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get id => $_getSZ(0);
  @$pb.TagNumber(1)
  set id($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get parentExecutionId => $_getSZ(1);
  @$pb.TagNumber(2)
  set parentExecutionId($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasParentExecutionId() => $_has(1);
  @$pb.TagNumber(2)
  void clearParentExecutionId() => clearField(2);

  @$pb.TagNumber(3)
  $core.String get parentState => $_getSZ(2);
  @$pb.TagNumber(3)
  set parentState($core.String v) { $_setString(2, v); }
  @$pb.TagNumber(3)
  $core.bool hasParentState() => $_has(2);
  @$pb.TagNumber(3)
  void clearParentState() => clearField(3);

  @$pb.TagNumber(4)
  $core.String get scopeType => $_getSZ(3);
  @$pb.TagNumber(4)
  set scopeType($core.String v) { $_setString(3, v); }
  @$pb.TagNumber(4)
  $core.bool hasScopeType() => $_has(3);
  @$pb.TagNumber(4)
  void clearScopeType() => clearField(4);

  @$pb.TagNumber(5)
  $core.String get status => $_getSZ(4);
  @$pb.TagNumber(5)
  set status($core.String v) { $_setString(4, v); }
  @$pb.TagNumber(5)
  $core.bool hasStatus() => $_has(4);
  @$pb.TagNumber(5)
  void clearStatus() => clearField(5);

  @$pb.TagNumber(6)
  $core.bool get waitAll => $_getBF(5);
  @$pb.TagNumber(6)
  set waitAll($core.bool v) { $_setBool(5, v); }
  @$pb.TagNumber(6)
  $core.bool hasWaitAll() => $_has(5);
  @$pb.TagNumber(6)
  void clearWaitAll() => clearField(6);

  @$pb.TagNumber(7)
  $core.int get totalChildren => $_getIZ(6);
  @$pb.TagNumber(7)
  set totalChildren($core.int v) { $_setSignedInt32(6, v); }
  @$pb.TagNumber(7)
  $core.bool hasTotalChildren() => $_has(6);
  @$pb.TagNumber(7)
  void clearTotalChildren() => clearField(7);

  @$pb.TagNumber(8)
  $core.int get completedChildren => $_getIZ(7);
  @$pb.TagNumber(8)
  set completedChildren($core.int v) { $_setSignedInt32(7, v); }
  @$pb.TagNumber(8)
  $core.bool hasCompletedChildren() => $_has(7);
  @$pb.TagNumber(8)
  void clearCompletedChildren() => clearField(8);

  @$pb.TagNumber(9)
  $core.int get failedChildren => $_getIZ(8);
  @$pb.TagNumber(9)
  set failedChildren($core.int v) { $_setSignedInt32(8, v); }
  @$pb.TagNumber(9)
  $core.bool hasFailedChildren() => $_has(8);
  @$pb.TagNumber(9)
  void clearFailedChildren() => clearField(9);

  @$pb.TagNumber(10)
  $core.int get nextChildIndex => $_getIZ(9);
  @$pb.TagNumber(10)
  set nextChildIndex($core.int v) { $_setSignedInt32(9, v); }
  @$pb.TagNumber(10)
  $core.bool hasNextChildIndex() => $_has(9);
  @$pb.TagNumber(10)
  void clearNextChildIndex() => clearField(10);

  @$pb.TagNumber(11)
  $core.int get maxConcurrency => $_getIZ(10);
  @$pb.TagNumber(11)
  set maxConcurrency($core.int v) { $_setSignedInt32(10, v); }
  @$pb.TagNumber(11)
  $core.bool hasMaxConcurrency() => $_has(10);
  @$pb.TagNumber(11)
  void clearMaxConcurrency() => clearField(11);

  @$pb.TagNumber(12)
  $core.String get itemVar => $_getSZ(11);
  @$pb.TagNumber(12)
  set itemVar($core.String v) { $_setString(11, v); }
  @$pb.TagNumber(12)
  $core.bool hasItemVar() => $_has(11);
  @$pb.TagNumber(12)
  void clearItemVar() => clearField(12);

  @$pb.TagNumber(13)
  $core.String get indexVar => $_getSZ(12);
  @$pb.TagNumber(13)
  set indexVar($core.String v) { $_setString(12, v); }
  @$pb.TagNumber(13)
  $core.bool hasIndexVar() => $_has(12);
  @$pb.TagNumber(13)
  void clearIndexVar() => clearField(13);

  @$pb.TagNumber(14)
  $6.Struct get itemsPayload => $_getN(13);
  @$pb.TagNumber(14)
  set itemsPayload($6.Struct v) { setField(14, v); }
  @$pb.TagNumber(14)
  $core.bool hasItemsPayload() => $_has(13);
  @$pb.TagNumber(14)
  void clearItemsPayload() => clearField(14);
  @$pb.TagNumber(14)
  $6.Struct ensureItemsPayload() => $_ensure(13);

  @$pb.TagNumber(15)
  $6.Struct get resultsPayload => $_getN(14);
  @$pb.TagNumber(15)
  set resultsPayload($6.Struct v) { setField(15, v); }
  @$pb.TagNumber(15)
  $core.bool hasResultsPayload() => $_has(14);
  @$pb.TagNumber(15)
  void clearResultsPayload() => clearField(15);
  @$pb.TagNumber(15)
  $6.Struct ensureResultsPayload() => $_ensure(14);

  @$pb.TagNumber(16)
  $2.Timestamp get createdAt => $_getN(15);
  @$pb.TagNumber(16)
  set createdAt($2.Timestamp v) { setField(16, v); }
  @$pb.TagNumber(16)
  $core.bool hasCreatedAt() => $_has(15);
  @$pb.TagNumber(16)
  void clearCreatedAt() => clearField(16);
  @$pb.TagNumber(16)
  $2.Timestamp ensureCreatedAt() => $_ensure(15);

  @$pb.TagNumber(17)
  $2.Timestamp get updatedAt => $_getN(16);
  @$pb.TagNumber(17)
  set updatedAt($2.Timestamp v) { setField(17, v); }
  @$pb.TagNumber(17)
  $core.bool hasUpdatedAt() => $_has(16);
  @$pb.TagNumber(17)
  void clearUpdatedAt() => clearField(17);
  @$pb.TagNumber(17)
  $2.Timestamp ensureUpdatedAt() => $_ensure(16);
}

class SignalWait extends $pb.GeneratedMessage {
  factory SignalWait({
    $core.String? id,
    $core.String? executionId,
    $core.String? state,
    $core.String? signalName,
    $core.String? outputVar,
    $core.String? status,
    $2.Timestamp? timeoutAt,
    $2.Timestamp? matchedAt,
    $2.Timestamp? timedOutAt,
    $core.String? messageId,
    $core.int? attempts,
    $2.Timestamp? createdAt,
    $2.Timestamp? updatedAt,
  }) {
    final $result = create();
    if (id != null) {
      $result.id = id;
    }
    if (executionId != null) {
      $result.executionId = executionId;
    }
    if (state != null) {
      $result.state = state;
    }
    if (signalName != null) {
      $result.signalName = signalName;
    }
    if (outputVar != null) {
      $result.outputVar = outputVar;
    }
    if (status != null) {
      $result.status = status;
    }
    if (timeoutAt != null) {
      $result.timeoutAt = timeoutAt;
    }
    if (matchedAt != null) {
      $result.matchedAt = matchedAt;
    }
    if (timedOutAt != null) {
      $result.timedOutAt = timedOutAt;
    }
    if (messageId != null) {
      $result.messageId = messageId;
    }
    if (attempts != null) {
      $result.attempts = attempts;
    }
    if (createdAt != null) {
      $result.createdAt = createdAt;
    }
    if (updatedAt != null) {
      $result.updatedAt = updatedAt;
    }
    return $result;
  }
  SignalWait._() : super();
  factory SignalWait.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory SignalWait.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'SignalWait', package: const $pb.PackageName(_omitMessageNames ? '' : 'runtime.v1'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'id')
    ..aOS(2, _omitFieldNames ? '' : 'executionId')
    ..aOS(3, _omitFieldNames ? '' : 'state')
    ..aOS(4, _omitFieldNames ? '' : 'signalName')
    ..aOS(5, _omitFieldNames ? '' : 'outputVar')
    ..aOS(6, _omitFieldNames ? '' : 'status')
    ..aOM<$2.Timestamp>(7, _omitFieldNames ? '' : 'timeoutAt', subBuilder: $2.Timestamp.create)
    ..aOM<$2.Timestamp>(8, _omitFieldNames ? '' : 'matchedAt', subBuilder: $2.Timestamp.create)
    ..aOM<$2.Timestamp>(9, _omitFieldNames ? '' : 'timedOutAt', subBuilder: $2.Timestamp.create)
    ..aOS(10, _omitFieldNames ? '' : 'messageId')
    ..a<$core.int>(11, _omitFieldNames ? '' : 'attempts', $pb.PbFieldType.O3)
    ..aOM<$2.Timestamp>(12, _omitFieldNames ? '' : 'createdAt', subBuilder: $2.Timestamp.create)
    ..aOM<$2.Timestamp>(13, _omitFieldNames ? '' : 'updatedAt', subBuilder: $2.Timestamp.create)
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  SignalWait clone() => SignalWait()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  SignalWait copyWith(void Function(SignalWait) updates) => super.copyWith((message) => updates(message as SignalWait)) as SignalWait;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SignalWait create() => SignalWait._();
  SignalWait createEmptyInstance() => create();
  static $pb.PbList<SignalWait> createRepeated() => $pb.PbList<SignalWait>();
  @$core.pragma('dart2js:noInline')
  static SignalWait getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<SignalWait>(create);
  static SignalWait? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get id => $_getSZ(0);
  @$pb.TagNumber(1)
  set id($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get executionId => $_getSZ(1);
  @$pb.TagNumber(2)
  set executionId($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasExecutionId() => $_has(1);
  @$pb.TagNumber(2)
  void clearExecutionId() => clearField(2);

  @$pb.TagNumber(3)
  $core.String get state => $_getSZ(2);
  @$pb.TagNumber(3)
  set state($core.String v) { $_setString(2, v); }
  @$pb.TagNumber(3)
  $core.bool hasState() => $_has(2);
  @$pb.TagNumber(3)
  void clearState() => clearField(3);

  @$pb.TagNumber(4)
  $core.String get signalName => $_getSZ(3);
  @$pb.TagNumber(4)
  set signalName($core.String v) { $_setString(3, v); }
  @$pb.TagNumber(4)
  $core.bool hasSignalName() => $_has(3);
  @$pb.TagNumber(4)
  void clearSignalName() => clearField(4);

  @$pb.TagNumber(5)
  $core.String get outputVar => $_getSZ(4);
  @$pb.TagNumber(5)
  set outputVar($core.String v) { $_setString(4, v); }
  @$pb.TagNumber(5)
  $core.bool hasOutputVar() => $_has(4);
  @$pb.TagNumber(5)
  void clearOutputVar() => clearField(5);

  @$pb.TagNumber(6)
  $core.String get status => $_getSZ(5);
  @$pb.TagNumber(6)
  set status($core.String v) { $_setString(5, v); }
  @$pb.TagNumber(6)
  $core.bool hasStatus() => $_has(5);
  @$pb.TagNumber(6)
  void clearStatus() => clearField(6);

  @$pb.TagNumber(7)
  $2.Timestamp get timeoutAt => $_getN(6);
  @$pb.TagNumber(7)
  set timeoutAt($2.Timestamp v) { setField(7, v); }
  @$pb.TagNumber(7)
  $core.bool hasTimeoutAt() => $_has(6);
  @$pb.TagNumber(7)
  void clearTimeoutAt() => clearField(7);
  @$pb.TagNumber(7)
  $2.Timestamp ensureTimeoutAt() => $_ensure(6);

  @$pb.TagNumber(8)
  $2.Timestamp get matchedAt => $_getN(7);
  @$pb.TagNumber(8)
  set matchedAt($2.Timestamp v) { setField(8, v); }
  @$pb.TagNumber(8)
  $core.bool hasMatchedAt() => $_has(7);
  @$pb.TagNumber(8)
  void clearMatchedAt() => clearField(8);
  @$pb.TagNumber(8)
  $2.Timestamp ensureMatchedAt() => $_ensure(7);

  @$pb.TagNumber(9)
  $2.Timestamp get timedOutAt => $_getN(8);
  @$pb.TagNumber(9)
  set timedOutAt($2.Timestamp v) { setField(9, v); }
  @$pb.TagNumber(9)
  $core.bool hasTimedOutAt() => $_has(8);
  @$pb.TagNumber(9)
  void clearTimedOutAt() => clearField(9);
  @$pb.TagNumber(9)
  $2.Timestamp ensureTimedOutAt() => $_ensure(8);

  @$pb.TagNumber(10)
  $core.String get messageId => $_getSZ(9);
  @$pb.TagNumber(10)
  set messageId($core.String v) { $_setString(9, v); }
  @$pb.TagNumber(10)
  $core.bool hasMessageId() => $_has(9);
  @$pb.TagNumber(10)
  void clearMessageId() => clearField(10);

  @$pb.TagNumber(11)
  $core.int get attempts => $_getIZ(10);
  @$pb.TagNumber(11)
  set attempts($core.int v) { $_setSignedInt32(10, v); }
  @$pb.TagNumber(11)
  $core.bool hasAttempts() => $_has(10);
  @$pb.TagNumber(11)
  void clearAttempts() => clearField(11);

  @$pb.TagNumber(12)
  $2.Timestamp get createdAt => $_getN(11);
  @$pb.TagNumber(12)
  set createdAt($2.Timestamp v) { setField(12, v); }
  @$pb.TagNumber(12)
  $core.bool hasCreatedAt() => $_has(11);
  @$pb.TagNumber(12)
  void clearCreatedAt() => clearField(12);
  @$pb.TagNumber(12)
  $2.Timestamp ensureCreatedAt() => $_ensure(11);

  @$pb.TagNumber(13)
  $2.Timestamp get updatedAt => $_getN(12);
  @$pb.TagNumber(13)
  set updatedAt($2.Timestamp v) { setField(13, v); }
  @$pb.TagNumber(13)
  $core.bool hasUpdatedAt() => $_has(12);
  @$pb.TagNumber(13)
  void clearUpdatedAt() => clearField(13);
  @$pb.TagNumber(13)
  $2.Timestamp ensureUpdatedAt() => $_ensure(12);
}

class SignalMessage extends $pb.GeneratedMessage {
  factory SignalMessage({
    $core.String? id,
    $core.String? signalName,
    $6.Struct? payload,
    $core.String? status,
    $2.Timestamp? deliveredAt,
    $core.String? waitId,
    $core.int? attempts,
    $2.Timestamp? createdAt,
    $2.Timestamp? updatedAt,
  }) {
    final $result = create();
    if (id != null) {
      $result.id = id;
    }
    if (signalName != null) {
      $result.signalName = signalName;
    }
    if (payload != null) {
      $result.payload = payload;
    }
    if (status != null) {
      $result.status = status;
    }
    if (deliveredAt != null) {
      $result.deliveredAt = deliveredAt;
    }
    if (waitId != null) {
      $result.waitId = waitId;
    }
    if (attempts != null) {
      $result.attempts = attempts;
    }
    if (createdAt != null) {
      $result.createdAt = createdAt;
    }
    if (updatedAt != null) {
      $result.updatedAt = updatedAt;
    }
    return $result;
  }
  SignalMessage._() : super();
  factory SignalMessage.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory SignalMessage.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'SignalMessage', package: const $pb.PackageName(_omitMessageNames ? '' : 'runtime.v1'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'id')
    ..aOS(2, _omitFieldNames ? '' : 'signalName')
    ..aOM<$6.Struct>(3, _omitFieldNames ? '' : 'payload', subBuilder: $6.Struct.create)
    ..aOS(4, _omitFieldNames ? '' : 'status')
    ..aOM<$2.Timestamp>(5, _omitFieldNames ? '' : 'deliveredAt', subBuilder: $2.Timestamp.create)
    ..aOS(6, _omitFieldNames ? '' : 'waitId')
    ..a<$core.int>(7, _omitFieldNames ? '' : 'attempts', $pb.PbFieldType.O3)
    ..aOM<$2.Timestamp>(8, _omitFieldNames ? '' : 'createdAt', subBuilder: $2.Timestamp.create)
    ..aOM<$2.Timestamp>(9, _omitFieldNames ? '' : 'updatedAt', subBuilder: $2.Timestamp.create)
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  SignalMessage clone() => SignalMessage()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  SignalMessage copyWith(void Function(SignalMessage) updates) => super.copyWith((message) => updates(message as SignalMessage)) as SignalMessage;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SignalMessage create() => SignalMessage._();
  SignalMessage createEmptyInstance() => create();
  static $pb.PbList<SignalMessage> createRepeated() => $pb.PbList<SignalMessage>();
  @$core.pragma('dart2js:noInline')
  static SignalMessage getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<SignalMessage>(create);
  static SignalMessage? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get id => $_getSZ(0);
  @$pb.TagNumber(1)
  set id($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get signalName => $_getSZ(1);
  @$pb.TagNumber(2)
  set signalName($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasSignalName() => $_has(1);
  @$pb.TagNumber(2)
  void clearSignalName() => clearField(2);

  @$pb.TagNumber(3)
  $6.Struct get payload => $_getN(2);
  @$pb.TagNumber(3)
  set payload($6.Struct v) { setField(3, v); }
  @$pb.TagNumber(3)
  $core.bool hasPayload() => $_has(2);
  @$pb.TagNumber(3)
  void clearPayload() => clearField(3);
  @$pb.TagNumber(3)
  $6.Struct ensurePayload() => $_ensure(2);

  @$pb.TagNumber(4)
  $core.String get status => $_getSZ(3);
  @$pb.TagNumber(4)
  set status($core.String v) { $_setString(3, v); }
  @$pb.TagNumber(4)
  $core.bool hasStatus() => $_has(3);
  @$pb.TagNumber(4)
  void clearStatus() => clearField(4);

  @$pb.TagNumber(5)
  $2.Timestamp get deliveredAt => $_getN(4);
  @$pb.TagNumber(5)
  set deliveredAt($2.Timestamp v) { setField(5, v); }
  @$pb.TagNumber(5)
  $core.bool hasDeliveredAt() => $_has(4);
  @$pb.TagNumber(5)
  void clearDeliveredAt() => clearField(5);
  @$pb.TagNumber(5)
  $2.Timestamp ensureDeliveredAt() => $_ensure(4);

  @$pb.TagNumber(6)
  $core.String get waitId => $_getSZ(5);
  @$pb.TagNumber(6)
  set waitId($core.String v) { $_setString(5, v); }
  @$pb.TagNumber(6)
  $core.bool hasWaitId() => $_has(5);
  @$pb.TagNumber(6)
  void clearWaitId() => clearField(6);

  @$pb.TagNumber(7)
  $core.int get attempts => $_getIZ(6);
  @$pb.TagNumber(7)
  set attempts($core.int v) { $_setSignedInt32(6, v); }
  @$pb.TagNumber(7)
  $core.bool hasAttempts() => $_has(6);
  @$pb.TagNumber(7)
  void clearAttempts() => clearField(7);

  @$pb.TagNumber(8)
  $2.Timestamp get createdAt => $_getN(7);
  @$pb.TagNumber(8)
  set createdAt($2.Timestamp v) { setField(8, v); }
  @$pb.TagNumber(8)
  $core.bool hasCreatedAt() => $_has(7);
  @$pb.TagNumber(8)
  void clearCreatedAt() => clearField(8);
  @$pb.TagNumber(8)
  $2.Timestamp ensureCreatedAt() => $_ensure(7);

  @$pb.TagNumber(9)
  $2.Timestamp get updatedAt => $_getN(8);
  @$pb.TagNumber(9)
  set updatedAt($2.Timestamp v) { setField(9, v); }
  @$pb.TagNumber(9)
  $core.bool hasUpdatedAt() => $_has(8);
  @$pb.TagNumber(9)
  void clearUpdatedAt() => clearField(9);
  @$pb.TagNumber(9)
  $2.Timestamp ensureUpdatedAt() => $_ensure(8);
}

class GetInstanceRunRequest extends $pb.GeneratedMessage {
  factory GetInstanceRunRequest({
    $core.String? instanceId,
    $core.bool? includePayloads,
    $core.int? executionLimit,
    $core.int? timelineLimit,
  }) {
    final $result = create();
    if (instanceId != null) {
      $result.instanceId = instanceId;
    }
    if (includePayloads != null) {
      $result.includePayloads = includePayloads;
    }
    if (executionLimit != null) {
      $result.executionLimit = executionLimit;
    }
    if (timelineLimit != null) {
      $result.timelineLimit = timelineLimit;
    }
    return $result;
  }
  GetInstanceRunRequest._() : super();
  factory GetInstanceRunRequest.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory GetInstanceRunRequest.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'GetInstanceRunRequest', package: const $pb.PackageName(_omitMessageNames ? '' : 'runtime.v1'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'instanceId')
    ..aOB(2, _omitFieldNames ? '' : 'includePayloads')
    ..a<$core.int>(3, _omitFieldNames ? '' : 'executionLimit', $pb.PbFieldType.O3)
    ..a<$core.int>(4, _omitFieldNames ? '' : 'timelineLimit', $pb.PbFieldType.O3)
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  GetInstanceRunRequest clone() => GetInstanceRunRequest()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  GetInstanceRunRequest copyWith(void Function(GetInstanceRunRequest) updates) => super.copyWith((message) => updates(message as GetInstanceRunRequest)) as GetInstanceRunRequest;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetInstanceRunRequest create() => GetInstanceRunRequest._();
  GetInstanceRunRequest createEmptyInstance() => create();
  static $pb.PbList<GetInstanceRunRequest> createRepeated() => $pb.PbList<GetInstanceRunRequest>();
  @$core.pragma('dart2js:noInline')
  static GetInstanceRunRequest getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<GetInstanceRunRequest>(create);
  static GetInstanceRunRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get instanceId => $_getSZ(0);
  @$pb.TagNumber(1)
  set instanceId($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasInstanceId() => $_has(0);
  @$pb.TagNumber(1)
  void clearInstanceId() => clearField(1);

  @$pb.TagNumber(2)
  $core.bool get includePayloads => $_getBF(1);
  @$pb.TagNumber(2)
  set includePayloads($core.bool v) { $_setBool(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasIncludePayloads() => $_has(1);
  @$pb.TagNumber(2)
  void clearIncludePayloads() => clearField(2);

  @$pb.TagNumber(3)
  $core.int get executionLimit => $_getIZ(2);
  @$pb.TagNumber(3)
  set executionLimit($core.int v) { $_setSignedInt32(2, v); }
  @$pb.TagNumber(3)
  $core.bool hasExecutionLimit() => $_has(2);
  @$pb.TagNumber(3)
  void clearExecutionLimit() => clearField(3);

  @$pb.TagNumber(4)
  $core.int get timelineLimit => $_getIZ(3);
  @$pb.TagNumber(4)
  set timelineLimit($core.int v) { $_setSignedInt32(3, v); }
  @$pb.TagNumber(4)
  $core.bool hasTimelineLimit() => $_has(3);
  @$pb.TagNumber(4)
  void clearTimelineLimit() => clearField(4);
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
    final $result = create();
    if (instance != null) {
      $result.instance = instance;
    }
    if (latestExecution != null) {
      $result.latestExecution = latestExecution;
    }
    if (traceId != null) {
      $result.traceId = traceId;
    }
    if (resumeStrategy != null) {
      $result.resumeStrategy = resumeStrategy;
    }
    if (executions != null) {
      $result.executions.addAll(executions);
    }
    if (timeline != null) {
      $result.timeline.addAll(timeline);
    }
    if (outputs != null) {
      $result.outputs.addAll(outputs);
    }
    if (scopeRuns != null) {
      $result.scopeRuns.addAll(scopeRuns);
    }
    if (signalWaits != null) {
      $result.signalWaits.addAll(signalWaits);
    }
    if (signalMessages != null) {
      $result.signalMessages.addAll(signalMessages);
    }
    return $result;
  }
  GetInstanceRunResponse._() : super();
  factory GetInstanceRunResponse.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory GetInstanceRunResponse.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'GetInstanceRunResponse', package: const $pb.PackageName(_omitMessageNames ? '' : 'runtime.v1'), createEmptyInstance: create)
    ..aOM<WorkflowInstance>(1, _omitFieldNames ? '' : 'instance', subBuilder: WorkflowInstance.create)
    ..aOM<WorkflowExecution>(2, _omitFieldNames ? '' : 'latestExecution', subBuilder: WorkflowExecution.create)
    ..aOS(3, _omitFieldNames ? '' : 'traceId')
    ..aOS(4, _omitFieldNames ? '' : 'resumeStrategy')
    ..pc<WorkflowExecution>(5, _omitFieldNames ? '' : 'executions', $pb.PbFieldType.PM, subBuilder: WorkflowExecution.create)
    ..pc<RunTimelineEntry>(6, _omitFieldNames ? '' : 'timeline', $pb.PbFieldType.PM, subBuilder: RunTimelineEntry.create)
    ..pc<StateOutput>(7, _omitFieldNames ? '' : 'outputs', $pb.PbFieldType.PM, subBuilder: StateOutput.create)
    ..pc<ScopeRun>(8, _omitFieldNames ? '' : 'scopeRuns', $pb.PbFieldType.PM, subBuilder: ScopeRun.create)
    ..pc<SignalWait>(9, _omitFieldNames ? '' : 'signalWaits', $pb.PbFieldType.PM, subBuilder: SignalWait.create)
    ..pc<SignalMessage>(10, _omitFieldNames ? '' : 'signalMessages', $pb.PbFieldType.PM, subBuilder: SignalMessage.create)
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  GetInstanceRunResponse clone() => GetInstanceRunResponse()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  GetInstanceRunResponse copyWith(void Function(GetInstanceRunResponse) updates) => super.copyWith((message) => updates(message as GetInstanceRunResponse)) as GetInstanceRunResponse;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetInstanceRunResponse create() => GetInstanceRunResponse._();
  GetInstanceRunResponse createEmptyInstance() => create();
  static $pb.PbList<GetInstanceRunResponse> createRepeated() => $pb.PbList<GetInstanceRunResponse>();
  @$core.pragma('dart2js:noInline')
  static GetInstanceRunResponse getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<GetInstanceRunResponse>(create);
  static GetInstanceRunResponse? _defaultInstance;

  @$pb.TagNumber(1)
  WorkflowInstance get instance => $_getN(0);
  @$pb.TagNumber(1)
  set instance(WorkflowInstance v) { setField(1, v); }
  @$pb.TagNumber(1)
  $core.bool hasInstance() => $_has(0);
  @$pb.TagNumber(1)
  void clearInstance() => clearField(1);
  @$pb.TagNumber(1)
  WorkflowInstance ensureInstance() => $_ensure(0);

  @$pb.TagNumber(2)
  WorkflowExecution get latestExecution => $_getN(1);
  @$pb.TagNumber(2)
  set latestExecution(WorkflowExecution v) { setField(2, v); }
  @$pb.TagNumber(2)
  $core.bool hasLatestExecution() => $_has(1);
  @$pb.TagNumber(2)
  void clearLatestExecution() => clearField(2);
  @$pb.TagNumber(2)
  WorkflowExecution ensureLatestExecution() => $_ensure(1);

  @$pb.TagNumber(3)
  $core.String get traceId => $_getSZ(2);
  @$pb.TagNumber(3)
  set traceId($core.String v) { $_setString(2, v); }
  @$pb.TagNumber(3)
  $core.bool hasTraceId() => $_has(2);
  @$pb.TagNumber(3)
  void clearTraceId() => clearField(3);

  @$pb.TagNumber(4)
  $core.String get resumeStrategy => $_getSZ(3);
  @$pb.TagNumber(4)
  set resumeStrategy($core.String v) { $_setString(3, v); }
  @$pb.TagNumber(4)
  $core.bool hasResumeStrategy() => $_has(3);
  @$pb.TagNumber(4)
  void clearResumeStrategy() => clearField(4);

  @$pb.TagNumber(5)
  $core.List<WorkflowExecution> get executions => $_getList(4);

  @$pb.TagNumber(6)
  $core.List<RunTimelineEntry> get timeline => $_getList(5);

  @$pb.TagNumber(7)
  $core.List<StateOutput> get outputs => $_getList(6);

  @$pb.TagNumber(8)
  $core.List<ScopeRun> get scopeRuns => $_getList(7);

  @$pb.TagNumber(9)
  $core.List<SignalWait> get signalWaits => $_getList(8);

  @$pb.TagNumber(10)
  $core.List<SignalMessage> get signalMessages => $_getList(9);
}

class RuntimeServiceApi {
  $pb.RpcClient _client;
  RuntimeServiceApi(this._client);

  $async.Future<ListInstancesResponse> listInstances($pb.ClientContext? ctx, ListInstancesRequest request) =>
    _client.invoke<ListInstancesResponse>(ctx, 'RuntimeService', 'ListInstances', request, ListInstancesResponse())
  ;
  $async.Future<RetryInstanceResponse> retryInstance($pb.ClientContext? ctx, RetryInstanceRequest request) =>
    _client.invoke<RetryInstanceResponse>(ctx, 'RuntimeService', 'RetryInstance', request, RetryInstanceResponse())
  ;
  $async.Future<ListExecutionsResponse> listExecutions($pb.ClientContext? ctx, ListExecutionsRequest request) =>
    _client.invoke<ListExecutionsResponse>(ctx, 'RuntimeService', 'ListExecutions', request, ListExecutionsResponse())
  ;
  $async.Future<GetExecutionResponse> getExecution($pb.ClientContext? ctx, GetExecutionRequest request) =>
    _client.invoke<GetExecutionResponse>(ctx, 'RuntimeService', 'GetExecution', request, GetExecutionResponse())
  ;
  $async.Future<RetryExecutionResponse> retryExecution($pb.ClientContext? ctx, RetryExecutionRequest request) =>
    _client.invoke<RetryExecutionResponse>(ctx, 'RuntimeService', 'RetryExecution', request, RetryExecutionResponse())
  ;
  $async.Future<ResumeExecutionResponse> resumeExecution($pb.ClientContext? ctx, ResumeExecutionRequest request) =>
    _client.invoke<ResumeExecutionResponse>(ctx, 'RuntimeService', 'ResumeExecution', request, ResumeExecutionResponse())
  ;
  $async.Future<GetInstanceRunResponse> getInstanceRun($pb.ClientContext? ctx, GetInstanceRunRequest request) =>
    _client.invoke<GetInstanceRunResponse>(ctx, 'RuntimeService', 'GetInstanceRun', request, GetInstanceRunResponse())
  ;
}


const _omitFieldNames = $core.bool.fromEnvironment('protobuf.omit_field_names');
const _omitMessageNames = $core.bool.fromEnvironment('protobuf.omit_message_names');
