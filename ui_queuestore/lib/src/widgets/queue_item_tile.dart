import 'package:flutter/material.dart';
import 'package:intl/intl.dart';

import '../models/queue_item.dart';
import 'item_status_badge.dart';

/// Compact card for a [QueueItem] in a list.
class QueueItemTile extends StatelessWidget {
  const QueueItemTile({
    super.key,
    required this.item,
    this.onTap,
  });

  final QueueItem item;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
        side: BorderSide(color: theme.colorScheme.outlineVariant),
      ),
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(12),
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
          child: Row(
            children: [
              Container(
                width: 40,
                height: 40,
                decoration: BoxDecoration(
                  color: _priorityColor(item.priority).withAlpha(25),
                  borderRadius: BorderRadius.circular(10),
                ),
                child: Center(
                  child: Text(
                    item.ticketNo.isNotEmpty ? item.ticketNo : '#',
                    style: TextStyle(
                      fontWeight: FontWeight.bold,
                      fontSize: 12,
                      color: _priorityColor(item.priority),
                    ),
                  ),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      item.customerId.isNotEmpty
                          ? item.customerId
                          : item.id.substring(0, 8),
                      style: theme.textTheme.titleSmall,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    ),
                    const SizedBox(height: 2),
                    Row(
                      children: [
                        if (item.category.isNotEmpty)
                          Text(
                            item.category,
                            style: theme.textTheme.bodySmall?.copyWith(
                              color: theme.colorScheme.onSurfaceVariant,
                            ),
                          ),
                        if (item.joinedAt != null) ...[
                          if (item.category.isNotEmpty)
                            Text(
                              ' · ',
                              style: theme.textTheme.bodySmall?.copyWith(
                                color: theme.colorScheme.onSurfaceVariant,
                              ),
                            ),
                          Text(
                            DateFormat.Hm().format(item.joinedAt!),
                            style: theme.textTheme.bodySmall?.copyWith(
                              color: theme.colorScheme.onSurfaceVariant,
                            ),
                          ),
                        ],
                      ],
                    ),
                  ],
                ),
              ),
              const SizedBox(width: 8),
              ItemStatusBadge(status: item.status),
            ],
          ),
        ),
      ),
    );
  }

  Color _priorityColor(int priority) {
    return switch (priority) {
      >= 3 => Colors.red,
      2 => Colors.orange,
      _ => Colors.blue,
    };
  }
}
