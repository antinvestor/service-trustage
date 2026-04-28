//
//  Generated code. Do not modify.
//  source: v1/workflow.proto
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

@$core.Deprecated('Use workflowStatusDescriptor instead')
const WorkflowStatus$json = {
  '1': 'WorkflowStatus',
  '2': [
    {'1': 'WORKFLOW_STATUS_UNSPECIFIED', '2': 0},
    {'1': 'WORKFLOW_STATUS_DRAFT', '2': 1},
    {'1': 'WORKFLOW_STATUS_ACTIVE', '2': 2},
    {'1': 'WORKFLOW_STATUS_ARCHIVED', '2': 3},
  ],
};

/// Descriptor for `WorkflowStatus`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List workflowStatusDescriptor = $convert.base64Decode(
    'Cg5Xb3JrZmxvd1N0YXR1cxIfChtXT1JLRkxPV19TVEFUVVNfVU5TUEVDSUZJRUQQABIZChVXT1'
    'JLRkxPV19TVEFUVVNfRFJBRlQQARIaChZXT1JLRkxPV19TVEFUVVNfQUNUSVZFEAISHAoYV09S'
    'S0ZMT1dfU1RBVFVTX0FSQ0hJVkVEEAM=');

@$core.Deprecated('Use workflowDefinitionDescriptor instead')
const WorkflowDefinition$json = {
  '1': 'WorkflowDefinition',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
    {'1': 'name', '3': 2, '4': 1, '5': 9, '10': 'name'},
    {'1': 'version', '3': 3, '4': 1, '5': 5, '10': 'version'},
    {'1': 'status', '3': 4, '4': 1, '5': 14, '6': '.workflow.v1.WorkflowStatus', '10': 'status'},
    {'1': 'dsl', '3': 5, '4': 1, '5': 11, '6': '.google.protobuf.Struct', '10': 'dsl'},
    {'1': 'input_schema_hash', '3': 6, '4': 1, '5': 9, '10': 'inputSchemaHash'},
    {'1': 'timeout_seconds', '3': 7, '4': 1, '5': 3, '10': 'timeoutSeconds'},
    {'1': 'created_at', '3': 8, '4': 1, '5': 11, '6': '.google.protobuf.Timestamp', '10': 'createdAt'},
    {'1': 'updated_at', '3': 9, '4': 1, '5': 11, '6': '.google.protobuf.Timestamp', '10': 'updatedAt'},
  ],
};

/// Descriptor for `WorkflowDefinition`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List workflowDefinitionDescriptor = $convert.base64Decode(
    'ChJXb3JrZmxvd0RlZmluaXRpb24SDgoCaWQYASABKAlSAmlkEhIKBG5hbWUYAiABKAlSBG5hbW'
    'USGAoHdmVyc2lvbhgDIAEoBVIHdmVyc2lvbhIzCgZzdGF0dXMYBCABKA4yGy53b3JrZmxvdy52'
    'MS5Xb3JrZmxvd1N0YXR1c1IGc3RhdHVzEikKA2RzbBgFIAEoCzIXLmdvb2dsZS5wcm90b2J1Zi'
    '5TdHJ1Y3RSA2RzbBIqChFpbnB1dF9zY2hlbWFfaGFzaBgGIAEoCVIPaW5wdXRTY2hlbWFIYXNo'
    'EicKD3RpbWVvdXRfc2Vjb25kcxgHIAEoA1IOdGltZW91dFNlY29uZHMSOQoKY3JlYXRlZF9hdB'
    'gIIAEoCzIaLmdvb2dsZS5wcm90b2J1Zi5UaW1lc3RhbXBSCWNyZWF0ZWRBdBI5Cgp1cGRhdGVk'
    'X2F0GAkgASgLMhouZ29vZ2xlLnByb3RvYnVmLlRpbWVzdGFtcFIJdXBkYXRlZEF0');

@$core.Deprecated('Use createWorkflowRequestDescriptor instead')
const CreateWorkflowRequest$json = {
  '1': 'CreateWorkflowRequest',
  '2': [
    {'1': 'dsl', '3': 1, '4': 1, '5': 11, '6': '.google.protobuf.Struct', '10': 'dsl'},
  ],
};

/// Descriptor for `CreateWorkflowRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List createWorkflowRequestDescriptor = $convert.base64Decode(
    'ChVDcmVhdGVXb3JrZmxvd1JlcXVlc3QSKQoDZHNsGAEgASgLMhcuZ29vZ2xlLnByb3RvYnVmLl'
    'N0cnVjdFIDZHNs');

@$core.Deprecated('Use createWorkflowResponseDescriptor instead')
const CreateWorkflowResponse$json = {
  '1': 'CreateWorkflowResponse',
  '2': [
    {'1': 'workflow', '3': 1, '4': 1, '5': 11, '6': '.workflow.v1.WorkflowDefinition', '10': 'workflow'},
  ],
};

/// Descriptor for `CreateWorkflowResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List createWorkflowResponseDescriptor = $convert.base64Decode(
    'ChZDcmVhdGVXb3JrZmxvd1Jlc3BvbnNlEjsKCHdvcmtmbG93GAEgASgLMh8ud29ya2Zsb3cudj'
    'EuV29ya2Zsb3dEZWZpbml0aW9uUgh3b3JrZmxvdw==');

@$core.Deprecated('Use scheduleDefinitionDescriptor instead')
const ScheduleDefinition$json = {
  '1': 'ScheduleDefinition',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
    {'1': 'name', '3': 2, '4': 1, '5': 9, '10': 'name'},
    {'1': 'cron_expr', '3': 3, '4': 1, '5': 9, '10': 'cronExpr'},
    {'1': 'workflow_name', '3': 4, '4': 1, '5': 9, '10': 'workflowName'},
    {'1': 'workflow_version', '3': 5, '4': 1, '5': 5, '10': 'workflowVersion'},
    {'1': 'active', '3': 6, '4': 1, '5': 8, '10': 'active'},
    {'1': 'next_fire_at', '3': 7, '4': 1, '5': 11, '6': '.google.protobuf.Timestamp', '10': 'nextFireAt'},
    {'1': 'last_fired_at', '3': 8, '4': 1, '5': 11, '6': '.google.protobuf.Timestamp', '10': 'lastFiredAt'},
    {'1': 'jitter_seconds', '3': 9, '4': 1, '5': 5, '10': 'jitterSeconds'},
    {'1': 'created_at', '3': 10, '4': 1, '5': 11, '6': '.google.protobuf.Timestamp', '10': 'createdAt'},
    {'1': 'updated_at', '3': 11, '4': 1, '5': 11, '6': '.google.protobuf.Timestamp', '10': 'updatedAt'},
    {'1': 'timezone', '3': 12, '4': 1, '5': 9, '10': 'timezone'},
  ],
};

/// Descriptor for `ScheduleDefinition`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List scheduleDefinitionDescriptor = $convert.base64Decode(
    'ChJTY2hlZHVsZURlZmluaXRpb24SDgoCaWQYASABKAlSAmlkEhIKBG5hbWUYAiABKAlSBG5hbW'
    'USGwoJY3Jvbl9leHByGAMgASgJUghjcm9uRXhwchIjCg13b3JrZmxvd19uYW1lGAQgASgJUgx3'
    'b3JrZmxvd05hbWUSKQoQd29ya2Zsb3dfdmVyc2lvbhgFIAEoBVIPd29ya2Zsb3dWZXJzaW9uEh'
    'YKBmFjdGl2ZRgGIAEoCFIGYWN0aXZlEjwKDG5leHRfZmlyZV9hdBgHIAEoCzIaLmdvb2dsZS5w'
    'cm90b2J1Zi5UaW1lc3RhbXBSCm5leHRGaXJlQXQSPgoNbGFzdF9maXJlZF9hdBgIIAEoCzIaLm'
    'dvb2dsZS5wcm90b2J1Zi5UaW1lc3RhbXBSC2xhc3RGaXJlZEF0EiUKDmppdHRlcl9zZWNvbmRz'
    'GAkgASgFUg1qaXR0ZXJTZWNvbmRzEjkKCmNyZWF0ZWRfYXQYCiABKAsyGi5nb29nbGUucHJvdG'
    '9idWYuVGltZXN0YW1wUgljcmVhdGVkQXQSOQoKdXBkYXRlZF9hdBgLIAEoCzIaLmdvb2dsZS5w'
    'cm90b2J1Zi5UaW1lc3RhbXBSCXVwZGF0ZWRBdBIaCgh0aW1lem9uZRgMIAEoCVIIdGltZXpvbm'
    'U=');

@$core.Deprecated('Use getWorkflowRequestDescriptor instead')
const GetWorkflowRequest$json = {
  '1': 'GetWorkflowRequest',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
  ],
};

/// Descriptor for `GetWorkflowRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getWorkflowRequestDescriptor = $convert.base64Decode(
    'ChJHZXRXb3JrZmxvd1JlcXVlc3QSDgoCaWQYASABKAlSAmlk');

@$core.Deprecated('Use getWorkflowResponseDescriptor instead')
const GetWorkflowResponse$json = {
  '1': 'GetWorkflowResponse',
  '2': [
    {'1': 'workflow', '3': 1, '4': 1, '5': 11, '6': '.workflow.v1.WorkflowDefinition', '10': 'workflow'},
    {'1': 'schedules', '3': 2, '4': 3, '5': 11, '6': '.workflow.v1.ScheduleDefinition', '10': 'schedules'},
  ],
};

/// Descriptor for `GetWorkflowResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getWorkflowResponseDescriptor = $convert.base64Decode(
    'ChNHZXRXb3JrZmxvd1Jlc3BvbnNlEjsKCHdvcmtmbG93GAEgASgLMh8ud29ya2Zsb3cudjEuV2'
    '9ya2Zsb3dEZWZpbml0aW9uUgh3b3JrZmxvdxI9CglzY2hlZHVsZXMYAiADKAsyHy53b3JrZmxv'
    'dy52MS5TY2hlZHVsZURlZmluaXRpb25SCXNjaGVkdWxlcw==');

@$core.Deprecated('Use listWorkflowsRequestDescriptor instead')
const ListWorkflowsRequest$json = {
  '1': 'ListWorkflowsRequest',
  '2': [
    {'1': 'name', '3': 1, '4': 1, '5': 9, '10': 'name'},
    {'1': 'status', '3': 2, '4': 1, '5': 14, '6': '.workflow.v1.WorkflowStatus', '10': 'status'},
    {'1': 'search', '3': 3, '4': 1, '5': 11, '6': '.common.v1.SearchRequest', '10': 'search'},
  ],
};

/// Descriptor for `ListWorkflowsRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List listWorkflowsRequestDescriptor = $convert.base64Decode(
    'ChRMaXN0V29ya2Zsb3dzUmVxdWVzdBISCgRuYW1lGAEgASgJUgRuYW1lEjMKBnN0YXR1cxgCIA'
    'EoDjIbLndvcmtmbG93LnYxLldvcmtmbG93U3RhdHVzUgZzdGF0dXMSMAoGc2VhcmNoGAMgASgL'
    'MhguY29tbW9uLnYxLlNlYXJjaFJlcXVlc3RSBnNlYXJjaA==');

@$core.Deprecated('Use listWorkflowsResponseDescriptor instead')
const ListWorkflowsResponse$json = {
  '1': 'ListWorkflowsResponse',
  '2': [
    {'1': 'items', '3': 1, '4': 3, '5': 11, '6': '.workflow.v1.WorkflowDefinition', '10': 'items'},
    {'1': 'next_cursor', '3': 2, '4': 1, '5': 11, '6': '.common.v1.PageCursor', '10': 'nextCursor'},
  ],
};

/// Descriptor for `ListWorkflowsResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List listWorkflowsResponseDescriptor = $convert.base64Decode(
    'ChVMaXN0V29ya2Zsb3dzUmVzcG9uc2USNQoFaXRlbXMYASADKAsyHy53b3JrZmxvdy52MS5Xb3'
    'JrZmxvd0RlZmluaXRpb25SBWl0ZW1zEjYKC25leHRfY3Vyc29yGAIgASgLMhUuY29tbW9uLnYx'
    'LlBhZ2VDdXJzb3JSCm5leHRDdXJzb3I=');

@$core.Deprecated('Use activateWorkflowRequestDescriptor instead')
const ActivateWorkflowRequest$json = {
  '1': 'ActivateWorkflowRequest',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
  ],
};

/// Descriptor for `ActivateWorkflowRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List activateWorkflowRequestDescriptor = $convert.base64Decode(
    'ChdBY3RpdmF0ZVdvcmtmbG93UmVxdWVzdBIOCgJpZBgBIAEoCVICaWQ=');

@$core.Deprecated('Use activateWorkflowResponseDescriptor instead')
const ActivateWorkflowResponse$json = {
  '1': 'ActivateWorkflowResponse',
  '2': [
    {'1': 'workflow', '3': 1, '4': 1, '5': 11, '6': '.workflow.v1.WorkflowDefinition', '10': 'workflow'},
  ],
};

/// Descriptor for `ActivateWorkflowResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List activateWorkflowResponseDescriptor = $convert.base64Decode(
    'ChhBY3RpdmF0ZVdvcmtmbG93UmVzcG9uc2USOwoId29ya2Zsb3cYASABKAsyHy53b3JrZmxvdy'
    '52MS5Xb3JrZmxvd0RlZmluaXRpb25SCHdvcmtmbG93');

@$core.Deprecated('Use archiveWorkflowRequestDescriptor instead')
const ArchiveWorkflowRequest$json = {
  '1': 'ArchiveWorkflowRequest',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
  ],
};

/// Descriptor for `ArchiveWorkflowRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List archiveWorkflowRequestDescriptor = $convert.base64Decode(
    'ChZBcmNoaXZlV29ya2Zsb3dSZXF1ZXN0Eg4KAmlkGAEgASgJUgJpZA==');

@$core.Deprecated('Use archiveWorkflowResponseDescriptor instead')
const ArchiveWorkflowResponse$json = {
  '1': 'ArchiveWorkflowResponse',
  '2': [
    {'1': 'workflow', '3': 1, '4': 1, '5': 11, '6': '.workflow.v1.WorkflowDefinition', '10': 'workflow'},
  ],
};

/// Descriptor for `ArchiveWorkflowResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List archiveWorkflowResponseDescriptor = $convert.base64Decode(
    'ChdBcmNoaXZlV29ya2Zsb3dSZXNwb25zZRI7Cgh3b3JrZmxvdxgBIAEoCzIfLndvcmtmbG93Ln'
    'YxLldvcmtmbG93RGVmaW5pdGlvblIId29ya2Zsb3c=');

const $core.Map<$core.String, $core.dynamic> WorkflowServiceBase$json = {
  '1': 'WorkflowService',
  '2': [
    {'1': 'CreateWorkflow', '2': '.workflow.v1.CreateWorkflowRequest', '3': '.workflow.v1.CreateWorkflowResponse', '4': {}},
    {'1': 'GetWorkflow', '2': '.workflow.v1.GetWorkflowRequest', '3': '.workflow.v1.GetWorkflowResponse', '4': {}},
    {'1': 'ListWorkflows', '2': '.workflow.v1.ListWorkflowsRequest', '3': '.workflow.v1.ListWorkflowsResponse', '4': {}},
    {'1': 'ActivateWorkflow', '2': '.workflow.v1.ActivateWorkflowRequest', '3': '.workflow.v1.ActivateWorkflowResponse', '4': {}},
    {'1': 'ArchiveWorkflow', '2': '.workflow.v1.ArchiveWorkflowRequest', '3': '.workflow.v1.ArchiveWorkflowResponse', '4': {}},
  ],
  '3': {},
};

@$core.Deprecated('Use workflowServiceDescriptor instead')
const $core.Map<$core.String, $core.Map<$core.String, $core.dynamic>> WorkflowServiceBase$messageJson = {
  '.workflow.v1.CreateWorkflowRequest': CreateWorkflowRequest$json,
  '.google.protobuf.Struct': $6.Struct$json,
  '.google.protobuf.Struct.FieldsEntry': $6.Struct_FieldsEntry$json,
  '.google.protobuf.Value': $6.Value$json,
  '.google.protobuf.ListValue': $6.ListValue$json,
  '.workflow.v1.CreateWorkflowResponse': CreateWorkflowResponse$json,
  '.workflow.v1.WorkflowDefinition': WorkflowDefinition$json,
  '.google.protobuf.Timestamp': $2.Timestamp$json,
  '.workflow.v1.GetWorkflowRequest': GetWorkflowRequest$json,
  '.workflow.v1.GetWorkflowResponse': GetWorkflowResponse$json,
  '.workflow.v1.ScheduleDefinition': ScheduleDefinition$json,
  '.workflow.v1.ListWorkflowsRequest': ListWorkflowsRequest$json,
  '.common.v1.SearchRequest': $7.SearchRequest$json,
  '.common.v1.PageCursor': $7.PageCursor$json,
  '.workflow.v1.ListWorkflowsResponse': ListWorkflowsResponse$json,
  '.workflow.v1.ActivateWorkflowRequest': ActivateWorkflowRequest$json,
  '.workflow.v1.ActivateWorkflowResponse': ActivateWorkflowResponse$json,
  '.workflow.v1.ArchiveWorkflowRequest': ArchiveWorkflowRequest$json,
  '.workflow.v1.ArchiveWorkflowResponse': ArchiveWorkflowResponse$json,
};

/// Descriptor for `WorkflowService`. Decode as a `google.protobuf.ServiceDescriptorProto`.
final $typed_data.Uint8List workflowServiceDescriptor = $convert.base64Decode(
    'Cg9Xb3JrZmxvd1NlcnZpY2UScAoOQ3JlYXRlV29ya2Zsb3cSIi53b3JrZmxvdy52MS5DcmVhdG'
    'VXb3JrZmxvd1JlcXVlc3QaIy53b3JrZmxvdy52MS5DcmVhdGVXb3JrZmxvd1Jlc3BvbnNlIhWC'
    'tRgRCg93b3JrZmxvd19tYW5hZ2USZQoLR2V0V29ya2Zsb3cSHy53b3JrZmxvdy52MS5HZXRXb3'
    'JrZmxvd1JlcXVlc3QaIC53b3JrZmxvdy52MS5HZXRXb3JrZmxvd1Jlc3BvbnNlIhOCtRgPCg13'
    'b3JrZmxvd192aWV3EmsKDUxpc3RXb3JrZmxvd3MSIS53b3JrZmxvdy52MS5MaXN0V29ya2Zsb3'
    'dzUmVxdWVzdBoiLndvcmtmbG93LnYxLkxpc3RXb3JrZmxvd3NSZXNwb25zZSITgrUYDwoNd29y'
    'a2Zsb3dfdmlldxJ2ChBBY3RpdmF0ZVdvcmtmbG93EiQud29ya2Zsb3cudjEuQWN0aXZhdGVXb3'
    'JrZmxvd1JlcXVlc3QaJS53b3JrZmxvdy52MS5BY3RpdmF0ZVdvcmtmbG93UmVzcG9uc2UiFYK1'
    'GBEKD3dvcmtmbG93X21hbmFnZRJzCg9BcmNoaXZlV29ya2Zsb3cSIy53b3JrZmxvdy52MS5Bcm'
    'NoaXZlV29ya2Zsb3dSZXF1ZXN0GiQud29ya2Zsb3cudjEuQXJjaGl2ZVdvcmtmbG93UmVzcG9u'
    'c2UiFYK1GBEKD3dvcmtmbG93X21hbmFnZRqLBoK1GIYGChBzZXJ2aWNlX3RydXN0YWdlEgxldm'
    'VudF9pbmdlc3QSDXdvcmtmbG93X3ZpZXcSD3dvcmtmbG93X21hbmFnZRINaW5zdGFuY2Vfdmll'
    'dxIOaW5zdGFuY2VfcmV0cnkSDmV4ZWN1dGlvbl92aWV3Eg9leGVjdXRpb25fcmV0cnkSEGV4ZW'
    'N1dGlvbl9yZXN1bWUSC3NpZ25hbF9zZW5kGo8BCAESDGV2ZW50X2luZ2VzdBINd29ya2Zsb3df'
    'dmlldxIPd29ya2Zsb3dfbWFuYWdlEg1pbnN0YW5jZV92aWV3Eg5pbnN0YW5jZV9yZXRyeRIOZX'
    'hlY3V0aW9uX3ZpZXcSD2V4ZWN1dGlvbl9yZXRyeRIQZXhlY3V0aW9uX3Jlc3VtZRILc2lnbmFs'
    'X3NlbmQajwEIAhIMZXZlbnRfaW5nZXN0Eg13b3JrZmxvd192aWV3Eg93b3JrZmxvd19tYW5hZ2'
    'USDWluc3RhbmNlX3ZpZXcSDmluc3RhbmNlX3JldHJ5Eg5leGVjdXRpb25fdmlldxIPZXhlY3V0'
    'aW9uX3JldHJ5EhBleGVjdXRpb25fcmVzdW1lEgtzaWduYWxfc2VuZBpLCAMSDGV2ZW50X2luZ2'
    'VzdBINd29ya2Zsb3dfdmlldxINaW5zdGFuY2VfdmlldxIOZXhlY3V0aW9uX3ZpZXcSC3NpZ25h'
    'bF9zZW5kGjAIBBINd29ya2Zsb3dfdmlldxINaW5zdGFuY2VfdmlldxIOZXhlY3V0aW9uX3ZpZX'
    'caMAgFEg13b3JrZmxvd192aWV3Eg1pbnN0YW5jZV92aWV3Eg5leGVjdXRpb25fdmlldxqPAQgG'
    'EgxldmVudF9pbmdlc3QSDXdvcmtmbG93X3ZpZXcSD3dvcmtmbG93X21hbmFnZRINaW5zdGFuY2'
    'VfdmlldxIOaW5zdGFuY2VfcmV0cnkSDmV4ZWN1dGlvbl92aWV3Eg9leGVjdXRpb25fcmV0cnkS'
    'EGV4ZWN1dGlvbl9yZXN1bWUSC3NpZ25hbF9zZW5k');

