import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import 'package:antinvestor_ui_core/navigation/nav_items.dart';
import 'package:antinvestor_ui_core/permissions/permission_manifest.dart';
import 'package:antinvestor_ui_core/routing/route_module.dart';

import '../screens/command_deck_screen.dart';
import '../screens/execution_queue_screen.dart';
import '../screens/run_explorer_screen.dart';
import '../screens/workflow_catalog_screen.dart';

/// Route module for the Trustage workflow orchestration UI.
class TrustageRouteModule extends RouteModule {
  @override
  String get moduleId => 'trustage';

  @override
  List<RouteBase> buildRoutes() {
    return [
      GoRoute(
        path: '/trustage',
        builder: (context, state) => const CommandDeckScreen(),
        routes: [
          GoRoute(
            path: 'runs',
            builder: (context, state) => const RunExplorerScreen(),
          ),
          GoRoute(
            path: 'runs/:instanceId',
            builder: (context, state) {
              final instanceId = state.pathParameters['instanceId'] ?? '';
              return RunExplorerScreen(initialInstanceId: instanceId);
            },
          ),
          GoRoute(
            path: 'executions',
            builder: (context, state) => const ExecutionQueueScreen(),
          ),
          GoRoute(
            path: 'workflows',
            builder: (context, state) => const WorkflowCatalogScreen(),
          ),
        ],
      ),
    ];
  }

  @override
  List<NavItem> buildNavItems() {
    return [
      const NavItem(
        id: 'trustage',
        label: 'Orchestrator',
        icon: Icons.hub_outlined,
        activeIcon: Icons.hub,
        route: '/trustage',
        requiredPermissions: {'trustage_read'},
        children: [
          NavItem(
            id: 'trustage_runs',
            label: 'Run Explorer',
            icon: Icons.play_circle_outline,
            route: '/trustage/runs',
            requiredPermissions: {'trustage_read'},
          ),
          NavItem(
            id: 'trustage_executions',
            label: 'Execution Queue',
            icon: Icons.list_alt,
            route: '/trustage/executions',
            requiredPermissions: {'trustage_read'},
          ),
          NavItem(
            id: 'trustage_workflows',
            label: 'Workflow Catalog',
            icon: Icons.schema_outlined,
            route: '/trustage/workflows',
            requiredPermissions: {'trustage_read'},
          ),
        ],
      ),
    ];
  }

  @override
  Map<String, Set<String>> get routePermissions => {
        '/trustage': {'trustage_read'},
        '/trustage/runs': {'trustage_read'},
        '/trustage/executions': {'trustage_read'},
        '/trustage/workflows': {'trustage_read'},
      };

  @override
  PermissionManifest get permissionManifest => const PermissionManifest(
        namespace: 'service_trustage',
        permissions: [
          PermissionEntry(
            key: 'trustage_read',
            label: 'View Workflows & Runs',
            scope: PermissionScope.service,
          ),
          PermissionEntry(
            key: 'trustage_operate',
            label: 'Retry, Resume, Send Signals',
            scope: PermissionScope.action,
          ),
          PermissionEntry(
            key: 'trustage_ingest',
            label: 'Trigger Events',
            scope: PermissionScope.action,
          ),
          PermissionEntry(
            key: 'trustage_manage',
            label: 'Create & Activate Workflows',
            scope: PermissionScope.action,
          ),
        ],
      );
}
