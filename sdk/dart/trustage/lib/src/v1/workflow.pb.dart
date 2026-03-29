//
//  Generated code. Do not modify.
//  source: v1/workflow.proto
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

import '../common/v1/common.pb.dart' as $8;
import '../google/protobuf/struct.pb.dart' as $2;
import '../google/protobuf/timestamp.pb.dart' as $3;
import 'workflow.pbenum.dart';

export 'workflow.pbenum.dart';

class WorkflowDefinition extends $pb.GeneratedMessage {
  factory WorkflowDefinition({
    $core.String? id,
    $core.String? name,
    $core.int? version,
    WorkflowStatus? status,
    $2.Struct? dsl,
    $core.String? inputSchemaHash,
    $fixnum.Int64? timeoutSeconds,
    $3.Timestamp? createdAt,
    $3.Timestamp? updatedAt,
  }) {
    final $result = create();
    if (id != null) {
      $result.id = id;
    }
    if (name != null) {
      $result.name = name;
    }
    if (version != null) {
      $result.version = version;
    }
    if (status != null) {
      $result.status = status;
    }
    if (dsl != null) {
      $result.dsl = dsl;
    }
    if (inputSchemaHash != null) {
      $result.inputSchemaHash = inputSchemaHash;
    }
    if (timeoutSeconds != null) {
      $result.timeoutSeconds = timeoutSeconds;
    }
    if (createdAt != null) {
      $result.createdAt = createdAt;
    }
    if (updatedAt != null) {
      $result.updatedAt = updatedAt;
    }
    return $result;
  }
  WorkflowDefinition._() : super();
  factory WorkflowDefinition.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory WorkflowDefinition.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'WorkflowDefinition', package: const $pb.PackageName(_omitMessageNames ? '' : 'workflow.v1'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'id')
    ..aOS(2, _omitFieldNames ? '' : 'name')
    ..a<$core.int>(3, _omitFieldNames ? '' : 'version', $pb.PbFieldType.O3)
    ..e<WorkflowStatus>(4, _omitFieldNames ? '' : 'status', $pb.PbFieldType.OE, defaultOrMaker: WorkflowStatus.WORKFLOW_STATUS_UNSPECIFIED, valueOf: WorkflowStatus.valueOf, enumValues: WorkflowStatus.values)
    ..aOM<$2.Struct>(5, _omitFieldNames ? '' : 'dsl', subBuilder: $2.Struct.create)
    ..aOS(6, _omitFieldNames ? '' : 'inputSchemaHash')
    ..aInt64(7, _omitFieldNames ? '' : 'timeoutSeconds')
    ..aOM<$3.Timestamp>(8, _omitFieldNames ? '' : 'createdAt', subBuilder: $3.Timestamp.create)
    ..aOM<$3.Timestamp>(9, _omitFieldNames ? '' : 'updatedAt', subBuilder: $3.Timestamp.create)
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  WorkflowDefinition clone() => WorkflowDefinition()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  WorkflowDefinition copyWith(void Function(WorkflowDefinition) updates) => super.copyWith((message) => updates(message as WorkflowDefinition)) as WorkflowDefinition;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static WorkflowDefinition create() => WorkflowDefinition._();
  WorkflowDefinition createEmptyInstance() => create();
  static $pb.PbList<WorkflowDefinition> createRepeated() => $pb.PbList<WorkflowDefinition>();
  @$core.pragma('dart2js:noInline')
  static WorkflowDefinition getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<WorkflowDefinition>(create);
  static WorkflowDefinition? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get id => $_getSZ(0);
  @$pb.TagNumber(1)
  set id($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get name => $_getSZ(1);
  @$pb.TagNumber(2)
  set name($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasName() => $_has(1);
  @$pb.TagNumber(2)
  void clearName() => clearField(2);

  @$pb.TagNumber(3)
  $core.int get version => $_getIZ(2);
  @$pb.TagNumber(3)
  set version($core.int v) { $_setSignedInt32(2, v); }
  @$pb.TagNumber(3)
  $core.bool hasVersion() => $_has(2);
  @$pb.TagNumber(3)
  void clearVersion() => clearField(3);

  @$pb.TagNumber(4)
  WorkflowStatus get status => $_getN(3);
  @$pb.TagNumber(4)
  set status(WorkflowStatus v) { setField(4, v); }
  @$pb.TagNumber(4)
  $core.bool hasStatus() => $_has(3);
  @$pb.TagNumber(4)
  void clearStatus() => clearField(4);

  @$pb.TagNumber(5)
  $2.Struct get dsl => $_getN(4);
  @$pb.TagNumber(5)
  set dsl($2.Struct v) { setField(5, v); }
  @$pb.TagNumber(5)
  $core.bool hasDsl() => $_has(4);
  @$pb.TagNumber(5)
  void clearDsl() => clearField(5);
  @$pb.TagNumber(5)
  $2.Struct ensureDsl() => $_ensure(4);

  @$pb.TagNumber(6)
  $core.String get inputSchemaHash => $_getSZ(5);
  @$pb.TagNumber(6)
  set inputSchemaHash($core.String v) { $_setString(5, v); }
  @$pb.TagNumber(6)
  $core.bool hasInputSchemaHash() => $_has(5);
  @$pb.TagNumber(6)
  void clearInputSchemaHash() => clearField(6);

  @$pb.TagNumber(7)
  $fixnum.Int64 get timeoutSeconds => $_getI64(6);
  @$pb.TagNumber(7)
  set timeoutSeconds($fixnum.Int64 v) { $_setInt64(6, v); }
  @$pb.TagNumber(7)
  $core.bool hasTimeoutSeconds() => $_has(6);
  @$pb.TagNumber(7)
  void clearTimeoutSeconds() => clearField(7);

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

  @$pb.TagNumber(9)
  $3.Timestamp get updatedAt => $_getN(8);
  @$pb.TagNumber(9)
  set updatedAt($3.Timestamp v) { setField(9, v); }
  @$pb.TagNumber(9)
  $core.bool hasUpdatedAt() => $_has(8);
  @$pb.TagNumber(9)
  void clearUpdatedAt() => clearField(9);
  @$pb.TagNumber(9)
  $3.Timestamp ensureUpdatedAt() => $_ensure(8);
}

class CreateWorkflowRequest extends $pb.GeneratedMessage {
  factory CreateWorkflowRequest({
    $2.Struct? dsl,
  }) {
    final $result = create();
    if (dsl != null) {
      $result.dsl = dsl;
    }
    return $result;
  }
  CreateWorkflowRequest._() : super();
  factory CreateWorkflowRequest.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory CreateWorkflowRequest.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'CreateWorkflowRequest', package: const $pb.PackageName(_omitMessageNames ? '' : 'workflow.v1'), createEmptyInstance: create)
    ..aOM<$2.Struct>(1, _omitFieldNames ? '' : 'dsl', subBuilder: $2.Struct.create)
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  CreateWorkflowRequest clone() => CreateWorkflowRequest()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  CreateWorkflowRequest copyWith(void Function(CreateWorkflowRequest) updates) => super.copyWith((message) => updates(message as CreateWorkflowRequest)) as CreateWorkflowRequest;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CreateWorkflowRequest create() => CreateWorkflowRequest._();
  CreateWorkflowRequest createEmptyInstance() => create();
  static $pb.PbList<CreateWorkflowRequest> createRepeated() => $pb.PbList<CreateWorkflowRequest>();
  @$core.pragma('dart2js:noInline')
  static CreateWorkflowRequest getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<CreateWorkflowRequest>(create);
  static CreateWorkflowRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $2.Struct get dsl => $_getN(0);
  @$pb.TagNumber(1)
  set dsl($2.Struct v) { setField(1, v); }
  @$pb.TagNumber(1)
  $core.bool hasDsl() => $_has(0);
  @$pb.TagNumber(1)
  void clearDsl() => clearField(1);
  @$pb.TagNumber(1)
  $2.Struct ensureDsl() => $_ensure(0);
}

class CreateWorkflowResponse extends $pb.GeneratedMessage {
  factory CreateWorkflowResponse({
    WorkflowDefinition? workflow,
  }) {
    final $result = create();
    if (workflow != null) {
      $result.workflow = workflow;
    }
    return $result;
  }
  CreateWorkflowResponse._() : super();
  factory CreateWorkflowResponse.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory CreateWorkflowResponse.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'CreateWorkflowResponse', package: const $pb.PackageName(_omitMessageNames ? '' : 'workflow.v1'), createEmptyInstance: create)
    ..aOM<WorkflowDefinition>(1, _omitFieldNames ? '' : 'workflow', subBuilder: WorkflowDefinition.create)
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  CreateWorkflowResponse clone() => CreateWorkflowResponse()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  CreateWorkflowResponse copyWith(void Function(CreateWorkflowResponse) updates) => super.copyWith((message) => updates(message as CreateWorkflowResponse)) as CreateWorkflowResponse;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CreateWorkflowResponse create() => CreateWorkflowResponse._();
  CreateWorkflowResponse createEmptyInstance() => create();
  static $pb.PbList<CreateWorkflowResponse> createRepeated() => $pb.PbList<CreateWorkflowResponse>();
  @$core.pragma('dart2js:noInline')
  static CreateWorkflowResponse getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<CreateWorkflowResponse>(create);
  static CreateWorkflowResponse? _defaultInstance;

  @$pb.TagNumber(1)
  WorkflowDefinition get workflow => $_getN(0);
  @$pb.TagNumber(1)
  set workflow(WorkflowDefinition v) { setField(1, v); }
  @$pb.TagNumber(1)
  $core.bool hasWorkflow() => $_has(0);
  @$pb.TagNumber(1)
  void clearWorkflow() => clearField(1);
  @$pb.TagNumber(1)
  WorkflowDefinition ensureWorkflow() => $_ensure(0);
}

class GetWorkflowRequest extends $pb.GeneratedMessage {
  factory GetWorkflowRequest({
    $core.String? id,
  }) {
    final $result = create();
    if (id != null) {
      $result.id = id;
    }
    return $result;
  }
  GetWorkflowRequest._() : super();
  factory GetWorkflowRequest.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory GetWorkflowRequest.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'GetWorkflowRequest', package: const $pb.PackageName(_omitMessageNames ? '' : 'workflow.v1'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'id')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  GetWorkflowRequest clone() => GetWorkflowRequest()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  GetWorkflowRequest copyWith(void Function(GetWorkflowRequest) updates) => super.copyWith((message) => updates(message as GetWorkflowRequest)) as GetWorkflowRequest;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetWorkflowRequest create() => GetWorkflowRequest._();
  GetWorkflowRequest createEmptyInstance() => create();
  static $pb.PbList<GetWorkflowRequest> createRepeated() => $pb.PbList<GetWorkflowRequest>();
  @$core.pragma('dart2js:noInline')
  static GetWorkflowRequest getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<GetWorkflowRequest>(create);
  static GetWorkflowRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get id => $_getSZ(0);
  @$pb.TagNumber(1)
  set id($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => clearField(1);
}

class GetWorkflowResponse extends $pb.GeneratedMessage {
  factory GetWorkflowResponse({
    WorkflowDefinition? workflow,
  }) {
    final $result = create();
    if (workflow != null) {
      $result.workflow = workflow;
    }
    return $result;
  }
  GetWorkflowResponse._() : super();
  factory GetWorkflowResponse.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory GetWorkflowResponse.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'GetWorkflowResponse', package: const $pb.PackageName(_omitMessageNames ? '' : 'workflow.v1'), createEmptyInstance: create)
    ..aOM<WorkflowDefinition>(1, _omitFieldNames ? '' : 'workflow', subBuilder: WorkflowDefinition.create)
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  GetWorkflowResponse clone() => GetWorkflowResponse()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  GetWorkflowResponse copyWith(void Function(GetWorkflowResponse) updates) => super.copyWith((message) => updates(message as GetWorkflowResponse)) as GetWorkflowResponse;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GetWorkflowResponse create() => GetWorkflowResponse._();
  GetWorkflowResponse createEmptyInstance() => create();
  static $pb.PbList<GetWorkflowResponse> createRepeated() => $pb.PbList<GetWorkflowResponse>();
  @$core.pragma('dart2js:noInline')
  static GetWorkflowResponse getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<GetWorkflowResponse>(create);
  static GetWorkflowResponse? _defaultInstance;

  @$pb.TagNumber(1)
  WorkflowDefinition get workflow => $_getN(0);
  @$pb.TagNumber(1)
  set workflow(WorkflowDefinition v) { setField(1, v); }
  @$pb.TagNumber(1)
  $core.bool hasWorkflow() => $_has(0);
  @$pb.TagNumber(1)
  void clearWorkflow() => clearField(1);
  @$pb.TagNumber(1)
  WorkflowDefinition ensureWorkflow() => $_ensure(0);
}

class ListWorkflowsRequest extends $pb.GeneratedMessage {
  factory ListWorkflowsRequest({
    $core.String? name,
    WorkflowStatus? status,
    $8.SearchRequest? search,
  }) {
    final $result = create();
    if (name != null) {
      $result.name = name;
    }
    if (status != null) {
      $result.status = status;
    }
    if (search != null) {
      $result.search = search;
    }
    return $result;
  }
  ListWorkflowsRequest._() : super();
  factory ListWorkflowsRequest.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory ListWorkflowsRequest.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'ListWorkflowsRequest', package: const $pb.PackageName(_omitMessageNames ? '' : 'workflow.v1'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'name')
    ..e<WorkflowStatus>(2, _omitFieldNames ? '' : 'status', $pb.PbFieldType.OE, defaultOrMaker: WorkflowStatus.WORKFLOW_STATUS_UNSPECIFIED, valueOf: WorkflowStatus.valueOf, enumValues: WorkflowStatus.values)
    ..aOM<$8.SearchRequest>(3, _omitFieldNames ? '' : 'search', subBuilder: $8.SearchRequest.create)
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  ListWorkflowsRequest clone() => ListWorkflowsRequest()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  ListWorkflowsRequest copyWith(void Function(ListWorkflowsRequest) updates) => super.copyWith((message) => updates(message as ListWorkflowsRequest)) as ListWorkflowsRequest;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ListWorkflowsRequest create() => ListWorkflowsRequest._();
  ListWorkflowsRequest createEmptyInstance() => create();
  static $pb.PbList<ListWorkflowsRequest> createRepeated() => $pb.PbList<ListWorkflowsRequest>();
  @$core.pragma('dart2js:noInline')
  static ListWorkflowsRequest getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<ListWorkflowsRequest>(create);
  static ListWorkflowsRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get name => $_getSZ(0);
  @$pb.TagNumber(1)
  set name($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasName() => $_has(0);
  @$pb.TagNumber(1)
  void clearName() => clearField(1);

  @$pb.TagNumber(2)
  WorkflowStatus get status => $_getN(1);
  @$pb.TagNumber(2)
  set status(WorkflowStatus v) { setField(2, v); }
  @$pb.TagNumber(2)
  $core.bool hasStatus() => $_has(1);
  @$pb.TagNumber(2)
  void clearStatus() => clearField(2);

  @$pb.TagNumber(3)
  $8.SearchRequest get search => $_getN(2);
  @$pb.TagNumber(3)
  set search($8.SearchRequest v) { setField(3, v); }
  @$pb.TagNumber(3)
  $core.bool hasSearch() => $_has(2);
  @$pb.TagNumber(3)
  void clearSearch() => clearField(3);
  @$pb.TagNumber(3)
  $8.SearchRequest ensureSearch() => $_ensure(2);
}

class ListWorkflowsResponse extends $pb.GeneratedMessage {
  factory ListWorkflowsResponse({
    $core.Iterable<WorkflowDefinition>? items,
    $8.PageCursor? nextCursor,
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
  ListWorkflowsResponse._() : super();
  factory ListWorkflowsResponse.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory ListWorkflowsResponse.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'ListWorkflowsResponse', package: const $pb.PackageName(_omitMessageNames ? '' : 'workflow.v1'), createEmptyInstance: create)
    ..pc<WorkflowDefinition>(1, _omitFieldNames ? '' : 'items', $pb.PbFieldType.PM, subBuilder: WorkflowDefinition.create)
    ..aOM<$8.PageCursor>(2, _omitFieldNames ? '' : 'nextCursor', subBuilder: $8.PageCursor.create)
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  ListWorkflowsResponse clone() => ListWorkflowsResponse()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  ListWorkflowsResponse copyWith(void Function(ListWorkflowsResponse) updates) => super.copyWith((message) => updates(message as ListWorkflowsResponse)) as ListWorkflowsResponse;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ListWorkflowsResponse create() => ListWorkflowsResponse._();
  ListWorkflowsResponse createEmptyInstance() => create();
  static $pb.PbList<ListWorkflowsResponse> createRepeated() => $pb.PbList<ListWorkflowsResponse>();
  @$core.pragma('dart2js:noInline')
  static ListWorkflowsResponse getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<ListWorkflowsResponse>(create);
  static ListWorkflowsResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.List<WorkflowDefinition> get items => $_getList(0);

  @$pb.TagNumber(2)
  $8.PageCursor get nextCursor => $_getN(1);
  @$pb.TagNumber(2)
  set nextCursor($8.PageCursor v) { setField(2, v); }
  @$pb.TagNumber(2)
  $core.bool hasNextCursor() => $_has(1);
  @$pb.TagNumber(2)
  void clearNextCursor() => clearField(2);
  @$pb.TagNumber(2)
  $8.PageCursor ensureNextCursor() => $_ensure(1);
}

class ActivateWorkflowRequest extends $pb.GeneratedMessage {
  factory ActivateWorkflowRequest({
    $core.String? id,
  }) {
    final $result = create();
    if (id != null) {
      $result.id = id;
    }
    return $result;
  }
  ActivateWorkflowRequest._() : super();
  factory ActivateWorkflowRequest.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory ActivateWorkflowRequest.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'ActivateWorkflowRequest', package: const $pb.PackageName(_omitMessageNames ? '' : 'workflow.v1'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'id')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  ActivateWorkflowRequest clone() => ActivateWorkflowRequest()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  ActivateWorkflowRequest copyWith(void Function(ActivateWorkflowRequest) updates) => super.copyWith((message) => updates(message as ActivateWorkflowRequest)) as ActivateWorkflowRequest;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ActivateWorkflowRequest create() => ActivateWorkflowRequest._();
  ActivateWorkflowRequest createEmptyInstance() => create();
  static $pb.PbList<ActivateWorkflowRequest> createRepeated() => $pb.PbList<ActivateWorkflowRequest>();
  @$core.pragma('dart2js:noInline')
  static ActivateWorkflowRequest getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<ActivateWorkflowRequest>(create);
  static ActivateWorkflowRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get id => $_getSZ(0);
  @$pb.TagNumber(1)
  set id($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => clearField(1);
}

class ActivateWorkflowResponse extends $pb.GeneratedMessage {
  factory ActivateWorkflowResponse({
    WorkflowDefinition? workflow,
  }) {
    final $result = create();
    if (workflow != null) {
      $result.workflow = workflow;
    }
    return $result;
  }
  ActivateWorkflowResponse._() : super();
  factory ActivateWorkflowResponse.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory ActivateWorkflowResponse.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'ActivateWorkflowResponse', package: const $pb.PackageName(_omitMessageNames ? '' : 'workflow.v1'), createEmptyInstance: create)
    ..aOM<WorkflowDefinition>(1, _omitFieldNames ? '' : 'workflow', subBuilder: WorkflowDefinition.create)
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  ActivateWorkflowResponse clone() => ActivateWorkflowResponse()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  ActivateWorkflowResponse copyWith(void Function(ActivateWorkflowResponse) updates) => super.copyWith((message) => updates(message as ActivateWorkflowResponse)) as ActivateWorkflowResponse;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ActivateWorkflowResponse create() => ActivateWorkflowResponse._();
  ActivateWorkflowResponse createEmptyInstance() => create();
  static $pb.PbList<ActivateWorkflowResponse> createRepeated() => $pb.PbList<ActivateWorkflowResponse>();
  @$core.pragma('dart2js:noInline')
  static ActivateWorkflowResponse getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<ActivateWorkflowResponse>(create);
  static ActivateWorkflowResponse? _defaultInstance;

  @$pb.TagNumber(1)
  WorkflowDefinition get workflow => $_getN(0);
  @$pb.TagNumber(1)
  set workflow(WorkflowDefinition v) { setField(1, v); }
  @$pb.TagNumber(1)
  $core.bool hasWorkflow() => $_has(0);
  @$pb.TagNumber(1)
  void clearWorkflow() => clearField(1);
  @$pb.TagNumber(1)
  WorkflowDefinition ensureWorkflow() => $_ensure(0);
}

class WorkflowServiceApi {
  $pb.RpcClient _client;
  WorkflowServiceApi(this._client);

  $async.Future<CreateWorkflowResponse> createWorkflow($pb.ClientContext? ctx, CreateWorkflowRequest request) =>
    _client.invoke<CreateWorkflowResponse>(ctx, 'WorkflowService', 'CreateWorkflow', request, CreateWorkflowResponse())
  ;
  $async.Future<GetWorkflowResponse> getWorkflow($pb.ClientContext? ctx, GetWorkflowRequest request) =>
    _client.invoke<GetWorkflowResponse>(ctx, 'WorkflowService', 'GetWorkflow', request, GetWorkflowResponse())
  ;
  $async.Future<ListWorkflowsResponse> listWorkflows($pb.ClientContext? ctx, ListWorkflowsRequest request) =>
    _client.invoke<ListWorkflowsResponse>(ctx, 'WorkflowService', 'ListWorkflows', request, ListWorkflowsResponse())
  ;
  $async.Future<ActivateWorkflowResponse> activateWorkflow($pb.ClientContext? ctx, ActivateWorkflowRequest request) =>
    _client.invoke<ActivateWorkflowResponse>(ctx, 'WorkflowService', 'ActivateWorkflow', request, ActivateWorkflowResponse())
  ;
}


const _omitFieldNames = $core.bool.fromEnvironment('protobuf.omit_field_names');
const _omitMessageNames = $core.bool.fromEnvironment('protobuf.omit_message_names');
