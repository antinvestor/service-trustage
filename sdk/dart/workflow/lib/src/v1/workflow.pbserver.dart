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

import 'workflow.pb.dart' as $8;
import 'workflow.pbjson.dart';

export 'workflow.pb.dart';

abstract class WorkflowServiceBase extends $pb.GeneratedService {
  $async.Future<$8.CreateWorkflowResponse> createWorkflow($pb.ServerContext ctx, $8.CreateWorkflowRequest request);
  $async.Future<$8.GetWorkflowResponse> getWorkflow($pb.ServerContext ctx, $8.GetWorkflowRequest request);
  $async.Future<$8.ListWorkflowsResponse> listWorkflows($pb.ServerContext ctx, $8.ListWorkflowsRequest request);
  $async.Future<$8.ActivateWorkflowResponse> activateWorkflow($pb.ServerContext ctx, $8.ActivateWorkflowRequest request);
  $async.Future<$8.ArchiveWorkflowResponse> archiveWorkflow($pb.ServerContext ctx, $8.ArchiveWorkflowRequest request);

  $pb.GeneratedMessage createRequest($core.String methodName) {
    switch (methodName) {
      case 'CreateWorkflow': return $8.CreateWorkflowRequest();
      case 'GetWorkflow': return $8.GetWorkflowRequest();
      case 'ListWorkflows': return $8.ListWorkflowsRequest();
      case 'ActivateWorkflow': return $8.ActivateWorkflowRequest();
      case 'ArchiveWorkflow': return $8.ArchiveWorkflowRequest();
      default: throw $core.ArgumentError('Unknown method: $methodName');
    }
  }

  $async.Future<$pb.GeneratedMessage> handleCall($pb.ServerContext ctx, $core.String methodName, $pb.GeneratedMessage request) {
    switch (methodName) {
      case 'CreateWorkflow': return this.createWorkflow(ctx, request as $8.CreateWorkflowRequest);
      case 'GetWorkflow': return this.getWorkflow(ctx, request as $8.GetWorkflowRequest);
      case 'ListWorkflows': return this.listWorkflows(ctx, request as $8.ListWorkflowsRequest);
      case 'ActivateWorkflow': return this.activateWorkflow(ctx, request as $8.ActivateWorkflowRequest);
      case 'ArchiveWorkflow': return this.archiveWorkflow(ctx, request as $8.ArchiveWorkflowRequest);
      default: throw $core.ArgumentError('Unknown method: $methodName');
    }
  }

  $core.Map<$core.String, $core.dynamic> get $json => WorkflowServiceBase$json;
  $core.Map<$core.String, $core.Map<$core.String, $core.dynamic>> get $messageJson => WorkflowServiceBase$messageJson;
}

