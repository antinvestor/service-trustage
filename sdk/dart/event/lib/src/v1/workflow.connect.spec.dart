//
//  Generated code. Do not modify.
//  source: v1/workflow.proto
//

import "package:connectrpc/connect.dart" as connect;
import "workflow.pb.dart" as v1workflow;

abstract final class WorkflowService {
  /// Fully-qualified name of the WorkflowService service.
  static const name = 'workflow.v1.WorkflowService';

  static const createWorkflow = connect.Spec(
    '/$name/CreateWorkflow',
    connect.StreamType.unary,
    v1workflow.CreateWorkflowRequest.new,
    v1workflow.CreateWorkflowResponse.new,
  );

  static const getWorkflow = connect.Spec(
    '/$name/GetWorkflow',
    connect.StreamType.unary,
    v1workflow.GetWorkflowRequest.new,
    v1workflow.GetWorkflowResponse.new,
  );

  static const listWorkflows = connect.Spec(
    '/$name/ListWorkflows',
    connect.StreamType.unary,
    v1workflow.ListWorkflowsRequest.new,
    v1workflow.ListWorkflowsResponse.new,
  );

  static const activateWorkflow = connect.Spec(
    '/$name/ActivateWorkflow',
    connect.StreamType.unary,
    v1workflow.ActivateWorkflowRequest.new,
    v1workflow.ActivateWorkflowResponse.new,
  );
}
