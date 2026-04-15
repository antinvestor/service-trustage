import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';

import 'package:antinvestor_ui_core/widgets/metadata_row.dart';

import '../models/form_definition.dart';
import '../providers/formstore_providers.dart';
import '../widgets/json_data_viewer.dart';
import '../widgets/submission_tile.dart';

/// Detail screen for a form definition, showing metadata and submissions.
class FormDetailScreen extends ConsumerStatefulWidget {
  const FormDetailScreen({
    super.key,
    required this.definitionId,
    this.initialDefinition,
  });

  final String definitionId;
  final FormDefinition? initialDefinition;

  @override
  ConsumerState<FormDetailScreen> createState() => _FormDetailScreenState();
}

class _FormDetailScreenState extends ConsumerState<FormDetailScreen> {
  bool _isDeleting = false;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    final asyncDef = ref.watch(formDefinitionProvider(widget.definitionId));

    final definition = asyncDef.value ?? widget.initialDefinition;

    if (definition == null && asyncDef.isLoading) {
      return const Scaffold(
        body: Center(child: CircularProgressIndicator()),
      );
    }

    if (definition == null) {
      return Scaffold(
        appBar: AppBar(title: const Text('Form')),
        body: const Center(child: Text('Form definition not found.')),
      );
    }

    return Scaffold(
      appBar: AppBar(
        title: Text(definition.name),
        actions: [
          IconButton(
            icon: const Icon(Icons.edit_outlined),
            tooltip: 'Edit',
            onPressed: () =>
                context.go('/formstore/edit/${definition.id}', extra: definition),
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
            onPressed: _isDeleting ? null : () => _confirmDelete(definition),
          ),
        ],
      ),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            _buildInfoCard(theme, definition),
            const SizedBox(height: 16),
            if (definition.jsonSchema != null &&
                definition.jsonSchema!.isNotEmpty)
              JsonDataViewer(
                data: definition.jsonSchema!,
                title: 'JSON Schema',
              ),
            const SizedBox(height: 24),
            Text('Submissions', style: theme.textTheme.titleMedium),
            const SizedBox(height: 12),
            _SubmissionsList(formId: definition.id),
          ],
        ),
      ),
    );
  }

  Widget _buildInfoCard(ThemeData theme, FormDefinition def) {
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
            MetadataRow(label: 'Form ID', value: def.formId),
            MetadataRow(label: 'Name', value: def.name),
            if (def.description.isNotEmpty)
              MetadataRow(label: 'Description', value: def.description),
            MetadataRow(
                label: 'Status', value: def.active ? 'Active' : 'Inactive'),
            if (def.createdAt != null)
              MetadataRow(
                label: 'Created',
                value: DateFormat.yMMMd().add_jm().format(def.createdAt!),
              ),
            if (def.modifiedAt != null)
              MetadataRow(
                label: 'Modified',
                value: DateFormat.yMMMd().add_jm().format(def.modifiedAt!),
              ),
          ],
        ),
      ),
    );
  }

  Future<void> _confirmDelete(FormDefinition def) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Delete Form'),
        content:
            Text('Are you sure you want to delete "${def.name}"?'),
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
      await ref.read(formDefinitionNotifierProvider.notifier).delete(def.id);
      if (mounted) context.go('/formstore');
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

class _SubmissionsList extends ConsumerWidget {
  const _SubmissionsList({required this.formId});
  final String formId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final asyncSubs = ref.watch(submissionListProvider(formId));

    return asyncSubs.when(
      loading: () => const Center(
        child: Padding(
          padding: EdgeInsets.all(24),
          child: CircularProgressIndicator(),
        ),
      ),
      error: (err, _) => Center(child: Text('Error: $err')),
      data: (submissions) {
        if (submissions.isEmpty) {
          return const Center(
            child: Padding(
              padding: EdgeInsets.all(24),
              child: Text('No submissions yet.'),
            ),
          );
        }
        return Column(
          children: submissions
              .map((sub) => SubmissionTile(
                    submission: sub,
                    onTap: () => context.go(
                      '/formstore/submission/${sub.id}',
                      extra: sub,
                    ),
                  ))
              .toList(),
        );
      },
    );
  }
}
