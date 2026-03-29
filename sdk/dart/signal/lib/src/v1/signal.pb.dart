//
//  Generated code. Do not modify.
//  source: v1/signal.proto
//
// @dart = 2.12

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_final_fields
// ignore_for_file: unnecessary_import, unnecessary_this, unused_import

import 'dart:async' as $async;
import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

import '../google/protobuf/struct.pb.dart' as $2;

class SendSignalRequest extends $pb.GeneratedMessage {
  factory SendSignalRequest({
    $core.String? instanceId,
    $core.String? signalName,
    $2.Struct? payload,
  }) {
    final $result = create();
    if (instanceId != null) {
      $result.instanceId = instanceId;
    }
    if (signalName != null) {
      $result.signalName = signalName;
    }
    if (payload != null) {
      $result.payload = payload;
    }
    return $result;
  }
  SendSignalRequest._() : super();
  factory SendSignalRequest.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory SendSignalRequest.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'SendSignalRequest', package: const $pb.PackageName(_omitMessageNames ? '' : 'signal.v1'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'instanceId')
    ..aOS(2, _omitFieldNames ? '' : 'signalName')
    ..aOM<$2.Struct>(3, _omitFieldNames ? '' : 'payload', subBuilder: $2.Struct.create)
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  SendSignalRequest clone() => SendSignalRequest()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  SendSignalRequest copyWith(void Function(SendSignalRequest) updates) => super.copyWith((message) => updates(message as SendSignalRequest)) as SendSignalRequest;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SendSignalRequest create() => SendSignalRequest._();
  SendSignalRequest createEmptyInstance() => create();
  static $pb.PbList<SendSignalRequest> createRepeated() => $pb.PbList<SendSignalRequest>();
  @$core.pragma('dart2js:noInline')
  static SendSignalRequest getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<SendSignalRequest>(create);
  static SendSignalRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get instanceId => $_getSZ(0);
  @$pb.TagNumber(1)
  set instanceId($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasInstanceId() => $_has(0);
  @$pb.TagNumber(1)
  void clearInstanceId() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get signalName => $_getSZ(1);
  @$pb.TagNumber(2)
  set signalName($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasSignalName() => $_has(1);
  @$pb.TagNumber(2)
  void clearSignalName() => clearField(2);

  @$pb.TagNumber(3)
  $2.Struct get payload => $_getN(2);
  @$pb.TagNumber(3)
  set payload($2.Struct v) { setField(3, v); }
  @$pb.TagNumber(3)
  $core.bool hasPayload() => $_has(2);
  @$pb.TagNumber(3)
  void clearPayload() => clearField(3);
  @$pb.TagNumber(3)
  $2.Struct ensurePayload() => $_ensure(2);
}

class SendSignalResponse extends $pb.GeneratedMessage {
  factory SendSignalResponse({
    $core.bool? delivered,
  }) {
    final $result = create();
    if (delivered != null) {
      $result.delivered = delivered;
    }
    return $result;
  }
  SendSignalResponse._() : super();
  factory SendSignalResponse.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory SendSignalResponse.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'SendSignalResponse', package: const $pb.PackageName(_omitMessageNames ? '' : 'signal.v1'), createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'delivered')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  SendSignalResponse clone() => SendSignalResponse()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  SendSignalResponse copyWith(void Function(SendSignalResponse) updates) => super.copyWith((message) => updates(message as SendSignalResponse)) as SendSignalResponse;

  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SendSignalResponse create() => SendSignalResponse._();
  SendSignalResponse createEmptyInstance() => create();
  static $pb.PbList<SendSignalResponse> createRepeated() => $pb.PbList<SendSignalResponse>();
  @$core.pragma('dart2js:noInline')
  static SendSignalResponse getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<SendSignalResponse>(create);
  static SendSignalResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get delivered => $_getBF(0);
  @$pb.TagNumber(1)
  set delivered($core.bool v) { $_setBool(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasDelivered() => $_has(0);
  @$pb.TagNumber(1)
  void clearDelivered() => clearField(1);
}

class SignalServiceApi {
  $pb.RpcClient _client;
  SignalServiceApi(this._client);

  $async.Future<SendSignalResponse> sendSignal($pb.ClientContext? ctx, SendSignalRequest request) =>
    _client.invoke<SendSignalResponse>(ctx, 'SignalService', 'SendSignal', request, SendSignalResponse())
  ;
}


const _omitFieldNames = $core.bool.fromEnvironment('protobuf.omit_field_names');
const _omitMessageNames = $core.bool.fromEnvironment('protobuf.omit_message_names');
