//
//  Generated code. Do not modify.
//  source: v1/runtime.proto
//
// @dart = 2.12

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_final_fields
// ignore_for_file: unnecessary_import, unnecessary_this, unused_import

import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

class InstanceStatus extends $pb.ProtobufEnum {
  static const InstanceStatus INSTANCE_STATUS_UNSPECIFIED = InstanceStatus._(0, _omitEnumNames ? '' : 'INSTANCE_STATUS_UNSPECIFIED');
  static const InstanceStatus INSTANCE_STATUS_RUNNING = InstanceStatus._(1, _omitEnumNames ? '' : 'INSTANCE_STATUS_RUNNING');
  static const InstanceStatus INSTANCE_STATUS_COMPLETED = InstanceStatus._(2, _omitEnumNames ? '' : 'INSTANCE_STATUS_COMPLETED');
  static const InstanceStatus INSTANCE_STATUS_FAILED = InstanceStatus._(3, _omitEnumNames ? '' : 'INSTANCE_STATUS_FAILED');
  static const InstanceStatus INSTANCE_STATUS_CANCELLED = InstanceStatus._(4, _omitEnumNames ? '' : 'INSTANCE_STATUS_CANCELLED');
  static const InstanceStatus INSTANCE_STATUS_SUSPENDED = InstanceStatus._(5, _omitEnumNames ? '' : 'INSTANCE_STATUS_SUSPENDED');

  static const $core.List<InstanceStatus> values = <InstanceStatus> [
    INSTANCE_STATUS_UNSPECIFIED,
    INSTANCE_STATUS_RUNNING,
    INSTANCE_STATUS_COMPLETED,
    INSTANCE_STATUS_FAILED,
    INSTANCE_STATUS_CANCELLED,
    INSTANCE_STATUS_SUSPENDED,
  ];

  static final $core.Map<$core.int, InstanceStatus> _byValue = $pb.ProtobufEnum.initByValue(values);
  static InstanceStatus? valueOf($core.int value) => _byValue[value];

  const InstanceStatus._($core.int v, $core.String n) : super(v, n);
}

class ExecutionStatus extends $pb.ProtobufEnum {
  static const ExecutionStatus EXECUTION_STATUS_UNSPECIFIED = ExecutionStatus._(0, _omitEnumNames ? '' : 'EXECUTION_STATUS_UNSPECIFIED');
  static const ExecutionStatus EXECUTION_STATUS_PENDING = ExecutionStatus._(1, _omitEnumNames ? '' : 'EXECUTION_STATUS_PENDING');
  static const ExecutionStatus EXECUTION_STATUS_DISPATCHED = ExecutionStatus._(2, _omitEnumNames ? '' : 'EXECUTION_STATUS_DISPATCHED');
  static const ExecutionStatus EXECUTION_STATUS_RUNNING = ExecutionStatus._(3, _omitEnumNames ? '' : 'EXECUTION_STATUS_RUNNING');
  static const ExecutionStatus EXECUTION_STATUS_COMPLETED = ExecutionStatus._(4, _omitEnumNames ? '' : 'EXECUTION_STATUS_COMPLETED');
  static const ExecutionStatus EXECUTION_STATUS_FAILED = ExecutionStatus._(5, _omitEnumNames ? '' : 'EXECUTION_STATUS_FAILED');
  static const ExecutionStatus EXECUTION_STATUS_FATAL = ExecutionStatus._(6, _omitEnumNames ? '' : 'EXECUTION_STATUS_FATAL');
  static const ExecutionStatus EXECUTION_STATUS_TIMED_OUT = ExecutionStatus._(7, _omitEnumNames ? '' : 'EXECUTION_STATUS_TIMED_OUT');
  static const ExecutionStatus EXECUTION_STATUS_INVALID_INPUT_CONTRACT = ExecutionStatus._(8, _omitEnumNames ? '' : 'EXECUTION_STATUS_INVALID_INPUT_CONTRACT');
  static const ExecutionStatus EXECUTION_STATUS_INVALID_OUTPUT_CONTRACT = ExecutionStatus._(9, _omitEnumNames ? '' : 'EXECUTION_STATUS_INVALID_OUTPUT_CONTRACT');
  static const ExecutionStatus EXECUTION_STATUS_STALE = ExecutionStatus._(10, _omitEnumNames ? '' : 'EXECUTION_STATUS_STALE');
  static const ExecutionStatus EXECUTION_STATUS_RETRY_SCHEDULED = ExecutionStatus._(11, _omitEnumNames ? '' : 'EXECUTION_STATUS_RETRY_SCHEDULED');
  static const ExecutionStatus EXECUTION_STATUS_WAITING = ExecutionStatus._(12, _omitEnumNames ? '' : 'EXECUTION_STATUS_WAITING');

  static const $core.List<ExecutionStatus> values = <ExecutionStatus> [
    EXECUTION_STATUS_UNSPECIFIED,
    EXECUTION_STATUS_PENDING,
    EXECUTION_STATUS_DISPATCHED,
    EXECUTION_STATUS_RUNNING,
    EXECUTION_STATUS_COMPLETED,
    EXECUTION_STATUS_FAILED,
    EXECUTION_STATUS_FATAL,
    EXECUTION_STATUS_TIMED_OUT,
    EXECUTION_STATUS_INVALID_INPUT_CONTRACT,
    EXECUTION_STATUS_INVALID_OUTPUT_CONTRACT,
    EXECUTION_STATUS_STALE,
    EXECUTION_STATUS_RETRY_SCHEDULED,
    EXECUTION_STATUS_WAITING,
  ];

  static final $core.Map<$core.int, ExecutionStatus> _byValue = $pb.ProtobufEnum.initByValue(values);
  static ExecutionStatus? valueOf($core.int value) => _byValue[value];

  const ExecutionStatus._($core.int v, $core.String n) : super(v, n);
}


const _omitEnumNames = $core.bool.fromEnvironment('protobuf.omit_enum_names');
