//
//  Generated code. Do not modify.
//  source: v1/runtime.proto
//

import "package:connectrpc/connect.dart" as connect;
import "runtime.pb.dart" as v1runtime;

abstract final class RuntimeService {
  /// Fully-qualified name of the RuntimeService service.
  static const name = 'runtime.v1.RuntimeService';

  static const listInstances = connect.Spec(
    '/$name/ListInstances',
    connect.StreamType.unary,
    v1runtime.ListInstancesRequest.new,
    v1runtime.ListInstancesResponse.new,
  );

  static const retryInstance = connect.Spec(
    '/$name/RetryInstance',
    connect.StreamType.unary,
    v1runtime.RetryInstanceRequest.new,
    v1runtime.RetryInstanceResponse.new,
  );

  static const listExecutions = connect.Spec(
    '/$name/ListExecutions',
    connect.StreamType.unary,
    v1runtime.ListExecutionsRequest.new,
    v1runtime.ListExecutionsResponse.new,
  );

  static const getExecution = connect.Spec(
    '/$name/GetExecution',
    connect.StreamType.unary,
    v1runtime.GetExecutionRequest.new,
    v1runtime.GetExecutionResponse.new,
  );

  static const retryExecution = connect.Spec(
    '/$name/RetryExecution',
    connect.StreamType.unary,
    v1runtime.RetryExecutionRequest.new,
    v1runtime.RetryExecutionResponse.new,
  );

  static const resumeExecution = connect.Spec(
    '/$name/ResumeExecution',
    connect.StreamType.unary,
    v1runtime.ResumeExecutionRequest.new,
    v1runtime.ResumeExecutionResponse.new,
  );

  static const getInstanceRun = connect.Spec(
    '/$name/GetInstanceRun',
    connect.StreamType.unary,
    v1runtime.GetInstanceRunRequest.new,
    v1runtime.GetInstanceRunResponse.new,
  );
}
