// This is a generated file - do not edit.
//
// Generated from v1/workflow.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:async' as $async;
import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

import 'workflow.pb.dart' as $3;
import 'workflow.pbjson.dart';

export 'workflow.pb.dart';

abstract class WorkflowServiceBase extends $pb.GeneratedService {
  $async.Future<$3.CreateWorkflowResponse> createWorkflow(
      $pb.ServerContext ctx, $3.CreateWorkflowRequest request);
  $async.Future<$3.GetWorkflowResponse> getWorkflow(
      $pb.ServerContext ctx, $3.GetWorkflowRequest request);
  $async.Future<$3.ListWorkflowsResponse> listWorkflows(
      $pb.ServerContext ctx, $3.ListWorkflowsRequest request);
  $async.Future<$3.ActivateWorkflowResponse> activateWorkflow(
      $pb.ServerContext ctx, $3.ActivateWorkflowRequest request);
  $async.Future<$3.ArchiveWorkflowResponse> archiveWorkflow(
      $pb.ServerContext ctx, $3.ArchiveWorkflowRequest request);

  $pb.GeneratedMessage createRequest($core.String methodName) {
    switch (methodName) {
      case 'CreateWorkflow':
        return $3.CreateWorkflowRequest();
      case 'GetWorkflow':
        return $3.GetWorkflowRequest();
      case 'ListWorkflows':
        return $3.ListWorkflowsRequest();
      case 'ActivateWorkflow':
        return $3.ActivateWorkflowRequest();
      case 'ArchiveWorkflow':
        return $3.ArchiveWorkflowRequest();
      default:
        throw $core.ArgumentError('Unknown method: $methodName');
    }
  }

  $async.Future<$pb.GeneratedMessage> handleCall($pb.ServerContext ctx,
      $core.String methodName, $pb.GeneratedMessage request) {
    switch (methodName) {
      case 'CreateWorkflow':
        return createWorkflow(ctx, request as $3.CreateWorkflowRequest);
      case 'GetWorkflow':
        return getWorkflow(ctx, request as $3.GetWorkflowRequest);
      case 'ListWorkflows':
        return listWorkflows(ctx, request as $3.ListWorkflowsRequest);
      case 'ActivateWorkflow':
        return activateWorkflow(ctx, request as $3.ActivateWorkflowRequest);
      case 'ArchiveWorkflow':
        return archiveWorkflow(ctx, request as $3.ArchiveWorkflowRequest);
      default:
        throw $core.ArgumentError('Unknown method: $methodName');
    }
  }

  $core.Map<$core.String, $core.dynamic> get $json => WorkflowServiceBase$json;
  $core.Map<$core.String, $core.Map<$core.String, $core.dynamic>>
      get $messageJson => WorkflowServiceBase$messageJson;
}
