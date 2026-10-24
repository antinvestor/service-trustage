// This is a generated file - do not edit.
//
// Generated from v1/workflow.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

class WorkflowStatus extends $pb.ProtobufEnum {
  static const WorkflowStatus WORKFLOW_STATUS_UNSPECIFIED =
      WorkflowStatus._(0, _omitEnumNames ? '' : 'WORKFLOW_STATUS_UNSPECIFIED');
  static const WorkflowStatus WORKFLOW_STATUS_DRAFT =
      WorkflowStatus._(1, _omitEnumNames ? '' : 'WORKFLOW_STATUS_DRAFT');
  static const WorkflowStatus WORKFLOW_STATUS_ACTIVE =
      WorkflowStatus._(2, _omitEnumNames ? '' : 'WORKFLOW_STATUS_ACTIVE');
  static const WorkflowStatus WORKFLOW_STATUS_ARCHIVED =
      WorkflowStatus._(3, _omitEnumNames ? '' : 'WORKFLOW_STATUS_ARCHIVED');

  static const $core.List<WorkflowStatus> values = <WorkflowStatus>[
    WORKFLOW_STATUS_UNSPECIFIED,
    WORKFLOW_STATUS_DRAFT,
    WORKFLOW_STATUS_ACTIVE,
    WORKFLOW_STATUS_ARCHIVED,
  ];

  static final $core.List<WorkflowStatus?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 3);
  static WorkflowStatus? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const WorkflowStatus._(super.value, super.name);
}

const $core.bool _omitEnumNames =
    $core.bool.fromEnvironment('protobuf.omit_enum_names');
