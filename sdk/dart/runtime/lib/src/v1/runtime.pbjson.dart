//
//  Generated code. Do not modify.
//  source: v1/runtime.proto
//
// @dart = 2.12

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_final_fields
// ignore_for_file: unnecessary_import, unnecessary_this, unused_import

import 'dart:convert' as $convert;
import 'dart:core' as $core;
import 'dart:typed_data' as $typed_data;

import '../common/v1/common.pbjson.dart' as $7;
import '../google/protobuf/struct.pbjson.dart' as $6;
import '../google/protobuf/timestamp.pbjson.dart' as $2;

@$core.Deprecated('Use instanceStatusDescriptor instead')
const InstanceStatus$json = {
  '1': 'InstanceStatus',
  '2': [
    {'1': 'INSTANCE_STATUS_UNSPECIFIED', '2': 0},
    {'1': 'INSTANCE_STATUS_RUNNING', '2': 1},
    {'1': 'INSTANCE_STATUS_COMPLETED', '2': 2},
    {'1': 'INSTANCE_STATUS_FAILED', '2': 3},
    {'1': 'INSTANCE_STATUS_CANCELLED', '2': 4},
    {'1': 'INSTANCE_STATUS_SUSPENDED', '2': 5},
  ],
};

/// Descriptor for `InstanceStatus`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List instanceStatusDescriptor = $convert.base64Decode(
    'Cg5JbnN0YW5jZVN0YXR1cxIfChtJTlNUQU5DRV9TVEFUVVNfVU5TUEVDSUZJRUQQABIbChdJTl'
    'NUQU5DRV9TVEFUVVNfUlVOTklORxABEh0KGUlOU1RBTkNFX1NUQVRVU19DT01QTEVURUQQAhIa'
    'ChZJTlNUQU5DRV9TVEFUVVNfRkFJTEVEEAMSHQoZSU5TVEFOQ0VfU1RBVFVTX0NBTkNFTExFRB'
    'AEEh0KGUlOU1RBTkNFX1NUQVRVU19TVVNQRU5ERUQQBQ==');

@$core.Deprecated('Use executionStatusDescriptor instead')
const ExecutionStatus$json = {
  '1': 'ExecutionStatus',
  '2': [
    {'1': 'EXECUTION_STATUS_UNSPECIFIED', '2': 0},
    {'1': 'EXECUTION_STATUS_PENDING', '2': 1},
    {'1': 'EXECUTION_STATUS_DISPATCHED', '2': 2},
    {'1': 'EXECUTION_STATUS_RUNNING', '2': 3},
    {'1': 'EXECUTION_STATUS_COMPLETED', '2': 4},
    {'1': 'EXECUTION_STATUS_FAILED', '2': 5},
    {'1': 'EXECUTION_STATUS_FATAL', '2': 6},
    {'1': 'EXECUTION_STATUS_TIMED_OUT', '2': 7},
    {'1': 'EXECUTION_STATUS_INVALID_INPUT_CONTRACT', '2': 8},
    {'1': 'EXECUTION_STATUS_INVALID_OUTPUT_CONTRACT', '2': 9},
    {'1': 'EXECUTION_STATUS_STALE', '2': 10},
    {'1': 'EXECUTION_STATUS_RETRY_SCHEDULED', '2': 11},
    {'1': 'EXECUTION_STATUS_WAITING', '2': 12},
  ],
};

/// Descriptor for `ExecutionStatus`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List executionStatusDescriptor = $convert.base64Decode(
    'Cg9FeGVjdXRpb25TdGF0dXMSIAocRVhFQ1VUSU9OX1NUQVRVU19VTlNQRUNJRklFRBAAEhwKGE'
    'VYRUNVVElPTl9TVEFUVVNfUEVORElORxABEh8KG0VYRUNVVElPTl9TVEFUVVNfRElTUEFUQ0hF'
    'RBACEhwKGEVYRUNVVElPTl9TVEFUVVNfUlVOTklORxADEh4KGkVYRUNVVElPTl9TVEFUVVNfQ0'
    '9NUExFVEVEEAQSGwoXRVhFQ1VUSU9OX1NUQVRVU19GQUlMRUQQBRIaChZFWEVDVVRJT05fU1RB'
    'VFVTX0ZBVEFMEAYSHgoaRVhFQ1VUSU9OX1NUQVRVU19USU1FRF9PVVQQBxIrCidFWEVDVVRJT0'
    '5fU1RBVFVTX0lOVkFMSURfSU5QVVRfQ09OVFJBQ1QQCBIsCihFWEVDVVRJT05fU1RBVFVTX0lO'
    'VkFMSURfT1VUUFVUX0NPTlRSQUNUEAkSGgoWRVhFQ1VUSU9OX1NUQVRVU19TVEFMRRAKEiQKIE'
    'VYRUNVVElPTl9TVEFUVVNfUkVUUllfU0NIRURVTEVEEAsSHAoYRVhFQ1VUSU9OX1NUQVRVU19X'
    'QUlUSU5HEAw=');

@$core.Deprecated('Use workflowInstanceDescriptor instead')
const WorkflowInstance$json = {
  '1': 'WorkflowInstance',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
    {'1': 'workflow_name', '3': 2, '4': 1, '5': 9, '10': 'workflowName'},
    {'1': 'workflow_version', '3': 3, '4': 1, '5': 5, '10': 'workflowVersion'},
    {'1': 'current_state', '3': 4, '4': 1, '5': 9, '10': 'currentState'},
    {'1': 'status', '3': 5, '4': 1, '5': 14, '6': '.runtime.v1.InstanceStatus', '10': 'status'},
    {'1': 'revision', '3': 6, '4': 1, '5': 3, '10': 'revision'},
    {'1': 'trigger_event_id', '3': 7, '4': 1, '5': 9, '10': 'triggerEventId'},
    {'1': 'metadata', '3': 8, '4': 1, '5': 11, '6': '.google.protobuf.Struct', '10': 'metadata'},
    {'1': 'started_at', '3': 9, '4': 1, '5': 11, '6': '.google.protobuf.Timestamp', '10': 'startedAt'},
    {'1': 'finished_at', '3': 10, '4': 1, '5': 11, '6': '.google.protobuf.Timestamp', '10': 'finishedAt'},
    {'1': 'created_at', '3': 11, '4': 1, '5': 11, '6': '.google.protobuf.Timestamp', '10': 'createdAt'},
    {'1': 'updated_at', '3': 12, '4': 1, '5': 11, '6': '.google.protobuf.Timestamp', '10': 'updatedAt'},
    {'1': 'parent_instance_id', '3': 13, '4': 1, '5': 9, '10': 'parentInstanceId'},
    {'1': 'parent_execution_id', '3': 14, '4': 1, '5': 9, '10': 'parentExecutionId'},
    {'1': 'scope_type', '3': 15, '4': 1, '5': 9, '10': 'scopeType'},
    {'1': 'scope_parent_state', '3': 16, '4': 1, '5': 9, '10': 'scopeParentState'},
    {'1': 'scope_entry_state', '3': 17, '4': 1, '5': 9, '10': 'scopeEntryState'},
    {'1': 'scope_index', '3': 18, '4': 1, '5': 5, '10': 'scopeIndex'},
  ],
};

/// Descriptor for `WorkflowInstance`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List workflowInstanceDescriptor = $convert.base64Decode(
    'ChBXb3JrZmxvd0luc3RhbmNlEg4KAmlkGAEgASgJUgJpZBIjCg13b3JrZmxvd19uYW1lGAIgAS'
    'gJUgx3b3JrZmxvd05hbWUSKQoQd29ya2Zsb3dfdmVyc2lvbhgDIAEoBVIPd29ya2Zsb3dWZXJz'
    'aW9uEiMKDWN1cnJlbnRfc3RhdGUYBCABKAlSDGN1cnJlbnRTdGF0ZRIyCgZzdGF0dXMYBSABKA'
    '4yGi5ydW50aW1lLnYxLkluc3RhbmNlU3RhdHVzUgZzdGF0dXMSGgoIcmV2aXNpb24YBiABKANS'
    'CHJldmlzaW9uEigKEHRyaWdnZXJfZXZlbnRfaWQYByABKAlSDnRyaWdnZXJFdmVudElkEjMKCG'
    '1ldGFkYXRhGAggASgLMhcuZ29vZ2xlLnByb3RvYnVmLlN0cnVjdFIIbWV0YWRhdGESOQoKc3Rh'
    'cnRlZF9hdBgJIAEoCzIaLmdvb2dsZS5wcm90b2J1Zi5UaW1lc3RhbXBSCXN0YXJ0ZWRBdBI7Cg'
    'tmaW5pc2hlZF9hdBgKIAEoCzIaLmdvb2dsZS5wcm90b2J1Zi5UaW1lc3RhbXBSCmZpbmlzaGVk'
    'QXQSOQoKY3JlYXRlZF9hdBgLIAEoCzIaLmdvb2dsZS5wcm90b2J1Zi5UaW1lc3RhbXBSCWNyZW'
    'F0ZWRBdBI5Cgp1cGRhdGVkX2F0GAwgASgLMhouZ29vZ2xlLnByb3RvYnVmLlRpbWVzdGFtcFIJ'
    'dXBkYXRlZEF0EiwKEnBhcmVudF9pbnN0YW5jZV9pZBgNIAEoCVIQcGFyZW50SW5zdGFuY2VJZB'
    'IuChNwYXJlbnRfZXhlY3V0aW9uX2lkGA4gASgJUhFwYXJlbnRFeGVjdXRpb25JZBIdCgpzY29w'
    'ZV90eXBlGA8gASgJUglzY29wZVR5cGUSLAoSc2NvcGVfcGFyZW50X3N0YXRlGBAgASgJUhBzY2'
    '9wZVBhcmVudFN0YXRlEioKEXNjb3BlX2VudHJ5X3N0YXRlGBEgASgJUg9zY29wZUVudHJ5U3Rh'
    'dGUSHwoLc2NvcGVfaW5kZXgYEiABKAVSCnNjb3BlSW5kZXg=');

@$core.Deprecated('Use workflowExecutionDescriptor instead')
const WorkflowExecution$json = {
  '1': 'WorkflowExecution',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
    {'1': 'instance_id', '3': 2, '4': 1, '5': 9, '10': 'instanceId'},
    {'1': 'state', '3': 3, '4': 1, '5': 9, '10': 'state'},
    {'1': 'state_version', '3': 4, '4': 1, '5': 5, '10': 'stateVersion'},
    {'1': 'attempt', '3': 5, '4': 1, '5': 5, '10': 'attempt'},
    {'1': 'status', '3': 6, '4': 1, '5': 14, '6': '.runtime.v1.ExecutionStatus', '10': 'status'},
    {'1': 'error_class', '3': 7, '4': 1, '5': 9, '10': 'errorClass'},
    {'1': 'error_message', '3': 8, '4': 1, '5': 9, '10': 'errorMessage'},
    {'1': 'next_retry_at', '3': 9, '4': 1, '5': 11, '6': '.google.protobuf.Timestamp', '10': 'nextRetryAt'},
    {'1': 'started_at', '3': 10, '4': 1, '5': 11, '6': '.google.protobuf.Timestamp', '10': 'startedAt'},
    {'1': 'finished_at', '3': 11, '4': 1, '5': 11, '6': '.google.protobuf.Timestamp', '10': 'finishedAt'},
    {'1': 'created_at', '3': 12, '4': 1, '5': 11, '6': '.google.protobuf.Timestamp', '10': 'createdAt'},
    {'1': 'updated_at', '3': 13, '4': 1, '5': 11, '6': '.google.protobuf.Timestamp', '10': 'updatedAt'},
    {'1': 'trace_id', '3': 14, '4': 1, '5': 9, '10': 'traceId'},
    {'1': 'input_schema_hash', '3': 15, '4': 1, '5': 9, '10': 'inputSchemaHash'},
    {'1': 'output_schema_hash', '3': 16, '4': 1, '5': 9, '10': 'outputSchemaHash'},
    {'1': 'input_payload', '3': 17, '4': 1, '5': 11, '6': '.google.protobuf.Struct', '10': 'inputPayload'},
    {'1': 'output', '3': 18, '4': 1, '5': 11, '6': '.google.protobuf.Struct', '10': 'output'},
  ],
};

/// Descriptor for `WorkflowExecution`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List workflowExecutionDescriptor = $convert.base64Decode(
    'ChFXb3JrZmxvd0V4ZWN1dGlvbhIOCgJpZBgBIAEoCVICaWQSHwoLaW5zdGFuY2VfaWQYAiABKA'
    'lSCmluc3RhbmNlSWQSFAoFc3RhdGUYAyABKAlSBXN0YXRlEiMKDXN0YXRlX3ZlcnNpb24YBCAB'
    'KAVSDHN0YXRlVmVyc2lvbhIYCgdhdHRlbXB0GAUgASgFUgdhdHRlbXB0EjMKBnN0YXR1cxgGIA'
    'EoDjIbLnJ1bnRpbWUudjEuRXhlY3V0aW9uU3RhdHVzUgZzdGF0dXMSHwoLZXJyb3JfY2xhc3MY'
    'ByABKAlSCmVycm9yQ2xhc3MSIwoNZXJyb3JfbWVzc2FnZRgIIAEoCVIMZXJyb3JNZXNzYWdlEj'
    '4KDW5leHRfcmV0cnlfYXQYCSABKAsyGi5nb29nbGUucHJvdG9idWYuVGltZXN0YW1wUgtuZXh0'
    'UmV0cnlBdBI5CgpzdGFydGVkX2F0GAogASgLMhouZ29vZ2xlLnByb3RvYnVmLlRpbWVzdGFtcF'
    'IJc3RhcnRlZEF0EjsKC2ZpbmlzaGVkX2F0GAsgASgLMhouZ29vZ2xlLnByb3RvYnVmLlRpbWVz'
    'dGFtcFIKZmluaXNoZWRBdBI5CgpjcmVhdGVkX2F0GAwgASgLMhouZ29vZ2xlLnByb3RvYnVmLl'
    'RpbWVzdGFtcFIJY3JlYXRlZEF0EjkKCnVwZGF0ZWRfYXQYDSABKAsyGi5nb29nbGUucHJvdG9i'
    'dWYuVGltZXN0YW1wUgl1cGRhdGVkQXQSGQoIdHJhY2VfaWQYDiABKAlSB3RyYWNlSWQSKgoRaW'
    '5wdXRfc2NoZW1hX2hhc2gYDyABKAlSD2lucHV0U2NoZW1hSGFzaBIsChJvdXRwdXRfc2NoZW1h'
    'X2hhc2gYECABKAlSEG91dHB1dFNjaGVtYUhhc2gSPAoNaW5wdXRfcGF5bG9hZBgRIAEoCzIXLm'
    'dvb2dsZS5wcm90b2J1Zi5TdHJ1Y3RSDGlucHV0UGF5bG9hZBIvCgZvdXRwdXQYEiABKAsyFy5n'
    'b29nbGUucHJvdG9idWYuU3RydWN0UgZvdXRwdXQ=');

@$core.Deprecated('Use listInstancesRequestDescriptor instead')
const ListInstancesRequest$json = {
  '1': 'ListInstancesRequest',
  '2': [
    {'1': 'workflow_name', '3': 1, '4': 1, '5': 9, '10': 'workflowName'},
    {'1': 'status', '3': 2, '4': 1, '5': 14, '6': '.runtime.v1.InstanceStatus', '10': 'status'},
    {'1': 'search', '3': 3, '4': 1, '5': 11, '6': '.common.v1.SearchRequest', '10': 'search'},
  ],
};

/// Descriptor for `ListInstancesRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List listInstancesRequestDescriptor = $convert.base64Decode(
    'ChRMaXN0SW5zdGFuY2VzUmVxdWVzdBIjCg13b3JrZmxvd19uYW1lGAEgASgJUgx3b3JrZmxvd0'
    '5hbWUSMgoGc3RhdHVzGAIgASgOMhoucnVudGltZS52MS5JbnN0YW5jZVN0YXR1c1IGc3RhdHVz'
    'EjAKBnNlYXJjaBgDIAEoCzIYLmNvbW1vbi52MS5TZWFyY2hSZXF1ZXN0UgZzZWFyY2g=');

@$core.Deprecated('Use listInstancesResponseDescriptor instead')
const ListInstancesResponse$json = {
  '1': 'ListInstancesResponse',
  '2': [
    {'1': 'items', '3': 1, '4': 3, '5': 11, '6': '.runtime.v1.WorkflowInstance', '10': 'items'},
    {'1': 'next_cursor', '3': 2, '4': 1, '5': 11, '6': '.common.v1.PageCursor', '10': 'nextCursor'},
  ],
};

/// Descriptor for `ListInstancesResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List listInstancesResponseDescriptor = $convert.base64Decode(
    'ChVMaXN0SW5zdGFuY2VzUmVzcG9uc2USMgoFaXRlbXMYASADKAsyHC5ydW50aW1lLnYxLldvcm'
    'tmbG93SW5zdGFuY2VSBWl0ZW1zEjYKC25leHRfY3Vyc29yGAIgASgLMhUuY29tbW9uLnYxLlBh'
    'Z2VDdXJzb3JSCm5leHRDdXJzb3I=');

@$core.Deprecated('Use retryInstanceRequestDescriptor instead')
const RetryInstanceRequest$json = {
  '1': 'RetryInstanceRequest',
  '2': [
    {'1': 'instance_id', '3': 1, '4': 1, '5': 9, '10': 'instanceId'},
  ],
};

/// Descriptor for `RetryInstanceRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List retryInstanceRequestDescriptor = $convert.base64Decode(
    'ChRSZXRyeUluc3RhbmNlUmVxdWVzdBIfCgtpbnN0YW5jZV9pZBgBIAEoCVIKaW5zdGFuY2VJZA'
    '==');

@$core.Deprecated('Use retryInstanceResponseDescriptor instead')
const RetryInstanceResponse$json = {
  '1': 'RetryInstanceResponse',
  '2': [
    {'1': 'execution', '3': 1, '4': 1, '5': 11, '6': '.runtime.v1.WorkflowExecution', '10': 'execution'},
  ],
};

/// Descriptor for `RetryInstanceResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List retryInstanceResponseDescriptor = $convert.base64Decode(
    'ChVSZXRyeUluc3RhbmNlUmVzcG9uc2USOwoJZXhlY3V0aW9uGAEgASgLMh0ucnVudGltZS52MS'
    '5Xb3JrZmxvd0V4ZWN1dGlvblIJZXhlY3V0aW9u');

@$core.Deprecated('Use listExecutionsRequestDescriptor instead')
const ListExecutionsRequest$json = {
  '1': 'ListExecutionsRequest',
  '2': [
    {'1': 'instance_id', '3': 1, '4': 1, '5': 9, '10': 'instanceId'},
    {'1': 'status', '3': 2, '4': 1, '5': 14, '6': '.runtime.v1.ExecutionStatus', '10': 'status'},
    {'1': 'search', '3': 3, '4': 1, '5': 11, '6': '.common.v1.SearchRequest', '10': 'search'},
  ],
};

/// Descriptor for `ListExecutionsRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List listExecutionsRequestDescriptor = $convert.base64Decode(
    'ChVMaXN0RXhlY3V0aW9uc1JlcXVlc3QSHwoLaW5zdGFuY2VfaWQYASABKAlSCmluc3RhbmNlSW'
    'QSMwoGc3RhdHVzGAIgASgOMhsucnVudGltZS52MS5FeGVjdXRpb25TdGF0dXNSBnN0YXR1cxIw'
    'CgZzZWFyY2gYAyABKAsyGC5jb21tb24udjEuU2VhcmNoUmVxdWVzdFIGc2VhcmNo');

@$core.Deprecated('Use listExecutionsResponseDescriptor instead')
const ListExecutionsResponse$json = {
  '1': 'ListExecutionsResponse',
  '2': [
    {'1': 'items', '3': 1, '4': 3, '5': 11, '6': '.runtime.v1.WorkflowExecution', '10': 'items'},
    {'1': 'next_cursor', '3': 2, '4': 1, '5': 11, '6': '.common.v1.PageCursor', '10': 'nextCursor'},
  ],
};

/// Descriptor for `ListExecutionsResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List listExecutionsResponseDescriptor = $convert.base64Decode(
    'ChZMaXN0RXhlY3V0aW9uc1Jlc3BvbnNlEjMKBWl0ZW1zGAEgAygLMh0ucnVudGltZS52MS5Xb3'
    'JrZmxvd0V4ZWN1dGlvblIFaXRlbXMSNgoLbmV4dF9jdXJzb3IYAiABKAsyFS5jb21tb24udjEu'
    'UGFnZUN1cnNvclIKbmV4dEN1cnNvcg==');

@$core.Deprecated('Use getExecutionRequestDescriptor instead')
const GetExecutionRequest$json = {
  '1': 'GetExecutionRequest',
  '2': [
    {'1': 'execution_id', '3': 1, '4': 1, '5': 9, '10': 'executionId'},
    {'1': 'include_output', '3': 2, '4': 1, '5': 8, '10': 'includeOutput'},
  ],
};

/// Descriptor for `GetExecutionRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getExecutionRequestDescriptor = $convert.base64Decode(
    'ChNHZXRFeGVjdXRpb25SZXF1ZXN0EiEKDGV4ZWN1dGlvbl9pZBgBIAEoCVILZXhlY3V0aW9uSW'
    'QSJQoOaW5jbHVkZV9vdXRwdXQYAiABKAhSDWluY2x1ZGVPdXRwdXQ=');

@$core.Deprecated('Use getExecutionResponseDescriptor instead')
const GetExecutionResponse$json = {
  '1': 'GetExecutionResponse',
  '2': [
    {'1': 'execution', '3': 1, '4': 1, '5': 11, '6': '.runtime.v1.WorkflowExecution', '10': 'execution'},
  ],
};

/// Descriptor for `GetExecutionResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getExecutionResponseDescriptor = $convert.base64Decode(
    'ChRHZXRFeGVjdXRpb25SZXNwb25zZRI7CglleGVjdXRpb24YASABKAsyHS5ydW50aW1lLnYxLl'
    'dvcmtmbG93RXhlY3V0aW9uUglleGVjdXRpb24=');

@$core.Deprecated('Use retryExecutionRequestDescriptor instead')
const RetryExecutionRequest$json = {
  '1': 'RetryExecutionRequest',
  '2': [
    {'1': 'execution_id', '3': 1, '4': 1, '5': 9, '10': 'executionId'},
  ],
};

/// Descriptor for `RetryExecutionRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List retryExecutionRequestDescriptor = $convert.base64Decode(
    'ChVSZXRyeUV4ZWN1dGlvblJlcXVlc3QSIQoMZXhlY3V0aW9uX2lkGAEgASgJUgtleGVjdXRpb2'
    '5JZA==');

@$core.Deprecated('Use retryExecutionResponseDescriptor instead')
const RetryExecutionResponse$json = {
  '1': 'RetryExecutionResponse',
  '2': [
    {'1': 'execution', '3': 1, '4': 1, '5': 11, '6': '.runtime.v1.WorkflowExecution', '10': 'execution'},
  ],
};

/// Descriptor for `RetryExecutionResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List retryExecutionResponseDescriptor = $convert.base64Decode(
    'ChZSZXRyeUV4ZWN1dGlvblJlc3BvbnNlEjsKCWV4ZWN1dGlvbhgBIAEoCzIdLnJ1bnRpbWUudj'
    'EuV29ya2Zsb3dFeGVjdXRpb25SCWV4ZWN1dGlvbg==');

@$core.Deprecated('Use resumeExecutionRequestDescriptor instead')
const ResumeExecutionRequest$json = {
  '1': 'ResumeExecutionRequest',
  '2': [
    {'1': 'execution_id', '3': 1, '4': 1, '5': 9, '10': 'executionId'},
    {'1': 'payload', '3': 2, '4': 1, '5': 11, '6': '.google.protobuf.Struct', '10': 'payload'},
  ],
};

/// Descriptor for `ResumeExecutionRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List resumeExecutionRequestDescriptor = $convert.base64Decode(
    'ChZSZXN1bWVFeGVjdXRpb25SZXF1ZXN0EiEKDGV4ZWN1dGlvbl9pZBgBIAEoCVILZXhlY3V0aW'
    '9uSWQSMQoHcGF5bG9hZBgCIAEoCzIXLmdvb2dsZS5wcm90b2J1Zi5TdHJ1Y3RSB3BheWxvYWQ=');

@$core.Deprecated('Use resumeExecutionResponseDescriptor instead')
const ResumeExecutionResponse$json = {
  '1': 'ResumeExecutionResponse',
  '2': [
    {'1': 'execution', '3': 1, '4': 1, '5': 11, '6': '.runtime.v1.WorkflowExecution', '10': 'execution'},
    {'1': 'action', '3': 2, '4': 1, '5': 9, '10': 'action'},
  ],
};

/// Descriptor for `ResumeExecutionResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List resumeExecutionResponseDescriptor = $convert.base64Decode(
    'ChdSZXN1bWVFeGVjdXRpb25SZXNwb25zZRI7CglleGVjdXRpb24YASABKAsyHS5ydW50aW1lLn'
    'YxLldvcmtmbG93RXhlY3V0aW9uUglleGVjdXRpb24SFgoGYWN0aW9uGAIgASgJUgZhY3Rpb24=');

@$core.Deprecated('Use runTimelineEntryDescriptor instead')
const RunTimelineEntry$json = {
  '1': 'RunTimelineEntry',
  '2': [
    {'1': 'event_type', '3': 1, '4': 1, '5': 9, '10': 'eventType'},
    {'1': 'state', '3': 2, '4': 1, '5': 9, '10': 'state'},
    {'1': 'from_state', '3': 3, '4': 1, '5': 9, '10': 'fromState'},
    {'1': 'to_state', '3': 4, '4': 1, '5': 9, '10': 'toState'},
    {'1': 'execution_id', '3': 5, '4': 1, '5': 9, '10': 'executionId'},
    {'1': 'trace_id', '3': 6, '4': 1, '5': 9, '10': 'traceId'},
    {'1': 'payload', '3': 7, '4': 1, '5': 11, '6': '.google.protobuf.Struct', '10': 'payload'},
    {'1': 'created_at', '3': 8, '4': 1, '5': 11, '6': '.google.protobuf.Timestamp', '10': 'createdAt'},
  ],
};

/// Descriptor for `RunTimelineEntry`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List runTimelineEntryDescriptor = $convert.base64Decode(
    'ChBSdW5UaW1lbGluZUVudHJ5Eh0KCmV2ZW50X3R5cGUYASABKAlSCWV2ZW50VHlwZRIUCgVzdG'
    'F0ZRgCIAEoCVIFc3RhdGUSHQoKZnJvbV9zdGF0ZRgDIAEoCVIJZnJvbVN0YXRlEhkKCHRvX3N0'
    'YXRlGAQgASgJUgd0b1N0YXRlEiEKDGV4ZWN1dGlvbl9pZBgFIAEoCVILZXhlY3V0aW9uSWQSGQ'
    'oIdHJhY2VfaWQYBiABKAlSB3RyYWNlSWQSMQoHcGF5bG9hZBgHIAEoCzIXLmdvb2dsZS5wcm90'
    'b2J1Zi5TdHJ1Y3RSB3BheWxvYWQSOQoKY3JlYXRlZF9hdBgIIAEoCzIaLmdvb2dsZS5wcm90b2'
    'J1Zi5UaW1lc3RhbXBSCWNyZWF0ZWRBdA==');

@$core.Deprecated('Use stateOutputDescriptor instead')
const StateOutput$json = {
  '1': 'StateOutput',
  '2': [
    {'1': 'execution_id', '3': 1, '4': 1, '5': 9, '10': 'executionId'},
    {'1': 'state', '3': 2, '4': 1, '5': 9, '10': 'state'},
    {'1': 'schema_hash', '3': 3, '4': 1, '5': 9, '10': 'schemaHash'},
    {'1': 'payload', '3': 4, '4': 1, '5': 11, '6': '.google.protobuf.Struct', '10': 'payload'},
    {'1': 'created_at', '3': 5, '4': 1, '5': 11, '6': '.google.protobuf.Timestamp', '10': 'createdAt'},
  ],
};

/// Descriptor for `StateOutput`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List stateOutputDescriptor = $convert.base64Decode(
    'CgtTdGF0ZU91dHB1dBIhCgxleGVjdXRpb25faWQYASABKAlSC2V4ZWN1dGlvbklkEhQKBXN0YX'
    'RlGAIgASgJUgVzdGF0ZRIfCgtzY2hlbWFfaGFzaBgDIAEoCVIKc2NoZW1hSGFzaBIxCgdwYXls'
    'b2FkGAQgASgLMhcuZ29vZ2xlLnByb3RvYnVmLlN0cnVjdFIHcGF5bG9hZBI5CgpjcmVhdGVkX2'
    'F0GAUgASgLMhouZ29vZ2xlLnByb3RvYnVmLlRpbWVzdGFtcFIJY3JlYXRlZEF0');

@$core.Deprecated('Use scopeRunDescriptor instead')
const ScopeRun$json = {
  '1': 'ScopeRun',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
    {'1': 'parent_execution_id', '3': 2, '4': 1, '5': 9, '10': 'parentExecutionId'},
    {'1': 'parent_state', '3': 3, '4': 1, '5': 9, '10': 'parentState'},
    {'1': 'scope_type', '3': 4, '4': 1, '5': 9, '10': 'scopeType'},
    {'1': 'status', '3': 5, '4': 1, '5': 9, '10': 'status'},
    {'1': 'wait_all', '3': 6, '4': 1, '5': 8, '10': 'waitAll'},
    {'1': 'total_children', '3': 7, '4': 1, '5': 5, '10': 'totalChildren'},
    {'1': 'completed_children', '3': 8, '4': 1, '5': 5, '10': 'completedChildren'},
    {'1': 'failed_children', '3': 9, '4': 1, '5': 5, '10': 'failedChildren'},
    {'1': 'next_child_index', '3': 10, '4': 1, '5': 5, '10': 'nextChildIndex'},
    {'1': 'max_concurrency', '3': 11, '4': 1, '5': 5, '10': 'maxConcurrency'},
    {'1': 'item_var', '3': 12, '4': 1, '5': 9, '10': 'itemVar'},
    {'1': 'index_var', '3': 13, '4': 1, '5': 9, '10': 'indexVar'},
    {'1': 'items_payload', '3': 14, '4': 1, '5': 11, '6': '.google.protobuf.Struct', '10': 'itemsPayload'},
    {'1': 'results_payload', '3': 15, '4': 1, '5': 11, '6': '.google.protobuf.Struct', '10': 'resultsPayload'},
    {'1': 'created_at', '3': 16, '4': 1, '5': 11, '6': '.google.protobuf.Timestamp', '10': 'createdAt'},
    {'1': 'updated_at', '3': 17, '4': 1, '5': 11, '6': '.google.protobuf.Timestamp', '10': 'updatedAt'},
  ],
};

/// Descriptor for `ScopeRun`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List scopeRunDescriptor = $convert.base64Decode(
    'CghTY29wZVJ1bhIOCgJpZBgBIAEoCVICaWQSLgoTcGFyZW50X2V4ZWN1dGlvbl9pZBgCIAEoCV'
    'IRcGFyZW50RXhlY3V0aW9uSWQSIQoMcGFyZW50X3N0YXRlGAMgASgJUgtwYXJlbnRTdGF0ZRId'
    'CgpzY29wZV90eXBlGAQgASgJUglzY29wZVR5cGUSFgoGc3RhdHVzGAUgASgJUgZzdGF0dXMSGQ'
    'oId2FpdF9hbGwYBiABKAhSB3dhaXRBbGwSJQoOdG90YWxfY2hpbGRyZW4YByABKAVSDXRvdGFs'
    'Q2hpbGRyZW4SLQoSY29tcGxldGVkX2NoaWxkcmVuGAggASgFUhFjb21wbGV0ZWRDaGlsZHJlbh'
    'InCg9mYWlsZWRfY2hpbGRyZW4YCSABKAVSDmZhaWxlZENoaWxkcmVuEigKEG5leHRfY2hpbGRf'
    'aW5kZXgYCiABKAVSDm5leHRDaGlsZEluZGV4EicKD21heF9jb25jdXJyZW5jeRgLIAEoBVIObW'
    'F4Q29uY3VycmVuY3kSGQoIaXRlbV92YXIYDCABKAlSB2l0ZW1WYXISGwoJaW5kZXhfdmFyGA0g'
    'ASgJUghpbmRleFZhchI8Cg1pdGVtc19wYXlsb2FkGA4gASgLMhcuZ29vZ2xlLnByb3RvYnVmLl'
    'N0cnVjdFIMaXRlbXNQYXlsb2FkEkAKD3Jlc3VsdHNfcGF5bG9hZBgPIAEoCzIXLmdvb2dsZS5w'
    'cm90b2J1Zi5TdHJ1Y3RSDnJlc3VsdHNQYXlsb2FkEjkKCmNyZWF0ZWRfYXQYECABKAsyGi5nb2'
    '9nbGUucHJvdG9idWYuVGltZXN0YW1wUgljcmVhdGVkQXQSOQoKdXBkYXRlZF9hdBgRIAEoCzIa'
    'Lmdvb2dsZS5wcm90b2J1Zi5UaW1lc3RhbXBSCXVwZGF0ZWRBdA==');

@$core.Deprecated('Use signalWaitDescriptor instead')
const SignalWait$json = {
  '1': 'SignalWait',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
    {'1': 'execution_id', '3': 2, '4': 1, '5': 9, '10': 'executionId'},
    {'1': 'state', '3': 3, '4': 1, '5': 9, '10': 'state'},
    {'1': 'signal_name', '3': 4, '4': 1, '5': 9, '10': 'signalName'},
    {'1': 'output_var', '3': 5, '4': 1, '5': 9, '10': 'outputVar'},
    {'1': 'status', '3': 6, '4': 1, '5': 9, '10': 'status'},
    {'1': 'timeout_at', '3': 7, '4': 1, '5': 11, '6': '.google.protobuf.Timestamp', '10': 'timeoutAt'},
    {'1': 'matched_at', '3': 8, '4': 1, '5': 11, '6': '.google.protobuf.Timestamp', '10': 'matchedAt'},
    {'1': 'timed_out_at', '3': 9, '4': 1, '5': 11, '6': '.google.protobuf.Timestamp', '10': 'timedOutAt'},
    {'1': 'message_id', '3': 10, '4': 1, '5': 9, '10': 'messageId'},
    {'1': 'attempts', '3': 11, '4': 1, '5': 5, '10': 'attempts'},
    {'1': 'created_at', '3': 12, '4': 1, '5': 11, '6': '.google.protobuf.Timestamp', '10': 'createdAt'},
    {'1': 'updated_at', '3': 13, '4': 1, '5': 11, '6': '.google.protobuf.Timestamp', '10': 'updatedAt'},
  ],
};

/// Descriptor for `SignalWait`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List signalWaitDescriptor = $convert.base64Decode(
    'CgpTaWduYWxXYWl0Eg4KAmlkGAEgASgJUgJpZBIhCgxleGVjdXRpb25faWQYAiABKAlSC2V4ZW'
    'N1dGlvbklkEhQKBXN0YXRlGAMgASgJUgVzdGF0ZRIfCgtzaWduYWxfbmFtZRgEIAEoCVIKc2ln'
    'bmFsTmFtZRIdCgpvdXRwdXRfdmFyGAUgASgJUglvdXRwdXRWYXISFgoGc3RhdHVzGAYgASgJUg'
    'ZzdGF0dXMSOQoKdGltZW91dF9hdBgHIAEoCzIaLmdvb2dsZS5wcm90b2J1Zi5UaW1lc3RhbXBS'
    'CXRpbWVvdXRBdBI5CgptYXRjaGVkX2F0GAggASgLMhouZ29vZ2xlLnByb3RvYnVmLlRpbWVzdG'
    'FtcFIJbWF0Y2hlZEF0EjwKDHRpbWVkX291dF9hdBgJIAEoCzIaLmdvb2dsZS5wcm90b2J1Zi5U'
    'aW1lc3RhbXBSCnRpbWVkT3V0QXQSHQoKbWVzc2FnZV9pZBgKIAEoCVIJbWVzc2FnZUlkEhoKCG'
    'F0dGVtcHRzGAsgASgFUghhdHRlbXB0cxI5CgpjcmVhdGVkX2F0GAwgASgLMhouZ29vZ2xlLnBy'
    'b3RvYnVmLlRpbWVzdGFtcFIJY3JlYXRlZEF0EjkKCnVwZGF0ZWRfYXQYDSABKAsyGi5nb29nbG'
    'UucHJvdG9idWYuVGltZXN0YW1wUgl1cGRhdGVkQXQ=');

@$core.Deprecated('Use signalMessageDescriptor instead')
const SignalMessage$json = {
  '1': 'SignalMessage',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
    {'1': 'signal_name', '3': 2, '4': 1, '5': 9, '10': 'signalName'},
    {'1': 'payload', '3': 3, '4': 1, '5': 11, '6': '.google.protobuf.Struct', '10': 'payload'},
    {'1': 'status', '3': 4, '4': 1, '5': 9, '10': 'status'},
    {'1': 'delivered_at', '3': 5, '4': 1, '5': 11, '6': '.google.protobuf.Timestamp', '10': 'deliveredAt'},
    {'1': 'wait_id', '3': 6, '4': 1, '5': 9, '10': 'waitId'},
    {'1': 'attempts', '3': 7, '4': 1, '5': 5, '10': 'attempts'},
    {'1': 'created_at', '3': 8, '4': 1, '5': 11, '6': '.google.protobuf.Timestamp', '10': 'createdAt'},
    {'1': 'updated_at', '3': 9, '4': 1, '5': 11, '6': '.google.protobuf.Timestamp', '10': 'updatedAt'},
  ],
};

/// Descriptor for `SignalMessage`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List signalMessageDescriptor = $convert.base64Decode(
    'Cg1TaWduYWxNZXNzYWdlEg4KAmlkGAEgASgJUgJpZBIfCgtzaWduYWxfbmFtZRgCIAEoCVIKc2'
    'lnbmFsTmFtZRIxCgdwYXlsb2FkGAMgASgLMhcuZ29vZ2xlLnByb3RvYnVmLlN0cnVjdFIHcGF5'
    'bG9hZBIWCgZzdGF0dXMYBCABKAlSBnN0YXR1cxI9CgxkZWxpdmVyZWRfYXQYBSABKAsyGi5nb2'
    '9nbGUucHJvdG9idWYuVGltZXN0YW1wUgtkZWxpdmVyZWRBdBIXCgd3YWl0X2lkGAYgASgJUgZ3'
    'YWl0SWQSGgoIYXR0ZW1wdHMYByABKAVSCGF0dGVtcHRzEjkKCmNyZWF0ZWRfYXQYCCABKAsyGi'
    '5nb29nbGUucHJvdG9idWYuVGltZXN0YW1wUgljcmVhdGVkQXQSOQoKdXBkYXRlZF9hdBgJIAEo'
    'CzIaLmdvb2dsZS5wcm90b2J1Zi5UaW1lc3RhbXBSCXVwZGF0ZWRBdA==');

@$core.Deprecated('Use getInstanceRunRequestDescriptor instead')
const GetInstanceRunRequest$json = {
  '1': 'GetInstanceRunRequest',
  '2': [
    {'1': 'instance_id', '3': 1, '4': 1, '5': 9, '10': 'instanceId'},
    {'1': 'include_payloads', '3': 2, '4': 1, '5': 8, '10': 'includePayloads'},
    {'1': 'execution_limit', '3': 3, '4': 1, '5': 5, '10': 'executionLimit'},
    {'1': 'timeline_limit', '3': 4, '4': 1, '5': 5, '10': 'timelineLimit'},
  ],
};

/// Descriptor for `GetInstanceRunRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getInstanceRunRequestDescriptor = $convert.base64Decode(
    'ChVHZXRJbnN0YW5jZVJ1blJlcXVlc3QSHwoLaW5zdGFuY2VfaWQYASABKAlSCmluc3RhbmNlSW'
    'QSKQoQaW5jbHVkZV9wYXlsb2FkcxgCIAEoCFIPaW5jbHVkZVBheWxvYWRzEicKD2V4ZWN1dGlv'
    'bl9saW1pdBgDIAEoBVIOZXhlY3V0aW9uTGltaXQSJQoOdGltZWxpbmVfbGltaXQYBCABKAVSDX'
    'RpbWVsaW5lTGltaXQ=');

@$core.Deprecated('Use getInstanceRunResponseDescriptor instead')
const GetInstanceRunResponse$json = {
  '1': 'GetInstanceRunResponse',
  '2': [
    {'1': 'instance', '3': 1, '4': 1, '5': 11, '6': '.runtime.v1.WorkflowInstance', '10': 'instance'},
    {'1': 'latest_execution', '3': 2, '4': 1, '5': 11, '6': '.runtime.v1.WorkflowExecution', '10': 'latestExecution'},
    {'1': 'trace_id', '3': 3, '4': 1, '5': 9, '10': 'traceId'},
    {'1': 'resume_strategy', '3': 4, '4': 1, '5': 9, '10': 'resumeStrategy'},
    {'1': 'executions', '3': 5, '4': 3, '5': 11, '6': '.runtime.v1.WorkflowExecution', '10': 'executions'},
    {'1': 'timeline', '3': 6, '4': 3, '5': 11, '6': '.runtime.v1.RunTimelineEntry', '10': 'timeline'},
    {'1': 'outputs', '3': 7, '4': 3, '5': 11, '6': '.runtime.v1.StateOutput', '10': 'outputs'},
    {'1': 'scope_runs', '3': 8, '4': 3, '5': 11, '6': '.runtime.v1.ScopeRun', '10': 'scopeRuns'},
    {'1': 'signal_waits', '3': 9, '4': 3, '5': 11, '6': '.runtime.v1.SignalWait', '10': 'signalWaits'},
    {'1': 'signal_messages', '3': 10, '4': 3, '5': 11, '6': '.runtime.v1.SignalMessage', '10': 'signalMessages'},
  ],
};

/// Descriptor for `GetInstanceRunResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getInstanceRunResponseDescriptor = $convert.base64Decode(
    'ChZHZXRJbnN0YW5jZVJ1blJlc3BvbnNlEjgKCGluc3RhbmNlGAEgASgLMhwucnVudGltZS52MS'
    '5Xb3JrZmxvd0luc3RhbmNlUghpbnN0YW5jZRJIChBsYXRlc3RfZXhlY3V0aW9uGAIgASgLMh0u'
    'cnVudGltZS52MS5Xb3JrZmxvd0V4ZWN1dGlvblIPbGF0ZXN0RXhlY3V0aW9uEhkKCHRyYWNlX2'
    'lkGAMgASgJUgd0cmFjZUlkEicKD3Jlc3VtZV9zdHJhdGVneRgEIAEoCVIOcmVzdW1lU3RyYXRl'
    'Z3kSPQoKZXhlY3V0aW9ucxgFIAMoCzIdLnJ1bnRpbWUudjEuV29ya2Zsb3dFeGVjdXRpb25SCm'
    'V4ZWN1dGlvbnMSOAoIdGltZWxpbmUYBiADKAsyHC5ydW50aW1lLnYxLlJ1blRpbWVsaW5lRW50'
    'cnlSCHRpbWVsaW5lEjEKB291dHB1dHMYByADKAsyFy5ydW50aW1lLnYxLlN0YXRlT3V0cHV0Ug'
    'dvdXRwdXRzEjMKCnNjb3BlX3J1bnMYCCADKAsyFC5ydW50aW1lLnYxLlNjb3BlUnVuUglzY29w'
    'ZVJ1bnMSOQoMc2lnbmFsX3dhaXRzGAkgAygLMhYucnVudGltZS52MS5TaWduYWxXYWl0UgtzaW'
    'duYWxXYWl0cxJCCg9zaWduYWxfbWVzc2FnZXMYCiADKAsyGS5ydW50aW1lLnYxLlNpZ25hbE1l'
    'c3NhZ2VSDnNpZ25hbE1lc3NhZ2Vz');

const $core.Map<$core.String, $core.dynamic> RuntimeServiceBase$json = {
  '1': 'RuntimeService',
  '2': [
    {'1': 'ListInstances', '2': '.runtime.v1.ListInstancesRequest', '3': '.runtime.v1.ListInstancesResponse', '4': {}},
    {'1': 'RetryInstance', '2': '.runtime.v1.RetryInstanceRequest', '3': '.runtime.v1.RetryInstanceResponse', '4': {}},
    {'1': 'ListExecutions', '2': '.runtime.v1.ListExecutionsRequest', '3': '.runtime.v1.ListExecutionsResponse', '4': {}},
    {'1': 'GetExecution', '2': '.runtime.v1.GetExecutionRequest', '3': '.runtime.v1.GetExecutionResponse', '4': {}},
    {'1': 'RetryExecution', '2': '.runtime.v1.RetryExecutionRequest', '3': '.runtime.v1.RetryExecutionResponse', '4': {}},
    {'1': 'ResumeExecution', '2': '.runtime.v1.ResumeExecutionRequest', '3': '.runtime.v1.ResumeExecutionResponse', '4': {}},
    {'1': 'GetInstanceRun', '2': '.runtime.v1.GetInstanceRunRequest', '3': '.runtime.v1.GetInstanceRunResponse', '4': {}},
  ],
  '3': {},
};

@$core.Deprecated('Use runtimeServiceDescriptor instead')
const $core.Map<$core.String, $core.Map<$core.String, $core.dynamic>> RuntimeServiceBase$messageJson = {
  '.runtime.v1.ListInstancesRequest': ListInstancesRequest$json,
  '.common.v1.SearchRequest': $7.SearchRequest$json,
  '.common.v1.PageCursor': $7.PageCursor$json,
  '.google.protobuf.Struct': $6.Struct$json,
  '.google.protobuf.Struct.FieldsEntry': $6.Struct_FieldsEntry$json,
  '.google.protobuf.Value': $6.Value$json,
  '.google.protobuf.ListValue': $6.ListValue$json,
  '.runtime.v1.ListInstancesResponse': ListInstancesResponse$json,
  '.runtime.v1.WorkflowInstance': WorkflowInstance$json,
  '.google.protobuf.Timestamp': $2.Timestamp$json,
  '.runtime.v1.RetryInstanceRequest': RetryInstanceRequest$json,
  '.runtime.v1.RetryInstanceResponse': RetryInstanceResponse$json,
  '.runtime.v1.WorkflowExecution': WorkflowExecution$json,
  '.runtime.v1.ListExecutionsRequest': ListExecutionsRequest$json,
  '.runtime.v1.ListExecutionsResponse': ListExecutionsResponse$json,
  '.runtime.v1.GetExecutionRequest': GetExecutionRequest$json,
  '.runtime.v1.GetExecutionResponse': GetExecutionResponse$json,
  '.runtime.v1.RetryExecutionRequest': RetryExecutionRequest$json,
  '.runtime.v1.RetryExecutionResponse': RetryExecutionResponse$json,
  '.runtime.v1.ResumeExecutionRequest': ResumeExecutionRequest$json,
  '.runtime.v1.ResumeExecutionResponse': ResumeExecutionResponse$json,
  '.runtime.v1.GetInstanceRunRequest': GetInstanceRunRequest$json,
  '.runtime.v1.GetInstanceRunResponse': GetInstanceRunResponse$json,
  '.runtime.v1.RunTimelineEntry': RunTimelineEntry$json,
  '.runtime.v1.StateOutput': StateOutput$json,
  '.runtime.v1.ScopeRun': ScopeRun$json,
  '.runtime.v1.SignalWait': SignalWait$json,
  '.runtime.v1.SignalMessage': SignalMessage$json,
};

/// Descriptor for `RuntimeService`. Decode as a `google.protobuf.ServiceDescriptorProto`.
final $typed_data.Uint8List runtimeServiceDescriptor = $convert.base64Decode(
    'Cg5SdW50aW1lU2VydmljZRJpCg1MaXN0SW5zdGFuY2VzEiAucnVudGltZS52MS5MaXN0SW5zdG'
    'FuY2VzUmVxdWVzdBohLnJ1bnRpbWUudjEuTGlzdEluc3RhbmNlc1Jlc3BvbnNlIhOCtRgPCg1p'
    'bnN0YW5jZV92aWV3EmoKDVJldHJ5SW5zdGFuY2USIC5ydW50aW1lLnYxLlJldHJ5SW5zdGFuY2'
    'VSZXF1ZXN0GiEucnVudGltZS52MS5SZXRyeUluc3RhbmNlUmVzcG9uc2UiFIK1GBAKDmluc3Rh'
    'bmNlX3JldHJ5Em0KDkxpc3RFeGVjdXRpb25zEiEucnVudGltZS52MS5MaXN0RXhlY3V0aW9uc1'
    'JlcXVlc3QaIi5ydW50aW1lLnYxLkxpc3RFeGVjdXRpb25zUmVzcG9uc2UiFIK1GBAKDmV4ZWN1'
    'dGlvbl92aWV3EmcKDEdldEV4ZWN1dGlvbhIfLnJ1bnRpbWUudjEuR2V0RXhlY3V0aW9uUmVxdW'
    'VzdBogLnJ1bnRpbWUudjEuR2V0RXhlY3V0aW9uUmVzcG9uc2UiFIK1GBAKDmV4ZWN1dGlvbl92'
    'aWV3Em4KDlJldHJ5RXhlY3V0aW9uEiEucnVudGltZS52MS5SZXRyeUV4ZWN1dGlvblJlcXVlc3'
    'QaIi5ydW50aW1lLnYxLlJldHJ5RXhlY3V0aW9uUmVzcG9uc2UiFYK1GBEKD2V4ZWN1dGlvbl9y'
    'ZXRyeRJyCg9SZXN1bWVFeGVjdXRpb24SIi5ydW50aW1lLnYxLlJlc3VtZUV4ZWN1dGlvblJlcX'
    'Vlc3QaIy5ydW50aW1lLnYxLlJlc3VtZUV4ZWN1dGlvblJlc3BvbnNlIhaCtRgSChBleGVjdXRp'
    'b25fcmVzdW1lEmwKDkdldEluc3RhbmNlUnVuEiEucnVudGltZS52MS5HZXRJbnN0YW5jZVJ1bl'
    'JlcXVlc3QaIi5ydW50aW1lLnYxLkdldEluc3RhbmNlUnVuUmVzcG9uc2UiE4K1GA8KDWluc3Rh'
    'bmNlX3ZpZXcaiwaCtRiGBgoQc2VydmljZV90cnVzdGFnZRIMZXZlbnRfaW5nZXN0Eg13b3JrZm'
    'xvd192aWV3Eg93b3JrZmxvd19tYW5hZ2USDWluc3RhbmNlX3ZpZXcSDmluc3RhbmNlX3JldHJ5'
    'Eg5leGVjdXRpb25fdmlldxIPZXhlY3V0aW9uX3JldHJ5EhBleGVjdXRpb25fcmVzdW1lEgtzaW'
    'duYWxfc2VuZBqPAQgBEgxldmVudF9pbmdlc3QSDXdvcmtmbG93X3ZpZXcSD3dvcmtmbG93X21h'
    'bmFnZRINaW5zdGFuY2VfdmlldxIOaW5zdGFuY2VfcmV0cnkSDmV4ZWN1dGlvbl92aWV3Eg9leG'
    'VjdXRpb25fcmV0cnkSEGV4ZWN1dGlvbl9yZXN1bWUSC3NpZ25hbF9zZW5kGo8BCAISDGV2ZW50'
    'X2luZ2VzdBINd29ya2Zsb3dfdmlldxIPd29ya2Zsb3dfbWFuYWdlEg1pbnN0YW5jZV92aWV3Eg'
    '5pbnN0YW5jZV9yZXRyeRIOZXhlY3V0aW9uX3ZpZXcSD2V4ZWN1dGlvbl9yZXRyeRIQZXhlY3V0'
    'aW9uX3Jlc3VtZRILc2lnbmFsX3NlbmQaSwgDEgxldmVudF9pbmdlc3QSDXdvcmtmbG93X3ZpZX'
    'cSDWluc3RhbmNlX3ZpZXcSDmV4ZWN1dGlvbl92aWV3EgtzaWduYWxfc2VuZBowCAQSDXdvcmtm'
    'bG93X3ZpZXcSDWluc3RhbmNlX3ZpZXcSDmV4ZWN1dGlvbl92aWV3GjAIBRINd29ya2Zsb3dfdm'
    'lldxINaW5zdGFuY2VfdmlldxIOZXhlY3V0aW9uX3ZpZXcajwEIBhIMZXZlbnRfaW5nZXN0Eg13'
    'b3JrZmxvd192aWV3Eg93b3JrZmxvd19tYW5hZ2USDWluc3RhbmNlX3ZpZXcSDmluc3RhbmNlX3'
    'JldHJ5Eg5leGVjdXRpb25fdmlldxIPZXhlY3V0aW9uX3JldHJ5EhBleGVjdXRpb25fcmVzdW1l'
    'EgtzaWduYWxfc2VuZA==');

