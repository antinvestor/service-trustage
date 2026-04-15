import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';

import 'package:antinvestor_ui_core/widgets/metadata_row.dart';

import '../models/queue_item.dart';
import '../providers/queuestore_providers.dart';
import '../widgets/item_status_badge.dart';

/// Detail screen for a single queue item with action buttons.
class QueueItemScreen extends ConsumerStatefulWidget {
  const QueueItemScreen({
    super.key,
    required this.itemId,
    this.initialItem,
  });

  final String itemId;
  final QueueItem? initialItem;

  @override
  ConsumerState<QueueItemScreen> createState() => _QueueItemScreenState();
}

class _QueueItemScreenState extends ConsumerState<QueueItemScreen> {
  bool _isActing = false;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final asyncItem = ref.watch(queueItemProvider(widget.itemId));
    final item = asyncItem.value ?? widget.initialItem;

    if (item == null && asyncItem.isLoading) {
      return const Scaffold(
        body: Center(child: CircularProgressIndicator()),
      );
    }

    if (item == null) {
      return Scaffold(
        appBar: AppBar(title: const Text('Queue Item')),
        body: const Center(child: Text('Item not found.')),
      );
    }

    return Scaffold(
      appBar: AppBar(
        title: Text(
            item.ticketNo.isNotEmpty ? item.ticketNo : 'Item ${item.id.substring(0, 8)}'),
        actions: [
          ItemStatusBadge(status: item.status),
          const SizedBox(width: 16),
        ],
      ),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            _buildInfoCard(theme, item),
            const SizedBox(height: 16),
            _buildTimestampsCard(theme, item),
            const SizedBox(height: 24),
            if (!_isTerminal(item.status)) _buildActions(theme, item),
          ],
        ),
      ),
    );
  }

  Widget _buildInfoCard(ThemeData theme, QueueItem item) {
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
            MetadataRow(label: 'ID', value: item.id),
            MetadataRow(label: 'Queue', value: item.queueId),
            MetadataRow(label: 'Priority', value: item.priority.toString()),
            MetadataRow(label: 'Status', value: item.status.toApiString()),
            if (item.ticketNo.isNotEmpty)
              MetadataRow(label: 'Ticket #', value: item.ticketNo),
            if (item.category.isNotEmpty)
              MetadataRow(label: 'Category', value: item.category),
            if (item.customerId.isNotEmpty)
              MetadataRow(label: 'Customer', value: item.customerId),
            if (item.counterId.isNotEmpty)
              MetadataRow(label: 'Counter', value: item.counterId),
            if (item.servedBy.isNotEmpty)
              MetadataRow(label: 'Served By', value: item.servedBy),
          ],
        ),
      ),
    );
  }

  Widget _buildTimestampsCard(ThemeData theme, QueueItem item) {
    final fmt = DateFormat.yMMMd().add_jm();
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
            Text('Timeline', style: theme.textTheme.titleSmall),
            const SizedBox(height: 12),
            if (item.joinedAt != null)
              MetadataRow(label: 'Joined', value: fmt.format(item.joinedAt!)),
            if (item.calledAt != null)
              MetadataRow(label: 'Called', value: fmt.format(item.calledAt!)),
            if (item.serviceStart != null)
              MetadataRow(
                  label: 'Service Start',
                  value: fmt.format(item.serviceStart!)),
            if (item.serviceEnd != null)
              MetadataRow(
                  label: 'Service End',
                  value: fmt.format(item.serviceEnd!)),
          ],
        ),
      ),
    );
  }

  Widget _buildActions(ThemeData theme, QueueItem item) {
    final notifier = ref.read(queueItemNotifierProvider.notifier);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text('Actions', style: theme.textTheme.titleMedium),
        const SizedBox(height: 12),
        Wrap(
          spacing: 8,
          runSpacing: 8,
          children: [
            if (item.status == QueueItemStatus.waiting) ...[
              FilledButton.icon(
                onPressed: _isActing ? null : () => _act(() => notifier.cancel(item.id)),
                icon: const Icon(Icons.cancel_outlined, size: 16),
                label: const Text('Cancel'),
              ),
              OutlinedButton.icon(
                onPressed: _isActing ? null : () => _act(() => notifier.noShow(item.id)),
                icon: const Icon(Icons.person_off_outlined, size: 16),
                label: const Text('No Show'),
              ),
            ],
            if (item.status == QueueItemStatus.noShow)
              FilledButton.tonal(
                onPressed: _isActing ? null : () => _act(() => notifier.requeue(item.id)),
                child: const Text('Requeue'),
              ),
          ],
        ),
      ],
    );
  }

  bool _isTerminal(QueueItemStatus status) {
    return status == QueueItemStatus.completed ||
        status == QueueItemStatus.cancelled ||
        status == QueueItemStatus.expired;
  }

  Future<void> _act(Future<void> Function() action) async {
    setState(() => _isActing = true);
    try {
      await action();
      ref.invalidate(queueItemProvider);
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Action failed: $e')),
        );
      }
    } finally {
      if (mounted) setState(() => _isActing = false);
    }
  }
}
