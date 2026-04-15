/// Workflow orchestration UI library for Antinvestor Trustage.
///
/// Provides the command deck, run explorer with execution graph,
/// execution queue, workflow catalog, and operator action center
/// for tracing and managing durable workflow executions.
///
/// ## Screens
///
/// - [CommandDeckScreen] — Overview dashboard with metrics, event trigger,
///   and hot executions needing attention.
/// - [RunExplorerScreen] — Instance browser with execution graph
///   visualization, causal timeline, detail panel, and action center.
/// - [ExecutionQueueScreen] — Execution list with status filtering and
///   one-click retry for failed/timed-out executions.
/// - [WorkflowCatalogScreen] — Active workflow definitions list.
///
/// ## Key widgets
///
/// - [ExecutionGraph] — Timeline visualization of execution attempts,
///   scope runs, signal waits/messages, and child instances.
/// - [ActionCenter] — Operator controls for retry, resume, and signal
///   delivery.
/// - [EventTriggerForm] — Form for ingesting external events.
/// - [ExecutionDetailPanel] — Input/output payload viewer for a selected
///   execution.
/// - [BlockedStatePanel] — Active signal waits and pending signals.
/// - [TimelineEntryTile] — Audit timeline entry display.
///
/// ## Embedding
///
/// Use [TrustageRouteModule] to compose this UI into a host app:
/// ```dart
/// final modules = [TrustageRouteModule(), ...];
/// ```
library;

// Providers
export 'src/providers/trustage_transport_provider.dart';
export 'src/providers/trustage_providers.dart';

// Screens
export 'src/screens/command_deck_screen.dart';
export 'src/screens/run_explorer_screen.dart';
export 'src/screens/execution_queue_screen.dart';
export 'src/screens/workflow_catalog_screen.dart';

// Widgets — Core
export 'src/widgets/trustage_status_badge.dart';
export 'src/widgets/metric_card.dart';
export 'src/widgets/json_block.dart';
export 'src/widgets/trustage_panel.dart';
export 'src/widgets/status_helpers.dart';

// Widgets — Execution tracing
export 'src/widgets/execution_graph.dart';
export 'src/widgets/execution_detail_panel.dart';
export 'src/widgets/timeline_entry_tile.dart';
export 'src/widgets/blocked_state_panel.dart';

// Widgets — Operator actions
export 'src/widgets/action_center.dart';
export 'src/widgets/event_trigger_form.dart';

// Routing
export 'src/routing/trustage_route_module.dart';
