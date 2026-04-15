/// Formstore management UI library for Antinvestor.
///
/// Provides screens, widgets, providers, and routing for the
/// form definition and submission management service.
library;

// Models
export 'src/models/form_definition.dart';
export 'src/models/form_submission.dart';

// API
export 'src/api/formstore_client.dart';
export 'src/api/auth_http_client.dart';

// Providers
export 'src/providers/formstore_providers.dart';

// Screens
export 'src/screens/form_list_screen.dart';
export 'src/screens/form_detail_screen.dart';
export 'src/screens/form_edit_screen.dart';
export 'src/screens/submission_detail_screen.dart';

// Widgets
export 'src/widgets/form_definition_tile.dart';
export 'src/widgets/submission_tile.dart';
export 'src/widgets/submission_status_badge.dart';
export 'src/widgets/json_data_viewer.dart';

// Routing
export 'src/routing/formstore_route_module.dart';
