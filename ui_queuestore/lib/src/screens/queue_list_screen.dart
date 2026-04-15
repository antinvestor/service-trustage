import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import 'package:antinvestor_ui_core/widgets/entity_list_page.dart';

import '../models/queue_definition.dart';
import '../providers/queuestore_providers.dart';
import '../widgets/queue_tile.dart';

/// Screen listing all queue definitions.
class QueueListScreen extends ConsumerStatefulWidget {
  const QueueListScreen({super.key});

  @override
  ConsumerState<QueueListScreen> createState() => _QueueListScreenState();
}

class _QueueListScreenState extends ConsumerState<QueueListScreen> {
  String _search = '';
  bool? _activeFilter;

  @override
  Widget build(BuildContext context) {
    final asyncQueues = ref.watch(queueListProvider(_activeFilter));

    return asyncQueues.when(
      loading: () => EntityListPage<QueueDefinition>(
        title: 'Queues',
        icon: Icons.queue_outlined,
        items: const [],
        isLoading: true,
        itemBuilder: (_, _) => const SizedBox.shrink(),
        searchHint: 'Search queues...',
        onSearchChanged: (v) => setState(() => _search = v),
        actionLabel: 'New Queue',
        onAction: () => context.go('/queuestore/create'),
        filterWidget: _buildFilterChip(),
      ),
      error: (err, _) => EntityListPage<QueueDefinition>(
        title: 'Queues',
        icon: Icons.queue_outlined,
        items: const [],
        error: err.toString(),
        onRetry: () => ref.invalidate(queueListProvider),
        itemBuilder: (_, _) => const SizedBox.shrink(),
        filterWidget: _buildFilterChip(),
      ),
      data: (queues) {
        final filtered = _search.isEmpty
            ? queues
            : queues
                .where((q) =>
                    q.name.toLowerCase().contains(_search.toLowerCase()))
                .toList();

        return EntityListPage<QueueDefinition>(
          title: 'Queues',
          icon: Icons.queue_outlined,
          items: filtered,
          itemBuilder: (context, queue) => QueueTile(
            queue: queue,
            onTap: () =>
                context.go('/queuestore/detail/${queue.id}', extra: queue),
          ),
          searchHint: 'Search queues...',
          onSearchChanged: (v) => setState(() => _search = v),
          actionLabel: 'New Queue',
          onAction: () => context.go('/queuestore/create'),
          filterWidget: _buildFilterChip(),
        );
      },
    );
  }

  Widget _buildFilterChip() {
    return SegmentedButton<bool?>(
      segments: const [
        ButtonSegment(value: null, label: Text('All')),
        ButtonSegment(value: true, label: Text('Active')),
        ButtonSegment(value: false, label: Text('Inactive')),
      ],
      selected: {_activeFilter},
      onSelectionChanged: (v) => setState(() => _activeFilter = v.first),
      showSelectedIcon: false,
      style: const ButtonStyle(
        visualDensity: VisualDensity.compact,
      ),
    );
  }
}
