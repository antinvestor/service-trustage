import 'package:flutter/material.dart';

import '../models/queue_counter.dart';
import 'counter_status_badge.dart';

/// Card for a [QueueCounter] with action buttons.
class CounterTile extends StatelessWidget {
  const CounterTile({
    super.key,
    required this.counter,
    this.onOpen,
    this.onClose,
    this.onPause,
    this.onCallNext,
    this.onBeginService,
    this.onCompleteService,
  });

  final QueueCounter counter;
  final VoidCallback? onOpen;
  final VoidCallback? onClose;
  final VoidCallback? onPause;
  final VoidCallback? onCallNext;
  final VoidCallback? onBeginService;
  final VoidCallback? onCompleteService;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

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
            Row(
              children: [
                Container(
                  width: 36,
                  height: 36,
                  decoration: BoxDecoration(
                    color: theme.colorScheme.secondaryContainer,
                    borderRadius: BorderRadius.circular(8),
                  ),
                  child: Icon(
                    Icons.point_of_sale_outlined,
                    size: 20,
                    color: theme.colorScheme.onSecondaryContainer,
                  ),
                ),
                const SizedBox(width: 10),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(counter.name, style: theme.textTheme.titleSmall),
                      if (counter.servedBy.isNotEmpty)
                        Text(
                          'Staff: ${counter.servedBy}',
                          style: theme.textTheme.bodySmall?.copyWith(
                            color: theme.colorScheme.onSurfaceVariant,
                          ),
                        ),
                    ],
                  ),
                ),
                CounterStatusBadge(status: counter.status),
              ],
            ),
            const SizedBox(height: 8),
            Row(
              children: [
                Text(
                  'Served: ${counter.totalServed}',
                  style: theme.textTheme.labelSmall?.copyWith(
                    color: theme.colorScheme.onSurfaceVariant,
                  ),
                ),
                if (counter.currentItemId.isNotEmpty) ...[
                  const SizedBox(width: 12),
                  Text(
                    'Current: ${counter.currentItemId.substring(0, 8)}...',
                    style: theme.textTheme.labelSmall?.copyWith(
                      color: theme.colorScheme.primary,
                    ),
                  ),
                ],
              ],
            ),
            const SizedBox(height: 12),
            Wrap(
              spacing: 8,
              runSpacing: 8,
              children: _buildActions(counter.status),
            ),
          ],
        ),
      ),
    );
  }

  List<Widget> _buildActions(CounterStatus status) {
    return switch (status) {
      CounterStatus.closed => [
          if (onOpen != null)
            FilledButton.tonal(
              onPressed: onOpen,
              child: const Text('Open'),
            ),
        ],
      CounterStatus.open => [
          if (counter.currentItemId.isEmpty && onCallNext != null)
            FilledButton.icon(
              onPressed: onCallNext,
              icon: const Icon(Icons.person_add, size: 16),
              label: const Text('Call Next'),
            ),
          if (counter.currentItemId.isNotEmpty && onBeginService != null)
            FilledButton.tonal(
              onPressed: onBeginService,
              child: const Text('Begin Service'),
            ),
          if (counter.currentItemId.isNotEmpty && onCompleteService != null)
            FilledButton.tonal(
              onPressed: onCompleteService,
              child: const Text('Complete'),
            ),
          if (onPause != null)
            OutlinedButton(
              onPressed: onPause,
              child: const Text('Pause'),
            ),
          if (onClose != null)
            OutlinedButton(
              onPressed: onClose,
              child: const Text('Close'),
            ),
        ],
      CounterStatus.paused => [
          if (onOpen != null)
            FilledButton.tonal(
              onPressed: onOpen,
              child: const Text('Resume'),
            ),
          if (onClose != null)
            OutlinedButton(
              onPressed: onClose,
              child: const Text('Close'),
            ),
        ],
    };
  }
}
