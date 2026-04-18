//
//  Generated code. Do not modify.
//  source: v1/runtime.proto
//
// @dart = 2.12

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_final_fields
// ignore_for_file: unnecessary_import, unnecessary_this, unused_import

import 'dart:async' as $async;
import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

import 'runtime.pb.dart' as $9;
import 'runtime.pbjson.dart';

export 'runtime.pb.dart';

abstract class RuntimeServiceBase extends $pb.GeneratedService {
  $async.Future<$9.ListInstancesResponse> listInstances($pb.ServerContext ctx, $9.ListInstancesRequest request);
  $async.Future<$9.RetryInstanceResponse> retryInstance($pb.ServerContext ctx, $9.RetryInstanceRequest request);
  $async.Future<$9.ListExecutionsResponse> listExecutions($pb.ServerContext ctx, $9.ListExecutionsRequest request);
  $async.Future<$9.GetExecutionResponse> getExecution($pb.ServerContext ctx, $9.GetExecutionRequest request);
  $async.Future<$9.RetryExecutionResponse> retryExecution($pb.ServerContext ctx, $9.RetryExecutionRequest request);
  $async.Future<$9.ResumeExecutionResponse> resumeExecution($pb.ServerContext ctx, $9.ResumeExecutionRequest request);
  $async.Future<$9.GetInstanceRunResponse> getInstanceRun($pb.ServerContext ctx, $9.GetInstanceRunRequest request);

  $pb.GeneratedMessage createRequest($core.String methodName) {
    switch (methodName) {
      case 'ListInstances': return $9.ListInstancesRequest();
      case 'RetryInstance': return $9.RetryInstanceRequest();
      case 'ListExecutions': return $9.ListExecutionsRequest();
      case 'GetExecution': return $9.GetExecutionRequest();
      case 'RetryExecution': return $9.RetryExecutionRequest();
      case 'ResumeExecution': return $9.ResumeExecutionRequest();
      case 'GetInstanceRun': return $9.GetInstanceRunRequest();
      default: throw $core.ArgumentError('Unknown method: $methodName');
    }
  }

  $async.Future<$pb.GeneratedMessage> handleCall($pb.ServerContext ctx, $core.String methodName, $pb.GeneratedMessage request) {
    switch (methodName) {
      case 'ListInstances': return this.listInstances(ctx, request as $9.ListInstancesRequest);
      case 'RetryInstance': return this.retryInstance(ctx, request as $9.RetryInstanceRequest);
      case 'ListExecutions': return this.listExecutions(ctx, request as $9.ListExecutionsRequest);
      case 'GetExecution': return this.getExecution(ctx, request as $9.GetExecutionRequest);
      case 'RetryExecution': return this.retryExecution(ctx, request as $9.RetryExecutionRequest);
      case 'ResumeExecution': return this.resumeExecution(ctx, request as $9.ResumeExecutionRequest);
      case 'GetInstanceRun': return this.getInstanceRun(ctx, request as $9.GetInstanceRunRequest);
      default: throw $core.ArgumentError('Unknown method: $methodName');
    }
  }

  $core.Map<$core.String, $core.dynamic> get $json => RuntimeServiceBase$json;
  $core.Map<$core.String, $core.Map<$core.String, $core.dynamic>> get $messageJson => RuntimeServiceBase$messageJson;
}

