import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';

import 'package:antinvestor_api_workflow/antinvestor_api_workflow.dart' as wf;

import 'package:antinvestor_ui_core/widgets/entity_list_page.dart';

import '../providers/trustage_providers.dart';
import '../widgets/status_helpers.dart';
import '../widgets/trustage_status_badge.dart';

/// Screen listing workflow definitions with search.
class WorkflowCatalogScreen extends ConsumerStatefulWidget {
  const WorkflowCatalogScreen({super.key});

  @override
  ConsumerState<WorkflowCatalogScreen> createState() =>
      _WorkflowCatalogScreenState();
}

class _WorkflowCatalogScreenState extends ConsumerState<WorkflowCatalogScreen> {
  String _search = '';

  @override
  Widget build(BuildContext context) {
    final asyncWorkflows = ref.watch(workflowListProvider(
      WorkflowQuery(
        query: _search.isEmpty ? null : _search,
        status: wf.WorkflowStatus.WORKFLOW_STATUS_ACTIVE,
      ),
    ));

    return asyncWorkflows.when(
      loading: () => EntityListPage<wf.WorkflowDefinition>(
        title: 'Workflow Catalog',
        icon: Icons.schema_outlined,
        items: const [],
        isLoading: true,
        itemBuilder: (_, _) => const SizedBox.shrink(),
        searchHint: 'Search workflows...',
        onSearchChanged: (v) => setState(() => _search = v),
      ),
      error: (e, _) => EntityListPage<wf.WorkflowDefinition>(
        title: 'Workflow Catalog',
        icon: Icons.schema_outlined,
        items: const [],
        error: e.toString(),
        onRetry: () => ref.invalidate(workflowListProvider),
        itemBuilder: (_, _) => const SizedBox.shrink(),
      ),
      data: (resp) => EntityListPage<wf.WorkflowDefinition>(
        title: 'Workflow Catalog',
        icon: Icons.schema_outlined,
        items: resp.items.toList(),
        itemBuilder: (ctx, w) => _WorkflowTile(workflow: w),
        searchHint: 'Search workflows...',
        onSearchChanged: (v) => setState(() => _search = v),
      ),
    );
  }
}

class _WorkflowTile extends StatelessWidget {
  const _WorkflowTile({required this.workflow});
  final wf.WorkflowDefinition workflow;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final fmt = DateFormat.yMMMd();

    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
        side: BorderSide(color: theme.colorScheme.outlineVariant),
      ),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
        child: Row(
          children: [
            Container(
              width: 40,
              height: 40,
              decoration: BoxDecoration(
                color: theme.colorScheme.primaryContainer,
                borderRadius: BorderRadius.circular(10),
              ),
              child: Icon(Icons.schema_outlined,
                  color: theme.colorScheme.onPrimaryContainer),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    workflow.name,
                    style: theme.textTheme.titleSmall
                        ?.copyWith(fontWeight: FontWeight.w600),
                    overflow: TextOverflow.ellipsis,
                  ),
                  Text(
                    shortId(workflow.id),
                    style: theme.textTheme.labelSmall?.copyWith(
                      fontFamily: 'monospace',
                      color: theme.colorScheme.onSurfaceVariant,
                    ),
                  ),
                ],
              ),
            ),
            const SizedBox(width: 8),
            Text(
              'v${workflow.version}',
              style: theme.textTheme.labelSmall?.copyWith(
                fontWeight: FontWeight.w600,
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
            const SizedBox(width: 12),
            if (workflow.hasUpdatedAt())
              Text(
                fmt.format(workflow.updatedAt.toDateTime()),
                style: theme.textTheme.labelSmall?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant,
                ),
              ),
            const SizedBox(width: 12),
            TrustageStatusBadge(status: workflow.status.name),
          ],
        ),
      ),
    );
  }
}
