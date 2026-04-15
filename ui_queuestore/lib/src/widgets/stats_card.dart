import 'package:flutter/material.dart';

import '../models/queue_stats.dart';

/// Dashboard card displaying queue statistics.
class StatsCard extends StatelessWidget {
  const StatsCard({super.key, required this.stats});

  final QueueStats stats;

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
            Text('Queue Statistics', style: theme.textTheme.titleSmall),
            const SizedBox(height: 16),
            Wrap(
              spacing: 16,
              runSpacing: 12,
              children: [
                _MetricChip(
                  label: 'Waiting',
                  value: stats.totalWaiting.toString(),
                  color: Colors.blue,
                  icon: Icons.hourglass_empty,
                ),
                _MetricChip(
                  label: 'Being Served',
                  value: stats.totalBeingServed.toString(),
                  color: Colors.orange,
                  icon: Icons.person_outline,
                ),
                _MetricChip(
                  label: 'Completed Today',
                  value: stats.completedToday.toString(),
                  color: Colors.green,
                  icon: Icons.check_circle_outline,
                ),
                _MetricChip(
                  label: 'Cancelled Today',
                  value: stats.cancelledToday.toString(),
                  color: Colors.red,
                  icon: Icons.cancel_outlined,
                ),
                _MetricChip(
                  label: 'Avg Wait',
                  value: _formatSeconds(stats.averageWaitTime),
                  color: Colors.purple,
                  icon: Icons.timer_outlined,
                ),
                _MetricChip(
                  label: 'Longest Wait',
                  value: _formatSeconds(stats.longestWaitTime),
                  color: Colors.deepOrange,
                  icon: Icons.timer,
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  String _formatSeconds(int seconds) {
    if (seconds < 60) return '${seconds}s';
    final minutes = seconds ~/ 60;
    final secs = seconds % 60;
    if (minutes < 60) return '${minutes}m ${secs}s';
    return '${minutes ~/ 60}h ${minutes % 60}m';
  }
}

class _MetricChip extends StatelessWidget {
  const _MetricChip({
    required this.label,
    required this.value,
    required this.color,
    required this.icon,
  });

  final String label;
  final String value;
  final Color color;
  final IconData icon;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Container(
      width: 140,
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: color.withAlpha(15),
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: color.withAlpha(40)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(icon, size: 18, color: color),
          const SizedBox(height: 6),
          Text(
            value,
            style: theme.textTheme.headlineSmall?.copyWith(
              fontWeight: FontWeight.bold,
              color: color,
            ),
          ),
          Text(
            label,
            style: theme.textTheme.labelSmall?.copyWith(
              color: theme.colorScheme.onSurfaceVariant,
            ),
          ),
        ],
      ),
    );
  }
}
