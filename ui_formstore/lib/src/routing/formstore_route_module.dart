import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import 'package:antinvestor_ui_core/navigation/nav_items.dart';
import 'package:antinvestor_ui_core/permissions/permission_manifest.dart';
import 'package:antinvestor_ui_core/routing/route_module.dart';

import '../models/form_definition.dart';
import '../models/form_submission.dart';
import '../screens/form_detail_screen.dart';
import '../screens/form_edit_screen.dart';
import '../screens/form_list_screen.dart';
import '../screens/submission_detail_screen.dart';

/// Route module for the Formstore service UI.
class FormstoreRouteModule extends RouteModule {
  @override
  String get moduleId => 'formstore';

  @override
  List<RouteBase> buildRoutes() {
    return [
      GoRoute(
        path: '/formstore',
        builder: (context, state) => const FormListScreen(),
        routes: [
          GoRoute(
            path: 'create',
            builder: (context, state) => const FormEditScreen(),
          ),
          GoRoute(
            path: 'detail/:id',
            builder: (context, state) {
              final id = state.pathParameters['id'] ?? '';
              final def = state.extra is FormDefinition
                  ? state.extra as FormDefinition
                  : null;
              return FormDetailScreen(
                definitionId: id,
                initialDefinition: def,
              );
            },
          ),
          GoRoute(
            path: 'edit/:id',
            builder: (context, state) {
              final def = state.extra is FormDefinition
                  ? state.extra as FormDefinition
                  : null;
              return FormEditScreen(definition: def);
            },
          ),
          GoRoute(
            path: 'submission/:id',
            builder: (context, state) {
              final id = state.pathParameters['id'] ?? '';
              final sub = state.extra is FormSubmission
                  ? state.extra as FormSubmission
                  : null;
              return SubmissionDetailScreen(
                submissionId: id,
                initialSubmission: sub,
              );
            },
          ),
        ],
      ),
    ];
  }

  @override
  List<NavItem> buildNavItems() {
    return [
      const NavItem(
        id: 'formstore',
        label: 'Forms',
        icon: Icons.description_outlined,
        activeIcon: Icons.description,
        route: '/formstore',
        requiredPermissions: {'formstore_read'},
      ),
    ];
  }

  @override
  Map<String, Set<String>> get routePermissions => {
        '/formstore': {'formstore_read'},
        '/formstore/create': {'formstore_create'},
        '/formstore/detail': {'formstore_read'},
        '/formstore/edit': {'formstore_update'},
        '/formstore/submission': {'formstore_read'},
      };

  @override
  PermissionManifest get permissionManifest => const PermissionManifest(
        namespace: 'service_formstore',
        permissions: [
          PermissionEntry(
            key: 'formstore_read',
            label: 'View Forms',
            scope: PermissionScope.service,
          ),
          PermissionEntry(
            key: 'formstore_create',
            label: 'Create Forms',
            scope: PermissionScope.action,
          ),
          PermissionEntry(
            key: 'formstore_update',
            label: 'Update Forms',
            scope: PermissionScope.action,
          ),
          PermissionEntry(
            key: 'formstore_delete',
            label: 'Delete Forms',
            scope: PermissionScope.action,
          ),
          PermissionEntry(
            key: 'formstore_submit',
            label: 'Submit Forms',
            scope: PermissionScope.action,
          ),
        ],
      );
}
