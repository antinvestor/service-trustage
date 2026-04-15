import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import 'package:antinvestor_api_runtime/antinvestor_api_runtime.dart' as rt;

import '../providers/trustage_providers.dart';
import '../widgets/event_trigger_form.dart';
import '../widgets/metric_card.dart';
import '../widgets/status_helpers.dart';
import '../widgets/trustage_panel.dart';
import '../widgets/trustage_status_badge.dart';

/// Overview dashboard showing summary metrics, event trigger, hot executions.
class CommandDeckScreen extends ConsumerWidget {
  const CommandDeckScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final asyncInstances = ref.watch(
        instanceListProvider(const InstanceQuery(limit: 50)));
    final asyncExecutions = ref.watch(
        executionListProvider(const ExecutionQuery(limit: 50)));

    return SingleChildScrollView(
      padding: const EdgeInsets.all(24),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // Header
          Text(
            'Command Deck',
            style: Theme.of(context).textTheme.headlineSmall?.copyWith(
                  fontWeight: FontWeight.w600,
                ),
          ),
          const SizedBox(height: 20),

          // Metrics row
          asyncInstances.when(
            loading: () => const Center(child: CircularProgressIndicator()),
            error: (e, _) => Text('Error loading metrics: $e'),
            data: (resp) {
              final instances = resp.items;
              final running = instances
                  .where((i) => i.status == rt.InstanceStatus.INSTANCE_STATUS_RUNNING)
                  .length;
              final waiting = instances
                  .where((i) => i.status == rt.InstanceStatus.INSTANCE_STATUS_SUSPENDED)
                  .length;

              return Wrap(
                spacing: 12,
                runSpacing: 12,
                children: [
                  SizedBox(
                    width: 160,
                    child: MetricCard(
                      label: 'Total Runs',
                      value: instances.length.toString(),
                      detail: '$running currently running',
                      icon: Icons.play_circle_outline,
                      color: const Color(0xFF0284C7),
                    ),
                  ),
                  SizedBox(
                    width: 160,
                    child: MetricCard(
                      label: 'Waiting',
                      value: waiting.toString(),
                      detail: 'timers, waits, or manual gates',
                      icon: Icons.hourglass_empty,
                      color: const Color(0xFFD97706),
                    ),
                  ),
                  asyncExecutions.when(
                    loading: () => const SizedBox(
                      width: 160,
                      child: MetricCard(
                        label: 'Needs Attention',
                        value: '...',
                        icon: Icons.error_outline,
                        color: Color(0xFFE11D48),
                      ),
                    ),
                    error: (_, _) => const SizedBox.shrink(),
                    data: (execResp) {
                      final needsAttention = execResp.items
                          .where((e) => canRetry(e.status.name) || isWaiting(e.status.name))
                          .length;
                      return SizedBox(
                        width: 160,
                        child: MetricCard(
                          label: 'Needs Attention',
                          value: needsAttention.toString(),
                          detail: 'retryable or failed executions',
                          icon: Icons.error_outline,
                          color: const Color(0xFFE11D48),
                        ),
                      );
                    },
                  ),
                ],
              );
            },
          ),

          const SizedBox(height: 24),

          // Event trigger form
          const EventTriggerForm(),

          const SizedBox(height: 24),

          // Hot executions
          asyncExecutions.when(
            loading: () => const SizedBox.shrink(),
            error: (_, _) => const SizedBox.shrink(),
            data: (execResp) {
              final hot = execResp.items
                  .where((e) => canRetry(e.status.name) || isWaiting(e.status.name))
                  .take(6)
                  .toList();

              if (hot.isEmpty) return const SizedBox.shrink();

              return TrustagePanel(
                eyebrow: 'Attention',
                title: 'Hot Executions',
                child: Column(
                  children: hot
                      .map((exec) => _HotExecutionTile(
                            execution: exec,
                            onTap: () => context.go(
                              '/trustage/runs/${exec.instanceId}',
                            ),
                          ))
                      .toList(),
                ),
              );
            },
          ),
        ],
      ),
    );
  }
}

class _HotExecutionTile extends StatelessWidget {
  const _HotExecutionTile({required this.execution, required this.onTap});
  final rt.WorkflowExecution execution;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(8),
      child: Padding(
        padding: const EdgeInsets.symmetric(vertical: 6, horizontal: 4),
        child: Row(
          children: [
            TrustageStatusBadge(status: execution.status.name),
            const SizedBox(width: 8),
            Expanded(
              child: Text(
                execution.state,
                style: theme.textTheme.bodySmall?.copyWith(
                  fontWeight: FontWeight.w600,
                ),
                overflow: TextOverflow.ellipsis,
              ),
            ),
            Text(
              shortId(execution.instanceId),
              style: theme.textTheme.labelSmall?.copyWith(
                fontFamily: 'monospace',
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
            const SizedBox(width: 4),
            Icon(Icons.chevron_right,
                size: 16, color: theme.colorScheme.onSurfaceVariant),
          ],
        ),
      ),
    );
  }
}
