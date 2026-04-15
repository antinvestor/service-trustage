import 'package:flutter/material.dart';

import 'package:antinvestor_ui_core/widgets/status_badge.dart';

import '../models/form_submission.dart';

/// Colored badge for [SubmissionStatus].
class SubmissionStatusBadge extends StatelessWidget {
  const SubmissionStatusBadge({super.key, required this.status});

  final SubmissionStatus status;

  @override
  Widget build(BuildContext context) {
    return StatusBadge.fromEnum(
      value: status,
      mapper: (s) => switch (s) {
        SubmissionStatus.pending => ('Pending', Colors.orange, Icons.schedule),
        SubmissionStatus.complete =>
          ('Complete', Colors.green, Icons.check_circle_outline),
        SubmissionStatus.archived =>
          ('Archived', Colors.grey, Icons.archive_outlined),
      },
    );
  }
}
