import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';

import 'package:antinvestor_ui_core/widgets/metadata_row.dart';

import '../models/form_submission.dart';
import '../providers/formstore_providers.dart';
import '../widgets/json_data_viewer.dart';
import '../widgets/submission_status_badge.dart';

/// Detail screen for a single form submission.
class SubmissionDetailScreen extends ConsumerWidget {
  const SubmissionDetailScreen({
    super.key,
    required this.submissionId,
    this.initialSubmission,
  });

  final String submissionId;
  final FormSubmission? initialSubmission;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final asyncSub = ref.watch(submissionProvider(submissionId));

    final submission = asyncSub.value ?? initialSubmission;

    if (submission == null && asyncSub.isLoading) {
      return const Scaffold(
        body: Center(child: CircularProgressIndicator()),
      );
    }

    if (submission == null) {
      return Scaffold(
        appBar: AppBar(title: const Text('Submission')),
        body: const Center(child: Text('Submission not found.')),
      );
    }

    return Scaffold(
      appBar: AppBar(
        title: const Text('Submission'),
        actions: [
          SubmissionStatusBadge(status: submission.status),
          const SizedBox(width: 16),
        ],
      ),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            _buildInfoCard(theme, submission),
            const SizedBox(height: 16),
            if (submission.data.isNotEmpty)
              JsonDataViewer(data: submission.data, title: 'Submission Data'),
            if (submission.metadata != null &&
                submission.metadata!.isNotEmpty) ...[
              const SizedBox(height: 16),
              JsonDataViewer(
                  data: submission.metadata!, title: 'Metadata'),
            ],
          ],
        ),
      ),
    );
  }

  Widget _buildInfoCard(ThemeData theme, FormSubmission sub) {
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
            MetadataRow(label: 'Submission ID', value: sub.id),
            MetadataRow(label: 'Form ID', value: sub.formId),
            if (sub.submitterId.isNotEmpty)
              MetadataRow(label: 'Submitter', value: sub.submitterId),
            MetadataRow(label: 'Status', value: sub.status.name),
            if (sub.fileCount > 0)
              MetadataRow(
                  label: 'Files', value: sub.fileCount.toString()),
            if (sub.idempotencyKey.isNotEmpty)
              MetadataRow(label: 'Idempotency Key', value: sub.idempotencyKey),
            if (sub.createdAt != null)
              MetadataRow(
                label: 'Created',
                value: DateFormat.yMMMd().add_jm().format(sub.createdAt!),
              ),
            if (sub.modifiedAt != null)
              MetadataRow(
                label: 'Modified',
                value: DateFormat.yMMMd().add_jm().format(sub.modifiedAt!),
              ),
          ],
        ),
      ),
    );
  }
}
