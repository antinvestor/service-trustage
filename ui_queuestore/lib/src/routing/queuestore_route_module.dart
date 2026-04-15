import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

import 'package:antinvestor_ui_core/navigation/nav_items.dart';
import 'package:antinvestor_ui_core/permissions/permission_manifest.dart';
import 'package:antinvestor_ui_core/routing/route_module.dart';

import '../models/queue_definition.dart';
import '../models/queue_item.dart';
import '../screens/enqueue_screen.dart';
import '../screens/queue_dashboard_screen.dart';
import '../screens/queue_edit_screen.dart';
import '../screens/queue_item_screen.dart';
import '../screens/queue_list_screen.dart';

/// Route module for the Queuestore service UI.
class QueuestoreRouteModule extends RouteModule {
  @override
  String get moduleId => 'queuestore';

  @override
  List<RouteBase> buildRoutes() {
    return [
      GoRoute(
        path: '/queuestore',
        builder: (context, state) => const QueueListScreen(),
        routes: [
          GoRoute(
            path: 'create',
            builder: (context, state) => const QueueEditScreen(),
          ),
          GoRoute(
            path: 'detail/:id',
            builder: (context, state) {
              final id = state.pathParameters['id'] ?? '';
              final queue = state.extra is QueueDefinition
                  ? state.extra as QueueDefinition
                  : null;
              return QueueDashboardScreen(
                queueId: id,
                initialQueue: queue,
              );
            },
          ),
          GoRoute(
            path: 'edit/:id',
            builder: (context, state) {
              final queue = state.extra is QueueDefinition
                  ? state.extra as QueueDefinition
                  : null;
              return QueueEditScreen(queue: queue);
            },
          ),
          GoRoute(
            path: 'enqueue/:queueId',
            builder: (context, state) {
              final queueId = state.pathParameters['queueId'] ?? '';
              return EnqueueScreen(queueId: queueId);
            },
          ),
          GoRoute(
            path: 'item/:id',
            builder: (context, state) {
              final id = state.pathParameters['id'] ?? '';
              final item = state.extra is QueueItem
                  ? state.extra as QueueItem
                  : null;
              return QueueItemScreen(itemId: id, initialItem: item);
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
        id: 'queuestore',
        label: 'Queues',
        icon: Icons.queue_outlined,
        activeIcon: Icons.queue,
        route: '/queuestore',
        requiredPermissions: {'queuestore_read'},
      ),
    ];
  }

  @override
  Map<String, Set<String>> get routePermissions => {
        '/queuestore': {'queuestore_read'},
        '/queuestore/create': {'queuestore_create'},
        '/queuestore/detail': {'queuestore_read'},
        '/queuestore/edit': {'queuestore_update'},
        '/queuestore/enqueue': {'queuestore_enqueue'},
        '/queuestore/item': {'queuestore_read'},
      };

  @override
  PermissionManifest get permissionManifest => const PermissionManifest(
        namespace: 'service_queuestore',
        permissions: [
          PermissionEntry(
            key: 'queuestore_read',
            label: 'View Queues',
            scope: PermissionScope.service,
          ),
          PermissionEntry(
            key: 'queuestore_create',
            label: 'Create Queues',
            scope: PermissionScope.action,
          ),
          PermissionEntry(
            key: 'queuestore_update',
            label: 'Update Queues',
            scope: PermissionScope.action,
          ),
          PermissionEntry(
            key: 'queuestore_delete',
            label: 'Delete Queues',
            scope: PermissionScope.action,
          ),
          PermissionEntry(
            key: 'queuestore_enqueue',
            label: 'Add Items to Queue',
            scope: PermissionScope.action,
          ),
          PermissionEntry(
            key: 'queuestore_counter',
            label: 'Manage Counters',
            scope: PermissionScope.action,
          ),
        ],
      );
}
