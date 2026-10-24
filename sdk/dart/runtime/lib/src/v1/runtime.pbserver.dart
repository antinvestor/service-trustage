// This is a generated file - do not edit.
//
// Generated from v1/runtime.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:async' as $async;
import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

import 'runtime.pb.dart' as $3;
import 'runtime.pbjson.dart';

export 'runtime.pb.dart';

abstract class RuntimeServiceBase extends $pb.GeneratedService {
  $async.Future<$3.ListInstancesResponse> listInstances(
      $pb.ServerContext ctx, $3.ListInstancesRequest request);
  $async.Future<$3.RetryInstanceResponse> retryInstance(
      $pb.ServerContext ctx, $3.RetryInstanceRequest request);
  $async.Future<$3.ListExecutionsResponse> listExecutions(
      $pb.ServerContext ctx, $3.ListExecutionsRequest request);
  $async.Future<$3.GetExecutionResponse> getExecution(
      $pb.ServerContext ctx, $3.GetExecutionRequest request);
  $async.Future<$3.RetryExecutionResponse> retryExecution(
      $pb.ServerContext ctx, $3.RetryExecutionRequest request);
  $async.Future<$3.ResumeExecutionResponse> resumeExecution(
      $pb.ServerContext ctx, $3.ResumeExecutionRequest request);
  $async.Future<$3.GetInstanceRunResponse> getInstanceRun(
      $pb.ServerContext ctx, $3.GetInstanceRunRequest request);

  $pb.GeneratedMessage createRequest($core.String methodName) {
    switch (methodName) {
      case 'ListInstances':
        return $3.ListInstancesRequest();
      case 'RetryInstance':
        return $3.RetryInstanceRequest();
      case 'ListExecutions':
        return $3.ListExecutionsRequest();
      case 'GetExecution':
        return $3.GetExecutionRequest();
      case 'RetryExecution':
        return $3.RetryExecutionRequest();
      case 'ResumeExecution':
        return $3.ResumeExecutionRequest();
      case 'GetInstanceRun':
        return $3.GetInstanceRunRequest();
      default:
        throw $core.ArgumentError('Unknown method: $methodName');
    }
  }

  $async.Future<$pb.GeneratedMessage> handleCall($pb.ServerContext ctx,
      $core.String methodName, $pb.GeneratedMessage request) {
    switch (methodName) {
      case 'ListInstances':
        return listInstances(ctx, request as $3.ListInstancesRequest);
      case 'RetryInstance':
        return retryInstance(ctx, request as $3.RetryInstanceRequest);
      case 'ListExecutions':
        return listExecutions(ctx, request as $3.ListExecutionsRequest);
      case 'GetExecution':
        return getExecution(ctx, request as $3.GetExecutionRequest);
      case 'RetryExecution':
        return retryExecution(ctx, request as $3.RetryExecutionRequest);
      case 'ResumeExecution':
        return resumeExecution(ctx, request as $3.ResumeExecutionRequest);
      case 'GetInstanceRun':
        return getInstanceRun(ctx, request as $3.GetInstanceRunRequest);
      default:
        throw $core.ArgumentError('Unknown method: $methodName');
    }
  }

  $core.Map<$core.String, $core.dynamic> get $json => RuntimeServiceBase$json;
  $core.Map<$core.String, $core.Map<$core.String, $core.dynamic>>
      get $messageJson => RuntimeServiceBase$messageJson;
}
