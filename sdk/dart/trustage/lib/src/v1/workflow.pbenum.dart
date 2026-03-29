//
//  Generated code. Do not modify.
//  source: v1/workflow.proto
//
// @dart = 2.12

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_final_fields
// ignore_for_file: unnecessary_import, unnecessary_this, unused_import

import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

class WorkflowStatus extends $pb.ProtobufEnum {
  static const WorkflowStatus WORKFLOW_STATUS_UNSPECIFIED = WorkflowStatus._(0, _omitEnumNames ? '' : 'WORKFLOW_STATUS_UNSPECIFIED');
  static const WorkflowStatus WORKFLOW_STATUS_DRAFT = WorkflowStatus._(1, _omitEnumNames ? '' : 'WORKFLOW_STATUS_DRAFT');
  static const WorkflowStatus WORKFLOW_STATUS_ACTIVE = WorkflowStatus._(2, _omitEnumNames ? '' : 'WORKFLOW_STATUS_ACTIVE');
  static const WorkflowStatus WORKFLOW_STATUS_ARCHIVED = WorkflowStatus._(3, _omitEnumNames ? '' : 'WORKFLOW_STATUS_ARCHIVED');

  static const $core.List<WorkflowStatus> values = <WorkflowStatus> [
    WORKFLOW_STATUS_UNSPECIFIED,
    WORKFLOW_STATUS_DRAFT,
    WORKFLOW_STATUS_ACTIVE,
    WORKFLOW_STATUS_ARCHIVED,
  ];

  static final $core.Map<$core.int, WorkflowStatus> _byValue = $pb.ProtobufEnum.initByValue(values);
  static WorkflowStatus? valueOf($core.int value) => _byValue[value];

  const WorkflowStatus._($core.int v, $core.String n) : super(v, n);
}


const _omitEnumNames = $core.bool.fromEnvironment('protobuf.omit_enum_names');
