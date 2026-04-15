import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';

import 'package:antinvestor_api_runtime/antinvestor_api_runtime.dart' as rt;

import '../providers/trustage_providers.dart';
import '../widgets/status_helpers.dart';
import '../widgets/trustage_status_badge.dart';

/// Screen listing all executions with search, status filter, and retry.
class ExecutionQueueScreen extends ConsumerStatefulWidget {
  const ExecutionQueueScreen({super.key});

  @override
  ConsumerState<ExecutionQueueScreen> createState() =>
      _ExecutionQueueScreenState();
}

class _ExecutionQueueScreenState extends ConsumerState<ExecutionQueueScreen> {
  String _query = '';
  rt.ExecutionStatus? _statusFilter;
  String? _retryingId;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final asyncExecs = ref.watch(executionListProvider(
      ExecutionQuery(
        query: _query.isEmpty ? null : _query,
        status: _statusFilter,
      ),
    ));

    return Column(
      children: [
        // Header
        Padding(
          padding: const EdgeInsets.fromLTRB(24, 20, 24, 0),
          child: Row(
            children: [
              Icon(Icons.list_alt, size: 24, color: theme.colorScheme.primary),
              const SizedBox(width: 8),
              Text('Execution Queue',
                  style: theme.textTheme.titleMedium
                      ?.copyWith(fontWeight: FontWeight.w600)),
              const Spacer(),
              IconButton(
                icon: const Icon(Icons.refresh, size: 20),
                onPressed: () => ref.invalidate(executionListProvider),
                tooltip: 'Refresh',
              ),
            ],
          ),
        ),

        // Search + filter
        Padding(
          padding: const EdgeInsets.fromLTRB(24, 12, 24, 8),
          child: Row(
            children: [
              Expanded(
                child: TextField(
                  onChanged: (v) => setState(() => _query = v),
                  decoration: const InputDecoration(
                    hintText: 'Search by state, ID, trace, error...',
                    prefixIcon: Icon(Icons.search, size: 20),
                    isDense: true,
                  ),
                ),
              ),
              const SizedBox(width: 12),
              DropdownButton<rt.ExecutionStatus?>(
                value: _statusFilter,
                hint: const Text('Status'),
                items: [
                  const DropdownMenuItem(value: null, child: Text('All')),
                  ...rt.ExecutionStatus.values
                      .where((s) => s != rt.ExecutionStatus.EXECUTION_STATUS_UNSPECIFIED)
                      .map((s) => DropdownMenuItem(
                            value: s,
                            child: Text(humanizeStatus(s.name)),
                          )),
                ],
                onChanged: (v) => setState(() => _statusFilter = v),
                underline: const SizedBox.shrink(),
              ),
            ],
          ),
        ),

        // Execution list
        Expanded(
          child: asyncExecs.when(
            loading: () => const Center(child: CircularProgressIndicator()),
            error: (e, _) => Center(child: Text('Error: $e')),
            data: (resp) {
              if (resp.items.isEmpty) {
                return const Center(child: Text('No executions found.'));
              }
              return ListView.separated(
                padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 8),
                itemCount: resp.items.length,
                separatorBuilder: (_, _) => const SizedBox(height: 4),
                itemBuilder: (ctx, i) {
                  final exec = resp.items[i];
                  return _ExecutionRow(
                    execution: exec,
                    isRetrying: _retryingId == exec.id,
                    onRetry: canRetry(exec.status.name)
                        ? () => _retry(exec.id)
                        : null,
                    onTap: () => context.go('/trustage/runs/${exec.instanceId}'),
                  );
                },
              );
            },
          ),
        ),
      ],
    );
  }

  Future<void> _retry(String executionId) async {
    setState(() => _retryingId = executionId);
    try {
      await ref.read(trustageActionProvider.notifier).retryExecution(executionId);
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Retry scheduled for ${shortId(executionId)}')),
        );
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Retry failed: $e')),
        );
      }
    } finally {
      if (mounted) setState(() => _retryingId = null);
    }
  }
}

class _ExecutionRow extends StatelessWidget {
  const _ExecutionRow({
    required this.execution,
    required this.isRetrying,
    this.onRetry,
    this.onTap,
  });
  final rt.WorkflowExecution execution;
  final bool isRetrying;
  final VoidCallback? onRetry;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final fmt = DateFormat.MMMd().add_Hm();

    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(10),
        side: BorderSide(color: theme.colorScheme.outlineVariant),
      ),
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(10),
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
          child: Row(
            children: [
              Expanded(
                flex: 3,
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      execution.state,
                      style: theme.textTheme.titleSmall
                          ?.copyWith(fontWeight: FontWeight.w600),
                      overflow: TextOverflow.ellipsis,
                    ),
                    if (execution.traceId.isNotEmpty)
                      Text(
                        'trace: ${shortId(execution.traceId)}',
                        style: theme.textTheme.labelSmall?.copyWith(
                          fontFamily: 'monospace',
                          color: theme.colorScheme.onSurfaceVariant,
                        ),
                      ),
                  ],
                ),
              ),
              SizedBox(
                width: 80,
                child: Text(
                  shortId(execution.instanceId),
                  style: theme.textTheme.labelSmall?.copyWith(
                    fontFamily: 'monospace',
                    color: theme.colorScheme.onSurfaceVariant,
                  ),
                ),
              ),
              SizedBox(
                width: 40,
                child: Text(
                  '#${execution.attempt}',
                  style: theme.textTheme.labelSmall?.copyWith(
                    color: theme.colorScheme.onSurfaceVariant,
                  ),
                ),
              ),
              SizedBox(
                width: 100,
                child: execution.hasStartedAt()
                    ? Text(
                        fmt.format(execution.startedAt.toDateTime()),
                        style: theme.textTheme.labelSmall?.copyWith(
                          color: theme.colorScheme.onSurfaceVariant,
                        ),
                      )
                    : const SizedBox.shrink(),
              ),
              SizedBox(
                width: 100,
                child: TrustageStatusBadge(status: execution.status.name),
              ),
              SizedBox(
                width: 60,
                child: onRetry != null
                    ? IconButton(
                        icon: isRetrying
                            ? const SizedBox(
                                width: 16,
                                height: 16,
                                child: CircularProgressIndicator(
                                    strokeWidth: 2),
                              )
                            : const Icon(Icons.replay, size: 18),
                        onPressed: isRetrying ? null : onRetry,
                        tooltip: 'Retry',
                        visualDensity: VisualDensity.compact,
                      )
                    : const SizedBox.shrink(),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
