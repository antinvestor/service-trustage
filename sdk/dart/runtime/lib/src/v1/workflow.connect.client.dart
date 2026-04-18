//
//  Generated code. Do not modify.
//  source: v1/workflow.proto
//

import "package:connectrpc/connect.dart" as connect;
import "workflow.pb.dart" as v1workflow;
import "workflow.connect.spec.dart" as specs;

extension type WorkflowServiceClient (connect.Transport _transport) {
  Future<v1workflow.CreateWorkflowResponse> createWorkflow(
    v1workflow.CreateWorkflowRequest input, {
    connect.Headers? headers,
    connect.AbortSignal? signal,
    Function(connect.Headers)? onHeader,
    Function(connect.Headers)? onTrailer,
  }) {
    return connect.Client(_transport).unary(
      specs.WorkflowService.createWorkflow,
      input,
      signal: signal,
      headers: headers,
      onHeader: onHeader,
      onTrailer: onTrailer,
    );
  }

  Future<v1workflow.GetWorkflowResponse> getWorkflow(
    v1workflow.GetWorkflowRequest input, {
    connect.Headers? headers,
    connect.AbortSignal? signal,
    Function(connect.Headers)? onHeader,
    Function(connect.Headers)? onTrailer,
  }) {
    return connect.Client(_transport).unary(
      specs.WorkflowService.getWorkflow,
      input,
      signal: signal,
      headers: headers,
      onHeader: onHeader,
      onTrailer: onTrailer,
    );
  }

  Future<v1workflow.ListWorkflowsResponse> listWorkflows(
    v1workflow.ListWorkflowsRequest input, {
    connect.Headers? headers,
    connect.AbortSignal? signal,
    Function(connect.Headers)? onHeader,
    Function(connect.Headers)? onTrailer,
  }) {
    return connect.Client(_transport).unary(
      specs.WorkflowService.listWorkflows,
      input,
      signal: signal,
      headers: headers,
      onHeader: onHeader,
      onTrailer: onTrailer,
    );
  }

  Future<v1workflow.ActivateWorkflowResponse> activateWorkflow(
    v1workflow.ActivateWorkflowRequest input, {
    connect.Headers? headers,
    connect.AbortSignal? signal,
    Function(connect.Headers)? onHeader,
    Function(connect.Headers)? onTrailer,
  }) {
    return connect.Client(_transport).unary(
      specs.WorkflowService.activateWorkflow,
      input,
      signal: signal,
      headers: headers,
      onHeader: onHeader,
      onTrailer: onTrailer,
    );
  }
}
