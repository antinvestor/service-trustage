//
//  Generated code. Do not modify.
//  source: v1/signal.proto
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

@$core.Deprecated('Use sendSignalRequestDescriptor instead')
const SendSignalRequest$json = {
  '1': 'SendSignalRequest',
  '2': [
    {'1': 'instance_id', '3': 1, '4': 1, '5': 9, '10': 'instanceId'},
    {'1': 'signal_name', '3': 2, '4': 1, '5': 9, '10': 'signalName'},
    {'1': 'payload', '3': 3, '4': 1, '5': 11, '6': '.google.protobuf.Struct', '10': 'payload'},
  ],
};

/// Descriptor for `SendSignalRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List sendSignalRequestDescriptor = $convert.base64Decode(
    'ChFTZW5kU2lnbmFsUmVxdWVzdBIfCgtpbnN0YW5jZV9pZBgBIAEoCVIKaW5zdGFuY2VJZBIfCg'
    'tzaWduYWxfbmFtZRgCIAEoCVIKc2lnbmFsTmFtZRIxCgdwYXlsb2FkGAMgASgLMhcuZ29vZ2xl'
    'LnByb3RvYnVmLlN0cnVjdFIHcGF5bG9hZA==');

@$core.Deprecated('Use sendSignalResponseDescriptor instead')
const SendSignalResponse$json = {
  '1': 'SendSignalResponse',
  '2': [
    {'1': 'delivered', '3': 1, '4': 1, '5': 8, '10': 'delivered'},
  ],
};

/// Descriptor for `SendSignalResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List sendSignalResponseDescriptor = $convert.base64Decode(
    'ChJTZW5kU2lnbmFsUmVzcG9uc2USHAoJZGVsaXZlcmVkGAEgASgIUglkZWxpdmVyZWQ=');

const $core.Map<$core.String, $core.dynamic> SignalServiceBase$json = {
  '1': 'SignalService',
  '2': [
    {'1': 'SendSignal', '2': '.signal.v1.SendSignalRequest', '3': '.signal.v1.SendSignalResponse', '4': {}},
  ],
  '3': {},
};

@$core.Deprecated('Use signalServiceDescriptor instead')
const $core.Map<$core.String, $core.Map<$core.String, $core.dynamic>> SignalServiceBase$messageJson = {
  '.signal.v1.SendSignalRequest': SendSignalRequest$json,
  '.google.protobuf.Struct': $2.Struct$json,
  '.google.protobuf.Struct.FieldsEntry': $2.Struct_FieldsEntry$json,
  '.google.protobuf.Value': $2.Value$json,
  '.google.protobuf.ListValue': $2.ListValue$json,
  '.signal.v1.SendSignalResponse': SendSignalResponse$json,
};

/// Descriptor for `SignalService`. Decode as a `google.protobuf.ServiceDescriptorProto`.
final $typed_data.Uint8List signalServiceDescriptor = $convert.base64Decode(
    'Cg1TaWduYWxTZXJ2aWNlElwKClNlbmRTaWduYWwSHC5zaWduYWwudjEuU2VuZFNpZ25hbFJlcX'
    'Vlc3QaHS5zaWduYWwudjEuU2VuZFNpZ25hbFJlc3BvbnNlIhGCtRgNCgtzaWduYWxfc2VuZBqL'
    'BoK1GIYGChBzZXJ2aWNlX3RydXN0YWdlEgxldmVudF9pbmdlc3QSDXdvcmtmbG93X3ZpZXcSD3'
    'dvcmtmbG93X21hbmFnZRINaW5zdGFuY2VfdmlldxIOaW5zdGFuY2VfcmV0cnkSDmV4ZWN1dGlv'
    'bl92aWV3Eg9leGVjdXRpb25fcmV0cnkSEGV4ZWN1dGlvbl9yZXN1bWUSC3NpZ25hbF9zZW5kGo'
    '8BCAESDGV2ZW50X2luZ2VzdBINd29ya2Zsb3dfdmlldxIPd29ya2Zsb3dfbWFuYWdlEg1pbnN0'
    'YW5jZV92aWV3Eg5pbnN0YW5jZV9yZXRyeRIOZXhlY3V0aW9uX3ZpZXcSD2V4ZWN1dGlvbl9yZX'
    'RyeRIQZXhlY3V0aW9uX3Jlc3VtZRILc2lnbmFsX3NlbmQajwEIAhIMZXZlbnRfaW5nZXN0Eg13'
    'b3JrZmxvd192aWV3Eg93b3JrZmxvd19tYW5hZ2USDWluc3RhbmNlX3ZpZXcSDmluc3RhbmNlX3'
    'JldHJ5Eg5leGVjdXRpb25fdmlldxIPZXhlY3V0aW9uX3JldHJ5EhBleGVjdXRpb25fcmVzdW1l'
    'EgtzaWduYWxfc2VuZBpLCAMSDGV2ZW50X2luZ2VzdBINd29ya2Zsb3dfdmlldxINaW5zdGFuY2'
    'VfdmlldxIOZXhlY3V0aW9uX3ZpZXcSC3NpZ25hbF9zZW5kGjAIBBINd29ya2Zsb3dfdmlldxIN'
    'aW5zdGFuY2VfdmlldxIOZXhlY3V0aW9uX3ZpZXcaMAgFEg13b3JrZmxvd192aWV3Eg1pbnN0YW'
    '5jZV92aWV3Eg5leGVjdXRpb25fdmlldxqPAQgGEgxldmVudF9pbmdlc3QSDXdvcmtmbG93X3Zp'
    'ZXcSD3dvcmtmbG93X21hbmFnZRINaW5zdGFuY2VfdmlldxIOaW5zdGFuY2VfcmV0cnkSDmV4ZW'
    'N1dGlvbl92aWV3Eg9leGVjdXRpb25fcmV0cnkSEGV4ZWN1dGlvbl9yZXN1bWUSC3NpZ25hbF9z'
    'ZW5k');

