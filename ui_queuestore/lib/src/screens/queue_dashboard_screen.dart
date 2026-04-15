import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';

import 'package:antinvestor_ui_core/widgets/metadata_row.dart';

import '../models/queue_definition.dart';
import '../models/queue_counter.dart';
import '../providers/queuestore_providers.dart';
import '../widgets/counter_tile.dart';
import '../widgets/queue_item_tile.dart';
import '../widgets/stats_card.dart';

/// Dashboard screen for a single queue — stats, waiting items, and counters.
class QueueDashboardScreen extends ConsumerStatefulWidget {
  const QueueDashboardScreen({
    super.key,
    required this.queueId,
    this.initialQueue,
  });

  final String queueId;
  final QueueDefinition? initialQueue;

  @override
  ConsumerState<QueueDashboardScreen> createState() =>
      _QueueDashboardScreenState();
}

class _QueueDashboardScreenState extends ConsumerState<QueueDashboardScreen> {
  bool _isDeleting = false;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final asyncQueue = ref.watch(queueDetailProvider(widget.queueId));
    final queue = asyncQueue.value ?? widget.initialQueue;

    if (queue == null && asyncQueue.isLoading) {
      return const Scaffold(
        body: Center(child: CircularProgressIndicator()),
      );
    }

    if (queue == null) {
      return Scaffold(
        appBar: AppBar(title: const Text('Queue')),
        body: const Center(child: Text('Queue not found.')),
      );
    }

    return Scaffold(
      appBar: AppBar(
        title: Text(queue.name),
        actions: [
          IconButton(
            icon: const Icon(Icons.edit_outlined),
            tooltip: 'Edit',
            onPressed: () =>
                context.go('/queuestore/edit/${queue.id}', extra: queue),
          ),
          IconButton(
            icon: _isDeleting
                ? const SizedBox(
                    width: 18,
                    height: 18,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  )
                : const Icon(Icons.delete_outline),
            tooltip: 'Delete',
            onPressed: _isDeleting ? null : () => _confirmDelete(queue),
          ),
          const SizedBox(width: 8),
        ],
      ),
      body: RefreshIndicator(
        onRefresh: () async {
          ref.invalidate(queueStatsProvider);
          ref.invalidate(queueItemListProvider);
          ref.invalidate(counterListProvider);
        },
        child: SingleChildScrollView(
          physics: const AlwaysScrollableScrollPhysics(),
          padding: const EdgeInsets.all(24),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              _buildInfoCard(theme, queue),
              const SizedBox(height: 16),
              _StatsSection(queueId: widget.queueId),
              const SizedBox(height: 24),
              _CountersSection(queueId: widget.queueId),
              const SizedBox(height: 24),
              _WaitingItemsSection(queueId: widget.queueId),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildInfoCard(ThemeData theme, QueueDefinition q) {
    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
        side: BorderSide(color: theme.colorScheme.outlineVariant),
      ),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('Details', style: theme.textTheme.titleSmall),
            const SizedBox(height: 12),
            MetadataRow(label: 'Name', value: q.name),
            if (q.description.isNotEmpty)
              MetadataRow(label: 'Description', value: q.description),
            MetadataRow(
                label: 'Status', value: q.active ? 'Active' : 'Inactive'),
            MetadataRow(
                label: 'Priority Levels',
                value: q.priorityLevels.toString()),
            MetadataRow(
                label: 'Max Capacity',
                value:
                    q.maxCapacity == 0 ? 'Unlimited' : q.maxCapacity.toString()),
            MetadataRow(label: 'SLA', value: '${q.slaMinutes} minutes'),
            if (q.createdAt != null)
              MetadataRow(
                label: 'Created',
                value: DateFormat.yMMMd().add_jm().format(q.createdAt!),
              ),
          ],
        ),
      ),
    );
  }

  Future<void> _confirmDelete(QueueDefinition q) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Delete Queue'),
        content: Text('Are you sure you want to delete "${q.name}"?'),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(ctx, false),
              child: const Text('Cancel')),
          FilledButton(
              onPressed: () => Navigator.pop(ctx, true),
              child: const Text('Delete')),
        ],
      ),
    );

    if (confirmed != true || !mounted) return;

    setState(() => _isDeleting = true);
    try {
      await ref.read(queueDefinitionNotifierProvider.notifier).delete(q.id);
      if (mounted) context.go('/queuestore');
    } catch (e) {
      if (mounted) {
        setState(() => _isDeleting = false);
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Delete failed: $e')),
        );
      }
    }
  }
}

class _StatsSection extends ConsumerWidget {
  const _StatsSection({required this.queueId});
  final String queueId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final asyncStats = ref.watch(queueStatsProvider(queueId));

    return asyncStats.when(
      loading: () => const Center(
        child: Padding(
          padding: EdgeInsets.all(16),
          child: CircularProgressIndicator(),
        ),
      ),
      error: (err, _) => Center(child: Text('Stats unavailable: $err')),
      data: (stats) => StatsCard(stats: stats),
    );
  }
}

class _CountersSection extends ConsumerWidget {
  const _CountersSection({required this.queueId});
  final String queueId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final asyncCounters = ref.watch(counterListProvider(queueId));

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            Text('Counters', style: theme.textTheme.titleMedium),
            const Spacer(),
            FilledButton.tonalIcon(
              onPressed: () =>
                  _showCreateCounterDialog(context, ref, queueId),
              icon: const Icon(Icons.add, size: 16),
              label: const Text('Add Counter'),
            ),
          ],
        ),
        const SizedBox(height: 12),
        asyncCounters.when(
          loading: () => const Center(child: CircularProgressIndicator()),
          error: (err, _) => Text('Error: $err'),
          data: (counters) {
            if (counters.isEmpty) {
              return const Center(
                child: Padding(
                  padding: EdgeInsets.all(16),
                  child: Text('No counters configured.'),
                ),
              );
            }
            return Column(
              children: counters.map((c) {
                final notifier =
                    ref.read(counterNotifierProvider.notifier);
                return Padding(
                  padding: const EdgeInsets.only(bottom: 8),
                  child: CounterTile(
                    counter: c,
                    onOpen: () => notifier.open(c.id),
                    onClose: () => notifier.close(c.id),
                    onPause: () => notifier.pause(c.id),
                    onCallNext: () => notifier.callNext(c.id),
                    onBeginService: () => notifier.beginService(c.id),
                    onCompleteService: () => notifier.completeService(c.id),
                  ),
                );
              }).toList(),
            );
          },
        ),
      ],
    );
  }

  Future<void> _showCreateCounterDialog(
      BuildContext context, WidgetRef ref, String queueId) async {
    final nameCtl = TextEditingController();
    final result = await showDialog<String>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Add Counter'),
        content: TextField(
          controller: nameCtl,
          decoration: const InputDecoration(
            labelText: 'Counter Name',
            hintText: 'e.g., Window 1',
          ),
          autofocus: true,
        ),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(ctx),
              child: const Text('Cancel')),
          FilledButton(
              onPressed: () => Navigator.pop(ctx, nameCtl.text),
              child: const Text('Create')),
        ],
      ),
    );
    nameCtl.dispose();

    if (result == null || result.isEmpty) return;

    final counter = QueueCounter(
      id: '',
      queueId: queueId,
      name: result,
    );

    try {
      await ref.read(counterNotifierProvider.notifier).create(queueId, counter);
    } catch (e) {
      if (context.mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Failed to create counter: $e')),
        );
      }
    }
  }
}

class _WaitingItemsSection extends ConsumerWidget {
  const _WaitingItemsSection({required this.queueId});
  final String queueId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final asyncItems = ref.watch(queueItemListProvider(queueId));

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            Text('Waiting Items', style: theme.textTheme.titleMedium),
            const Spacer(),
            FilledButton.tonalIcon(
              onPressed: () => context.go('/queuestore/enqueue/$queueId'),
              icon: const Icon(Icons.person_add, size: 16),
              label: const Text('Enqueue'),
            ),
          ],
        ),
        const SizedBox(height: 12),
        asyncItems.when(
          loading: () => const Center(child: CircularProgressIndicator()),
          error: (err, _) => Text('Error: $err'),
          data: (items) {
            if (items.isEmpty) {
              return const Center(
                child: Padding(
                  padding: EdgeInsets.all(16),
                  child: Text('Queue is empty.'),
                ),
              );
            }
            return Column(
              children: items
                  .map((item) => Padding(
                        padding: const EdgeInsets.only(bottom: 8),
                        child: QueueItemTile(
                          item: item,
                          onTap: () => context.go(
                            '/queuestore/item/${item.id}',
                            extra: item,
                          ),
                        ),
                      ))
                  .toList(),
            );
          },
        ),
      ],
    );
  }
}
