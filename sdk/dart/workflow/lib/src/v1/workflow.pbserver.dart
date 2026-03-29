//
//  Generated code. Do not modify.
//  source: v1/workflow.proto
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

import 'workflow.pb.dart' as $11;
import 'workflow.pbjson.dart';

export 'workflow.pb.dart';

abstract class WorkflowServiceBase extends $pb.GeneratedService {
  $async.Future<$11.CreateWorkflowResponse> createWorkflow($pb.ServerContext ctx, $11.CreateWorkflowRequest request);
  $async.Future<$11.GetWorkflowResponse> getWorkflow($pb.ServerContext ctx, $11.GetWorkflowRequest request);
  $async.Future<$11.ListWorkflowsResponse> listWorkflows($pb.ServerContext ctx, $11.ListWorkflowsRequest request);
  $async.Future<$11.ActivateWorkflowResponse> activateWorkflow($pb.ServerContext ctx, $11.ActivateWorkflowRequest request);

  $pb.GeneratedMessage createRequest($core.String methodName) {
    switch (methodName) {
      case 'CreateWorkflow': return $11.CreateWorkflowRequest();
      case 'GetWorkflow': return $11.GetWorkflowRequest();
      case 'ListWorkflows': return $11.ListWorkflowsRequest();
      case 'ActivateWorkflow': return $11.ActivateWorkflowRequest();
      default: throw $core.ArgumentError('Unknown method: $methodName');
    }
  }

  $async.Future<$pb.GeneratedMessage> handleCall($pb.ServerContext ctx, $core.String methodName, $pb.GeneratedMessage request) {
    switch (methodName) {
      case 'CreateWorkflow': return this.createWorkflow(ctx, request as $11.CreateWorkflowRequest);
      case 'GetWorkflow': return this.getWorkflow(ctx, request as $11.GetWorkflowRequest);
      case 'ListWorkflows': return this.listWorkflows(ctx, request as $11.ListWorkflowsRequest);
      case 'ActivateWorkflow': return this.activateWorkflow(ctx, request as $11.ActivateWorkflowRequest);
      default: throw $core.ArgumentError('Unknown method: $methodName');
    }
  }

  $core.Map<$core.String, $core.dynamic> get $json => WorkflowServiceBase$json;
  $core.Map<$core.String, $core.Map<$core.String, $core.dynamic>> get $messageJson => WorkflowServiceBase$messageJson;
}

