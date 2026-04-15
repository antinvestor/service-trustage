import 'package:flutter/material.dart';

import 'package:antinvestor_ui_core/widgets/status_badge.dart';

import '../models/queue_item.dart';

/// Colored badge for [QueueItemStatus].
class ItemStatusBadge extends StatelessWidget {
  const ItemStatusBadge({super.key, required this.status});

  final QueueItemStatus status;

  @override
  Widget build(BuildContext context) {
    return StatusBadge.fromEnum(
      value: status,
      mapper: (s) => switch (s) {
        QueueItemStatus.waiting =>
          ('Waiting', Colors.blue, Icons.hourglass_empty),
        QueueItemStatus.serving =>
          ('Serving', Colors.orange, Icons.person_outline),
        QueueItemStatus.completed =>
          ('Completed', Colors.green, Icons.check_circle_outline),
        QueueItemStatus.cancelled =>
          ('Cancelled', Colors.red, Icons.cancel_outlined),
        QueueItemStatus.noShow =>
          ('No Show', Colors.grey, Icons.person_off_outlined),
        QueueItemStatus.expired =>
          ('Expired', Colors.brown, Icons.timer_off_outlined),
      },
    );
  }
}
