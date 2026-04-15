/// Queue management UI library for Antinvestor.
///
/// Provides screens, widgets, providers, and routing for the
/// queue management service — definitions, items, counters, and statistics.
library;

// Models
export 'src/models/queue_definition.dart';
export 'src/models/queue_item.dart';
export 'src/models/queue_counter.dart';
export 'src/models/queue_stats.dart';

// API
export 'src/api/queuestore_client.dart';
export 'src/api/auth_http_client.dart';

// Providers
export 'src/providers/queuestore_providers.dart';

// Screens
export 'src/screens/queue_list_screen.dart';
export 'src/screens/queue_dashboard_screen.dart';
export 'src/screens/queue_edit_screen.dart';
export 'src/screens/queue_item_screen.dart';
export 'src/screens/enqueue_screen.dart';

// Widgets
export 'src/widgets/queue_tile.dart';
export 'src/widgets/queue_item_tile.dart';
export 'src/widgets/counter_tile.dart';
export 'src/widgets/item_status_badge.dart';
export 'src/widgets/counter_status_badge.dart';
export 'src/widgets/stats_card.dart';

// Routing
export 'src/routing/queuestore_route_module.dart';
