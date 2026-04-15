import 'package:flutter/material.dart';
import 'package:intl/intl.dart';

import 'package:antinvestor_api_event/antinvestor_api_event.dart' as ev;

import 'trustage_status_badge.dart';

/// Compact display of a single audit timeline entry.
class TimelineEntryTile extends StatelessWidget {
  const TimelineEntryTile({super.key, required this.entry});

  final ev.TimelineEntry entry;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final fmt = DateFormat.Hm();

    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // Dot
          Padding(
            padding: const EdgeInsets.only(top: 4, right: 10),
            child: Container(
              width: 8,
              height: 8,
              decoration: BoxDecoration(
                shape: BoxShape.circle,
                color: theme.colorScheme.primary.withAlpha(120),
              ),
            ),
          ),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    TrustageStatusBadge(status: entry.eventType),
                    const SizedBox(width: 6),
                    if (entry.state.isNotEmpty)
                      Text(
                        entry.state,
                        style: theme.textTheme.labelSmall?.copyWith(
                          fontFamily: 'monospace',
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                  ],
                ),
                if (entry.fromState.isNotEmpty || entry.toState.isNotEmpty)
                  Padding(
                    padding: const EdgeInsets.only(top: 2),
                    child: Row(
                      children: [
                        if (entry.fromState.isNotEmpty)
                          Text(
                            entry.fromState,
                            style: theme.textTheme.bodySmall?.copyWith(
                              fontSize: 11,
                              color: theme.colorScheme.onSurfaceVariant,
                            ),
                          ),
                        if (entry.fromState.isNotEmpty &&
                            entry.toState.isNotEmpty)
                          Padding(
                            padding: const EdgeInsets.symmetric(horizontal: 4),
                            child: Icon(Icons.arrow_forward,
                                size: 12,
                                color: theme.colorScheme.onSurfaceVariant),
                          ),
                        if (entry.toState.isNotEmpty)
                          Text(
                            entry.toState,
                            style: theme.textTheme.bodySmall?.copyWith(
                              fontSize: 11,
                              fontWeight: FontWeight.w600,
                            ),
                          ),
                      ],
                    ),
                  ),
              ],
            ),
          ),
          if (entry.hasCreatedAt())
            Text(
              fmt.format(entry.createdAt.toDateTime()),
              style: theme.textTheme.labelSmall?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
        ],
      ),
    );
  }
}
