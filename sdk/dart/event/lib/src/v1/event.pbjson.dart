//
//  Generated code. Do not modify.
//  source: v1/event.proto
//
// @dart = 2.12

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_final_fields
// ignore_for_file: unnecessary_import, unnecessary_this, unused_import

import 'dart:convert' as $convert;
import 'dart:core' as $core;
import 'dart:typed_data' as $typed_data;

import '../google/protobuf/struct.pbjson.dart' as $2;
import '../google/protobuf/timestamp.pbjson.dart' as $3;

@$core.Deprecated('Use eventRecordDescriptor instead')
const EventRecord$json = {
  '1': 'EventRecord',
  '2': [
    {'1': 'event_id', '3': 1, '4': 1, '5': 9, '10': 'eventId'},
    {'1': 'event_type', '3': 2, '4': 1, '5': 9, '10': 'eventType'},
    {'1': 'source', '3': 3, '4': 1, '5': 9, '10': 'source'},
    {'1': 'idempotency_key', '3': 4, '4': 1, '5': 9, '10': 'idempotencyKey'},
    {'1': 'payload', '3': 5, '4': 1, '5': 11, '6': '.google.protobuf.Struct', '10': 'payload'},
  ],
};

/// Descriptor for `EventRecord`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List eventRecordDescriptor = $convert.base64Decode(
    'CgtFdmVudFJlY29yZBIZCghldmVudF9pZBgBIAEoCVIHZXZlbnRJZBIdCgpldmVudF90eXBlGA'
    'IgASgJUglldmVudFR5cGUSFgoGc291cmNlGAMgASgJUgZzb3VyY2USJwoPaWRlbXBvdGVuY3lf'
    'a2V5GAQgASgJUg5pZGVtcG90ZW5jeUtleRIxCgdwYXlsb2FkGAUgASgLMhcuZ29vZ2xlLnByb3'
    'RvYnVmLlN0cnVjdFIHcGF5bG9hZA==');

@$core.Deprecated('Use ingestEventRequestDescriptor instead')
const IngestEventRequest$json = {
  '1': 'IngestEventRequest',
  '2': [
    {'1': 'event_type', '3': 1, '4': 1, '5': 9, '10': 'eventType'},
    {'1': 'source', '3': 2, '4': 1, '5': 9, '10': 'source'},
    {'1': 'idempotency_key', '3': 3, '4': 1, '5': 9, '10': 'idempotencyKey'},
    {'1': 'payload', '3': 4, '4': 1, '5': 11, '6': '.google.protobuf.Struct', '10': 'payload'},
  ],
};

/// Descriptor for `IngestEventRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List ingestEventRequestDescriptor = $convert.base64Decode(
    'ChJJbmdlc3RFdmVudFJlcXVlc3QSHQoKZXZlbnRfdHlwZRgBIAEoCVIJZXZlbnRUeXBlEhYKBn'
    'NvdXJjZRgCIAEoCVIGc291cmNlEicKD2lkZW1wb3RlbmN5X2tleRgDIAEoCVIOaWRlbXBvdGVu'
    'Y3lLZXkSMQoHcGF5bG9hZBgEIAEoCzIXLmdvb2dsZS5wcm90b2J1Zi5TdHJ1Y3RSB3BheWxvYW'
    'Q=');

@$core.Deprecated('Use ingestEventResponseDescriptor instead')
const IngestEventResponse$json = {
  '1': 'IngestEventResponse',
  '2': [
    {'1': 'event', '3': 1, '4': 1, '5': 11, '6': '.event.v1.EventRecord', '10': 'event'},
    {'1': 'idempotent', '3': 2, '4': 1, '5': 8, '10': 'idempotent'},
  ],
};

/// Descriptor for `IngestEventResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List ingestEventResponseDescriptor = $convert.base64Decode(
    'ChNJbmdlc3RFdmVudFJlc3BvbnNlEisKBWV2ZW50GAEgASgLMhUuZXZlbnQudjEuRXZlbnRSZW'
    'NvcmRSBWV2ZW50Eh4KCmlkZW1wb3RlbnQYAiABKAhSCmlkZW1wb3RlbnQ=');

@$core.Deprecated('Use getInstanceTimelineRequestDescriptor instead')
const GetInstanceTimelineRequest$json = {
  '1': 'GetInstanceTimelineRequest',
  '2': [
    {'1': 'instance_id', '3': 1, '4': 1, '5': 9, '10': 'instanceId'},
  ],
};

/// Descriptor for `GetInstanceTimelineRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getInstanceTimelineRequestDescriptor = $convert.base64Decode(
    'ChpHZXRJbnN0YW5jZVRpbWVsaW5lUmVxdWVzdBIfCgtpbnN0YW5jZV9pZBgBIAEoCVIKaW5zdG'
    'FuY2VJZA==');

@$core.Deprecated('Use timelineEntryDescriptor instead')
const TimelineEntry$json = {
  '1': 'TimelineEntry',
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

/// Descriptor for `TimelineEntry`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List timelineEntryDescriptor = $convert.base64Decode(
    'Cg1UaW1lbGluZUVudHJ5Eh0KCmV2ZW50X3R5cGUYASABKAlSCWV2ZW50VHlwZRIUCgVzdGF0ZR'
    'gCIAEoCVIFc3RhdGUSHQoKZnJvbV9zdGF0ZRgDIAEoCVIJZnJvbVN0YXRlEhkKCHRvX3N0YXRl'
    'GAQgASgJUgd0b1N0YXRlEiEKDGV4ZWN1dGlvbl9pZBgFIAEoCVILZXhlY3V0aW9uSWQSGQoIdH'
    'JhY2VfaWQYBiABKAlSB3RyYWNlSWQSMQoHcGF5bG9hZBgHIAEoCzIXLmdvb2dsZS5wcm90b2J1'
    'Zi5TdHJ1Y3RSB3BheWxvYWQSOQoKY3JlYXRlZF9hdBgIIAEoCzIaLmdvb2dsZS5wcm90b2J1Zi'
    '5UaW1lc3RhbXBSCWNyZWF0ZWRBdA==');

@$core.Deprecated('Use getInstanceTimelineResponseDescriptor instead')
const GetInstanceTimelineResponse$json = {
  '1': 'GetInstanceTimelineResponse',
  '2': [
    {'1': 'items', '3': 1, '4': 3, '5': 11, '6': '.event.v1.TimelineEntry', '10': 'items'},
  ],
};

/// Descriptor for `GetInstanceTimelineResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List getInstanceTimelineResponseDescriptor = $convert.base64Decode(
    'ChtHZXRJbnN0YW5jZVRpbWVsaW5lUmVzcG9uc2USLQoFaXRlbXMYASADKAsyFy5ldmVudC52MS'
    '5UaW1lbGluZUVudHJ5UgVpdGVtcw==');

const $core.Map<$core.String, $core.dynamic> EventServiceBase$json = {
  '1': 'EventService',
  '2': [
    {'1': 'IngestEvent', '2': '.event.v1.IngestEventRequest', '3': '.event.v1.IngestEventResponse', '4': {}},
    {'1': 'GetInstanceTimeline', '2': '.event.v1.GetInstanceTimelineRequest', '3': '.event.v1.GetInstanceTimelineResponse', '4': {}},
  ],
  '3': {},
};

@$core.Deprecated('Use eventServiceDescriptor instead')
const $core.Map<$core.String, $core.Map<$core.String, $core.dynamic>> EventServiceBase$messageJson = {
  '.event.v1.IngestEventRequest': IngestEventRequest$json,
  '.google.protobuf.Struct': $2.Struct$json,
  '.google.protobuf.Struct.FieldsEntry': $2.Struct_FieldsEntry$json,
  '.google.protobuf.Value': $2.Value$json,
  '.google.protobuf.ListValue': $2.ListValue$json,
  '.event.v1.IngestEventResponse': IngestEventResponse$json,
  '.event.v1.EventRecord': EventRecord$json,
  '.event.v1.GetInstanceTimelineRequest': GetInstanceTimelineRequest$json,
  '.event.v1.GetInstanceTimelineResponse': GetInstanceTimelineResponse$json,
  '.event.v1.TimelineEntry': TimelineEntry$json,
  '.google.protobuf.Timestamp': $3.Timestamp$json,
};

/// Descriptor for `EventService`. Decode as a `google.protobuf.ServiceDescriptorProto`.
final $typed_data.Uint8List eventServiceDescriptor = $convert.base64Decode(
    'CgxFdmVudFNlcnZpY2USXgoLSW5nZXN0RXZlbnQSHC5ldmVudC52MS5Jbmdlc3RFdmVudFJlcX'
    'Vlc3QaHS5ldmVudC52MS5Jbmdlc3RFdmVudFJlc3BvbnNlIhKCtRgOCgxldmVudF9pbmdlc3QS'
    'dwoTR2V0SW5zdGFuY2VUaW1lbGluZRIkLmV2ZW50LnYxLkdldEluc3RhbmNlVGltZWxpbmVSZX'
    'F1ZXN0GiUuZXZlbnQudjEuR2V0SW5zdGFuY2VUaW1lbGluZVJlc3BvbnNlIhOCtRgPCg1pbnN0'
    'YW5jZV92aWV3GosGgrUYhgYKEHNlcnZpY2VfdHJ1c3RhZ2USDGV2ZW50X2luZ2VzdBINd29ya2'
    'Zsb3dfdmlldxIPd29ya2Zsb3dfbWFuYWdlEg1pbnN0YW5jZV92aWV3Eg5pbnN0YW5jZV9yZXRy'
    'eRIOZXhlY3V0aW9uX3ZpZXcSD2V4ZWN1dGlvbl9yZXRyeRIQZXhlY3V0aW9uX3Jlc3VtZRILc2'
    'lnbmFsX3NlbmQajwEIARIMZXZlbnRfaW5nZXN0Eg13b3JrZmxvd192aWV3Eg93b3JrZmxvd19t'
    'YW5hZ2USDWluc3RhbmNlX3ZpZXcSDmluc3RhbmNlX3JldHJ5Eg5leGVjdXRpb25fdmlldxIPZX'
    'hlY3V0aW9uX3JldHJ5EhBleGVjdXRpb25fcmVzdW1lEgtzaWduYWxfc2VuZBqPAQgCEgxldmVu'
    'dF9pbmdlc3QSDXdvcmtmbG93X3ZpZXcSD3dvcmtmbG93X21hbmFnZRINaW5zdGFuY2Vfdmlldx'
    'IOaW5zdGFuY2VfcmV0cnkSDmV4ZWN1dGlvbl92aWV3Eg9leGVjdXRpb25fcmV0cnkSEGV4ZWN1'
    'dGlvbl9yZXN1bWUSC3NpZ25hbF9zZW5kGksIAxIMZXZlbnRfaW5nZXN0Eg13b3JrZmxvd192aW'
    'V3Eg1pbnN0YW5jZV92aWV3Eg5leGVjdXRpb25fdmlldxILc2lnbmFsX3NlbmQaMAgEEg13b3Jr'
    'Zmxvd192aWV3Eg1pbnN0YW5jZV92aWV3Eg5leGVjdXRpb25fdmlldxowCAUSDXdvcmtmbG93X3'
    'ZpZXcSDWluc3RhbmNlX3ZpZXcSDmV4ZWN1dGlvbl92aWV3Go8BCAYSDGV2ZW50X2luZ2VzdBIN'
    'd29ya2Zsb3dfdmlldxIPd29ya2Zsb3dfbWFuYWdlEg1pbnN0YW5jZV92aWV3Eg5pbnN0YW5jZV'
    '9yZXRyeRIOZXhlY3V0aW9uX3ZpZXcSD2V4ZWN1dGlvbl9yZXRyeRIQZXhlY3V0aW9uX3Jlc3Vt'
    'ZRILc2lnbmFsX3NlbmQ=');

