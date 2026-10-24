// This is a generated file - do not edit.
//
// Generated from v1/signal.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:async' as $async;
import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;
import 'package:protobuf/well_known_types/google/protobuf/struct.pb.dart' as $0;

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

class SendSignalRequest extends $pb.GeneratedMessage {
  factory SendSignalRequest({
    $core.String? instanceId,
    $core.String? signalName,
    $0.Struct? payload,
  }) {
    final result = SendSignalRequest._();
    if (instanceId != null) result.instanceId = instanceId;
    if (signalName != null) result.signalName = signalName;
    if (payload != null) result.payload = payload;
    return result;
  }

  SendSignalRequest._();

  factory SendSignalRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      SendSignalRequest()..mergeFromBuffer(data, registry);
  factory SendSignalRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      SendSignalRequest()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SendSignalRequest',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'signal.v1'),
      createEmptyInstance: SendSignalRequest.$_createMessage)
    ..aOS(1, _omitFieldNames ? '' : 'instanceId')
    ..aOS(2, _omitFieldNames ? '' : 'signalName')
    ..aOM<$0.Struct>(3, _omitFieldNames ? '' : 'payload',
        subBuilder: $0.Struct.$_createMessage)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SendSignalRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SendSignalRequest copyWith(void Function(SendSignalRequest) updates) =>
      super.copyWith((message) => updates(message as SendSignalRequest))
          as SendSignalRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use SendSignalRequest() / SendSignalRequest.new instead')
  static SendSignalRequest create() => SendSignalRequest._();
  static $pb.GeneratedMessage $_createMessage() => SendSignalRequest._();
  @$core.override
  SendSignalRequest createEmptyInstance() => SendSignalRequest._();
  @$core.pragma('dart2js:noInline')
  static SendSignalRequest getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<SendSignalRequest>(
          SendSignalRequest.$_createMessage);
  static SendSignalRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get instanceId => $_getSZ(0);
  @$pb.TagNumber(1)
  set instanceId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasInstanceId() => $_has(0);
  @$pb.TagNumber(1)
  void clearInstanceId() => $_clearField(1);

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
}

class SendSignalResponse extends $pb.GeneratedMessage {
  factory SendSignalResponse({
    $core.bool? delivered,
  }) {
    final result = SendSignalResponse._();
    if (delivered != null) result.delivered = delivered;
    return result;
  }

  SendSignalResponse._();

  factory SendSignalResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      SendSignalResponse()..mergeFromBuffer(data, registry);
  factory SendSignalResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      SendSignalResponse()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SendSignalResponse',
      package: const $pb.PackageName(_omitMessageNames ? '' : 'signal.v1'),
      createEmptyInstance: SendSignalResponse.$_createMessage)
    ..aOB(1, _omitFieldNames ? '' : 'delivered')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SendSignalResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SendSignalResponse copyWith(void Function(SendSignalResponse) updates) =>
      super.copyWith((message) => updates(message as SendSignalResponse))
          as SendSignalResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  @$core.Deprecated('Use SendSignalResponse() / SendSignalResponse.new instead')
  static SendSignalResponse create() => SendSignalResponse._();
  static $pb.GeneratedMessage $_createMessage() => SendSignalResponse._();
  @$core.override
  SendSignalResponse createEmptyInstance() => SendSignalResponse._();
  @$core.pragma('dart2js:noInline')
  static SendSignalResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SendSignalResponse>(
          SendSignalResponse.$_createMessage);
  static SendSignalResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get delivered => $_getBF(0);
  @$pb.TagNumber(1)
  set delivered($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasDelivered() => $_has(0);
  @$pb.TagNumber(1)
  void clearDelivered() => $_clearField(1);
}

class SignalServiceApi {
  final $pb.RpcClient _client;

  SignalServiceApi(this._client);

  $async.Future<SendSignalResponse> sendSignal(
          $pb.ClientContext? ctx, SendSignalRequest request) =>
      _client.invoke<SendSignalResponse>(
          ctx, 'SignalService', 'SendSignal', request, SendSignalResponse());
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
