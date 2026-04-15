import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';

import 'package:antinvestor_api_runtime/antinvestor_api_runtime.dart' as rt;

import '../providers/trustage_providers.dart';
import '../widgets/action_center.dart';
import '../widgets/blocked_state_panel.dart';
import '../widgets/execution_detail_panel.dart';
import '../widgets/execution_graph.dart';
import '../widgets/metric_card.dart';
import '../widgets/status_helpers.dart';
import '../widgets/timeline_entry_tile.dart';
import '../widgets/trustage_panel.dart';
import '../widgets/trustage_status_badge.dart';

/// The main run exploration screen with instance list, execution graph,
/// timeline, detail panel, and action center.
class RunExplorerScreen extends ConsumerStatefulWidget {
  const RunExplorerScreen({super.key, this.initialInstanceId});
  final String? initialInstanceId;

  @override
  ConsumerState<RunExplorerScreen> createState() => _RunExplorerScreenState();
}

class _RunExplorerScreenState extends ConsumerState<RunExplorerScreen> {
  String _query = '';
  String? _selectedInstanceId;
  String? _selectedExecutionId;

  @override
  void initState() {
    super.initState();
    if (widget.initialInstanceId != null) {
      _selectedInstanceId = widget.initialInstanceId;
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final asyncInstances = ref.watch(
        instanceListProvider(InstanceQuery(query: _query.isEmpty ? null : _query)));

    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        // Left: Instance list
        SizedBox(
          width: 360,
          child: Column(
            children: [
              Padding(
                padding: const EdgeInsets.fromLTRB(16, 16, 16, 8),
                child: Row(
                  children: [
                    Icon(Icons.play_circle_outline,
                        size: 24, color: theme.colorScheme.primary),
                    const SizedBox(width: 8),
                    Text('Run Explorer',
                        style: theme.textTheme.titleMedium
                            ?.copyWith(fontWeight: FontWeight.w600)),
                    const Spacer(),
                    IconButton(
                      icon: const Icon(Icons.refresh, size: 20),
                      onPressed: () {
                        ref.invalidate(instanceListProvider);
                        if (_selectedInstanceId != null) {
                          ref.invalidate(instanceRunProvider);
                          ref.invalidate(instanceTimelineProvider);
                        }
                      },
                      tooltip: 'Refresh',
                    ),
                  ],
                ),
              ),
              Padding(
                padding: const EdgeInsets.symmetric(horizontal: 16),
                child: TextField(
                  onChanged: (v) => setState(() => _query = v),
                  decoration: const InputDecoration(
                    hintText: 'Search by workflow, ID, state...',
                    prefixIcon: Icon(Icons.search, size: 20),
                    isDense: true,
                  ),
                ),
              ),
              const SizedBox(height: 8),
              Expanded(
                child: asyncInstances.when(
                  loading: () =>
                      const Center(child: CircularProgressIndicator()),
                  error: (e, _) => Center(child: Text('Error: $e')),
                  data: (resp) {
                    if (resp.items.isEmpty) {
                      return const Center(child: Text('No instances found.'));
                    }
                    return ListView.builder(
                      padding: const EdgeInsets.symmetric(horizontal: 12),
                      itemCount: resp.items.length,
                      itemBuilder: (ctx, i) {
                        final inst = resp.items[i];
                        final isSelected = inst.id == _selectedInstanceId;
                        return _InstanceListTile(
                          instance: inst,
                          isSelected: isSelected,
                          onTap: () => setState(() {
                            _selectedInstanceId = inst.id;
                            _selectedExecutionId = null;
                          }),
                        );
                      },
                    );
                  },
                ),
              ),
            ],
          ),
        ),

        // Vertical divider
        VerticalDivider(width: 1, color: theme.colorScheme.outlineVariant),

        // Center: Execution graph + timeline
        Expanded(
          child: _selectedInstanceId == null
              ? const Center(child: Text('Select an instance to explore.'))
              : _RunDetailView(
                  instanceId: _selectedInstanceId!,
                  selectedExecutionId: _selectedExecutionId,
                  onSelectExecution: (id) =>
                      setState(() => _selectedExecutionId = id),
                  onSelectInstance: (id) => setState(() {
                    _selectedInstanceId = id;
                    _selectedExecutionId = null;
                  }),
                ),
        ),

        // Right sidebar: detail panel + actions
        if (_selectedInstanceId != null)
          SizedBox(
            width: 380,
            child: _RightSidebar(
              instanceId: _selectedInstanceId!,
              selectedExecutionId: _selectedExecutionId,
            ),
          ),
      ],
    );
  }
}

class _InstanceListTile extends StatelessWidget {
  const _InstanceListTile({
    required this.instance,
    required this.isSelected,
    required this.onTap,
  });
  final rt.WorkflowInstance instance;
  final bool isSelected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final fmt = DateFormat.MMMd().add_Hm();
    final color = statusColor(instance.status.name);

    return Card(
      elevation: 0,
      color: isSelected ? color.withAlpha(15) : null,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(10),
        side: BorderSide(
          color: isSelected ? color.withAlpha(60) : theme.colorScheme.outlineVariant,
        ),
      ),
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(10),
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  Expanded(
                    child: Text(
                      instance.workflowName,
                      style: theme.textTheme.titleSmall?.copyWith(
                        fontWeight: FontWeight.w600,
                      ),
                      overflow: TextOverflow.ellipsis,
                    ),
                  ),
                  TrustageStatusBadge(status: instance.status.name),
                ],
              ),
              const SizedBox(height: 4),
              Row(
                children: [
                  Text(
                    shortId(instance.id),
                    style: theme.textTheme.labelSmall?.copyWith(
                      fontFamily: 'monospace',
                      color: theme.colorScheme.onSurfaceVariant,
                    ),
                  ),
                  const Spacer(),
                  Text(
                    instance.currentState,
                    style: theme.textTheme.labelSmall?.copyWith(
                      color: theme.colorScheme.onSurfaceVariant,
                    ),
                  ),
                  if (instance.hasStartedAt()) ...[
                    const SizedBox(width: 8),
                    Text(
                      fmt.format(instance.startedAt.toDateTime()),
                      style: theme.textTheme.labelSmall?.copyWith(
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                    ),
                  ],
                ],
              ),
              if (instance.parentInstanceId.isNotEmpty)
                Padding(
                  padding: const EdgeInsets.only(top: 2),
                  child: Row(
                    children: [
                      Icon(Icons.subdirectory_arrow_right,
                          size: 12, color: theme.colorScheme.onSurfaceVariant),
                      const SizedBox(width: 2),
                      Text(
                        'child of ${shortId(instance.parentInstanceId)}',
                        style: theme.textTheme.labelSmall?.copyWith(
                          fontSize: 10,
                          color: theme.colorScheme.primary,
                        ),
                      ),
                    ],
                  ),
                ),
            ],
          ),
        ),
      ),
    );
  }
}

class _RunDetailView extends ConsumerWidget {
  const _RunDetailView({
    required this.instanceId,
    this.selectedExecutionId,
    this.onSelectExecution,
    this.onSelectInstance,
  });
  final String instanceId;
  final String? selectedExecutionId;
  final ValueChanged<String>? onSelectExecution;
  final ValueChanged<String>? onSelectInstance;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final asyncRun = ref.watch(instanceRunProvider(instanceId));
    final asyncTimeline = ref.watch(instanceTimelineProvider(instanceId));

    return asyncRun.when(
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) => Center(child: Text('Error loading run: $e')),
      data: (run) => SingleChildScrollView(
        padding: const EdgeInsets.all(20),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            // Run metrics
            Wrap(
              spacing: 10,
              runSpacing: 10,
              children: [
                SizedBox(
                  width: 130,
                  child: MetricCard(
                    label: 'Executions',
                    value: run.executions.length.toString(),
                    color: const Color(0xFF0284C7),
                  ),
                ),
                SizedBox(
                  width: 130,
                  child: MetricCard(
                    label: 'Scope Runs',
                    value: run.scopeRuns.length.toString(),
                    color: const Color(0xFF059669),
                  ),
                ),
                SizedBox(
                  width: 130,
                  child: MetricCard(
                    label: 'Signal Waits',
                    value: run.signalWaits
                        .where((w) => isWaiting(w.status))
                        .length
                        .toString(),
                    color: const Color(0xFFD97706),
                  ),
                ),
                SizedBox(
                  width: 130,
                  child: MetricCard(
                    label: 'Signals',
                    value: run.signalMessages.length.toString(),
                    color: const Color(0xFF7C3AED),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 20),

            // Execution graph
            ExecutionGraph(
              run: run,
              selectedExecutionId: selectedExecutionId,
              onSelectExecution: onSelectExecution,
              onSelectInstance: onSelectInstance,
            ),

            const SizedBox(height: 24),

            // Causal timeline
            asyncTimeline.when(
              loading: () => const SizedBox.shrink(),
              error: (_, _) => const SizedBox.shrink(),
              data: (tl) {
                if (tl.items.isEmpty) return const SizedBox.shrink();
                final entries = tl.items.take(20).toList();
                return TrustagePanel(
                  eyebrow: 'Audit',
                  title: 'Causal Timeline',
                  subtitle: '${tl.items.length} events',
                  child: Column(
                    children: entries
                        .map((e) => TimelineEntryTile(entry: e))
                        .toList(),
                  ),
                );
              },
            ),
          ],
        ),
      ),
    );
  }
}

class _RightSidebar extends ConsumerWidget {
  const _RightSidebar({
    required this.instanceId,
    this.selectedExecutionId,
  });
  final String instanceId;
  final String? selectedExecutionId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final asyncRun = ref.watch(instanceRunProvider(instanceId));

    return asyncRun.when(
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) => Center(child: Text('Error: $e')),
      data: (run) {
        // Find selected execution
        rt.WorkflowExecution? selectedExec;
        rt.StateOutput? selectedOutput;
        if (selectedExecutionId != null) {
          try {
            selectedExec = run.executions
                .firstWhere((e) => e.id == selectedExecutionId);
            selectedOutput = run.outputs
                .where((o) => o.executionId == selectedExecutionId)
                .firstOrNull;
          } catch (_) {}
        }
        selectedExec ??= run.latestExecution;

        return SingleChildScrollView(
          padding: const EdgeInsets.all(16),
          child: Column(
            children: [
              // Execution detail
              if (selectedExec.id.isNotEmpty)
                ExecutionDetailPanel(
                  execution: selectedExec,
                  output: selectedOutput,
                ),

              const SizedBox(height: 12),

              // Blocked states
              BlockedStatePanel(
                signalWaits: run.signalWaits.toList(),
                signalMessages: run.signalMessages.toList(),
              ),

              const SizedBox(height: 12),

              // Action center
              ActionCenter(
                instance: run.instance,
                selectedExecution: selectedExec.id.isNotEmpty ? selectedExec : null,
                signalWaits: run.signalWaits.toList(),
              ),
            ],
          ),
        );
      },
    );
  }
}
