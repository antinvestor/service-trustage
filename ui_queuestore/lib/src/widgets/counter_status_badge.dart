import 'package:flutter/material.dart';

import 'package:antinvestor_ui_core/widgets/status_badge.dart';

import '../models/queue_counter.dart';

/// Colored badge for [CounterStatus].
class CounterStatusBadge extends StatelessWidget {
  const CounterStatusBadge({super.key, required this.status});

  final CounterStatus status;

  @override
  Widget build(BuildContext context) {
    return StatusBadge.fromEnum(
      value: status,
      mapper: (s) => switch (s) {
        CounterStatus.open =>
          ('Open', Colors.green, Icons.check_circle_outline),
        CounterStatus.closed =>
          ('Closed', Colors.red, Icons.block_outlined),
        CounterStatus.paused =>
          ('Paused', Colors.orange, Icons.pause_circle_outline),
      },
    );
  }
}
