/// Dart client library for Ant Investor Trustage Service.
///
/// Provides Workflow, Event, Signal, and Runtime service functionality
/// using Connect RPC protocol.
library;

// Workflow service
export 'src/v1/workflow.pb.dart';
export 'src/v1/workflow.pbenum.dart';
export 'src/v1/workflow.pbjson.dart';
export 'src/v1/workflow.connect.client.dart';
export 'src/v1/workflow.connect.spec.dart';

// Event service
export 'src/v1/event.pb.dart';
export 'src/v1/event.pbenum.dart';
export 'src/v1/event.pbjson.dart';
export 'src/v1/event.connect.client.dart';
export 'src/v1/event.connect.spec.dart';

// Signal service
export 'src/v1/signal.pb.dart';
export 'src/v1/signal.pbenum.dart';
export 'src/v1/signal.pbjson.dart';
export 'src/v1/signal.connect.client.dart';
export 'src/v1/signal.connect.spec.dart';

// Runtime service
export 'src/v1/runtime.pb.dart';
export 'src/v1/runtime.pbenum.dart';
export 'src/v1/runtime.pbjson.dart';
export 'src/v1/runtime.connect.client.dart';
export 'src/v1/runtime.connect.spec.dart';

// Common types
export 'src/common/v1/common.pb.dart';
export 'src/common/v1/common.pbenum.dart';
export 'src/google/protobuf/struct.pb.dart';
export 'src/google/protobuf/timestamp.pb.dart';
